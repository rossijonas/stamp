package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

type checkResult struct {
	Name    string
	Updates []manager.UpdateInfo
	Err     error
}

func needsSudo(adapters []manager.Adapter) bool {
	for _, a := range adapters {
		if a.Name() == "dnf" {
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

			// Check phase (skipped when -y)
			if !app.yes {
				results := runCheck(cmd.Context(), adapters, packageFlag, errOut)

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

			// Pre-run: sudo re-auth
			if needsSudo(adapters) && isTerminal(cmd.InOrStdin()) {
				_, _ = fmt.Fprintln(errOut, "▪ Authentication required for system package managers")
				sudo := exec.CommandContext(cmd.Context(), "sudo", "-v")
				sudo.Stdin = cmd.InOrStdin()
				sudo.Stderr = errOut
				sudo.Stdout = errOut
				if err := sudo.Run(); err != nil {
					return fmt.Errorf("sudo authentication failed: %w", err)
				}
			}

			// Run phase
			if runUpdates(cmd.Context(), errOut, adapters, packageFlag, serial) {
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
