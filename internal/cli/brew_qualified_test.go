package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

const qualifiedRef = "nklmilojevic/sofka/sofka"

func TestInstallCmd_QualifiedRoutesToBrew(t *testing.T) {
	buf, err := execCmd(t, []string{"install", qualifiedRef, "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "installed "+qualifiedRef+" via brew")
}

func TestInstallCmd_QualifiedWithoutBrewFails(t *testing.T) {
	_, err := execCmd(t, []string{"install", qualifiedRef, "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires brew")
}

func TestInstallCmd_TapOnlyRefHintsTap(t *testing.T) {
	_, err := execCmd(t, []string{"install", "nklmilojevic/sofka", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stamp tap")
}

func TestRemoveCmd_QualifiedRoutesToBrew(t *testing.T) {
	buf, err := execCmd(t, []string{"remove", qualifiedRef, "-y"}, []manager.Adapter{
		&mockAdapter{name: "brew"}, &mockAdapter{name: "dnf"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed "+qualifiedRef+" via brew")
}

func TestReinstallCmd_QualifiedRecordsInManifest(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	_, err := execCmdAt(t, []string{"reinstall", qualifiedRef, "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}}, mPath, cPath)
	require.NoError(t, err)

	m, err := manifest.Load(mPath)
	require.NoError(t, err)
	assert.True(t, m.HasPackage(qualifiedRef, "brew"), "reinstall must track the qualified name under brew")
}

func TestInstallCmd_HintSkippedForExplicitNonBrewManager(t *testing.T) {
	buf, err := execCmd(t, []string{"install", "foo/bar", "-m", "go", "-y"}, []manager.Adapter{&mockAdapter{name: "go"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "installed foo/bar via go")
}

func TestInstallCmd_TapOnlyRefWithBrewManagerHints(t *testing.T) {
	_, err := execCmd(t, []string{"install", "nklmilojevic/sofka", "-m", "brew", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stamp tap")
}

func TestSearchCmd_QualifiedScopesToBrew(t *testing.T) {
	buf, err := execCmd(t, []string{"search", qualifiedRef}, []manager.Adapter{
		&mockAdapter{name: "brew"}, &mockAdapter{name: "dnf"},
	})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "(brew)")
	assert.NotContains(t, output, "(dnf)")
}

func TestSearchCmd_QualifiedWithoutBrewFails(t *testing.T) {
	_, err := execCmd(t, []string{"search", qualifiedRef}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires brew")
}
