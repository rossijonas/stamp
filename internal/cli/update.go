package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

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
		case "dnf", "apt", "zypper", "pacman", "paru", "macports", "snap":
			return true
		}
	}
	return false
}

func runUpdate(ctx context.Context, w io.Writer, a manager.Adapter, pkg string, prefix string) bool {
	ctx = manager.WithOutputPrefix(ctx, prefix)
	if err := a.Update(ctx, pkg); err != nil {
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

func runCheck(ctx context.Context, adapters []manager.Adapter, pkg string, w io.Writer) []checkResult {
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
		Aliases: []string{"upgrade"},
		Short:   "Run system upgrades across all package managers",
		Example: "  stamp update\n  stamp update --check\n  stamp update -m apt\n  stamp update -p htop -m brew\n  stamp update --serial\n  stamp upgrade",
		Long: `Run system upgrade commands for each available package manager using a safe two-phase (check + confirm) flow.

By default, checks for available updates, displays them, and prompts for confirmation before upgrading.
Use --check to only run the check phase (dry-run).
Use -y to skip the check phase and auto-confirm for maximum speed.
Use -m to scope to a single package manager.
Use -p to update a single package (requires -m).
Use --serial to run updates one manager at a time (default: parallel).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			errOut := cmd.ErrOrStderr()

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// SIGINT handler: restore terminal state and cancel context
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT)
			go func() {
				<-sigCh
				signal.Stop(sigCh)
				_, _ = fmt.Fprintln(errOut)
				if stty, err := exec.LookPath("stty"); err == nil {
					//nolint:gosec // stty path comes from LookPath, not user input
					_ = exec.Command(stty, "echo").Run()
				}
				cancel()
			}()
			defer signal.Stop(sigCh)

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

				// Prompt
				if isTerminal(cmd.InOrStdin()) {
					if !promptYesNo(errOut, cmd.InOrStdin(), "Proceed with updates? [Y/n]: ", true) {
						return nil
					}
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
