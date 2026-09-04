package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

// collectProvidesResults queries every target adapter for the file, skipping
// managers that do not support provides and warning on other failures.
func collectProvidesResults(ctx context.Context, w io.Writer, targets []manager.Adapter, query string) []string {
	var results []string
	for _, a := range targets {
		out, err := a.Provides(ctx, query)
		if err != nil {
			if errors.Is(err, manager.ErrNotSupported) {
				continue
			}
			_, _ = fmt.Fprintf(w, "warning: %s provides failed: %v\n", a.Name(), err)
			continue
		}
		for _, line := range out {
			results = append(results, fmt.Sprintf("%s (%s)", line, a.Name()))
		}
	}
	return results
}

func newProvidesCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "provides <file>",
		Short: "Find which package provides a given file",
		Example: `  # find which package owns a binary across all managers
  stamp provides /usr/bin/htop

  # scope to a single manager for faster results
  stamp provides libssl.so -m dnf

  # no results returns a clear message
  stamp provides /usr/bin/nonexistent`,
		Long: `Search across all system package managers to find which package
owns the specified file. Supports both absolute and relative paths.

Scoped to a single manager with the --manager flag.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			query := args[0]

			targets, err := resolveManagerTarget(app.adapters, managerFlag)
			if err != nil {
				return err
			}

			results := collectProvidesResults(cmd.Context(), cmd.ErrOrStderr(), targets, query)

			if len(results) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no packages provide "+query)
				return nil
			}

			for _, r := range results {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), r)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to query")
	return cmd
}
