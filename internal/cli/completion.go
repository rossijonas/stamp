package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	var stdout bool

	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate and install shell completion script",
		Long: `Generate and install shell completion scripts for stamp.

Without arguments, auto-detects the current shell and installs to the
correct system path. Use --stdout to print the script instead.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var shell string
			if len(args) == 1 {
				shell = args[0]
			} else {
				shell = detectShell()
				if shell == "" {
					return fmt.Errorf("cannot detect shell, specify: stamp completion [bash|zsh|fish|powershell]")
				}
			}

			if stdout {
				return writeCompletion(cmd, shell, cmd.OutOrStdout())
			}

			return installCompletion(cmd, shell)
		},
	}

	cmd.Flags().BoolVarP(&stdout, "stdout", "s", false, "print completion script to stdout instead of installing")
	return cmd
}

func detectShell() string {
	path := os.Getenv("SHELL")
	if path == "" {
		return ""
	}
	base := filepath.Base(path)
	return strings.TrimPrefix(base, "-")
}

func writeCompletion(cmd *cobra.Command, shell string, w io.Writer) error {
	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(w)
	case "zsh":
		return cmd.Root().GenZshCompletion(w)
	case "fish":
		return cmd.Root().GenFishCompletion(w, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("unknown shell %q, supported: bash, zsh, fish, powershell", shell)
	}
}

func completionPath(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch shell {
	case "bash":
		return filepath.Join(home, ".local", "share", "bash-completion", "completions", "stamp")
	case "zsh":
		// Prefer XDG path, fallback to ~/.zfunc
		dir := filepath.Join(home, ".local", "share", "zsh", "site-functions")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return filepath.Join(dir, "_stamp")
		}
		return filepath.Join(home, ".zfunc", "_stamp")
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions", "stamp.fish")
	}
	return ""
}

func installCompletion(cmd *cobra.Command, shell string) error {
	path := completionPath(shell)
	if path == "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: auto-install not supported for %s, use --stdout\n", shell)
		return writeCompletion(cmd, shell, cmd.OutOrStdout())
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create completion directory %s: %w", dir, err)
	}

	// Generate to temp file for atomic write
	tmpFile, err := os.CreateTemp(dir, "stamp-completion-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	var writeErr error
	switch shell {
	case "bash":
		writeErr = cmd.Root().GenBashCompletion(tmpFile)
	case "zsh":
		writeErr = cmd.Root().GenZshCompletion(tmpFile)
	case "fish":
		writeErr = cmd.Root().GenFishCompletion(tmpFile, true)
	case "powershell":
		writeErr = cmd.Root().GenPowerShellCompletionWithDesc(tmpFile)
	}

	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	if writeErr != nil {
		return fmt.Errorf("failed to generate completion: %w", writeErr)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to install completion to %s: %w", path, err)
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "completion installed to %s\n", path)
	return nil
}
