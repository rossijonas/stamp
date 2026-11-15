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
func resolveReinstallAdapter(ctx context.Context, app *AppContext, pkgName, managerFlag string) (adapter manager.Adapter, canonical string, err error) {
	// Tracked packages always use their recorded manager; -m is for
	// pre-existing packages only. Falling back to -m here would silently move
	// manifest intent to another manager.
	var recordedManager string
	for _, p := range app.manifest.Packages {
		if p.Name == pkgName || p.Name == manager.TrackedName(p.Manager, pkgName) {
			recordedManager = p.Manager
			canonical = p.Name
			break
		}
	}

	if recordedManager == "" && managerFlag == "" {
		recordedManager, canonical = resolveTrackedManager(ctx, app, pkgName)
	}

	if recordedManager != "" {
		for _, a := range app.adapters {
			if a.Name() == recordedManager {
				return a, canonical, nil
			}
		}
		return nil, "", catErr(ErrUnavailable, "manager %q is not available on this system", recordedManager)
	}

	resolver := NewResolver(app.adapters, app.config)
	resolved, err := resolver.Resolve(pkgName, managerFlag)
	if err != nil {
		return nil, "", fmt.Errorf("cannot resolve manager for %q: %w", pkgName, err)
	}
	return resolved, "", nil
}

// resolveTrackedManager asks resolver-capable adapters for a canonical installed
// name and returns the manager and manifest identity whose entry matches it.
func resolveTrackedManager(ctx context.Context, app *AppContext, pkgName string) (managerName, canonical string) {
	for _, a := range app.adapters {
		resolver, ok := a.(manager.InstalledNameResolver)
		if !ok {
			continue
		}
		resolved, err := resolver.ResolveInstalledName(ctx, pkgName)
		if err != nil {
			continue
		}
		if _, _, found := findTrackedAdapter(app.manifest.Packages, []manager.Adapter{a}, resolved); found {
			return a.Name(), resolved
		}
	}
	return "", ""
}

// executeSingleReinstall runs the native reinstall, refreshes snapshots, and
// records the package (or its note) in the manifest.
func executeSingleReinstall(cmd *cobra.Command, app *AppContext, adapter manager.Adapter, pkgName, nativeName, note string, isPreExisting bool) error {
	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	if line := statusLine(tty, false, "reinstalling", pkgName, adapter.Name(), ""); line != "" {
		_, _ = fmt.Fprintln(errOut, line)
	}

	if err := adapter.Reinstall(manager.WithYes(cmd.Context()), nativeName); err != nil {
		return fmt.Errorf("reinstall failed: %w", err)
	}
	if isPreExisting && nativeName != manager.TrackedName(adapter.Name(), pkgName) {
		_, _ = fmt.Fprintf(errOut, "  note: %s is provided by %s; tracking %s\n", pkgName, nativeName, nativeName)
	}

	// Save snapshots to align baseline
	restoreSaveSnapshots(cmd.Context(), errOut, app.adapters)

	// Add to manifest if pre-existing
	trackedName := nativeName
	if isPreExisting {
		app.manifest.AddPackage(manifest.Package{
			Name:    trackedName,
			Manager: adapter.Name(),
			Notes:   note,
			Origin:  manifest.OriginStamped,
		})
	} else if note != "" {
		// Tracked package: refresh its note to capture reinstall intent.
		app.manifest.SetNote(trackedName, adapter.Name(), note)
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

  # reinstall a tap-qualified Homebrew formula
  stamp reinstall nklmilojevic/sofka/sofka

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

			adapter, canonical, err := resolveReinstallAdapter(cmd.Context(), app, pkgName, managerFlag)
			if err != nil {
				return err
			}
			if err := manager.ValidatePackageForManager(adapter.Name(), pkgName); err != nil {
				return err
			}
			nativeName := canonical
			if nativeName == "" {
				nativeName = maintenanceName(cmd.Context(), app, adapter, pkgName)
			}
			isPreExisting := !hasRecordedManager(app.manifest.Packages, nativeName)

			// Confirmation gate: prompts unless -y is passed. Non-interactive
			// runs without -y abort (fail closed, non-zero exit).
			if err := confirmDestructive(cmd.Context(), cmd.ErrOrStderr(), cmd.InOrStdin(), app.yes,
				adapter, previewReinstall, "Reinstall", pkgName); err != nil {
				return handleConsent(err)
			}

			return executeSingleReinstall(cmd, app, adapter, pkgName, nativeName, note, isPreExisting)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "manager for pre-existing packages (tracked packages use their recorded manager)")
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
	pkgs = maintenanceNames(cmd.Context(), app, adapter, pkgs)
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
		tracked := manager.TrackedName(adapter.Name(), p)
		for _, rec := range packages {
			if rec.Name == tracked && rec.Manager != adapter.Name() {
				return catErr(ErrUsage, "package %s is tracked under %s, not %s; reinstall it with -m %s", tracked, rec.Manager, adapter.Name(), rec.Manager)
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
		tracked := manager.TrackedName(mgr, p)
		if !m.HasPackage(tracked, mgr) {
			m.AddPackage(manifest.Package{
				Name:    tracked,
				Manager: mgr,
				Notes:   note,
				Origin:  manifest.OriginStamped,
			})
		} else if note != "" {
			m.SetNote(tracked, mgr, note)
		}
	}
}
