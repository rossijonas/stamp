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
	_, _ = fmt.Fprintf(w, "▪ Refreshing package metadata...\n")
	refreshMetadata(ctx, adapters, w)
	_, _ = fmt.Fprintf(w, "▪ Checking for updates...\n")
	var results []checkResult
	for _, a := range adapters {
		updates, err := a.CheckUpdate(ctx, pkg)
		results = append(results, checkResult{Name: a.Name(), Updates: updates, Err: err})
	}

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
	return results
}

func runUpdates(ctx context.Context, w io.Writer, adapters []manager.Adapter, pkg string, serial bool) bool {
	var hasErr bool
	var mu sync.Mutex
	if serial {
		for _, a := range adapters {
			_, _ = fmt.Fprintf(w, "▪ updating via %s...\n", a.Name())
			if !runUpdate(ctx, w, a, pkg, "") {
				hasErr = true
			}
		}
	} else {
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
	}
	return hasErr
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
			errOut := cmd.ErrOrStderr()
			ctx := cmd.Context()

			if checkOnly && app.yes {
				return fmt.Errorf("--check and --yes are mutually exclusive")
			}

			adapters := app.adapters
			if managerFlag != "" {
				resolved := manager.ResolveManager(managerFlag)
				var found bool
				for _, a := range adapters {
					if a.Name() == resolved {
						adapters = []manager.Adapter{a}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("manager %q not available", managerFlag)
				}
			}

			if packageFlag != "" {
				if managerFlag == "" {
					return fmt.Errorf("specify --manager to update a specific package")
				}
				if err := manager.ValidatePackageForManager(adapters[0].Name(), packageFlag); err != nil {
					return err
				}
			}

			if len(adapters) == 0 {
				return fmt.Errorf("no package managers available")
			}

			// Pre-run: sudo re-auth (caches password for all sudo commands via sudo -S)
			if needsSudo(adapters) && isTerminal(cmd.InOrStdin()) {
				_, _ = fmt.Fprint(errOut, "▪ sudo password: ")
				//nolint:gosec // uintptr -> int conversion is safe on all target platforms
				pw, err := term.ReadPassword(int(os.Stdin.Fd()))
				_, _ = fmt.Fprintln(errOut)
				if err != nil {
					// Non-interactive environment — sudo -n will fail fast if password needed
					_, _ = fmt.Fprintf(errOut, "  warning: cannot read password in non-interactive mode: %v\n", err)
				} else {
					manager.SetSudoPassword(pw)
					defer manager.ClearSudoPassword()
				}
			}

			// Check phase (skipped when -y)
			if !app.yes {
				results := runCheck(ctx, adapters, packageFlag, errOut)

				if checkOnly {
					return nil
				}

				// Decide whether to run
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

				shouldRun := hasUpdates || hasUnsupported

				if !shouldRun && !checkFailed {
					_, _ = fmt.Fprintf(errOut, "  ✓ All up to date\n")
					return nil
				}

				// Prompt (fail closed: non-interactive without -y aborts; an
				// interrupted run aborts too).
				if ctx.Err() != nil {
					_, _ = fmt.Fprintln(errOut, "aborted")
					return nil
				}
				if !promptYesNo(ctx, errOut, cmd.InOrStdin(), "Proceed with updates? [y/N]: ", false) {
					_, _ = fmt.Fprintln(errOut, "aborted")
					return nil
				}
			}

			// Run phase
			if runUpdates(ctx, errOut, adapters, packageFlag, serial) {
				return fmt.Errorf("one or more managers failed to update")
			}
			return nil
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
