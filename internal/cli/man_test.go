package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestMan_Output(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"man"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "STAMP")
	assert.Contains(t, buf.String(), ".TH")
}

func TestMan_InstallDryRun(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"man", "--install", "--prefix", t.TempDir()}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "man page installed to")
}

func TestMan_InvalidFlag(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"man", "--nonexistent"}, []manager.Adapter{})
	require.Error(t, err)
}
