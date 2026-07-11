package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestCompletion_AllShells(t *testing.T) {
	t.Parallel()
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			t.Parallel()
			buf, err := execCmd(t, []string{"completion", shell}, []manager.Adapter{})
			require.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestCompletion_InvalidShell(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"completion", "invalid"}, []manager.Adapter{})
	require.Error(t, err)
}

func TestCompletion_NoArgs(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"completion"}, []manager.Adapter{})
	require.Error(t, err)
}

func TestCompletion_ExtraArgs(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"completion", "bash", "extra"}, []manager.Adapter{})
	require.Error(t, err)
}
