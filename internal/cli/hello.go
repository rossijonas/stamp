package cli

import (
	"bufio"
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

func newHelloCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "setup",
		Aliases: []string{"hello"},
		Short:   "Run first-time setup wizard",
		Long: `Guided setup for new stamp installations.
Runs completion installation, man page setup, initialization, and diagnostics.
Use -y to skip all prompts for scripting.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			autoAccept := app.yes
			errOut := cmd.ErrOrStderr()

			if autoAccept {
				_, _ = fmt.Fprintln(errOut, "▪ Stamp Setup Wizard (auto-accept)")
			} else {
				_, _ = fmt.Fprintln(errOut, "▪ Stamp Setup Wizard")
			}
			_, _ = fmt.Fprintln(errOut)

			// Step 1: Shell Completions
			_, _ = fmt.Fprintln(errOut, "Step 1 of 4: Shell Completions")
			if autoAccept || promptYesNo(errOut, cmd.InOrStdin(), "  Install shell completions? [Y/n]: ", true) {
				runCompletion(cmd)
			} else {
				_, _ = fmt.Fprintln(errOut, "  Run 'stamp completion' later")
			}
			_, _ = fmt.Fprintln(errOut)

			// Step 2: Man Pages
			_, _ = fmt.Fprintln(errOut, "Step 2 of 4: Man Pages")
			if autoAccept || promptYesNo(errOut, cmd.InOrStdin(), "  Install man pages? [Y/n]: ", true) {
				runSubcommand(cmd, "man", "install")
			} else {
				_, _ = fmt.Fprintln(errOut, "  Run 'stamp man install' later")
			}
			_, _ = fmt.Fprintln(errOut)

			// Step 3: Init
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
			if autoAccept || promptYesNo(errOut, cmd.InOrStdin(), promptText, promptDefault) {
				if isInit {
					runSubcommand(cmd, "init", "--yes")
				} else {
					runSubcommand(cmd, "init")
				}
			} else {
				_, _ = fmt.Fprintln(errOut, "  ⚠ stamp requires initialization to work properly")
			}
			_, _ = fmt.Fprintln(errOut)

			// Step 4: Doctor
			_, _ = fmt.Fprintln(errOut, "Step 4 of 4: System Diagnosis")
			runSubcommand(cmd, "doctor")

			_, _ = fmt.Fprintln(errOut)
			_, _ = fmt.Fprintln(errOut, "▪ Setup complete!")
			return nil
		},
	}

	return cmd
}

func promptYesNo(out io.Writer, in io.Reader, msg string, defaultYes bool) bool {
	if !isTerminal(in) {
		return defaultYes
	}
	_, _ = fmt.Fprint(out, msg)
	response, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		return defaultYes
	}
	response = strings.TrimSpace(response)
	if defaultYes {
		return response == "" || strings.EqualFold(response, "y") || strings.EqualFold(response, "yes")
	}
	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes")
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
