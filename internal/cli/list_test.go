package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

// writeManifestWithOrigins writes a manifest containing packages and repos
// with mixed origins, plus a stubbed config path. Returns both paths.
func writeManifestWithOrigins(t *testing.T) (manifestPath, configPath string) {
	t.Helper()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"
notes = "system monitor"
origin = "stamped"

[[packages]]
name = "NetworkManager"
manager = "dnf"
origin = "reconciled"

[[packages]]
name = "lazygit"
manager = "brew"
origin = "stamped"

[[repositories]]
name = "my-tap"
manager = "brew"
origin = "stamped"

[[repositories]]
name = "brave-browser"
manager = "dnf"
url = "https://example.com/brave"
origin = "reconciled"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))
	return mPath, cPath
}

// writeEmptyManifest writes a manifest with no packages or repos.
func writeEmptyManifest(t *testing.T) (manifestPath, configPath string) {
	t.Helper()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))
	return mPath, cPath
}

// runListCmd executes `stamp list` with the given args against the given
// manifest/config and returns combined stdout+stderr output.
func runListCmd(t *testing.T, mPath, cPath string, args ...string) string {
	t.Helper()
	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(append([]string{"list"}, args...))
	require.NoError(t, root.Execute())
	return buf.String()
}

// runListCmdWithAdapters is runListCmd with injected mock adapters, needed by
// system-aware views like --type missing.
func runListCmdWithAdapters(t *testing.T, mPath, cPath string, adapters []manager.Adapter, args ...string) string {
	t.Helper()
	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(cPath), WithAdapters(adapters))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(append([]string{"list"}, args...))
	require.NoError(t, root.Execute())
	return buf.String()
}

func TestValidateListType(t *testing.T) {
	t.Parallel()
	for _, valid := range append([]string{""}, validListTypes...) {
		require.NoError(t, validateListType(valid), "valid type %q", valid)
	}
	for _, invalid := range []string{"xyz", "package", "stamped package", "Stamped", "repos-packages", "-t"} {
		err := validateListType(invalid)
		require.Error(t, err, "invalid type %q", invalid)
		assert.Contains(t, err.Error(), "unknown type")
		assert.Contains(t, err.Error(), "packages")
		assert.Contains(t, err.Error(), "reconciled-repos")
	}
}

func TestFilterPackages(t *testing.T) {
	t.Parallel()
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "dnf", Origin: manifest.OriginStamped},
		{Name: "NetworkManager", Manager: "dnf", Origin: manifest.OriginReconciled},
		{Name: "lazygit", Manager: "brew", Origin: manifest.OriginStamped},
		{Name: "jq", Manager: "brew"}, // pre-origin: defaults to stamped
	}
	t.Run("no filter", func(t *testing.T) {
		assert.Len(t, filterPackages(pkgs, "", ""), 4)
	})
	t.Run("manager only", func(t *testing.T) {
		got := filterPackages(pkgs, "dnf", "")
		require.Len(t, got, 2)
		assert.Equal(t, "htop", got[0].Name)
		assert.Equal(t, "NetworkManager", got[1].Name)
	})
	t.Run("origin stamped", func(t *testing.T) {
		got := filterPackages(pkgs, "", manifest.OriginStamped)
		require.Len(t, got, 3)
		assert.Equal(t, "htop", got[0].Name)
		assert.Equal(t, "lazygit", got[1].Name)
		assert.Equal(t, "jq", got[2].Name, "pre-origin entry counts as stamped")
	})
	t.Run("origin reconciled", func(t *testing.T) {
		got := filterPackages(pkgs, "", manifest.OriginReconciled)
		require.Len(t, got, 1)
		assert.Equal(t, "NetworkManager", got[0].Name)
	})
	t.Run("manager and origin", func(t *testing.T) {
		got := filterPackages(pkgs, "brew", manifest.OriginStamped)
		require.Len(t, got, 2)
	})
	t.Run("no match", func(t *testing.T) {
		assert.Empty(t, filterPackages(pkgs, "flatpak", ""))
	})
}

func TestFilterRepositories(t *testing.T) {
	t.Parallel()
	repos := []manifest.Repository{
		{Name: "my-tap", Manager: "brew", Origin: manifest.OriginStamped},
		{Name: "brave-browser", Manager: "dnf", URL: "https://example.com/brave", Origin: manifest.OriginReconciled},
		{Name: "old-tap", Manager: "brew"}, // pre-origin: defaults to stamped
	}
	t.Run("no filter", func(t *testing.T) {
		assert.Len(t, filterRepositories(repos, "", ""), 3)
	})
	t.Run("origin stamped", func(t *testing.T) {
		got := filterRepositories(repos, "", manifest.OriginStamped)
		require.Len(t, got, 2)
		assert.Equal(t, "my-tap", got[0].Name)
		assert.Equal(t, "old-tap", got[1].Name, "pre-origin entry counts as stamped")
	})
	t.Run("manager and origin", func(t *testing.T) {
		got := filterRepositories(repos, "brew", manifest.OriginReconciled)
		assert.Empty(t, got)
	})
}

func TestListCmd_TypeTTY(t *testing.T) {
	mPath, cPath := writeManifestWithOrigins(t)
	tests := []struct {
		typeFlag string
		want     []string
		notWant  []string
	}{
		{
			typeFlag: "packages",
			want:     []string{"htop (dnf) — system monitor", "NetworkManager (dnf)", "lazygit (brew)"},
			notWant:  []string{"my-tap"},
		},
		{
			typeFlag: "repos",
			want:     []string{"my-tap (brew)", "brave-browser (dnf) https://example.com/brave"},
			notWant:  []string{"htop"},
		},
		{
			typeFlag: "stamped",
			want:     []string{"htop (dnf) — system monitor", "lazygit (brew)", "my-tap (brew)"},
			notWant:  []string{"NetworkManager", "brave-browser"},
		},
		{
			typeFlag: "reconciled",
			want:     []string{"NetworkManager (dnf)", "brave-browser (dnf) https://example.com/brave"},
			notWant:  []string{"htop", "my-tap", "lazygit"},
		},
		{
			typeFlag: "stamped-packages",
			want:     []string{"htop (dnf) — system monitor", "lazygit (brew)"},
			notWant:  []string{"NetworkManager", "my-tap"},
		},
		{
			typeFlag: "stamped-repos",
			want:     []string{"my-tap (brew)"},
			notWant:  []string{"htop", "brave-browser"},
		},
		{
			typeFlag: "reconciled-packages",
			want:     []string{"NetworkManager (dnf)"},
			notWant:  []string{"htop", "my-tap"},
		},
		{
			typeFlag: "reconciled-repos",
			want:     []string{"brave-browser (dnf) https://example.com/brave"},
			notWant:  []string{"htop", "my-tap", "NetworkManager"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.typeFlag, func(t *testing.T) {
			out := runListCmd(t, mPath, cPath, "-t", tt.typeFlag)
			for _, w := range tt.want {
				assert.Contains(t, out, w, "want %q in output for -t %s", w, tt.typeFlag)
			}
			for _, nw := range tt.notWant {
				assert.NotContains(t, out, nw, "must not contain %q for -t %s", nw, tt.typeFlag)
			}
		})
	}
}

func TestListCmd_TypeJSON(t *testing.T) {
	mPath, cPath := writeManifestWithOrigins(t)

	t.Run("packages", func(t *testing.T) {
		out := runListCmd(t, mPath, cPath, "-t", "packages", "-j")
		assert.Contains(t, out, `"Name": "htop"`)
		assert.Contains(t, out, `"Origin": "stamped"`)
		assert.Contains(t, out, `"Notes": "system monitor"`)
	})

	t.Run("repos", func(t *testing.T) {
		out := runListCmd(t, mPath, cPath, "-t", "repos", "-j")
		assert.Contains(t, out, `"Name": "my-tap"`)
		assert.Contains(t, out, `"URL": "https://example.com/brave"`)
	})

	t.Run("stamped combined flat array", func(t *testing.T) {
		out := runListCmd(t, mPath, cPath, "-t", "stamped", "-j")
		assert.Contains(t, out, `"Name": "htop"`)
		assert.Contains(t, out, `"Name": "my-tap"`)
		htopIdx := strings.Index(out, `"Name": "htop"`)
		tapIdx := strings.Index(out, `"Name": "my-tap"`)
		require.Greater(t, htopIdx, -1)
		assert.Greater(t, tapIdx, htopIdx, "packages must precede repositories in combined output")
	})

	t.Run("reconciled excludes stamped", func(t *testing.T) {
		out := runListCmd(t, mPath, cPath, "-t", "reconciled", "-j")
		assert.Contains(t, out, `"Name": "NetworkManager"`)
		assert.NotContains(t, out, `"Name": "htop"`)
	})
}

func TestListCmd_TypeManager(t *testing.T) {
	mPath, cPath := writeManifestWithOrigins(t)

	out := runListCmd(t, mPath, cPath, "-t", "stamped", "-m", "brew")
	assert.Contains(t, out, "lazygit (brew)")
	assert.Contains(t, out, "my-tap (brew)")
	assert.NotContains(t, out, "htop")

	out = runListCmd(t, mPath, cPath, "-t", "stamped-repos", "-m", "dnf")
	assert.Contains(t, out, "no repositories tracked")
}

func TestListCmd_TypeInvalid(t *testing.T) {
	mPath, cPath := writeManifestWithOrigins(t)
	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"list", "-t", "xyz"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown type "xyz"`)
	assert.Contains(t, err.Error(), "reconciled-repos")
	assert.Equal(t, ExitUsage, exitCodeFor(err))
}

func TestListCmd_TypeEmpty(t *testing.T) {
	mPath, cPath := writeEmptyManifest(t)
	tests := []struct{ typeFlag, want string }{
		{"packages", "no packages tracked"},
		{"repos", "no repositories tracked"},
		{"stamped", "nothing tracked"},
		{"reconciled", "nothing tracked"},
		{"stamped-packages", "no packages tracked"},
		{"stamped-repos", "no repositories tracked"},
		{"reconciled-packages", "no packages tracked"},
		{"reconciled-repos", "no repositories tracked"},
	}
	for _, tt := range tests {
		t.Run(tt.typeFlag, func(t *testing.T) {
			assert.Contains(t, runListCmd(t, mPath, cPath, "-t", tt.typeFlag), tt.want)
		})
	}
}

func TestListCmd_TypeEmptyJSON(t *testing.T) {
	mPath, cPath := writeEmptyManifest(t)
	for _, typeFlag := range []string{"packages", "repos", "stamped", "reconciled", "stamped-packages", "stamped-repos", "reconciled-packages", "reconciled-repos"} {
		t.Run(typeFlag, func(t *testing.T) {
			assert.Contains(t, runListCmd(t, mPath, cPath, "-t", typeFlag, "-j"), "[]")
		})
	}
}

func TestListCmd_TypePreOriginManifest(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"

[[packages]]
name = "lazygit"
manager = "brew"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))

	out := runListCmd(t, mPath, cPath, "-t", "stamped-packages")
	assert.Contains(t, out, "htop (dnf)")
	assert.Contains(t, out, "lazygit (brew)")

	out = runListCmd(t, mPath, cPath, "-t", "reconciled-packages")
	assert.Contains(t, out, "no packages tracked")
}

func TestListCmd_TypePackagesEqualsDefault(t *testing.T) {
	mPath, cPath := writeManifestWithOrigins(t)
	defaultOut := runListCmd(t, mPath, cPath)
	typedOut := runListCmd(t, mPath, cPath, "-t", "packages")
	assert.Equal(t, defaultOut, typedOut)
}

func TestListCmd_NoteRendering(t *testing.T) {
	mPath, cPath := writeManifestWithOrigins(t)
	out := runListCmd(t, mPath, cPath)
	assert.Contains(t, out, "htop (dnf) — system monitor")
	assert.Contains(t, out, "lazygit (brew)\n")
}

func TestListTypeCompletion(t *testing.T) {
	t.Parallel()
	got, directive := listTypeCompletion(nil, nil, "")
	assert.Equal(t, validListTypes, got)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

// writeMissingManifest writes a manifest with packages spread across managers.
func writeMissingManifest(t *testing.T) (manifestPath, configPath string) {
	t.Helper()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"

[[packages]]
name = "lazygit"
manager = "brew"

[[packages]]
name = "jq"
manager = "brew"
`
	require.NoError(t, os.WriteFile(mPath, []byte(content), 0600))
	return mPath, cPath
}

func TestListCmd_TypeMissing_TTY(t *testing.T) {
	mPath, cPath := writeMissingManifest(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}},
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"jq"}},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing")
	assert.Contains(t, out, "lazygit (brew)")
	assert.NotContains(t, out, "htop")
	assert.NotContains(t, out, "jq")
}

func TestListCmd_TypeMissing_JSON(t *testing.T) {
	mPath, cPath := writeMissingManifest(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}},
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"jq"}},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing", "--json")

	var pkgs []manifest.Package
	require.NoError(t, json.Unmarshal([]byte(out), &pkgs))
	require.Len(t, pkgs, 1)
	assert.Equal(t, "lazygit", pkgs[0].Name)
	assert.Equal(t, "brew", pkgs[0].Manager)
}

func TestListCmd_TypeMissing_NoneMissing(t *testing.T) {
	mPath, cPath := writeMissingManifest(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}},
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"lazygit", "jq"}},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing")
	assert.Contains(t, out, "no missing packages")
}

func TestListCmd_TypeMissing_NoneMissing_JSON(t *testing.T) {
	mPath, cPath := writeMissingManifest(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}},
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"lazygit", "jq"}},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing", "--json")
	assert.Contains(t, out, "[]")
	assert.NotContains(t, out, "no missing packages")
}

func TestListCmd_TypeMissing_ManagerFilter(t *testing.T) {
	mPath, cPath := writeMissingManifest(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "dnf", InstalledPkgs: nil},
		&manager.Mock{ManagerName: "brew", InstalledPkgs: nil},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing", "-m", "brew")
	assert.Contains(t, out, "lazygit (brew)")
	assert.Contains(t, out, "jq (brew)")
	assert.NotContains(t, out, "htop")
}

func TestListCmd_TypeMissing_ListErrorSkipped(t *testing.T) {
	mPath, cPath := writeMissingManifest(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", ListErr: errors.New("boom")},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing")
	assert.Contains(t, out, "no missing packages")
}

func TestListCmd_TypeMissing_GroupsAndCasksExcluded(t *testing.T) {
	mPath, cPath := writeManifestWithOrigins(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "dnf", InstalledPkgs: nil},
		&manager.Mock{ManagerName: "brew", InstalledPkgs: nil},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing")
	assert.Contains(t, out, "htop (dnf)")
	assert.NotContains(t, out, "no missing packages")
}

func TestListCmd_TypeMissing_EmptyManifest(t *testing.T) {
	mPath, cPath := writeEmptyManifest(t)
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: nil},
	}
	out := runListCmdWithAdapters(t, mPath, cPath, adapters, "-t", "missing")
	assert.Contains(t, out, "no missing packages")
}
