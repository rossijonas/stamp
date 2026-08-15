package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func newReinstallCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "reinstall <package>",
		Short: "Reinstall a package and record it in the manifest",
		Example: `  # reinstall a package using the manager recorded in the manifest
  stamp reinstall htop

  # reinstall a pre-existing package from a specific manager
  stamp reinstall lazygit -m brew

  # reinstall multiple packages in one command (per-manager batch, -m required)
  stamp reinstall lazygit jq -m brew`,
		Long: `Look up the package in the manifest to find its recorded package manager,
then execute the native reinstallation command. If the package is not
tracked in the manifest, resolve the manager and track it.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			if len(args) > 1 {
				return reinstallMany(cmd, app, args, managerFlag)
			}
			pkgName := args[0]
			if err := manager.ValidatePackageName(pkgName); err != nil {
				return fmt.Errorf("invalid package name: %w", err)
			}

			// Look up in manifest
			var recordedManager string
			for _, p := range app.manifest.Packages {
				if p.Name == pkgName {
					recordedManager = p.Manager
					break
				}
			}

			var adapter manager.Adapter
			isPreExisting := recordedManager == ""

			if !isPreExisting {
				// Manifest-tracked: find adapter by recorded manager
				for _, a := range app.adapters {
					if a.Name() == recordedManager {
						adapter = a
						break
					}
				}
				if adapter == nil {
					return catErr(ErrUnavailable, "manager %q is not available on this system", recordedManager)
				}
			} else {
				// Pre-existing: resolve via 3-tier engine
				resolver := NewResolver(app.adapters, app.config)
				resolved, err := resolver.Resolve(pkgName, managerFlag)
				if err != nil {
					return fmt.Errorf("cannot resolve manager for %q: %w", pkgName, err)
				}
				adapter = resolved
			}

			// Confirmation gate: prompts unless -y is passed. Non-interactive
			// runs without -y abort (fail closed, non-zero exit).
			if err := confirmDestructive(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
				adapter, previewReinstall, "Reinstall", pkgName); err != nil {
				return handleConsent(err)
			}

			errOut := cmd.ErrOrStderr()
			tty := isOutputTerminal(errOut)
			if line := statusLine(tty, false, "reinstalling", pkgName, adapter.Name(), ""); line != "" {
				_, _ = fmt.Fprintln(errOut, line)
			}

			// Execute native reinstall
			if err := adapter.Reinstall(manager.WithYes(cmd.Context()), pkgName); err != nil {
				return fmt.Errorf("reinstall failed: %w", err)
			}

			// Save snapshots to align baseline
			restoreSaveSnapshots(cmd.Context(), cmd.ErrOrStderr(), app.adapters)

			// Add to manifest if pre-existing
			if isPreExisting {
				app.manifest.AddPackage(manifest.Package{
					Name:    pkgName,
					Manager: adapter.Name(),
					Origin:  manifest.OriginStamped,
				})
			}

			// Save manifest
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			if line := statusLine(tty, true, "reinstalled", pkgName, adapter.Name(), ""); line != "" {
				_, _ = fmt.Fprintln(errOut, line)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use (pre-existing packages only)")
	return cmd
}

// reinstallMany reinstalls multiple packages in a single native invocation.
// Requires -m so every package goes to the same manager; only managers with
// native multi-package reinstall support participate (see
// manager.BatchReinstaller — snap is excluded: reinstall is remove+install
// there with no native batch form).
func reinstallMany(cmd *cobra.Command, app *AppContext, pkgs []string, managerFlag string) error {
	if managerFlag == "" {
		return catErr(ErrUsage, "multiple packages require --manager")
	}

	adapter, err := resolveAdapterByFlag(app.adapters, managerFlag)
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		if err := manager.ValidatePackageForManager(adapter.Name(), p); err != nil {
			return fmt.Errorf("invalid package name %q: %w", p, err)
		}
	}

	// A batch is per-manager: if any package is already tracked under a
	// different manager, fail fast before any confirmation or execution. The
	// single-package path honors the recorded manager; a batch cannot.
	for _, p := range pkgs {
		for _, rec := range app.manifest.Packages {
			if rec.Name == p && rec.Manager != adapter.Name() {
				return catErr(ErrUsage, "package %s is tracked under %s, not %s; reinstall it with -m %s", p, rec.Manager, adapter.Name(), rec.Manager)
			}
		}
	}

	br, ok := adapter.(manager.BatchReinstaller)
	if !ok {
		return catErr(ErrUnavailable, "manager %s does not support reinstalling multiple packages at once", adapter.Name())
	}

	// Brew: --cask is batch-wide; a mixed cask/formula batch falls back to
	// per-package single reinstalls.
	casks := brewCasks(cmd.Context(), adapter, pkgs)
	caskCount := 0
	for _, isCask := range casks {
		if isCask {
			caskCount++
		}
	}
	mixed := caskCount > 0 && caskCount < len(pkgs)

	reinstallCtx := cmd.Context()
	if caskCount == len(pkgs) {
		reinstallCtx = manager.WithCask(cmd.Context())
	}

	if err := confirmDestructiveMany(reinstallCtx, cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
		adapter, previewReinstall, "Reinstall", pkgs); err != nil {
		return handleConsent(err)
	}

	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	target := fmt.Sprintf("%d package(s)", len(pkgs))
	if line := statusLine(tty, false, "reinstalling", target, adapter.Name(), ""); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}

	if mixed {
		for _, p := range pkgs {
			ctx := manager.WithYes(cmd.Context())
			if casks[p] {
				ctx = manager.WithYes(manager.WithCask(cmd.Context()))
			}
			if err := adapter.Reinstall(ctx, p); err != nil {
				return fmt.Errorf("reinstall failed: %w", err)
			}
		}
	} else {
		if err := br.ReinstallMany(manager.WithYes(reinstallCtx), pkgs...); err != nil {
			return fmt.Errorf("reinstall failed: %w", err)
		}
	}

	// Align snapshots with the current system state, mirroring the single
	// reinstall path.
	restoreSaveSnapshots(cmd.Context(), cmd.ErrOrStderr(), app.adapters)

	// Track any packages not already in the manifest.
	for _, p := range pkgs {
		if !app.manifest.HasPackage(p, adapter.Name()) {
			app.manifest.AddPackage(manifest.Package{
				Name:    p,
				Manager: adapter.Name(),
				Origin:  manifest.OriginStamped,
			})
		}
	}
	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	if line := statusLine(tty, true, "reinstalled", target, adapter.Name(), ""); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}
	return nil
}
