package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

type checkResult struct {
	Name    string
	Updates []manager.UpdateInfo
	Err     error
}

func runUpdate(ctx context.Context, w io.Writer, a manager.Adapter, pkg string, prefix string, userPhase bool) bool {
	ctx = manager.WithOutputPrefix(ctx, prefix)
	if userPhase {
		// Parallel user-phase children must never read the terminal: a prompt
		// from one manager would consume input intended for another.
		ctx = manager.WithNoStdin(ctx)
	}

	var err error
	phaser, phased := a.(manager.UpdatePhaser)
	switch {
	case userPhase && phased:
		err = phaser.UpdateUser(manager.WithYes(ctx), pkg)
	case phased:
		err = phaser.UpdateSystem(manager.WithYes(ctx), pkg)
	default:
		err = a.Update(manager.WithYes(ctx), pkg)
	}
	if err != nil {
		_, _ = fmt.Fprintf(w, "⚠ update failed for %s: %v\n", a.Name(), err)
		return false
	}

	scope := ""
	if phased {
		if userPhase {
			scope = " user"
		} else {
			scope = " system"
		}
	}
	if pkg != "" {
		_, _ = fmt.Fprintf(w, "updated %s via %s%s\n", pkg, a.Name(), scope)
	} else {
		_, _ = fmt.Fprintf(w, "updated packages via %s%s\n", a.Name(), scope)
	}
	return true
}

// refreshMetadata calls Refresh on each adapter before checking for updates.
func refreshMetadata(ctx context.Context, adapters []manager.Adapter, w io.Writer) {
	for _, a := range adapters {
		_, _ = fmt.Fprintf(w, "  refreshing %s...\n", a.Name())
		if err := a.Refresh(ctx); err != nil {
			_, _ = fmt.Fprintf(w, "  ⚠ %s: refresh failed: %v\n", a.Name(), err)
		}
	}
}

func runCheck(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkg string, stdin io.Reader, flatpakUserOnly bool) []checkResult {
	tty := isOutputTerminal(w)
	printProgress(w, tty, "▪", "Refreshing package metadata...")
	refreshMetadata(ctx, adapters, w)
	printProgress(w, tty, "▪", "Checking for updates...")
	results := make([]checkResult, 0, len(adapters))
	warnedPolkit := false
	if flatpakUserOnly {
		noteSystemSkipped(w, adapters)
	}

	for _, a := range adapters {
		phaser, phased := a.(manager.UpdatePhaser)
		if !phased {
			updates, err := a.CheckUpdate(ctx, pkg)
			results = append(results, checkResult{Name: a.Name(), Updates: updates, Err: err})
			continue
		}

		updates, err := phaser.CheckUser(ctx, pkg)
		results = append(results, checkResult{Name: a.Name(), Updates: updates, Err: err})

		if flatpakUserOnly {
			continue
		}
		hasSystemRefs, refErr := phaser.HasSystemRefs(ctx, pkg)
		if refErr != nil {
			_, _ = fmt.Fprintf(w, "  ⚠ %s: could not check system scope: %v\n", a.Name(), refErr)
			continue
		}
		if !hasSystemRefs {
			continue
		}
		if !isTerminal(stdin) {
			_, _ = fmt.Fprintf(w, "  %s: system check skipped (authentication required)\n", a.Name())
			continue
		}
		if !warnedPolkit && needsPolkitAuth(ctx) {
			warnedPolkit = true
			_, _ = fmt.Fprintf(w, "  warning: polkit authentication may be required for %s system checks; without an agent the prompt appears in this terminal\n", a.Name())
		}
		_, _ = fmt.Fprintf(w, "▪ %s system check — authentication may be required (polkit)\n", a.Name())
		sysUpdates, sysErr := phaser.CheckSystem(ctx, pkg)
		results = append(results, checkResult{Name: a.Name() + " (system)", Updates: sysUpdates, Err: sysErr})
	}

	renderCheckResults(w, results)
	return results
}

// renderCheckResults prints the per-manager check outcome and reports whether
// any manager has updates or cannot preview them.
func renderCheckResults(w io.Writer, results []checkResult) bool {
	hasUpdates := false
	for _, r := range results {
		switch {
		case errors.Is(r.Err, manager.ErrCheckUnsupported):
			_, _ = fmt.Fprintf(w, "  %s: cannot preview updates\n", r.Name)
			hasUpdates = true
		case r.Err != nil:
			_, _ = fmt.Fprintf(w, "  ⚠ %s: check failed: %v\n", r.Name, r.Err)
		case len(r.Updates) == 0:
			_, _ = fmt.Fprintf(w, "  %s: No updates available\n", r.Name)
		default:
			hasUpdates = true
			for _, u := range r.Updates {
				if u.CurrentVersion != "" && u.AvailableVersion != "" {
					_, _ = fmt.Fprintf(w, "  %s: %s %s → %s\n", r.Name, u.Package, u.CurrentVersion, u.AvailableVersion)
				} else {
					_, _ = fmt.Fprintf(w, "  %s: %s\n", r.Name, u.Package)
				}
			}
		}
	}
	if !hasUpdates {
		_, _ = fmt.Fprintln(w, "  No updates available")
	}
	return hasUpdates
}

func runUpdatesSerial(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkg string) bool {
	tty := isOutputTerminal(w)
	var hasErr bool
	for _, a := range adapters {
		printProgress(w, tty, "▪", fmt.Sprintf("updating via %s...", a.Name()))
		if !runUpdate(ctx, w, a, pkg, "", true) {
			hasErr = true
		}
	}
	return hasErr
}

func runUpdatesParallel(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkg string) bool {
	var hasErr bool
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, a := range adapters {
		a := a
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !runUpdate(ctx, w, a, pkg, "["+a.Name()+"] ", true) {
				mu.Lock()
				hasErr = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return hasErr
}

// runUpdates runs the prompt-free user phase, then serializes any authenticated
// system phases behind a banner so prompts never interleave with other output.
func runUpdates(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkg string, serial, flatpakUserOnly bool, stdin io.Reader) (bool, error) {
	hasErr := false
	if serial {
		hasErr = runUpdatesSerial(ctx, w, adapters, pkg)
	} else {
		hasErr = runUpdatesParallel(ctx, w, adapters, pkg)
	}

	if flatpakUserOnly {
		noteSystemSkipped(w, adapters)
		return hasErr, nil
	}

	systemErr, err := runSystemPhases(ctx, w, adapters, pkg, stdin)
	return hasErr || systemErr, err
}

// noteSystemSkipped tells the operator that system-scope work was requested
// off, so a scripted run does not look like a full update.
func noteSystemSkipped(w io.Writer, adapters []manager.Adapter) {
	for _, a := range adapters {
		if _, ok := a.(manager.UpdatePhaser); ok {
			_, _ = fmt.Fprintf(w, "  %s: system scope skipped (--flatpak-user-only)\n", a.Name())
			return
		}
	}
}

// runSystemPhases runs authenticated phases serially, after every prompt-free
// update has finished, so their prompts cannot interleave with other output.
func runSystemPhases(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkg string, stdin io.Reader) (bool, error) {
	hasErr := false
	warnedPolkit := false

	for _, a := range adapters {
		phaser, ok := a.(manager.UpdatePhaser)
		if !ok {
			continue
		}

		hasSystemRefs, err := phaser.HasSystemRefs(ctx, pkg)
		if err != nil {
			_, _ = fmt.Fprintf(w, "  ⚠ %s: could not check system scope: %v\n", a.Name(), err)
			continue
		}
		if !hasSystemRefs {
			continue
		}

		if !isTerminal(stdin) {
			return hasErr, fmt.Errorf("%s system updates need a terminal for authentication; rerun with --flatpak-user-only", a.Name())
		}

		if !warnedPolkit && needsPolkitAuth(ctx) {
			warnedPolkit = true
			_, _ = fmt.Fprintf(w, "  warning: polkit authentication may be required for %s system updates; without an agent the prompt appears in this terminal\n", a.Name())
		}

		_, _ = fmt.Fprintf(w, "▪ %s system update — authentication may be required (polkit)\n", a.Name())
		if !runUpdate(ctx, w, a, pkg, "", false) {
			hasErr = true
		}
	}

	return hasErr, nil
}

// runCheckPhase runs the check phase and decides whether to proceed. It
// returns true when updates should run, false when the command should stop.
func runCheckPhase(cmd *cobra.Command, adapters []manager.Adapter, packageFlag string, checkOnly, flatpakUserOnly bool, errOut io.Writer) (proceed bool, err error) {
	results := runCheck(cmd.Context(), errOut, adapters, packageFlag, cmd.InOrStdin(), flatpakUserOnly)
	if checkOnly {
		return false, nil
	}

	hasUpdates := false
	hasUnsupported := false
	checkFailed := false
	for _, r := range results {
		if len(r.Updates) > 0 {
			hasUpdates = true
		}
		if errors.Is(r.Err, manager.ErrCheckUnsupported) {
			hasUnsupported = true
		}
		if r.Err != nil && !errors.Is(r.Err, manager.ErrCheckUnsupported) {
			checkFailed = true
		}
	}

	if !hasUpdates && !hasUnsupported && !checkFailed {
		tty := isOutputTerminal(errOut)
		_, _ = fmt.Fprintf(errOut, "  %s\n", iconLine(tty, "✓", "All up to date"))
		return false, nil
	}

	// Prompt (fail closed: non-interactive without -y exits
	// non-zero; an interrupted run aborts cleanly).
	if cmd.Context().Err() != nil {
		_, _ = fmt.Fprintln(errOut, "aborted")
		return false, nil
	}
	if err := consentPrompt(cmd.Context(), errOut, cmd.InOrStdin(), "Proceed with updates? [y/N]: "); err != nil {
		return false, handleConsent(err)
	}
	return true, nil
}

// validateUpdateArgs checks the --check/--yes and --package/--manager flag
// combinations and validates the package name against the manager.
func validateUpdateArgs(app *AppContext, checkOnly bool, managerFlag, packageFlag string, adapters []manager.Adapter) error {
	if checkOnly && app.yes {
		return fmt.Errorf("--check and --yes are mutually exclusive")
	}
	if packageFlag != "" {
		if managerFlag == "" {
			return catErr(ErrUsage, "specify --manager to update a specific package")
		}
		if err := manager.ValidatePackageForManager(adapters[0].Name(), packageFlag); err != nil {
			return err
		}
	}
	return nil
}

// executeUpdate runs the check phase (unless -y) and then the update phase.
func executeUpdate(cmd *cobra.Command, app *AppContext, adapters []manager.Adapter, packageFlag string, checkOnly, serial, flatpakUserOnly bool) error {
	errOut := cmd.ErrOrStderr()
	ctx := cmd.Context()

	// Authenticate before the metadata refresh; the check phase may itself run sudo.
	sudoPreflight(cmd, adapters, errOut)

	// Check phase (skipped when -y)
	if !app.yes {
		proceed, err := runCheckPhase(cmd, adapters, packageFlag, checkOnly, flatpakUserOnly, errOut)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}
	}

	// Re-validate right before the run phase. The returned flag forces serial
	// execution when sudo cannot cache credentials, so prompts never race.
	parallelOK := sudoPreflight(cmd, adapters, errOut)

	// Interrupted (SIGINT) during auth: abort cleanly rather than start the run.
	if ctx.Err() != nil {
		_, _ = fmt.Fprintln(errOut, "aborted")
		return nil
	}

	// Run phase
	hasErr, err := runUpdates(ctx, errOut, adapters, packageFlag, serial || !parallelOK, flatpakUserOnly, cmd.InOrStdin())
	if err != nil {
		return err
	}
	if hasErr {
		return fmt.Errorf("one or more managers failed to update")
	}
	return nil
}

func newUpdateCmd() *cobra.Command {
	var managerFlag, packageFlag string
	var serial, checkOnly, flatpakUserOnly bool

	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"upgrade", "refresh"},
		Short:   "Run system upgrades across all package managers",
		Example: `  # default two-phase flow (check + confirm, then update)
  stamp update

  # dry-run: check for updates without applying them
  # (refreshes package metadata first — may require sudo and network access)
  stamp update --check

  # skip check phase, auto-confirm (useful in scripts)
  stamp update -y

  # update a specific package (requires --manager)
  stamp update -p htop -m brew

  # update a tap-qualified Homebrew formula
  stamp update -p nklmilojevic/sofka/sofka -m brew

  # run updates one manager at a time instead of parallel
  stamp update --serial

  # skip system-scope flatpak updates (avoids polkit prompts)
  stamp update -y --flatpak-user-only

  # update a versioned apt package (tracked as htop)
  stamp update -p htop=3.2.1 -m apt

  # alias
  stamp upgrade`,
		Long: `Run system upgrade commands for each available package manager using a safe two-phase (check + confirm) flow.

By default, checks for available updates, displays them, and prompts for confirmation before upgrading.
Use --check to only run the check phase (dry-run).
Note: the check phase refreshes package metadata first, which may require sudo and network access.
Use -y to skip the check phase and auto-confirm for maximum speed.
Use -m to scope to a single package manager.
Use -p to update a single package (requires -m).
Use --serial to run updates one manager at a time (default: parallel).
Use --flatpak-user-only to skip system-scope flatpak updates and system checks,
which avoids polkit authentication prompts on headless or scripted hosts.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			adapters, err := resolveManagerTarget(app.adapters, managerFlag)
			if err != nil {
				return err
			}

			if err := validateUpdateArgs(app, checkOnly, managerFlag, packageFlag, adapters); err != nil {
				return err
			}

			if packageFlag != "" {
				packageFlag = maintenanceName(cmd.Context(), app, adapters[0], packageFlag)
			}

			if len(adapters) == 0 {
				return catErr(ErrUnavailable, "no package managers available")
			}

			return executeUpdate(cmd, app, adapters, packageFlag, checkOnly, serial, flatpakUserOnly)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to update")
	cmd.Flags().StringVarP(&packageFlag, "package", "p", "", "update a single package (requires --manager)")
	cmd.Flags().BoolVarP(&serial, "serial", "s", false, "run updates one at a time (sequential)")
	cmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "check for available updates without applying them")
	cmd.Flags().BoolVar(&flatpakUserOnly, "flatpak-user-only", false, "skip system-scope flatpak updates (avoids polkit prompts)")
	return cmd
}

func newOutdatedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "outdated",
		Short: "Check for available updates (alias for stamp update --check)",
		Long: `Check across all package managers for outdated packages.
Alias for "stamp update --check".`,
		Example: `  # check for outdated packages (alias form)
  stamp outdated

  # equivalent canonical command
  stamp update --check`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			runCheck(cmd.Context(), cmd.ErrOrStderr(), app.adapters, "", cmd.InOrStdin(), false)
			return nil
		},
	}
}

func newCheckUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-update",
		Short: "Check for available updates (alias for stamp update --check)",
		Long: `Check across all package managers for available updates.
Alias for "stamp update --check".`,
		Example: `  # check for available updates (alias form)
  stamp check-update

  # equivalent canonical command
  stamp update --check`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			runCheck(cmd.Context(), cmd.ErrOrStderr(), app.adapters, "", cmd.InOrStdin(), false)
			return nil
		},
	}
}
