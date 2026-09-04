package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

// hasRecordedManager reports whether the package is tracked in the manifest.
func hasRecordedManager(packages []manifest.Package, pkgName string) bool {
	for _, p := range packages {
		if p.Name == pkgName {
			return true
		}
	}
	return false
}

// resolveReinstallAdapter finds the adapter for a single reinstall. Packages
// tracked in the manifest use their recorded manager; pre-existing packages
// go through the 3-tier resolver.
func resolveReinstallAdapter(app *AppContext, pkgName, managerFlag string) (adapter manager.Adapter, err error) {
	var recordedManager string
	for _, p := range app.manifest.Packages {
		if p.Name == pkgName {
			recordedManager = p.Manager
			break
		}
	}

	if recordedManager != "" {
		for _, a := range app.adapters {
			if a.Name() == recordedManager {
				return a, nil
			}
		}
		return nil, catErr(ErrUnavailable, "manager %q is not available on this system", recordedManager)
	}

	resolver := NewResolver(app.adapters, app.config)
	resolved, err := resolver.Resolve(pkgName, managerFlag)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve manager for %q: %w", pkgName, err)
	}
	return resolved, nil
}

// executeSingleReinstall runs the native reinstall, refreshes snapshots, and
// records the package (or its note) in the manifest.
func executeSingleReinstall(cmd *cobra.Command, app *AppContext, adapter manager.Adapter, pkgName, note string, isPreExisting bool) error {
	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	if line := statusLine(tty, false, "reinstalling", pkgName, adapter.Name(), ""); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}

	if err := adapter.Reinstall(manager.WithYes(cmd.Context()), pkgName); err != nil {
		return fmt.Errorf("reinstall failed: %w", err)
	}

	// Save snapshots to align baseline
	restoreSaveSnapshots(cmd.Context(), errOut, app.adapters)

	// Add to manifest if pre-existing
	if isPreExisting {
		app.manifest.AddPackage(manifest.Package{
			Name:    pkgName,
			Manager: adapter.Name(),
			Notes:   note,
			Origin:  manifest.OriginStamped,
		})
	} else if note != "" {
		// Tracked package: refresh its note to capture reinstall intent.
		app.manifest.SetNote(pkgName, adapter.Name(), note)
	}

	// Save manifest
	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	if line := statusLine(tty, true, "reinstalled", pkgName, adapter.Name(), note); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}
	return nil
}

func newReinstallCmd() *cobra.Command {
	var managerFlag string
	var note string

	cmd := &cobra.Command{
		Use:   "reinstall <package>",
		Short: "Reinstall a package and record it in the manifest",
		Example: `  # reinstall a package using the manager recorded in the manifest
  stamp reinstall htop

  # reinstall a pre-existing package from a specific manager
  stamp reinstall lazygit -m brew

  # reinstall multiple packages in one command (per-manager batch, -m required)
  stamp reinstall lazygit jq -m brew

  # annotate a reinstall so you remember why later
  stamp reinstall lazygit -m brew --note "refresh intent"`,
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
				return reinstallMany(cmd, app, args, managerFlag, note)
			}
			pkgName := args[0]
			if err := manager.ValidatePackageName(pkgName); err != nil {
				return fmt.Errorf("invalid package name: %w", err)
			}

			adapter, err := resolveReinstallAdapter(app, pkgName, managerFlag)
			if err != nil {
				return err
			}
			isPreExisting := !hasRecordedManager(app.manifest.Packages, pkgName)

			// Confirmation gate: prompts unless -y is passed. Non-interactive
			// runs without -y abort (fail closed, non-zero exit).
			if err := confirmDestructive(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
				adapter, previewReinstall, "Reinstall", pkgName); err != nil {
				return handleConsent(err)
			}

			return executeSingleReinstall(cmd, app, adapter, pkgName, note, isPreExisting)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use (pre-existing packages only)")
	cmd.Flags().StringVarP(&note, "note", "n", "", "annotation for this package")
	return cmd
}

// reinstallMany reinstalls multiple packages in a single native invocation.
// Requires -m so every package goes to the same manager; only managers with
// native multi-package reinstall support participate (see
// manager.BatchReinstaller — snap is excluded: reinstall is remove+install
// there with no native batch form).
func reinstallMany(cmd *cobra.Command, app *AppContext, pkgs []string, managerFlag, note string) error {
	if managerFlag == "" {
		return catErr(ErrUsage, "multiple packages require --manager")
	}

	adapter, err := resolveAdapterByFlag(app.adapters, managerFlag)
	if err != nil {
		return err
	}
	if err := validateBatchPackages(adapter, pkgs); err != nil {
		return err
	}
	if err := validateBatchReinstall(adapter, pkgs, app.manifest.Packages); err != nil {
		return err
	}

	br, ok := adapter.(manager.BatchReinstaller)
	if !ok {
		return catErr(ErrUnavailable, "manager %s does not support reinstalling multiple packages at once", adapter.Name())
	}

	// Brew: --cask is batch-wide; a mixed cask/formula batch falls back to
	// per-package single reinstalls.
	casks := brewCasks(cmd.Context(), adapter, pkgs)
	caskCount := countCasks(casks)

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

	if err := execBatchReinstall(reinstallCtx, adapter, br, caskCount, casks, pkgs); err != nil {
		return err
	}

	// Align snapshots with the current system state, mirroring the single
	// reinstall path.
	restoreSaveSnapshots(cmd.Context(), cmd.ErrOrStderr(), app.adapters)

	trackBatchReinstalls(app.manifest, pkgs, adapter.Name(), note)
	if err := app.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	if line := statusLine(tty, true, "reinstalled", target, adapter.Name(), note); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}
	return nil
}

// validateBatchReinstall fails fast when any batch package is tracked under a
// different manager. A batch is per-manager, so a conflicting record cannot be
// honored the way the single-package path does.
func validateBatchReinstall(adapter manager.Adapter, pkgs []string, packages []manifest.Package) error {
	for _, p := range pkgs {
		for _, rec := range packages {
			if rec.Name == p && rec.Manager != adapter.Name() {
				return catErr(ErrUsage, "package %s is tracked under %s, not %s; reinstall it with -m %s", p, rec.Manager, adapter.Name(), rec.Manager)
			}
		}
	}
	return nil
}

// execBatchReinstall runs the native reinstall. A mixed cask/formula batch
// falls back to per-package single reinstalls (cask stacking per package); a
// uniform batch uses one native invocation.
func execBatchReinstall(ctx context.Context, adapter manager.Adapter, br manager.BatchReinstaller, caskCount int, casks map[string]bool, pkgs []string) error {
	if mixed := caskCount > 0 && caskCount < len(pkgs); mixed {
		for _, p := range pkgs {
			pkgCtx := manager.WithYes(ctx)
			if casks[p] {
				pkgCtx = manager.WithYes(manager.WithCask(ctx))
			}
			if err := adapter.Reinstall(pkgCtx, p); err != nil {
				return fmt.Errorf("reinstall failed: %w", err)
			}
		}
		return nil
	}
	if err := br.ReinstallMany(manager.WithYes(ctx), pkgs...); err != nil {
		return fmt.Errorf("reinstall failed: %w", err)
	}
	return nil
}

// trackBatchReinstalls records new packages in the manifest and refreshes the
// notes of already-tracked ones when a note was given.
func trackBatchReinstalls(m *manifest.Manifest, pkgs []string, mgr, note string) {
	for _, p := range pkgs {
		if !m.HasPackage(p, mgr) {
			m.AddPackage(manifest.Package{
				Name:    p,
				Manager: mgr,
				Notes:   note,
				Origin:  manifest.OriginStamped,
			})
		} else if note != "" {
			m.SetNote(p, mgr, note)
		}
	}
}
