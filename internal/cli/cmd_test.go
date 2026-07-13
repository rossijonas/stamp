// Package cli provides the Cobra command tree for stamp.
package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// mockAdapter is a test implementation of manager.Adapter.
type mockAdapter struct {
	name          string
	err           error
	searchResults []string // when nil, defaults to []string{q + "-found"}
}

func (m *mockAdapter) Name() string                                      { return m.name }
func (m *mockAdapter) ListInstalled(_ context.Context) ([]string, error) { return nil, m.err }
func (m *mockAdapter) Install(_ context.Context, _ string) error         { return m.err }
func (m *mockAdapter) Remove(_ context.Context, _ string) error          { return m.err }
func (m *mockAdapter) Search(_ context.Context, q string) ([]string, error) {
	if m.searchResults != nil {
		return m.searchResults, m.err
	}
	return []string{q + "-found"}, m.err
}
func (m *mockAdapter) AddRepo(_ context.Context, _, _ string) error { return m.err }
func (m *mockAdapter) RemoveRepo(_ context.Context, _ string) error { return m.err }
func (m *mockAdapter) Info(_ context.Context, q string) (string, error) {
	return "Name: " + q + "\nVersion: 1.0.0", m.err
}
func (m *mockAdapter) Doctor(_ context.Context) (string, error) {
	return "mock doctor: all good", m.err
}

// execCmd builds a root with injected mock adapters and isolated temp paths, executes, returns output.
func execCmd(t *testing.T, args []string, adapters []manager.Adapter) (*bytes.Buffer, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	tmpDir := t.TempDir()
	cPath := filepath.Join(tmpDir, "config.toml")
	mPath := filepath.Join(tmpDir, "manifest.toml")
	root := NewRootCmd(WithAdapters(adapters), WithConfigPath(cPath), WithManifestPath(mPath))
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	root.SetIn(r)
	_ = w.Close() // stdin will read EOF immediately (non-interactive)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf, err
}

func lookupCmd(cmds []*cobra.Command, name string) *cobra.Command {
	for _, c := range cmds {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestSaveManifestError(t *testing.T) {
	readonlyDir := t.TempDir()
	//nolint:gosec // intentionally restrictive permissions for test
	if err := os.Chmod(readonlyDir, 0500); err != nil {
		t.Skip("cannot change permissions:", err)
	}
	mPath := filepath.Join(readonlyDir, "manifest.toml")
	cPath := filepath.Join(readonlyDir, "config.toml")

	buf := new(bytes.Buffer)
	root := NewRootCmd(WithAdapters([]manager.Adapter{&mockAdapter{name: "dnf"}}), WithConfigPath(cPath), WithManifestPath(mPath))
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"install", "htop"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save manifest")
}

func TestXDGConfigDir_Fallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := xdgConfigDir()
	assert.Contains(t, dir, ".config")
	assert.Contains(t, dir, "stamp")
}

func TestXDGConfigDir_FromEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/path")
	dir := xdgConfigDir()
	assert.Equal(t, "/custom/path/stamp", dir)
}

func TestManifestPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/test")
	assert.Contains(t, manifestPath(), "/test/stamp/manifest.toml")
}

func TestConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/test")
	assert.Contains(t, configPath(), "/test/stamp/config.toml")
}

func TestAppFromCtx_Nil(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	ctx := appFromCtx(cmd)
	assert.Nil(t, ctx)
}

func TestWithAdapters(t *testing.T) {
	t.Parallel()
	a := []manager.Adapter{&mockAdapter{name: "test"}}
	opt := WithAdapters(a)
	cfg := &rootConfig{}
	opt(cfg)
	require.Len(t, cfg.adapters, 1)
	assert.Equal(t, "test", cfg.adapters[0].Name())
}

func TestNewRootCmdWithoutOptions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})
	err := root.Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestNewRootCmd_CustomPaths(t *testing.T) {
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("version = 1\n[[packages]]\nname = \"htop\"\nmanager = \"dnf\"\n"), 0600))

	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "--json"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "packages_count\": 1")
}

func TestDetectAdapters_Runs(t *testing.T) {
	t.Parallel()
	adapters := detectAdapters()
	// Should return a slice (possibly empty) without error
	assert.NotNil(t, adapters)
}

// --- Command structure tests ---

func TestRootCmd_CommandRegistration(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	for _, name := range []string{"install", "remove", "search", "repo", "reconcile", "restore"} {
		require.NotNil(t, lookupCmd(root.Commands(), name), "missing command: %s", name)
	}
}

func TestInstallCmd_Aliases(t *testing.T) {
	t.Parallel()
	cmd := lookupCmd(NewRootCmd().Commands(), "install")
	require.NotNil(t, cmd)
	assert.Contains(t, cmd.Aliases, "add")
	require.NoError(t, cmd.ValidateArgs([]string{"pkg"}))
	require.Error(t, cmd.ValidateArgs([]string{}))
	require.Error(t, cmd.ValidateArgs([]string{"a", "b"}))
}

func TestRemoveCmd_Aliases(t *testing.T) {
	t.Parallel()
	cmd := lookupCmd(NewRootCmd().Commands(), "remove")
	require.NotNil(t, cmd)
	assert.Subset(t, cmd.Aliases, []string{"uninstall", "rm", "delete", "del"})
	require.NoError(t, cmd.ValidateArgs([]string{"pkg"}))
	require.Error(t, cmd.ValidateArgs([]string{}))
}

func TestSearchCmd_Flags(t *testing.T) {
	t.Parallel()
	cmd := lookupCmd(NewRootCmd().Commands(), "search")
	require.NotNil(t, cmd)
	require.NoError(t, cmd.ValidateArgs([]string{"q"}))
	require.Error(t, cmd.ValidateArgs([]string{}))

	mFlag := cmd.Flag("manager")
	require.NotNil(t, mFlag)
	assert.Equal(t, "m", mFlag.Shorthand)
}

func TestRepoCmd_Subcommands(t *testing.T) {
	t.Parallel()
	repo := lookupCmd(NewRootCmd().Commands(), "repo")
	require.NotNil(t, repo)
	require.NoError(t, repo.ValidateArgs([]string{}))
	require.Error(t, repo.ValidateArgs([]string{"x"}))

	for _, name := range []string{"add", "remove", "list"} {
		require.NotNil(t, lookupCmd(repo.Commands(), name), "missing repo subcommand: %s", name)
	}
}

func TestRepoAddCmd_AliasesAndFlags(t *testing.T) {
	t.Parallel()
	repo := lookupCmd(NewRootCmd().Commands(), "repo")
	require.NotNil(t, repo)
	add := lookupCmd(repo.Commands(), "add")
	require.NotNil(t, add)
	assert.Contains(t, add.Aliases, "install")

	mFlag := add.Flag("manager")
	require.NotNil(t, mFlag)
	assert.Equal(t, "m", mFlag.Shorthand)
}

func TestRepoRemoveCmd_Aliases(t *testing.T) {
	t.Parallel()
	repo := lookupCmd(NewRootCmd().Commands(), "repo")
	rm := lookupCmd(repo.Commands(), "remove")
	require.NotNil(t, rm)
	assert.Subset(t, rm.Aliases, []string{"uninstall", "delete", "del"})
}

func TestRepoListCmd_Aliases(t *testing.T) {
	t.Parallel()
	repo := lookupCmd(NewRootCmd().Commands(), "repo")
	ls := lookupCmd(repo.Commands(), "list")
	require.NotNil(t, ls)
	assert.Contains(t, ls.Aliases, "ls")
}

func TestGlobalFlags(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	for _, tc := range []struct{ name, short string }{
		{"verbose", "v"}, {"yes", "y"},
	} {
		f := root.PersistentFlags().Lookup(tc.name)
		require.NotNil(t, f, "missing flag: %s", tc.name)
		assert.Equal(t, tc.short, f.Shorthand)
	}
	f := root.PersistentFlags().Lookup("json")
	require.NotNil(t, f)
}

// --- Execution tests with mock adapters ---

func TestInstallCmd_ExecutesInstall(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"install", "htop"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "installed htop via dnf")
}

func TestRemoveCmd_ExecutesRemove(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"remove", "htop"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed htop via brew")
}

func TestRemoveCmd_WithManagerFlag(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"remove", "htop", "-m", "dnf"}, []manager.Adapter{
		&mockAdapter{name: "dnf"}, &mockAdapter{name: "brew"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed htop via dnf")
}

func TestRemoveCmd_UnknownManager(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"remove", "htop", "-m", "nonexistent"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown manager")
}

func TestSearchCmd_FindsResults(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"search", "ripgrep"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "ripgrep-found (dnf)")
}

func TestSearchCmd_ScopedByManager(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"search", "jq", "-m", "brew"}, []manager.Adapter{
		&mockAdapter{name: "dnf"}, &mockAdapter{name: "brew"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "jq-found (brew)")
	assert.NotContains(t, buf.String(), "dnf")
}

func TestSearchCmd_UnknownManager(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"search", "foo", "-m", "nonexistent"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown manager")
}

func TestRemoveCmd_NoAdapters(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"remove", "htop"}, []manager.Adapter{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no package managers available")
}

func TestSearchCmd_NoResults(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"search", "foo"}, []manager.Adapter{&mockAdapter{name: "flatpak", err: errors.New("fail")}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no results found")
}

func TestSearchCmd_NoResultsWithEmpty(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"search", "foo"}, []manager.Adapter{&mockAdapter{name: "flatpak", searchResults: []string{}}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no results found")
}

func TestRepoAddCmd_Executes(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"repo", "add", "mytap", "-m", "brew"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "added repo mytap via brew")
}

func TestRepoRemoveCmd_Executes(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"repo", "remove", "mytap", "-m", "flatpak"}, []manager.Adapter{&mockAdapter{name: "flatpak"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed repo mytap via flatpak")
}

func TestInstallCmd_Error(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"install", "htop"}, []manager.Adapter{&mockAdapter{name: "dnf", err: errors.New("install failed")}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "install failed")
}

func TestInstallCmd_CorruptedManifest(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	root.SetArgs([]string{"install", "htop"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestRemoveCmd_Error(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"remove", "htop"}, []manager.Adapter{&mockAdapter{name: "brew", err: errors.New("remove failed")}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove failed")
}

func TestRemoveCmd_CorruptedManifest(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	root.SetArgs([]string{"remove", "htop"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestRepoAddCmd_Error(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"repo", "add", "mytap", "-m", "brew"}, []manager.Adapter{&mockAdapter{name: "brew", err: errors.New("add repo failed")}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add repo failed")
}

func TestRepoAddCmd_CorruptedManifest(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	root.SetArgs([]string{"repo", "add", "mytap", "-m", "brew"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestRepoRemoveCmd_Error(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"repo", "remove", "mytap", "-m", "flatpak"}, []manager.Adapter{&mockAdapter{name: "flatpak", err: errors.New("remove repo failed")}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove repo failed")
}

func TestRepoRemoveCmd_CorruptedManifest(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&mockAdapter{name: "flatpak"}}
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	root.SetArgs([]string{"repo", "remove", "mytap", "-m", "flatpak"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestRepoListCmd_Empty(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"repo", "list"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no repositories tracked")
}

func TestRepoListCmd_WithEntries(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&mockAdapter{name: "dnf"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "hashicorp/tap"
manager = "brew"
url = "https://github.com/hashicorp/tap"

[[repositories]]
name = "flathub"
manager = "flatpak"
`
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "list"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "hashicorp/tap (brew) https://github.com/hashicorp/tap")
	assert.Contains(t, output, "flathub (flatpak)")
}

func TestRepoAddCmd_MissingManager(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"repo", "add", "mytap"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.Error(t, err)
}

func TestRepoAddCmd_InvalidURL(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"repo", "add", "mytap", "invalid-url", "-m", "brew"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repository URL")
}

func TestRepoRemoveCmd_InvalidName(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"repo", "remove", "-invalid", "-m", "brew"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.Error(t, err)
}

func TestInstallCmd_InvalidName(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"install", "foo;rm"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestAliasesWork(t *testing.T) {
	t.Parallel()
	for _, alias := range []struct{ cmd, pkg string }{
		{"add", "htop"}, {"uninstall", "htop"}, {"rm", "htop"}, {"delete", "htop"}, {"del", "htop"},
	} {
		t.Run(alias.cmd, func(t *testing.T) {
			buf, err := execCmd(t, []string{alias.cmd, alias.pkg}, []manager.Adapter{&mockAdapter{name: "dnf"}})
			require.NoError(t, err)
			assert.True(t, strings.Contains(buf.String(), "installed") || strings.Contains(buf.String(), "removed"),
				"alias %s produced unexpected output: %s", alias.cmd, buf.String())
		})
	}
}

func TestRepoAliasesWork(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, args string }{
		{"repo install", "install"},
		{"repo uninstall", "uninstall"},
		{"repo delete", "delete"},
		{"repo del", "del"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf, err := execCmd(t, []string{"repo", tc.args, "mytap", "-m", "brew"}, []manager.Adapter{&mockAdapter{name: "brew"}})
			require.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestValidateRepoName(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateRepoName("mytap"))
	assert.NoError(t, validateRepoName("hashicorp/tap"))
	require.Error(t, validateRepoName("-invalid"))
	require.Error(t, validateRepoName(""))
}

func TestValidateRepoURL(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateRepoURL(""))
	assert.NoError(t, validateRepoURL("https://example.com/repo"))
	assert.NoError(t, validateRepoURL("http://example.com/repo"))
	require.Error(t, validateRepoURL("ftp://bad.com"))
	assert.Error(t, validateRepoURL("not-a-url"))
}

func TestSearchResultsToStdout(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"search", "htop"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestInstallCmd_WithManagerFlag(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"install", "lazygit", "-m", "brew"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "installed lazygit via brew")
}

func TestInstallCmd_WithNote(t *testing.T) {
	t.Parallel()
	adapter := &mockAdapter{name: "dnf"}
	buf, err := execCmd(t, []string{"install", "htop", "--note", "needed for work"}, []manager.Adapter{adapter})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "installed htop via dnf")
}

func TestRemoveCmd_WithManifestLookup(t *testing.T) {
	t.Parallel()
	// First install adds package to manifest with its manager
	installBuf := new(bytes.Buffer)
	tmpDir := t.TempDir()
	cPath := filepath.Join(tmpDir, "config.toml")
	mPath := filepath.Join(tmpDir, "manifest.toml")
	adapters := []manager.Adapter{&mockAdapter{name: "dnf"}}

	root1 := NewRootCmd(WithAdapters(adapters), WithConfigPath(cPath), WithManifestPath(mPath))
	root1.SetOut(installBuf)
	root1.SetErr(installBuf)
	root1.SetArgs([]string{"install", "htop"})
	require.NoError(t, root1.Execute())

	// Now remove should find htop in manifest and use dnf
	removeBuf := new(bytes.Buffer)
	root2 := NewRootCmd(WithAdapters(adapters), WithConfigPath(cPath), WithManifestPath(mPath))
	root2.SetOut(removeBuf)
	root2.SetErr(removeBuf)
	root2.SetArgs([]string{"remove", "htop"})
	require.NoError(t, root2.Execute())
	assert.Contains(t, removeBuf.String(), "removed htop via dnf")
}

func TestRepoAddCmd_WithURL(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"repo", "add", "flathub", "https://dl.flathub.org/repo/flathub.flatpakrepo", "-m", "flatpak"},
		[]manager.Adapter{&mockAdapter{name: "flatpak"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "added repo flathub via flatpak")
}

func TestResolveAmbiguousInInstall(t *testing.T) {
	t.Parallel()
	// Adapter "apt" not in default precedence → Tier 3 returns "specify --manager"
	_, err := execCmd(t, []string{"install", "htop"}, []manager.Adapter{&mockAdapter{name: "apt"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify --manager")
}

func TestRepoListCmd_WithFlags(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}, &mockAdapter{name: "dnf"}}

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "my-tap"
manager = "brew"
url = "https://github.com/my-tap"

[[repositories]]
name = "flathub"
manager = "flatpak"
`
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	// Test filtering with -m dnf (nothing should be shown since dnf is not in manifest repos)
	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "list", "-m", "dnf"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no repositories tracked")

	// Test filtering with -m brew
	buf2 := new(bytes.Buffer)
	root2 := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	root2.SetOut(buf2)
	root2.SetErr(buf2)
	root2.SetArgs([]string{"repo", "list", "-m", "brew"})
	err2 := root2.Execute()
	require.NoError(t, err2)
	assert.Contains(t, buf2.String(), "my-tap (brew)")
	assert.NotContains(t, buf2.String(), "flathub")

	// Test JSON output
	buf3 := new(bytes.Buffer)
	root3 := NewRootCmd(WithAdapters(adapters), WithManifestPath(mPath), WithConfigPath(cPath))
	root3.SetOut(buf3)
	root3.SetErr(buf3)
	root3.SetArgs([]string{"repo", "list", "--json"})
	err3 := root3.Execute()
	require.NoError(t, err3)
	assert.Contains(t, buf3.String(), `"Name": "my-tap"`)
	assert.Contains(t, buf3.String(), `"Manager": "brew"`)
}

func TestListCmd_Empty(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"list"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no packages tracked")
}

func TestListCmd_WithEntries(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"

[[packages]]
name = "lazygit"
manager = "brew"
`
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"list"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "htop (dnf)")
	assert.Contains(t, output, "lazygit (brew)")
}

func TestListCmd_JSON(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"
`
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"list", "--json"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"Name": "htop"`)
	assert.Contains(t, buf.String(), `"Manager": "dnf"`)
}

func TestListCmd_ManagerFlag(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"

[[packages]]
name = "lazygit"
manager = "brew"
`
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"list", "-m", "brew"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "lazygit (brew)")
	assert.NotContains(t, output, "htop")
}

func TestListCmd_ManagerFlagNoMatch(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"
`
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"list", "-m", "brew"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no packages tracked")
}

func TestListCmd_Aliases(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "dnf"
`
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"ls"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "htop (dnf)")
}

func TestListCmd_EmptyJSON(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"list", "--json"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "[]")
}

func TestListCmd_CorruptedManifest(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	require.NoError(t, os.WriteFile(mPath, []byte("invalid [[toml\n"), 0600))
	root := NewRootCmd(WithManifestPath(mPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	root.SetArgs([]string{"list"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}
