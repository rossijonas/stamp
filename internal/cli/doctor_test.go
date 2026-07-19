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
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	buf, err := execCmd(t, []string{"doctor"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "▪ System Diagnosis (Stamp Doctor)")
	assert.Contains(t, output, "Package Managers:")
	assert.Contains(t, output, "Manifest Integrity:")
	assert.Contains(t, output, "Path:")
	assert.Contains(t, output, "Man Page: ❌ Not found")
}

func TestDoctor_JSON(t *testing.T) {
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	buf, err := execCmd(t, []string{"doctor", "--json"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.Equal(t, runtime.GOOS, report.System)
	assert.Len(t, report.PackageManagers, 4)

	names := make(map[string]bool)
	for _, m := range report.PackageManagers {
		names[m.Name] = true
	}
	assert.True(t, names["apt"])
	assert.True(t, names["dnf"])
	assert.True(t, names["brew"])
	assert.True(t, names["flatpak"])

	assert.NotEmpty(t, report.Manifest.Path)
	assert.False(t, report.NoColor) // NO_COLOR not set in tests
	assert.False(t, report.ManPage.Installed)
}

func TestDoctor_NOCOLOR_Set(t *testing.T) {
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	t.Setenv("NO_COLOR", "1")
	buf, err := execCmd(t, []string{"doctor", "--json"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.True(t, report.NoColor)
}

func TestDoctor_Manifest_Healthy(t *testing.T) {
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

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
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

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

func TestDoctor_Manifest_Missing(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

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
	assert.Contains(t, report.Manifest.Error, "manifest not found")
}

func TestDoctor_Manifest_Missing_TTY(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "❌ manifest not found")
}

func TestDoctor_Completions_NotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	comps := checkCompletionStatus()
	assert.False(t, comps.Installed)
}

func TestDoctor_Completions_Installed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	bashDir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
	require.NoError(t, os.MkdirAll(bashDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(bashDir, "stamp"), []byte("#!/bin/bash"), 0600))

	comps := checkCompletionStatus()
	assert.True(t, comps.Installed)
	assert.Contains(t, comps.Shells, "bash")
}

func TestDoctor_Completions_TTY(t *testing.T) {
	// No completions installed — doctor should report not installed
	buf, err := execCmd(t, []string{"doctor"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Completions:")
}

func TestDoctor_ManPage_Healthy(t *testing.T) {
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	// Pre-create the man page with matching current version ("dev")
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp dev" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manFile, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"doctor", "--json"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.True(t, report.ManPage.Installed)
	assert.Equal(t, "dev", report.ManPage.Version)
}

func TestDoctor_ManPage_UserLocal_Detected(t *testing.T) {
	home := t.TempDir()
	manPath := filepath.Join(home, ".local", "share", "man", "man1", "stamp.1")

	oldCandidates := manPageCandidates
	manPageCandidates = []string{manPath}
	defer func() { manPageCandidates = oldCandidates }()

	require.NoError(t, os.MkdirAll(filepath.Dir(manPath), 0750))
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp ` + Version + `" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manPath, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"doctor", "--json"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)
	assert.True(t, report.ManPage.Installed)
	assert.Equal(t, Version, report.ManPage.Version)
}

func TestDoctor_ManagerFlag_Active(t *testing.T) {
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew"},
		&manager.Mock{ManagerName: "dnf"},
	}

	buf, err := execCmd(t, []string{"doctor", "-m", "brew"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "brew")
	assert.NotContains(t, output, "dnf")
}

func TestDoctor_ManagerFlag_NotFound(t *testing.T) {
	_, err := execCmd(t, []string{"doctor", "-m", "nonexistent"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available on this system")
}

func TestDoctor_ManagerFlag_NativeOutput(t *testing.T) {
	buf, err := execCmd(t, []string{"doctor", "-m", "brew"}, []manager.Adapter{&manager.Mock{ManagerName: "brew", DoctorResult: "Your system is ready to brew."}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "brew doctor:")
	assert.Contains(t, output, "Your system is ready to brew.")
}
