package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHelloCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hello",
		Short: "Print welcome message and recommended next steps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()

			_, _ = fmt.Fprint(out, `
                              
         █▄
        ▄██▄      ▄
   ▄██▀█ ██ ▄▀▀█▄ ███▄███▄ ████▄
   ▀███▄ ██ ▄█▀██ ██ ██ ██ ██ ██
  █▄▄██▀▄██▄▀█▄██▄██ ██ ▀█▄████▀
                           ██
                           ▀

  stamp — A lightweight yet powerful wrapper for your native package managers.

  For a fresh installation, try:

    stamp init          — Create manifest and take initial snapshot
    stamp doctor        — Verify system configuration
    stamp man install   — Install offline documentation

  Need help? Run:  stamp --help
`)
			return nil
		},
	}

	return cmd
}
