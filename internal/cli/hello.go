package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func alreadyInitialized(manifestPath string) bool {
	_, err := os.Stat(manifestPath)
	return err == nil
}

// runSetupCompletions offers shell completion installation.
func runSetupCompletions(cmd *cobra.Command, autoAccept bool, errOut io.Writer) {
	_, _ = fmt.Fprintln(errOut, "Step 1 of 4: Shell Completions")
	if autoAccept || promptYesNo(cmd.Context(), errOut, cmd.InOrStdin(), "  Install shell completions? [Y/n]: ", true) {
		runCompletion(cmd)
	} else {
		_, _ = fmt.Fprintln(errOut, "  Run 'stamp completion' later")
	}
	_, _ = fmt.Fprintln(errOut)
}

// runSetupManPages offers man page installation.
func runSetupManPages(cmd *cobra.Command, autoAccept bool, errOut io.Writer) {
	_, _ = fmt.Fprintln(errOut, "Step 2 of 4: Man Pages")
	if autoAccept || promptYesNo(cmd.Context(), errOut, cmd.InOrStdin(), "  Install man pages? [Y/n]: ", true) {
		runSubcommand(cmd, "man", "install")
	} else {
		_, _ = fmt.Fprintln(errOut, "  Run 'stamp man install' later")
	}
	_, _ = fmt.Fprintln(errOut)
}

// runSetupInit offers manifest initialization (or re-initialization).
func runSetupInit(cmd *cobra.Command, app *AppContext, autoAccept bool, errOut io.Writer) {
	_, _ = fmt.Fprintln(errOut, "Step 3 of 4: Initialize")
	isInit := alreadyInitialized(app.manifestPath)
	if isInit {
		_, _ = fmt.Fprintln(errOut, "  ⚠ Stamp is already initialized on this system.")
		_, _ = fmt.Fprintln(errOut, "  This will re-write manifest.toml and baseline snapshots.")
	}
	promptText := "  Create manifest and baseline snapshot? [Y/n]: "
	promptDefault := true
	if isInit {
		promptText = "  Re-initialize (backup old configuration)? [y/N]: "
		promptDefault = false
	}
	if autoAccept || promptYesNo(cmd.Context(), errOut, cmd.InOrStdin(), promptText, promptDefault) {
		if isInit {
			runSubcommand(cmd, "init", "--yes")
		} else {
			runSubcommand(cmd, "init")
		}
	} else {
		_, _ = fmt.Fprintln(errOut, "  ⚠ stamp requires initialization to work properly")
	}
	_, _ = fmt.Fprintln(errOut)
}

// runSetupDoctor runs the final system diagnosis step.
func runSetupDoctor(cmd *cobra.Command, errOut io.Writer) {
	_, _ = fmt.Fprintln(errOut, "Step 4 of 4: System Diagnosis")
	runSubcommand(cmd, "doctor")
	_, _ = fmt.Fprintln(errOut)
}

func newHelloCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "setup",
		Aliases: []string{"hello"},
		Short:   "Run first-time setup wizard",
		Example: `  # run the interactive first-time setup wizard
  stamp setup

  # non-interactive setup for scripting
  stamp setup -y`,
		Long: `Guided setup for new stamp installations.
Runs completion installation, man page setup, initialization, and diagnostics.
Use -y to skip all prompts for scripting.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			autoAccept := app.yes
			errOut := cmd.ErrOrStderr()
			tty := isOutputTerminal(errOut)
			_, _ = fmt.Fprint(errOut, stampBanner)

			if autoAccept {
				_, _ = fmt.Fprintln(errOut, iconLine(tty, "▪", "Stamp Setup Wizard (auto-accept)"))
			} else {
				_, _ = fmt.Fprintln(errOut, iconLine(tty, "▪", "Stamp Setup Wizard"))
			}
			_, _ = fmt.Fprintln(errOut)

			runSetupCompletions(cmd, autoAccept, errOut)
			runSetupManPages(cmd, autoAccept, errOut)
			runSetupInit(cmd, app, autoAccept, errOut)
			runSetupDoctor(cmd, errOut)

			_, _ = fmt.Fprintln(errOut, iconLine(tty, "▪", "Setup complete!"))
			return nil
		},
	}

	return cmd
}

func promptYesNo(ctx context.Context, out io.Writer, in io.Reader, msg string, defaultYes bool) bool {
	if !isTerminal(in) {
		return defaultYes
	}
	// Already interrupted (SIGINT canceled the context): fail closed.
	if ctx.Err() != nil {
		return false
	}
	_, _ = fmt.Fprint(out, msg)
	ch := make(chan string, 1)
	go func() {
		response, err := bufio.NewReader(in).ReadString('\n')
		if err != nil {
			ch <- ""
			return
		}
		ch <- strings.TrimSpace(response)
	}()
	select {
	case response := <-ch:
		if defaultYes {
			return response == "" || strings.EqualFold(response, "y") || strings.EqualFold(response, "yes")
		}
		return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes")
	case <-ctx.Done():
		return false
	}
}

func runCompletion(cmd *cobra.Command) {
	shell := detectShell()
	if shell == "" {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ⚠ cannot detect shell, run 'stamp completion <shell>' manually\n")
		return
	}
	if err := installCompletion(cmd, shell); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ⚠ completion install failed: %v\n", err)
	}
}

func runSubcommand(cmd *cobra.Command, args ...string) {
	subCmd, _, err := cmd.Root().Find(args)
	if err != nil || subCmd == nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ⚠ %s command not found\n", args[0])
		return
	}
	subCmd.SetContext(cmd.Context())
	subCmd.SetOut(cmd.OutOrStdout())
	subCmd.SetErr(cmd.ErrOrStderr())
	subCmd.SetIn(cmd.InOrStdin())

	for _, f := range args[1:] {
		name := strings.TrimLeft(f, "-")
		if strings.Contains(name, "=") {
			parts := strings.SplitN(name, "=", 2)
			_ = subCmd.Flags().Set(parts[0], parts[1])
		} else {
			_ = subCmd.Flags().Set(name, "true")
		}
	}

	if err := subCmd.RunE(subCmd, nil); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ⚠ %s failed: %v\n", args[0], err)
	}
}
