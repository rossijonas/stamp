package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestDoctor_TTY(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"doctor"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "▪ System Diagnosis (Stamp Doctor)")
	assert.Contains(t, output, "Package Managers:")
	assert.Contains(t, output, "Manifest Integrity:")
	assert.Contains(t, output, "Path:")
}

func TestDoctor_JSON(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"doctor", "--json"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.Equal(t, runtime.GOOS, report.System)
	assert.Len(t, report.PackageManagers, 3)

	names := make(map[string]bool)
	for _, m := range report.PackageManagers {
		names[m.Name] = true
	}
	assert.True(t, names["dnf"])
	assert.True(t, names["brew"])
	assert.True(t, names["flatpak"])

	assert.NotEmpty(t, report.Manifest.Path)
}

func TestDoctor_Manifest_Healthy(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.True(t, report.Manifest.Valid)
	assert.Equal(t, 1, report.Manifest.PackagesCount)
}

func TestDoctor_Manifest_Corrupt(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	err := root.Execute()
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.False(t, report.Manifest.Valid)
	assert.Contains(t, report.Manifest.Error, "failed to parse manifest")
}
