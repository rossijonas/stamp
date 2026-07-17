package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// syncBuffer wraps bytes.Buffer with a mutex for concurrent-safe writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.b.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.b.String()
}

// execUpdateCmd is like execCmd but uses syncBuffer for concurrent-safe output.
func execUpdateCmd(t *testing.T, args []string, adapters []manager.Adapter) (*syncBuffer, error) {
	t.Helper()
	buf := new(syncBuffer)
	tmpDir := t.TempDir()
	cPath := filepath.Join(tmpDir, "config.toml")
	mPath := filepath.Join(tmpDir, "manifest.toml")
	root := NewRootCmd(WithAdapters(adapters), WithConfigPath(cPath), WithManifestPath(mPath))
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	root.SetIn(r)
	_ = w.Close()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf, err
}

func TestUpdateCmd_AllManagers(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf"},
		&mockAdapter{name: "flatpak"},
	}
	buf, err := execUpdateCmd(t, []string{"update"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via brew")
	assert.Contains(t, output, "updated packages via dnf")
	assert.Contains(t, output, "updated packages via flatpak")
}

func TestUpdateCmd_ManagerFlag(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf"},
		&mockAdapter{name: "flatpak"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-m", "dnf"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via dnf")
	assert.NotContains(t, output, "brew")
	assert.NotContains(t, output, "flatpak")
}

func TestUpdateCmd_UnknownManager(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	_, err := execUpdateCmd(t, []string{"update", "-m", "nonexistent"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestUpdateCmd_OneFails(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf", err: assert.AnError},
		&mockAdapter{name: "flatpak"},
	}
	buf, err := execUpdateCmd(t, []string{"update"}, adapters)
	require.Error(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via brew")
	assert.Contains(t, output, "⚠ update failed for dnf")
	assert.Contains(t, output, "updated packages via flatpak")
}

func TestUpdateCmd_NoAdapters(t *testing.T) {
	_, err := execUpdateCmd(t, []string{"update"}, []manager.Adapter{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no package managers available")
}

func TestUpdateCmd_AllFail(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew", err: assert.AnError},
		&mockAdapter{name: "dnf", err: assert.AnError},
	}
	buf, err := execUpdateCmd(t, []string{"update"}, adapters)
	require.Error(t, err)
	output := buf.String()
	assert.Contains(t, output, "⚠ update failed for brew")
	assert.Contains(t, output, "⚠ update failed for dnf")
	assert.Contains(t, err.Error(), "one or more managers failed to update")
}

func TestUpdateCmd_UsesAdapterUpdate(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	buf, err := execUpdateCmd(t, []string{"update"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated packages via brew")
}

func TestUpdateCmd_UpgradeAlias(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	buf, err := execUpdateCmd(t, []string{"upgrade"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated packages via brew")
}

func TestUpdateCmd_WithManifestNotRequired(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	buf, err := execUpdateCmd(t, []string{"update"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated packages via brew")
}
