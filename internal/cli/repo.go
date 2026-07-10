package cli

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

var validRepoName = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_\-\.\/\+]*$`)

func validateRepoName(name string) error {
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("repository name %q cannot start with '-'", name)
	}
	if !validRepoName.MatchString(name) {
		return fmt.Errorf("repository name %q contains invalid characters", name)
	}
	return nil
}

func validateRepoURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q; must be http or https", parsed.Scheme)
	}
	return nil
}

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage third-party repositories",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newRepoAddCmd())
	cmd.AddCommand(newRepoRemoveCmd())
	cmd.AddCommand(newRepoListCmd())

	return cmd
}

func newRepoAddCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "add <name> [url]",
		Aliases: []string{"install"},
		Short:   "Add a third-party repository",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			name := args[0]
			if err := validateRepoName(name); err != nil {
				return err
			}
			url := ""
			if len(args) > 1 {
				url = args[1]
			}
			if err := validateRepoURL(url); err != nil {
				return err
			}

			var adapter manager.Adapter
			for _, a := range app.adapters {
				if a.Name() == managerFlag {
					adapter = a
					break
				}
			}
			if adapter == nil {
				return fmt.Errorf("manager %q not found (required)", managerFlag)
			}

			if err := adapter.AddRepo(cmd.Context(), name, url); err != nil {
				return fmt.Errorf("failed to add repo: %w", err)
			}

			app.manifest.AddRepository(manifest.Repository{
				Name:    name,
				Manager: adapter.Name(),
				URL:     url,
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
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			name := args[0]
			if err := validateRepoName(name); err != nil {
				return err
			}

			var adapter manager.Adapter
			for _, a := range app.adapters {
				if a.Name() == managerFlag {
					adapter = a
					break
				}
			}
			if adapter == nil {
				return fmt.Errorf("manager %q not found (required)", managerFlag)
			}

			if err := adapter.RemoveRepo(cmd.Context(), name); err != nil {
				return fmt.Errorf("failed to remove repo: %w", err)
			}

			app.manifest.RemoveRepository(name, adapter.Name())
			if err := app.saveManifest(); err != nil {
				return fmt.Errorf("failed to save manifest: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "removed repo %s via %s\n", name, managerFlag)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager (required)")
	_ = cmd.MarkFlagRequired("manager")
	return cmd
}

func newRepoListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all tracked repositories",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			if len(app.manifest.Repositories) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no repositories tracked")
				return nil
			}

			for _, r := range app.manifest.Repositories {
				line := fmt.Sprintf("%s (%s)", r.Name, r.Manager)
				if r.URL != "" {
					line += " " + r.URL
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
	return cmd
}
