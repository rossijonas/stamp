package manager

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSudoCmd_NonTTY(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_ = w.Close()

	oldStdin := stdIn
	stdIn = r
	defer func() {
		stdIn = oldStdin
		_ = r.Close()
	}()

	result := sudoCmd("install", "-y", "htop")
	assert.Equal(t, []string{"sudo", "-n", "install", "-y", "htop"}, result)
}

func TestSudoCmd_StatError(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_ = r.Close()
	_ = w.Close()

	oldStdin := stdIn
	stdIn = r
	defer func() { stdIn = oldStdin }()

	result := sudoCmd("update")
	// Stat returns error on closed pipe → interactive assumed, no -n.
	assert.Equal(t, []string{"sudo", "update"}, result)
}

func TestSudoReady(t *testing.T) {
	oldProbe := sudoProbe
	defer func() { sudoProbe = oldProbe }()

	sudoProbe = func(context.Context) bool { return true }
	assert.True(t, SudoReady(context.Background()))

	sudoProbe = func(context.Context) bool { return false }
	assert.False(t, SudoReady(context.Background()))
}

func TestEnsureSudo(t *testing.T) {
	oldValidate := sudoValidate
	defer func() { sudoValidate = oldValidate }()

	sudoValidate = func(context.Context) error { return nil }
	require.NoError(t, EnsureSudo(context.Background()))

	sudoValidate = func(context.Context) error { return assert.AnError }
	assert.ErrorIs(t, EnsureSudo(context.Background()), assert.AnError)
}
