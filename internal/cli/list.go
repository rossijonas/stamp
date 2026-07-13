package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manifest"
)

func newListCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all intentionally installed packages",
		Long: `Read the manifest and display all tracked packages.
By default prints a table of package names and their managers.
Use --json for machine-readable output.
Use -m to filter by a specific package manager.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			pkgs := app.manifest.Packages
			if managerFlag != "" {
				filtered := []manifest.Package{}
				for _, p := range pkgs {
					if p.Manager == managerFlag {
						filtered = append(filtered, p)
					}
				}
				pkgs = filtered
			}

			if len(pkgs) == 0 {
				if app.json {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[]")
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no packages tracked")
				}
				return nil
			}

			if app.json {
				data, err := json.MarshalIndent(pkgs, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal packages: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			for _, p := range pkgs {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s (%s)\n", p.Name, p.Manager)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to list")
	return cmd
}
