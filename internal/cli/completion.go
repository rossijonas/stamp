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
		Use:     "completion [bash|zsh|fish|powershell]",
		Short:   "Generate and install shell completion script",
		Example: "  stamp completion\n  stamp completion --stdout bash\n  stamp completion fish",
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
		// Check common user fpath directories in preference order
		candidates := []string{
			filepath.Join(home, ".zsh", ".zfunc"),
			filepath.Join(home, ".local", "share", "zsh", "site-functions"),
		}
		for _, dir := range candidates {
			if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
				return filepath.Join(dir, "_stamp")
			}
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

	// Strip redundant compdef line from Zsh completion — #compdef is sufficient
	if shell == "zsh" {
		if err := stripZshCompdef(tmpPath); err != nil {
			return fmt.Errorf("failed to process completion: %w", err)
		}
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to install completion to %s: %w", path, err)
	}

	// Remove stale completions from other common directories (not the target)
	home, _ := os.UserHomeDir()
	destDir := filepath.Dir(path)
	staleDirs := []string{
		filepath.Join(home, ".zsh", "completions"),
		filepath.Join(home, ".zfunc"),
		filepath.Join(home, ".oh-my-zsh", "custom", "completions"),
	}
	for _, dir := range staleDirs {
		if dir == destDir {
			continue
		}
		stalePath := filepath.Join(dir, "_stamp")
		if fi, err := os.Stat(stalePath); err == nil && !fi.IsDir() {
			_ = os.Remove(stalePath)
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  removed stale completion from %s\n", stalePath)
		}
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "completion installed to %s\n", path)

	if shell == "zsh" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "\nTo enable completions, add BEFORE compinit in ~/.zshrc:\n")
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  fpath=(%s $fpath)\n", filepath.Dir(path))
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  autoload -U compinit; compinit\n")
	}

	return nil
}

func stripZshCompdef(path string) error {
	//nolint:gosec // path is a temp file in the completion directory
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "compdef ") {
			continue
		}
		filtered = append(filtered, line)
	}
	//nolint:gosec // completion files must be world-readable
	return os.WriteFile(path, []byte(strings.Join(filtered, "\n")), 0644)
}
