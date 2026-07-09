package cli

import "github.com/spf13/cobra"

func newInstallCmd() *cobra.Command {
	var managerFlag string
	var note string

	cmd := &cobra.Command{
		Use:     "install <package>",
		Aliases: []string{"add"},
		Short:   "Install a package and record intent",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = appFromCtx(cmd)
			cmd.OutOrStdout()
			// TODO: implement full install logic in follow-up
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	cmd.Flags().StringVar(&note, "note", "", "annotation for this package")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:     "remove <package>",
		Aliases: []string{"uninstall", "rm", "delete", "del"},
		Short:   "Remove a package and untrack it",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = appFromCtx(cmd)
			cmd.OutOrStdout()
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to use")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for packages across managers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = appFromCtx(cmd)
			cmd.OutOrStdout()
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to search")
	return cmd
}
