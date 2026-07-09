package cli

import (
	"github.com/spf13/cobra"
)

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = appFromCtx(cmd)
			cmd.OutOrStdout()
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = appFromCtx(cmd)
			cmd.OutOrStdout()
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
			_ = appFromCtx(cmd)
			cmd.OutOrStdout()
			return nil
		},
	}
	return cmd
}
