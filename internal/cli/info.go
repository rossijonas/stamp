package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

var brewDescriptorRegex = regexp.MustCompile(`^==> \S+: (.+)$`)

type infoReportItem struct {
	Manager string `json:"manager"`
	Found   bool   `json:"found"`
	Info    string `json:"info,omitempty"`
}

type infoReport struct {
	Package string           `json:"package"`
	Results []infoReportItem `json:"results"`
}

func newInfoCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "info <package>",
		Short: "Show package information across managers",
		Long: `Query detailed information about a package.
By default, queries all available managers and outputs a summary table.
If -m, --manager is specified, displays the native manager's full raw info block.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}

			pkgName := args[0]
			if err := manager.ValidatePackageName(pkgName); err != nil {
				return fmt.Errorf("invalid package name: %w", err)
			}

			targets := app.adapters
			if managerFlag != "" {
				var found manager.Adapter
				for _, a := range app.adapters {
					if a.Name() == managerFlag {
						found = a
						break
					}
				}
				if found == nil {
					return fmt.Errorf("manager %q not found (required)", managerFlag)
				}
				targets = []manager.Adapter{found}
			}

			if len(targets) == 0 {
				return fmt.Errorf("no package managers available")
			}

			type rawResult struct {
				manager string
				found   bool
				info    string
			}
			var results []rawResult

			for _, a := range targets {
				info, err := a.Info(cmd.Context(), pkgName)
				if err != nil {
					results = append(results, rawResult{manager: a.Name(), found: false})
				} else {
					results = append(results, rawResult{manager: a.Name(), found: true, info: info})
				}
			}

			// Validate if package was found anywhere
			anyFound := false
			for _, r := range results {
				if r.found {
					anyFound = true
					break
				}
			}

			if app.json {
				report := infoReport{
					Package: pkgName,
				}
				for _, r := range results {
					report.Results = append(report.Results, infoReportItem{
						Manager: r.manager,
						Found:   r.found,
						Info:    r.info,
					})
				}
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal info report: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if !anyFound {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: not found in any package manager\n", pkgName)
				return nil
			}

			// If managerFlag is specified and package was found, print raw block
			if managerFlag != "" {
				for _, r := range results {
					if r.found {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s via %s:\n\n%s\n", pkgName, r.manager, r.info)
						return nil
					}
				}
			}

			// Multi-manager TTY Table-like Output
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", pkgName)
			for _, r := range results {
				if r.found {
					// Extract version from first few lines of raw info if possible, or print standard string
					lines := strings.Split(r.info, "\n")
					version := "available"
					for _, l := range lines {
						lLower := strings.ToLower(l)
						if strings.HasPrefix(lLower, "version") || strings.Contains(lLower, "version:") {
							parts := strings.Split(l, ":")
							if len(parts) > 1 {
								version = "v" + strings.TrimSpace(parts[1])
								break
							}
						}
					}
					// Fallback for brew-style output: "==> htop: stable 3.4.1 (bottled), HEAD"
					if version == "available" {
						for _, l := range lines {
							if m := brewDescriptorRegex.FindStringSubmatch(l); m != nil {
								version = m[1]
								break
							}
						}
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-10s %s\n", r.manager+":", version)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-10s %s\n", r.manager+":", "not available")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to query")
	return cmd
}
