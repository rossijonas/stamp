package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestDoctor_TTY(t *testing.T) {
	stubManExec(t)
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
	assert.Contains(t, output, "Man Page: ✗ Not found")
}

func TestDoctor_JSON(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	buf, err := execCmd(t, []string{"doctor", "--json"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)

	var report doctorReport
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.Equal(t, runtime.GOOS, report.System)
	assert.Len(t, report.PackageManagers, 14)

	names := make(map[string]bool)
	for _, m := range report.PackageManagers {
		names[m.Name] = true
	}
	assert.True(t, names["apt"])
	assert.True(t, names["dnf"])
	assert.True(t, names["pacman"])
	assert.True(t, names["paru"])
	assert.True(t, names["zypper"])
	assert.True(t, names["snap"])
	assert.True(t, names["flatpak"])
	assert.True(t, names["brew"])
	assert.True(t, names["macports"])
	assert.True(t, names["go"])
	assert.True(t, names["npm"])
	assert.True(t, names["cargo"])
	assert.True(t, names["pipx"])
	assert.True(t, names["uv"])

	assert.NotEmpty(t, report.Manifest.Path)
	assert.True(t, report.Config.Valid) // no config.toml -> defaults
	assert.False(t, report.NoColor)     // NO_COLOR not set in tests
	assert.False(t, report.ManPage.Installed)
}

func TestDoctor_NOCOLOR_Set(t *testing.T) {
	stubManExec(t)
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

func TestDoctor_Config_Valid(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "Configuration:")
	assert.Contains(t, output, "✓ Valid")
	assert.NotContains(t, output, "invalid [backup]")
}

func TestDoctor_Config_MissingFileDefaultsValid(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml") // never written
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, root.Execute())

	var report doctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	assert.True(t, report.Config.Valid)
	assert.Empty(t, report.Config.Error)
}

func TestDoctor_Config_InvalidMinExceedsMax(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))
	require.NoError(t, os.WriteFile(cPath, []byte(`[backup]
max_manifest_backups = 10
min_manifest_backups = 30
`), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "Configuration:")
	assert.Contains(t, output, "✗ invalid [backup] config:")
	assert.Contains(t, output, "min_manifest_backups (30) exceeds max_manifest_backups (10)")
	assert.Contains(t, output, "fixing config.toml")
}

func TestDoctor_Config_InvalidJSON(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))
	require.NoError(t, os.WriteFile(cPath, []byte(`[backup]
min_manifest_backup_age_days = 90
max_manifest_backup_age_days = 30
`), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, root.Execute())

	var report doctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	assert.False(t, report.Config.Valid)
	assert.Contains(t, report.Config.Error, "min_manifest_backup_age_days (90) exceeds max_manifest_backup_age_days (30)")
}

func TestDoctor_SystemMissing_TTY(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"lazygit"}}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	content := `version = 1
system = "linux"

[[packages]]
name = "lazygit"
manager = "brew"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "✓ Healthy (2 package(s))")
	assert.Contains(t, output, "Missing:")
	assert.Contains(t, output, "- htop (brew)")
	assert.Contains(t, output, "stamp restore")
	assert.Contains(t, output, "stamp ls --type missing")
	assert.NotContains(t, output, "lazygit (brew)")
}

func TestDoctor_SystemMissing_JSON(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"lazygit"}}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	content := `version = 1
system = "linux"

[[packages]]
name = "lazygit"
manager = "brew"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, root.Execute())

	var report doctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.True(t, report.Manifest.Valid)
	require.Len(t, report.Manifest.Missing, 1)
	assert.Equal(t, "htop", report.Manifest.Missing[0].Name)
	assert.Equal(t, "brew", report.Manifest.Missing[0].Manager)
}

func TestDoctor_SystemMissing_ListErrorSkipped(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", ListErr: errors.New("boom")}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, root.Execute())

	var report doctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.True(t, report.Manifest.Valid)
	assert.Empty(t, report.Manifest.Missing)
}

func TestDoctor_SystemMissing_NoneMissing(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"htop"}}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, root.Execute())

	var report doctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.True(t, report.Manifest.Valid)
	assert.Empty(t, report.Manifest.Missing)
}

func TestDoctor_SystemMissing_ProvidesAlias(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	// nodejs resolves to nodejs22 through provides — the checker must keep it
	// out of the Missing list even though the raw listing lacks the name.
	checker := &checkerMock{
		Mock:   manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"nodejs22"}},
		absent: map[string]bool{"ghost": true},
	}
	adapters := []manager.Adapter{checker}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	content := `version = 1
system = "linux"

[[packages]]
name = "nodejs"
manager = "dnf"

[[packages]]
name = "ghost"
manager = "dnf"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, root.Execute())

	var report doctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.True(t, report.Manifest.Valid)
	require.Len(t, report.Manifest.Missing, 1)
	assert.Equal(t, "ghost", report.Manifest.Missing[0].Name)
}

func TestDoctor_Manifest_Healthy(t *testing.T) {
	stubManExec(t)
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
	// Healthy refers to manifest integrity; a manifest entry absent from the
	// (empty) mock installed list still surfaces as missing drift.
	require.Len(t, report.Manifest.Missing, 1)
	assert.Equal(t, "htop", report.Manifest.Missing[0].Name)
}

func TestDoctor_Manifest_Corrupt(t *testing.T) {
	stubManExec(t)
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
	assert.Contains(t, output, "✗ manifest not found")
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

	stubManExec(t)
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

	stubManExec(t)
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

func TestDoctor_ManPage_Outdated(t *testing.T) {
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	// Man page version differs from current Version ("dev")
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp 0.5.0" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manFile, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"doctor"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "⚠ Outdated")
	assert.Contains(t, output, "run 'stamp man install'")
}

func TestDoctor_ManagerFlag_Active(t *testing.T) {
	stubManExec(t)
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
	assert.Equal(t, ExitUnavailable, exitCodeFor(err))
}

func TestDoctor_ManagerFlag_NativeOutput(t *testing.T) {
	buf, err := execCmd(t, []string{"doctor", "-m", "brew"}, []manager.Adapter{&manager.Mock{ManagerName: "brew", DoctorResult: "Your system is ready to brew."}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "brew doctor:")
	assert.Contains(t, output, "Your system is ready to brew.")
}
