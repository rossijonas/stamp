package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rossijonas/stamp/internal/manager"
)

type checkResult struct {
	Name    string
	Updates []manager.UpdateInfo
	Err     error
}

func needsSudo(adapters []manager.Adapter) bool {
	for _, a := range adapters {
		switch a.Name() {
		case "dnf", "apt", "zypper", "pacman", "paru", "macports", "snap", "npm":
			return true
		}
	}
	return false
}

func runUpdate(ctx context.Context, w io.Writer, a manager.Adapter, pkg string, prefix string) bool {
	ctx = manager.WithOutputPrefix(ctx, prefix)
	if err := a.Update(manager.WithYes(ctx), pkg); err != nil {
		_, _ = fmt.Fprintf(w, "⚠ update failed for %s: %v\n", a.Name(), err)
		return false
	}
	if pkg != "" {
		_, _ = fmt.Fprintf(w, "updated %s via %s\n", pkg, a.Name())
	} else {
		_, _ = fmt.Fprintf(w, "updated packages via %s\n", a.Name())
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

func runCheck(ctx context.Context, adapters []manager.Adapter, pkg string, w io.Writer) []checkResult {
	tty := isOutputTerminal(w)
	printProgress(w, tty, "▪", "Refreshing package metadata...")
	refreshMetadata(ctx, adapters, w)
	printProgress(w, tty, "▪", "Checking for updates...")
	var results []checkResult
	for _, a := range adapters {
		updates, err := a.CheckUpdate(ctx, pkg)
		results = append(results, checkResult{Name: a.Name(), Updates: updates, Err: err})
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
		if !runUpdate(ctx, w, a, pkg, "") {
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
			if !runUpdate(ctx, w, a, pkg, "["+a.Name()+"] ") {
				mu.Lock()
				hasErr = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return hasErr
}

func runUpdates(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkg string, serial bool) bool {
	if serial {
		return runUpdatesSerial(ctx, w, adapters, pkg)
	}
	return runUpdatesParallel(ctx, w, adapters, pkg)
}

// promptSudoPassword re-authenticates sudo when needed, caching the password
// for all subsequent sudo commands. Returns a cleanup func or nil.
func promptSudoPassword(cmd *cobra.Command, adapters []manager.Adapter, errOut io.Writer, tty bool) (cleanup func()) {
	if !needsSudo(adapters) || !isTerminal(cmd.InOrStdin()) {
		return nil
	}
	_, _ = fmt.Fprint(errOut, iconLine(tty, "▪", "sudo password:")+" ")
	//nolint:gosec // uintptr -> int conversion is safe on all target platforms
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(errOut)
	if err != nil {
		// Non-interactive environment — sudo -n will fail fast if password needed
		_, _ = fmt.Fprintf(errOut, "  warning: cannot read password in non-interactive mode: %v\n", err)
		return nil
	}
	manager.SetSudoPassword(pw)
	return manager.ClearSudoPassword
}

// runCheckPhase runs the check phase and decides whether to proceed. It
// returns true when updates should run, false when the command should stop.
func runCheckPhase(cmd *cobra.Command, adapters []manager.Adapter, packageFlag string, checkOnly bool, errOut io.Writer) (proceed bool, err error) {
	results := runCheck(cmd.Context(), adapters, packageFlag, errOut)
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
func executeUpdate(cmd *cobra.Command, app *AppContext, adapters []manager.Adapter, packageFlag string, checkOnly, serial bool) error {
	errOut := cmd.ErrOrStderr()
	tty := isOutputTerminal(errOut)
	ctx := cmd.Context()

	if cleanup := promptSudoPassword(cmd, adapters, errOut, tty); cleanup != nil {
		defer cleanup()
	}

	// Check phase (skipped when -y)
	if !app.yes {
		proceed, err := runCheckPhase(cmd, adapters, packageFlag, checkOnly, errOut)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}
	}

	// Run phase
	if runUpdates(ctx, errOut, adapters, packageFlag, serial) {
		return fmt.Errorf("one or more managers failed to update")
	}
	return nil
}

func newUpdateCmd() *cobra.Command {
	var managerFlag, packageFlag string
	var serial, checkOnly bool

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

  # run updates one manager at a time instead of parallel
  stamp update --serial

  # alias
  stamp upgrade`,
		Long: `Run system upgrade commands for each available package manager using a safe two-phase (check + confirm) flow.

By default, checks for available updates, displays them, and prompts for confirmation before upgrading.
Use --check to only run the check phase (dry-run).
Note: the check phase refreshes package metadata first, which may require sudo and network access.
Use -y to skip the check phase and auto-confirm for maximum speed.
Use -m to scope to a single package manager.
Use -p to update a single package (requires -m).
Use --serial to run updates one manager at a time (default: parallel).`,
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

			if len(adapters) == 0 {
				return catErr(ErrUnavailable, "no package managers available")
			}

			return executeUpdate(cmd, app, adapters, packageFlag, checkOnly, serial)
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to update")
	cmd.Flags().StringVarP(&packageFlag, "package", "p", "", "update a single package (requires --manager)")
	cmd.Flags().BoolVarP(&serial, "serial", "s", false, "run updates one at a time (sequential)")
	cmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "check for available updates without applying them")
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
			runCheck(cmd.Context(), app.adapters, "", cmd.ErrOrStderr())
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
			runCheck(cmd.Context(), app.adapters, "", cmd.ErrOrStderr())
			return nil
		},
	}
}
