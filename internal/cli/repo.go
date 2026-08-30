package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

var validRepoName = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_\-\.\/\+\:]*$`)

// repoManagerFlagDesc is the shared flag description for the -m/--manager
// flag on repository subcommands that accept an optional manager.
const repoManagerFlagDesc = "package manager to use (optional if the repo is tracked in the manifest)"

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

// parseRepoAddArgs parses and validates the positional arguments for
// `repo add`. It handles the URL shorthand (single URL → derived name),
// and returns the final name and URL with validation errors.
func parseRepoAddArgs(args []string) (name, url string, err error) {
	name = args[0]
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
			return "", "", catErr(ErrUsage, "cannot derive repository name from URL %q", url)
		}
		return "", "", err
	}
	if err := validateRepoURL(url); err != nil {
		return "", "", err
	}
	return name, url, nil
}

// resolveAddAdapter resolves the adapter for a repo add command using
// the explicit -m flag. The add command always requires an explicit manager.
func resolveAddAdapter(app *AppContext, managerFlag string) (manager.Adapter, error) {
	for _, a := range app.adapters {
		if a.Name() == manager.ResolveManager(managerFlag) {
			return a, nil
		}
	}
	return nil, fmt.Errorf("manager %q not found (required)", managerFlag)
}

// resolveAdapterFromManifest resolves an adapter from the manifest's
// recorded manager for a named repository. Returns nil if no match.
func resolveAdapterFromManifest(app *AppContext, name string) manager.Adapter {
	for _, r := range app.manifest.Repositories {
		if r.Name != name {
			continue
		}
		for _, a := range app.adapters {
			if a.Name() == r.Manager {
				return a
			}
		}
		break
	}
	return nil
}

// resolveAdapterFromFlag resolves an adapter using the explicit -m flag.
func resolveAdapterFromFlag(app *AppContext, managerFlag string) manager.Adapter {
	for _, a := range app.adapters {
		if a.Name() == manager.ResolveManager(managerFlag) {
			return a
		}
	}
	return nil
}

// filterReposByManager returns repos matching managerFlag, or all if empty.
func filterReposByManager(repos []manifest.Repository, managerFlag string) []manifest.Repository {
	if managerFlag == "" {
		return repos
	}
	var filtered []manifest.Repository
	for _, r := range repos {
		if r.Manager == managerFlag {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// renderRepoListJSON writes repos as indented JSON to w.
func renderRepoListJSON(w io.Writer, repos []manifest.Repository) error {
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal repositories: %w", err)
	}
	_, _ = fmt.Fprintln(w, string(data))
	return nil
}

// renderRepoListText writes the human-readable repo list to w.
func renderRepoListText(w io.Writer, repos []manifest.Repository) {
	for _, r := range repos {
		line := fmt.Sprintf("%s (%s)", r.Name, r.Manager)
		if r.URL != "" {
			line += " " + r.URL
		}
		_, _ = fmt.Fprintln(w, line)
	}
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
	cmd.AddCommand(newRepoTrustCmd())
	cmd.AddCommand(newRepoUntrustCmd())

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

			name, url, err := parseRepoAddArgs(args)
			if err != nil {
				return err
			}

			adapter, err := resolveAddAdapter(app, managerFlag)
			if err != nil {
				return err
			}

			verb := fmt.Sprintf("Add repo %s via %s", name, managerFlag)
			if manager.ResolveManager(managerFlag) == "brew" {
				verb = fmt.Sprintf("Add and trust repo %s via brew", name)
			}
			if err := requireConsent(cmd, verb); err != nil {
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

			adapter, err := resolveRepoAdapter(cmd, name, managerFlag)
			if err != nil {
				return err
			}

			verb := fmt.Sprintf("Remove repo %s via %s", name, adapter.Name())
			if adapter.Name() == "brew" {
				verb = fmt.Sprintf("Remove and untrust repo %s via brew", name)
			}
			if err := requireConsent(cmd, verb); err != nil {
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

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", repoManagerFlagDesc)
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

			repos := filterReposByManager(app.manifest.Repositories, managerFlag)

			if len(repos) == 0 {
				if app.json {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[]")
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no repositories tracked")
				}
				return nil
			}

			if app.json {
				return renderRepoListJSON(cmd.OutOrStdout(), repos)
			}

			renderRepoListText(cmd.OutOrStdout(), repos)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to list")
	return cmd
}

// resolveRepoAdapter resolves the adapter for a repository operation. It
// resolves via the explicit -m flag when given, otherwise from the manager
// recorded in the manifest for the named repository. Shared by `repo remove`
// and the `repo trust`/`repo untrust` brew operations.
func resolveRepoAdapter(cmd *cobra.Command, name, managerFlag string) (manager.Adapter, error) {
	app := appFromCtx(cmd)
	if app.manifestErr != nil {
		return nil, app.manifestErr
	}

	// Check the manifest first: if the repo is tracked, use its recorded
	// manager so `-m` is optional.
	if managerFlag == "" {
		adapter := resolveAdapterFromManifest(app, name)
		if adapter != nil {
			return adapter, nil
		}
		return nil, catErr(ErrUsage, "repository %q is not tracked; specify its manager with --manager", name)
	}

	// Fall back to the explicit flag.
	adapter := resolveAdapterFromFlag(app, managerFlag)
	if adapter == nil {
		return nil, fmt.Errorf("manager %q not found", managerFlag)
	}
	return adapter, nil
}

// resolveBrewAdapter resolves the brew adapter for a repo trust/untrust
// operation. It returns an error if the resolved manager is not brew.
func resolveBrewAdapter(cmd *cobra.Command, name, managerFlag string) (manager.Adapter, error) {
	adapter, err := resolveRepoAdapter(cmd, name, managerFlag)
	if err != nil {
		return nil, err
	}
	if adapter.Name() != "brew" {
		return nil, fmt.Errorf("trust/untrust is only supported for the brew manager, got %q", adapter.Name())
	}
	return adapter, nil
}

func newRepoTrustCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "trust <name>",
		Short: "Trust a Homebrew tap",
		Example: `  # trust a tap recorded in the manifest
  stamp repo trust homebrew/cask

  # specify the manager explicitly
  stamp repo trust anomalyco/tap -m brew`,
		Long: `Mark a Homebrew tap as trusted so Homebrew 6.0.0+ loads its formulae,
casks, and commands. Only brew taps can be trusted.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			adapter, err := resolveBrewAdapter(cmd, name, managerFlag)
			if err != nil {
				return err
			}
			if err := requireConsent(cmd, fmt.Sprintf("Trust repo %s via brew", name)); err != nil {
				return handleConsent(err)
			}
			brew, ok := adapter.(manager.TapTrustManager)
			if !ok {
				return fmt.Errorf("brew adapter is unavailable")
			}
			if err := brew.Trust(manager.WithYes(cmd.Context()), name); err != nil {
				return fmt.Errorf("failed to trust repo: %w", err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "trusted repo %s via brew\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", repoManagerFlagDesc)
	return cmd
}

func newRepoUntrustCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "untrust <name>",
		Short: "Stop trusting a Homebrew tap",
		Example: `  # untrust a tap recorded in the manifest
  stamp repo untrust homebrew/cask

  # specify the manager explicitly
  stamp repo untrust anomalyco/tap -m brew`,
		Long: `Stop trusting a Homebrew tap. Only brew taps can be untrusted.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			adapter, err := resolveBrewAdapter(cmd, name, managerFlag)
			if err != nil {
				return err
			}
			if err := requireConsent(cmd, fmt.Sprintf("Untrust repo %s via brew", name)); err != nil {
				return handleConsent(err)
			}
			brew, ok := adapter.(manager.TapTrustManager)
			if !ok {
				return fmt.Errorf("brew adapter is unavailable")
			}
			if err := brew.Untrust(manager.WithYes(cmd.Context()), name); err != nil {
				return fmt.Errorf("failed to untrust repo: %w", err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "untrusted repo %s via brew\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", repoManagerFlagDesc)
	return cmd
}
