package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// infoRawResult is one adapter's info lookup outcome during collection.
type infoRawResult struct {
	manager string
	found   bool
	info    string
}

// resolveInfoTargets narrows adapters to a single manager when managerFlag is
// set, returning the plain "not found (required)" error so the exit code stays 1.
func resolveInfoTargets(adapters []manager.Adapter, managerFlag string) ([]manager.Adapter, error) {
	if managerFlag == "" {
		return adapters, nil
	}
	for _, a := range adapters {
		if a.Name() == manager.ResolveManager(managerFlag) {
			return []manager.Adapter{a}, nil
		}
	}
	return nil, fmt.Errorf("manager %q not found (required)", managerFlag)
}

// collectInfoResults queries every target adapter for package info. Adapters
// that error are recorded as not-found without surfacing the error.
func collectInfoResults(ctx context.Context, targets []manager.Adapter, pkg string, groupInfo bool) []infoRawResult {
	results := make([]infoRawResult, 0, len(targets))
	for _, a := range targets {
		infoCtx := ctx
		if groupInfo {
			infoCtx = manager.WithGroup(infoCtx)
		}
		info, err := a.Info(infoCtx, pkg)
		if err != nil {
			results = append(results, infoRawResult{manager: a.Name(), found: false})
		} else {
			results = append(results, infoRawResult{manager: a.Name(), found: true, info: info})
		}
	}
	return results
}

// anyInfoFound reports whether any adapter resolved the package.
func anyInfoFound(results []infoRawResult) bool {
	for _, r := range results {
		if r.found {
			return true
		}
	}
	return false
}

// extractVersion pulls a version out of a manager's raw info block. It first
// looks for a "version:" line, then falls back to brew-style "==> pkg: ...".
func extractVersion(info string) string {
	lines := strings.Split(info, "\n")
	for _, l := range lines {
		lLower := strings.ToLower(l)
		if strings.HasPrefix(lLower, "version") || strings.Contains(lLower, "version:") {
			parts := strings.Split(l, ":")
			if len(parts) > 1 {
				return "v" + strings.TrimSpace(parts[1])
			}
		}
	}
	for _, l := range lines {
		if m := brewDescriptorRegex.FindStringSubmatch(l); m != nil {
			return m[1]
		}
	}
	return "available"
}

// renderInfoJSON writes the info report as indented JSON to w.
func renderInfoJSON(w io.Writer, pkg string, results []infoRawResult) error {
	report := infoReport{Package: pkg}
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
	_, _ = fmt.Fprintln(w, string(data))
	return nil
}

// renderInfoText writes the human-readable info summary. With managerFlag set
// and the package found, the native raw block is printed; otherwise a summary
// table across all managers is shown.
func renderInfoText(w io.Writer, pkg string, results []infoRawResult, managerFlag string) {
	if !anyInfoFound(results) {
		_, _ = fmt.Fprintf(w, "%s: not found in any package manager\n", pkg)
		return
	}

	if managerFlag != "" {
		for _, r := range results {
			if r.found {
				_, _ = fmt.Fprintf(w, "%s via %s:\n\n%s\n", pkg, r.manager, r.info)
				return
			}
		}
	}

	_, _ = fmt.Fprintf(w, "%s:\n", pkg)
	for _, r := range results {
		if r.found {
			_, _ = fmt.Fprintf(w, "  %-10s %s\n", r.manager+":", extractVersion(r.info))
		} else {
			_, _ = fmt.Fprintf(w, "  %-10s %s\n", r.manager+":", "not available")
		}
	}
}

// validateInfoArgs validates the package name and the --group/--manager flag
// combination for stamp info.
func validateInfoArgs(pkgName, managerFlag string, groupInfo bool, targets []manager.Adapter) error {
	if !groupInfo {
		if err := manager.ValidatePackageName(pkgName); err != nil {
			return fmt.Errorf("invalid package name: %w", err)
		}
	}
	if groupInfo {
		if managerFlag == "" {
			return fmt.Errorf("--group requires --manager <name>")
		}
		for _, a := range targets {
			if a.Name() != "dnf" && a.Name() != "yum" {
				return fmt.Errorf("--group is only supported for dnf")
			}
		}
	}
	return nil
}

func newInfoCmd() *cobra.Command {
	var managerFlag string
	var groupInfo bool

	cmd := &cobra.Command{
		Use:     "info <package>",
		Aliases: []string{"show", "view"},
		Short:   "Show package information across managers",
		Example: `  # show package info across all managers (summary table)
  stamp info htop

  # show full raw output from a specific manager
  stamp info htop -m dnf

  # machine-readable JSON output
  stamp info htop --json

  # query info about a DNF package group (by group ID)
  stamp info development-tools -m dnf --group`,
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
			targets, err := resolveInfoTargets(app.adapters, managerFlag)
			if err != nil {
				return err
			}
			if err := validateInfoArgs(pkgName, managerFlag, groupInfo, targets); err != nil {
				return err
			}
			if len(targets) == 0 {
				return catErr(ErrUnavailable, "no package managers available")
			}

			results := collectInfoResults(cmd.Context(), targets, pkgName, groupInfo)

			if app.json {
				return renderInfoJSON(cmd.OutOrStdout(), pkgName, results)
			}

			renderInfoText(cmd.OutOrStdout(), pkgName, results, managerFlag)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to query")
	cmd.Flags().BoolVarP(&groupInfo, "group", "g", false, "query a DNF package group")
	return cmd
}
