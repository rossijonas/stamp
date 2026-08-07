package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCommandsHaveExamples guards the man-page/documentation contract: every
// command (groups included) must ship a cobra Example so `task docs` renders a
// commented EXAMPLES section on the corresponding man page (see issue #184).
// The reference format is `stamp install`: a "# description" line followed by
// the command, one pair per variation.
func TestCommandsHaveExamples(t *testing.T) {
	root := NewRootCmd()
	walk(t, root)
}

func walk(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	if strings.TrimSpace(cmd.Example) == "" {
		t.Errorf("command %q has no Example; add a comment-style example (see stamp install)", cmd.CommandPath())
	}
	for _, sub := range cmd.Commands() {
		walk(t, sub)
	}
}
