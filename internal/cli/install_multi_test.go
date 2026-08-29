package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestInstallMany_RequiresManager(t *testing.T) {
	_, err := execCmd(t, []string{"install", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple packages require --manager")
}

func TestInstallMany_HappyPath(t *testing.T) {
	buf, err := execCmd(t, []string{"install", "-m", "brew", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "installed 2 package(s) via brew")
}

func TestInstallMany_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"install", "-m", "nonexistent", "htop", "atop", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available on this system")
}

func TestInstallMany_CapabilityError(t *testing.T) {
	// mockAdapter implements manager.Adapter but not manager.BatchInstaller.
	_, err := execCmd(t, []string{"install", "-m", "dnf", "htop", "atop", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support installing multiple packages at once")
}

func TestInstallMany_GroupRejected(t *testing.T) {
	_, err := execCmd(t, []string{"install", "-m", "dnf", "htop", "atop", "--group", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--group supports a single package")
}

func TestInstallCmd_GroupNameWithSpacesRejected(t *testing.T) {
	_, err := execCmd(t, []string{"install", "VideoLAN Client", "-m", "dnf", "--group", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group IDs contain only a-z0-9_-")
}

func TestInstallMany_InvalidNameAborts(t *testing.T) {
	_, err := execCmd(t, []string{"install", "-m", "brew", "htop", "bad name", "-y"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

// caskMock embeds manager.Mock (full Adapter + BatchInstaller) and reports
// cask status purely by name so the brew mixed-cask fallback path is testable
// without the real brew executor.
type caskMock struct {
	*manager.Mock
	installs []string
}

func (c *caskMock) IsCask(_ context.Context, pkg string) (bool, error) {
	return pkg == "cask-app", nil
}

func (c *caskMock) Install(ctx context.Context, pkg string) error {
	c.installs = append(c.installs, pkg)
	return c.Mock.Install(ctx, pkg)
}

func TestInstallMany_BrewMixedCaskFallsBackToSingle(t *testing.T) {
	adapters := &caskMock{Mock: &manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	root := NewRootCmd(
		WithAdapters([]manager.Adapter{adapters}),
		WithManifestPath(tmpDir+"/manifest.toml"),
		WithConfigPath(tmpDir+"/config.toml"),
	)
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"install", "-m", "brew", "cask-app", "formula", "-y"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "installed 2 package(s) via brew")
	require.Len(t, adapters.installs, 2, "mixed cask/formula batch must fall back to per-package installs")
	assert.Equal(t, []string{"cask-app", "formula"}, adapters.installs)
}

func TestInstallCmd_FlatpakRemoteHint(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		adapter    manager.Adapter
		wantHint   bool
		wantSuffix string
	}{
		{
			name:     "remote-first with -m flatpak",
			args:     []string{"install", "flathub", "com.obsproject.Studio", "-m", "flatpak"},
			adapter:  &manager.Mock{ManagerName: "flatpak"},
			wantHint: true,
		},
		{
			name:       "dotted-first batch -m flatpak",
			args:       []string{"install", "com.foo.Bar", "com.baz.Qux", "-m", "flatpak"},
			adapter:    &manager.Mock{ManagerName: "flatpak"},
			wantHint:   false,
			wantSuffix: "refusing to run without -y",
		},
		{
			name:       "no -m flag remote-first",
			args:       []string{"install", "flathub", "com.obsproject.Studio"},
			adapter:    &manager.Mock{ManagerName: "flatpak"},
			wantHint:   false,
			wantSuffix: "multiple packages require --manager",
		},
		{
			name:       "-m dnf undotted first",
			args:       []string{"install", "htop", "vim", "-m", "dnf"},
			adapter:    &manager.Mock{ManagerName: "dnf"},
			wantHint:   false,
			wantSuffix: "refusing to run without -y",
		},
		{
			name:       "single arg -m flatpak",
			args:       []string{"install", "flathub", "-m", "flatpak"},
			adapter:    &manager.Mock{ManagerName: "flatpak"},
			wantHint:   false,
			wantSuffix: "refusing to run without -y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execCmd(t, tt.args, []manager.Adapter{tt.adapter})
			require.Error(t, err)
			if tt.wantHint {
				assert.Contains(t, err.Error(), "flatpak install takes a remote and app ID separately")
			} else {
				assert.NotContains(t, err.Error(), "flatpak install takes a remote and app ID separately")
				assert.Contains(t, err.Error(), tt.wantSuffix)
			}
		})
	}
}
