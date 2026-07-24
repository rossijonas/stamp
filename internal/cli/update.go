package cli

import (
	"fmt"
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

func newUpdateCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"upgrade"},
		Short:   "Run system upgrades across all package managers",
		Example: "  stamp update\n  stamp update -m apt\n  stamp upgrade",
		Long: `Run system upgrade commands for each available package manager.
Updates and upgrades all packages to their latest versions.
Use -m to scope to a single package manager.`,
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
			var wg sync.WaitGroup

			for _, a := range adapters {
				a := a
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx := manager.WithOutputPrefix(cmd.Context(), "["+a.Name()+"] ")
					if err := a.Update(ctx); err != nil {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "⚠ update failed for %s: %v\n", a.Name(), err)
						mu.Lock()
						hasErr = true
						mu.Unlock()
						return
					}
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "updated packages via %s\n", a.Name())
				}()
			}

			wg.Wait()

			if hasErr {
				return fmt.Errorf("one or more managers failed to update")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to update")
	return cmd
}
