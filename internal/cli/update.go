package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

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
				// adapters has been filtered to a single entry for managerFlag
				if err := manager.ValidatePackageForManager(adapters[0].Name(), packageFlag); err != nil {
					return err
				}
			}

			if len(adapters) == 0 {
				return fmt.Errorf("no package managers available")
			}

			if needsSudo(adapters) && isTerminal(cmd.InOrStdin()) {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "▪ Authentication required for system package managers")
				sudo := exec.CommandContext(cmd.Context(), "sudo", "-v")
				sudo.Stdin = cmd.InOrStdin()
				sudo.Stderr = cmd.ErrOrStderr()
				sudo.Stdout = cmd.ErrOrStderr()
				if err := sudo.Run(); err != nil {
					return fmt.Errorf("sudo authentication failed: %w", err)
				}
			}

			var hasErr bool
			var mu sync.Mutex

			if serial {
				for _, a := range adapters {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "▪ updating via %s...\n", a.Name())
					if !runUpdate(cmd.Context(), cmd.ErrOrStderr(), a, packageFlag, "") {
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
						if !runUpdate(cmd.Context(), cmd.ErrOrStderr(), a, packageFlag, "["+a.Name()+"] ") {
							mu.Lock()
							hasErr = true
							mu.Unlock()
						}
					}()
				}
				wg.Wait()
			}

			if hasErr {
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
