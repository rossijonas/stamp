package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manifest"
)

type ctxKey struct{}

// AppContext holds the runtime dependencies for CLI commands.
type AppContext struct {
	Repos []manifest.Repository
}

func newAppContext() (*AppContext, error) {
	return &AppContext{}, nil
}

func appFromCtx(cmd *cobra.Command) *AppContext {
	v := cmd.Context().Value(ctxKey{})
	if v == nil {
		return nil
	}
	return v.(*AppContext)
}

// NewRootCmd creates a new root command with all subcommands registered.
// Returns a fresh tree each call — safe for parallel testing.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "stamp",
		Short:         "Track package installation intent across multiple package managers",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			appCtx, err := newAppContext()
			if err != nil {
				return fmt.Errorf("initialization failed: %w", err)
			}
			cmd.SetContext(context.WithValue(cmd.Context(), ctxKey{}, appCtx))
			return nil
		},
	}

	root.PersistentFlags().BoolP("verbose", "v", false, "enable debug logging")
	root.PersistentFlags().Bool("json", false, "output results in JSON format")
	root.PersistentFlags().BoolP("yes", "y", false, "auto-accept all prompts")

	root.AddCommand(newInstallCmd())
	root.AddCommand(newRemoveCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newRepoCmd())

	return root
}

var rootCmd = NewRootCmd()

// Execute is the entry point for the CLI, called from cmd/stamp/main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
