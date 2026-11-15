package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func TestInstallCmd_QualifiedSpecsTrackBaseName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    string
		manager string
		tracked string
	}{
		{"apt version", "htop=3.2.1", "apt", "htop"},
		{"pacman repo", "extra/htop", "pacman", "htop"},
		{"zypper repo", "oss:htop", "zypper", "htop"},
		{"cargo version", "ripgrep@14.1.0", "cargo", "ripgrep"},
		{"uv version", "ruff@0.6.0", "uv", "ruff"},
		{"pipx extras", "black[colorama]", "pipx", "black"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mPath := seedManifest(t)
			cPath := filepath.Join(t.TempDir(), "config.toml")
			adapter := &mockAdapter{name: tt.manager}

			_, err := execCmdAt(t, []string{"install", tt.spec, "-m", tt.manager, "-y"},
				[]manager.Adapter{adapter}, mPath, cPath)
			require.NoError(t, err)

			m := loadManifest(t, mPath)
			assert.True(t, m.HasPackage(tt.tracked, tt.manager), "expected %s tracked", tt.tracked)
			assert.False(t, m.HasPackage(tt.spec, tt.manager), "raw spec must not be tracked")
		})
	}
}

func TestRemoveCmd_QualifiedSpecUntracksBaseName(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t, manifest.Package{Name: "htop", Manager: "apt"})
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"remove", "htop=3.2.1", "-m", "apt", "-y"},
		[]manager.Adapter{&mockAdapter{name: "apt"}}, mPath, cPath)
	require.NoError(t, err)
	assert.False(t, loadManifest(t, mPath).HasPackage("htop", "apt"))
}

func TestInstallCmd_GroupSyntaxHints(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"install", "@development-tools", "-m", "dnf", "-y"},
		[]manager.Adapter{&mockAdapter{name: "dnf"}}, mPath, cPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use --group")
}

func TestInstallCmd_AptModifierHints(t *testing.T) {
	t.Parallel()
	mPath := seedManifest(t)
	cPath := filepath.Join(t.TempDir(), "config.toml")

	_, err := execCmdAt(t, []string{"install", "htop+", "-m", "apt", "-y"},
		[]manager.Adapter{&mockAdapter{name: "apt"}}, mPath, cPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apt modifier")
}
