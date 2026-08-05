package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manifest"
)

// validListTypes enumerates the accepted values for the list --type flag.
var validListTypes = []string{
	"packages",
	"repos",
	"stamped",
	"reconciled",
	"stamped-packages",
	"stamped-repos",
	"reconciled-packages",
	"reconciled-repos",
}

// listTypeCompletion completes the --type flag. Declared as a variable so it
// can be exercised directly in tests.
var listTypeCompletion = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return append([]string{}, validListTypes...), cobra.ShellCompDirectiveNoFileComp
}

// validateListType rejects any --type value outside the allowlist. An empty
// string is the default (packages) and is always valid.
func validateListType(t string) error {
	if t == "" {
		return nil
	}
	for _, v := range validListTypes {
		if t == v {
			return nil
		}
	}
	return catErr(ErrUsage, "unknown type %q; valid types: %s", t, strings.Join(validListTypes, ", "))
}

// filterPackages returns packages matching the manager and origin filters.
// An empty manager or origin disables that filter. Origin matching uses
// OriginEffective so pre-origin manifests (entries without an origin field)
// count as stamped.
func filterPackages(pkgs []manifest.Package, manager, origin string) []manifest.Package {
	filtered := []manifest.Package{}
	for _, p := range pkgs {
		if manager != "" && p.Manager != manager {
			continue
		}
		if origin != "" && p.OriginEffective() != origin {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// filterRepositories returns repositories matching the manager and origin
// filters. See filterPackages for filter semantics.
func filterRepositories(repos []manifest.Repository, manager, origin string) []manifest.Repository {
	filtered := []manifest.Repository{}
	for _, r := range repos {
		if manager != "" && r.Manager != manager {
			continue
		}
		if origin != "" && r.OriginEffective() != origin {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func printPackagesTTY(cmd *cobra.Command, pkgs []manifest.Package) {
	for _, p := range pkgs {
		line := fmt.Sprintf("%s (%s)", p.Name, p.Manager)
		if p.Notes != "" {
			line += " — " + p.Notes
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
	}
}

func printReposTTY(cmd *cobra.Command, repos []manifest.Repository) {
	for _, r := range repos {
		line := fmt.Sprintf("%s (%s)", r.Name, r.Manager)
		if r.URL != "" {
			line += " " + r.URL
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
	}
}

func renderPackages(cmd *cobra.Command, app *AppContext, pkgs []manifest.Package) error {
	if app.json {
		data, err := json.MarshalIndent(pkgs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal packages: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}
	if len(pkgs) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no packages tracked")
		return nil
	}
	printPackagesTTY(cmd, pkgs)
	return nil
}

func renderRepositories(cmd *cobra.Command, app *AppContext, repos []manifest.Repository) error {
	if app.json {
		data, err := json.MarshalIndent(repos, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal repositories: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}
	if len(repos) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no repositories tracked")
		return nil
	}
	printReposTTY(cmd, repos)
	return nil
}

// renderCombined renders packages followed by repositories for the combined
// stamped and reconciled views. JSON output is a flat array mixing package
// and repository objects. The empty message only fires when both slices are
// empty — a combined view may legitimately have no repos.
func renderCombined(cmd *cobra.Command, app *AppContext, pkgs []manifest.Package, repos []manifest.Repository) error {
	if app.json {
		items := make([]any, 0, len(pkgs)+len(repos))
		for i := range pkgs {
			items = append(items, pkgs[i])
		}
		for i := range repos {
			items = append(items, repos[i])
		}
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal list items: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}
	if len(pkgs) == 0 && len(repos) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "nothing tracked")
		return nil
	}
	printPackagesTTY(cmd, pkgs)
	printReposTTY(cmd, repos)
	return nil
}

func newListCmd() *cobra.Command {
	var (
		managerFlag string
		typeFlag    string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all intentionally installed packages",
		Example: "  stamp list\n  stamp list --json\n  stamp list -m brew\n  stamp list -t stamped-packages",
		Long: `Read the manifest and display all tracked packages.
By default prints a table of package names and their managers.
Use --json for machine-readable output.
Use -m to filter by a specific package manager.
Use --type to filter by entity type (packages/repos) and origin
(stamped/reconciled).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}
			if err := validateListType(typeFlag); err != nil {
				return err
			}

			switch typeFlag {
			case "", "packages":
				return renderPackages(cmd, app, filterPackages(app.manifest.Packages, managerFlag, ""))
			case "repos":
				return renderRepositories(cmd, app, filterRepositories(app.manifest.Repositories, managerFlag, ""))
			case "stamped":
				pkgs := filterPackages(app.manifest.Packages, managerFlag, manifest.OriginStamped)
				repos := filterRepositories(app.manifest.Repositories, managerFlag, manifest.OriginStamped)
				return renderCombined(cmd, app, pkgs, repos)
			case "reconciled":
				pkgs := filterPackages(app.manifest.Packages, managerFlag, manifest.OriginReconciled)
				repos := filterRepositories(app.manifest.Repositories, managerFlag, manifest.OriginReconciled)
				return renderCombined(cmd, app, pkgs, repos)
			case "stamped-packages":
				return renderPackages(cmd, app, filterPackages(app.manifest.Packages, managerFlag, manifest.OriginStamped))
			case "stamped-repos":
				return renderRepositories(cmd, app, filterRepositories(app.manifest.Repositories, managerFlag, manifest.OriginStamped))
			case "reconciled-packages":
				return renderPackages(cmd, app, filterPackages(app.manifest.Packages, managerFlag, manifest.OriginReconciled))
			case "reconciled-repos":
				return renderRepositories(cmd, app, filterRepositories(app.manifest.Repositories, managerFlag, manifest.OriginReconciled))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to list")
	cmd.Flags().StringVarP(&typeFlag, "type", "t", "", `Filter by entity type and origin. Valid types:
  packages           All packages (default)
  repos              All repositories
  stamped            Everything installed via stamp (packages + repos)
  reconciled         Everything discovered by reconcile (packages + repos)
  stamped-packages   Packages installed via stamp
  stamped-repos      Repos added via stamp
  reconciled-packages Packages discovered by reconcile
  reconciled-repos    Repos discovered by reconcile

Origin meanings:
  "stamped"    = installed explicitly via stamp install/reinstall
  "reconciled" = installed outside stamp, auto-discovered by reconcile`)
	//nolint:errcheck // the --type flag is registered above, so this cannot fail
	_ = cmd.RegisterFlagCompletionFunc("type", listTypeCompletion)
	return cmd
}
