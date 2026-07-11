package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func newManCmd() *cobra.Command {
	var install bool
	var prefix string

	cmd := &cobra.Command{
		Use:   "man",
		Short: "Generate the stamp man page",
		Long: `Generate the troff man page for stamp.

By default prints the man page to stdout. Use --install to copy to the system
man page directory so 'man stamp' works.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			header := &doc.GenManHeader{
				Title:   "STAMP",
				Section: "1",
				Source:  "stamp https://github.com/rossijonas/stamp",
			}

			if !install {
				return doc.GenMan(cmd.Root(), header, cmd.OutOrStdout())
			}

			if prefix == "" {
				prefix = defaultManPrefix()
			}

			manDir := filepath.Join(prefix, "share", "man", "man1")
			if err := os.MkdirAll(manDir, 0750); err != nil {
				return fmt.Errorf("failed to create man directory %s: %w", manDir, err)
			}

			manPath := filepath.Join(manDir, "stamp.1")
			//nolint:gosec // path is controlled by --prefix flag
			f, err := os.Create(manPath)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", manPath, err)
			}
			defer func() { _ = f.Close() }()

			if err := doc.GenMan(cmd.Root(), header, f); err != nil {
				return fmt.Errorf("failed to generate man page: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "man page installed to %s\n", manPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&install, "install", false, "install man page to system directory")
	cmd.Flags().StringVar(&prefix, "prefix", "", "install prefix (default: ~/.local)")
	return cmd
}

func defaultManPrefix() string {
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".local")
	}
	return "/usr/local"
}
