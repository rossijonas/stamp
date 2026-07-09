package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_FileNotFound(t *testing.T) {
	t.Parallel()
	cfg, err := LoadConfig("/nonexistent/path/config.toml")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, []string{"dnf", "flatpak", "brew"}, cfg.Precedence)
	assert.Empty(t, cfg.Rules)
}

func TestLoadConfig_ValidFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	content := []byte(`
precedence = ["brew", "dnf"]

[[rules]]
pattern = "^com\\..*"
prefer = "flatpak"
`)
	err := os.WriteFile(path, content, 0600)
	require.NoError(t, err)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"brew", "dnf"}, cfg.Precedence)
	require.Len(t, cfg.Rules, 1)
	assert.Equal(t, "^com\\..*", cfg.Rules[0].Pattern)
	assert.Equal(t, "flatpak", cfg.Rules[0].Prefer)
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.toml")
	err := os.WriteFile(path, []byte("invalid [[toml\n"), 0600)
	require.NoError(t, err)

	_, err = LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestLoadConfig_ReadError(t *testing.T) {
	t.Parallel()
	_, err := LoadConfig("/proc/1/root/config.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config")
}
