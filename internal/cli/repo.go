package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

var validRepoName = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_\-\.\/\+\:]*$`)

func validateRepoName(name string) error {
	if strings.HasPrefix(name, "-") {
		return catErr(ErrUsage, "repository name %q cannot start with '-'", name)
	}
	if !validRepoName.MatchString(name) {
		return catErr(ErrUsage, "repository name %q contains invalid characters", name)
	}
	return nil
}

func validateRepoURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return catErr(ErrUsage, "invalid repository URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return catErr(ErrUsage, "unsupported URL scheme %q; must be http or https", parsed.Scheme)
	}
	return nil
}

// isRepoURL reports whether rawURL parses as an http(s) URL, mirroring
// validateRepoURL without surfacing an error. Used to detect a single-argument
// `stamp repo add <url>` invocation so the name can be derived.
func isRepoURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// deriveRepoName derives a repository name from a URL. The basename of the
// URL path is used with a trailing .repo extension stripped; a pathless URL
// falls back to the host. Example: https://yum.enpass.io/enpass-yum.repo → "enpass-yum".
func deriveRepoName(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	base := path.Base(parsed.Path)
	if base == "" || base == "." || base == "/" {
		return parsed.Host
	}
	if strings.HasSuffix(strings.ToLower(base), ".repo") {
		base = base[:len(base)-len(".repo")]
	}
	return base
}

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage third-party repositories",
		Example: `  # add a third-party repository or tap
  stamp repo add my-tap

  # remove a repository
  stamp repo remove my-tap

  # list tracked repositories
  stamp repo list`,
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(newRepoAddCmd())
	cmd.AddCommand(newRepoRemoveCmd())
	cmd.AddCommand(newRepoListCmd())

	return cmd
}

func newRepoAddCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "add [name] [url]",
		Aliases: []string{"install"},
		Short:   "Add a third-party repository",
		Example: `  # add a PPA on Debian/Ubuntu systems
  stamp repo add ppa:git-core/ppa -m apt

  # add a COPR repository on Fedora/RHEL
  stamp repo add petersen/cava -m dnf

  # add a .repo file URL (e.g. Brave or Enpass) on Fedora/RHEL
  stamp repo add brave https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo -m dnf

  # add a repo by URL, deriving the name from the URL
  stamp repo add https://yum.enpass.io/enpass-yum.repo -m dnf

  # add a flatpak remote by URL
  stamp repo add flathub https://dl.flathub.org/repo/flathub.flatpakrepo -m flatpak

  # add a homebrew tap
  stamp repo add homebrew/cask -m brew`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}
			name := args[0]
			url := ""
			if len(args) > 1 {
				url = args[1]
			}
			// A single URL argument is shorthand: derive the name from the URL
			// (e.g. stamp repo add <repofile-url> -m dnf).
			if len(args) == 1 && isRepoURL(name) {
				url = name
				name = deriveRepoName(url)
			}
			if err := validateRepoName(name); err != nil {
				if url != "" && isRepoURL(url) {
					return catErr(ErrUsage, "cannot derive repository name from URL %q", url)
				}
				return err
			}
			if err := validateRepoURL(url); err != nil {
				return err
			}

			var adapter manager.Adapter
			for _, a := range app.adapters {
				if a.Name() == manager.ResolveManager(managerFlag) {
					adapter = a
					break
				}
			}
			if adapter == nil {
				return fmt.Errorf("manager %q not found (required)", managerFlag)
			}

			if err := requireConsent(cmd, fmt.Sprintf("Add repo %s via %s", name, managerFlag)); err != nil {
				return handleConsent(err)
			}
			if err := adapter.AddRepo(manager.WithYes(cmd.Context()), name, url); err != nil {
				return fmt.Errorf("failed to add repo: %w", err)
			}

			app.manifest.AddRepository(manifest.Repository{
				Name:    name,
				Manager: adapter.Name(),
				URL:     url,
				Origin:  manifest.OriginStamped,
			})

			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "added repo %s via %s\n", name, managerFlag)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager (required)")
	_ = cmd.MarkFlagRequired("manager")
	return cmd
}

func newRepoRemoveCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"uninstall", "rm", "delete", "del"},
		Short:   "Remove a third-party repository",
		Example: `  # remove a repository using the manager recorded in the manifest
  stamp repo remove ppa:git-core/ppa

  # specify a manager explicitly
  stamp repo remove ppa:git-core/ppa -m apt

  # aliases behave the same way
  stamp repo rm ppa:git-core/ppa -m apt`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}
			name := args[0]
			if err := validateRepoName(name); err != nil {
				return err
			}

			var adapter manager.Adapter

			// Check the manifest first: if the repo is tracked, use its
			// recorded manager so `-m` is optional.
			if managerFlag == "" {
				for _, r := range app.manifest.Repositories {
					if r.Name != name {
						continue
					}
					for _, a := range app.adapters {
						if a.Name() == r.Manager {
							adapter = a
							break
						}
					}
					break
				}
				if adapter == nil {
					return catErr(ErrUsage, "repository %q is not tracked; specify its manager with --manager", name)
				}
			}

			// Fall back to the explicit flag.
			if adapter == nil {
				for _, a := range app.adapters {
					if a.Name() == manager.ResolveManager(managerFlag) {
						adapter = a
						break
					}
				}
				if adapter == nil {
					return fmt.Errorf("manager %q not found", managerFlag)
				}
			}

			if err := requireConsent(cmd, fmt.Sprintf("Remove repo %s via %s", name, adapter.Name())); err != nil {
				return handleConsent(err)
			}
			if err := adapter.RemoveRepo(manager.WithYes(cmd.Context()), name); err != nil {
				return fmt.Errorf("failed to remove repo: %w", err)
			}

			app.manifest.RemoveRepository(name, adapter.Name())
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "removed repo %s via %s\n", name, adapter.Name())
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use (optional if the repo is tracked in the manifest)")
	return cmd
}

func newRepoListCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all tracked repositories",
		Example: `  # list all tracked repositories
  stamp repo list

  # filter by package manager
  stamp repo list -m flatpak

  # machine-readable JSON output
  stamp repo list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			repos := app.manifest.Repositories
			if managerFlag != "" {
				var filtered []manifest.Repository
				for _, r := range repos {
					if r.Manager == managerFlag {
						filtered = append(filtered, r)
					}
				}
				repos = filtered
			}

			if len(repos) == 0 {
				if app.json {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[]")
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no repositories tracked")
				}
				return nil
			}

			if app.json {
				data, err := json.MarshalIndent(repos, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal repositories: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			for _, r := range repos {
				line := fmt.Sprintf("%s (%s)", r.Name, r.Manager)
				if r.URL != "" {
					line += " " + r.URL
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to list")
	return cmd
}
