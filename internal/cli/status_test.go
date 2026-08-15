package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		tty    bool
		done   bool
		verb   string
		target string
		mgr    string
		note   string
		want   string
	}{
		{"tty progress", true, false, "installing", "lazygit", "brew", "", "▪ installing lazygit via brew..."},
		{"tty result", true, true, "installed", "lazygit", "brew", "", "✓ installed lazygit via brew"},
		{"tty result with note", true, true, "installed", "lazygit", "brew", "better git TUI", "✓ installed lazygit via brew (note: better git TUI)"},
		{"plain result", false, true, "installed", "lazygit", "brew", "", "installed lazygit via brew"},
		{"plain result with note", false, true, "installed", "lazygit", "brew", "better git TUI", "installed lazygit via brew (note: better git TUI)"},
		{"plain progress is empty", false, false, "installing", "lazygit", "brew", "", ""},
		{"tty batch progress", true, false, "installing", "3 package(s)", "dnf", "", "▪ installing 3 package(s) via dnf..."},
		{"plain batch result", false, true, "installed", "3 package(s)", "dnf", "", "installed 3 package(s) via dnf"},
		{"tty batch result", true, true, "installed", "3 package(s)", "dnf", "", "✓ installed 3 package(s) via dnf"},
		{"tty remove", true, true, "removed", "htop", "apt", "", "✓ removed htop via apt"},
		{"plain reinstall", false, true, "reinstalled", "lazygit", "brew", "", "reinstalled lazygit via brew"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusLine(tt.tty, tt.done, tt.verb, tt.target, tt.mgr, tt.note)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsOutputTerminal(t *testing.T) {
	t.Parallel()
	// In-memory buffers are never terminals.
	assert.False(t, isOutputTerminal(&bytes.Buffer{}))

	// A regular file is not a terminal either.
	f, err := os.CreateTemp(t.TempDir(), "stamp-status-*")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	assert.False(t, isOutputTerminal(f))
}
