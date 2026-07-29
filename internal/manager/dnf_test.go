package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNF_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		operation   string
		pkgName     string
		mockOutput  string
		mockErr     error
		expectedErr bool
		expectedRes []string
	}{
		{
			name:        "list installed success",
			operation:   "list",
			mockOutput:  "htop\nripgrep\n",
			expectedRes: []string{"htop", "ripgrep"},
		},
		{
			name:        "list installed error",
			operation:   "list",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "install success",
			operation: "install",
			pkgName:   "htop",
			mockErr:   nil,
		},
		{
			name:        "install error",
			operation:   "install",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "htop",
			mockErr:   nil,
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "htop",
			mockErr:   nil,
		},
		{
			name:        "remove error",
			operation:   "remove",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "htop",
			mockOutput:  "htop\nhtop-debuginfo\n",
			expectedRes: []string{"htop", "htop-debuginfo"},
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "htop",
			mockOutput:  "htop\nhtop-debuginfo\n",
			expectedRes: []string{"htop", "htop-debuginfo"},
		},
		{
			name:        "search error",
			operation:   "search",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "add repo success (copr)",
			operation: "addrepo",
			pkgName:   "petersen/cava",
			mockErr:   nil,
		},
		{
			name:        "add repo error",
			operation:   "addrepo",
			pkgName:     "petersen/cava",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "add repo success (url)",
			operation: "addrepo_url",
			pkgName:   "google-chrome",
			mockErr:   nil,
		},
		{
			name:        "add repo error (url)",
			operation:   "addrepo_url",
			pkgName:     "google-chrome",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove repo success",
			operation: "removerepo",
			pkgName:   "petersen/cava",
			mockErr:   nil,
		},
		{
			name:        "remove repo error",
			operation:   "removerepo",
			pkgName:     "petersen/cava",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "install validation error",
			operation:   "install",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:        "remove validation error",
			operation:   "remove",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:        "search validation error",
			operation:   "search",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:       "info success",
			operation:  "info",
			pkgName:    "htop",
			mockOutput: "Name: htop\nVersion: 3.4.1\n",
		},
		{
			name:        "info error",
			operation:   "info",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "info validation error",
			operation:   "info",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:      "update success",
			operation: "update",
		},
		{
			name:        "update error",
			operation:   "update",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "update single package",
			operation: "update_single",
			pkgName:   "htop",
		},
		{
			name:        "doctor not supported",
			operation:   "doctor",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewDNF("dnf")
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "dnf", manager.Name())

			var err error
			ctx := context.Background()

			switch tt.operation {
			case "list":
				res, err := manager.ListInstalled(ctx)
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedRes, res)
				}
			case "install":
				err = manager.Install(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
				}
			case "reinstall":
				err = manager.Reinstall(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
				}
			case "remove":
				err = manager.Remove(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
				}
			case "search":
				res, err := manager.Search(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedRes, res)
				}
			case "addrepo":
				err = manager.AddRepo(ctx, tt.pkgName, "")
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
				}
			case "addrepo_url":
				err = manager.AddRepo(ctx, tt.pkgName, "http://example.com/repo")
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
				}
			case "removerepo":
				err = manager.RemoveRepo(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
				}
			case "info":
				res, err := manager.Info(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
					if tt.mockErr != nil {
						require.ErrorIs(t, err, tt.mockErr)
					}
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.mockOutput, res)
				}
			case "doctor":
				_, err = manager.Doctor(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "update":
				err = manager.Update(ctx, "")
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "update_single":
				err = manager.Update(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestParseDNFHistoryUserInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "standard NEVRA lines",
			input:    "htop-3.2.2-1.fc37.x86_64\nripgrep-13.0.0-4.fc38.x86_64\n",
			expected: []string{"htop", "ripgrep"},
		},
		{
			name:     "with header line",
			input:    "Packages installed by the user:\nhtop-3.2.2-1.fc37.x86_64\n",
			expected: []string{"htop"},
		},
		{
			name:     "complex name",
			input:    "google-chrome-stable-114.0.5735.196-1.x86_64\n",
			expected: []string{"google-chrome-stable"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "short names skipped",
			input:    "foo\n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseDNFHistoryUserInstalled([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestParseDNFRepos(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "filters system repos, keeps custom",
			input: "repo id                     repo name\n" +
				"fedora                      Fedora 44 - x86_64\n" +
				"fedora-updates              Fedora 44 - x86_64 - Updates\n" +
				"copr:copr.fedorainfracloud.org:petersen:cava Copr repo\n" +
				"google-chrome               Google Chrome repo\n",
			expected: []string{
				"copr:copr.fedorainfracloud.org:petersen:cava",
				"google-chrome",
			},
		},
		{
			name:     "only system repos",
			input:    "repo id     repo name\nfedora      Fedora 44\nupdates     Updates\n",
			expected: []string{},
		},
		{
			name:     "empty output",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseDNFRepos([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestDNF_ListRepos(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "enpass-yum.repo"), []byte(
		"[enpass]\n"+
			"name=Enpass\n"+
			"baseurl=https://yum.enpass.io/\n"+
			"enabled=1\n",
	), 0o644))
	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "google-chrome.repo"), []byte(
		"[google-chrome]\n"+
			"name=Google Chrome\n"+
			"baseurl=https://dl.google.com/linux/chrome/rpm/stable/x86_64\n"+
			"enabled=1\n",
	), 0o644))
	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "fedora.repo"), []byte(
		"[fedora]\n"+
			"name=Fedora $releasever - $basearch\n"+
			"metalink=https://mirrors.fedoraproject.org/metalink?repo=fedora-$releasever\n"+
			"enabled=1\n",
	), 0o644))

	repos, err := parseDNFSources()
	require.NoError(t, err)
	require.Len(t, repos, 2)

	names := make(map[string]string)
	for _, r := range repos {
		names[r.Name] = r.URL
	}

	assert.Equal(t, "https://yum.enpass.io/", names["enpass"])
	assert.Equal(t, "https://dl.google.com/linux/chrome/rpm/stable/x86_64", names["google-chrome"])
	assert.NotContains(t, names, "fedora")
}

func TestDNF_ListRepos_EmptyDir(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	repos, err := parseDNFSources()
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestDNF_ListRepos_MissingDir(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = "/nonexistent/repos"
	defer func() { dnfReposDir = oldDir }()

	_, err := parseDNFSources()
	require.Error(t, err)
}

func TestParseDNFCheckUpdate(t *testing.T) {
	t.Parallel()
	input := []byte("htop.x86_64 3.2.1 updates\ngit.noarch 2.43.0 updates\n")
	updates := parseDNFCheckUpdate(input)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "git", updates[1].Package)
	assert.Equal(t, "2.43.0", updates[1].CurrentVersion)
}

func TestDNF_ListInstalledFallback(t *testing.T) {
	t.Parallel()
	calls := 0
	manager := NewDNF("dnf")
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		if calls == 1 {
			return nil, assert.AnError
		}
		return []byte("htop-3.2.2-1.fc37.x86_64\n"), nil
	}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop"}, pkgs)
	assert.Equal(t, 2, calls)
}

func TestDNF_ListInstalledFallbackError(t *testing.T) {
	t.Parallel()
	calls := 0
	manager := NewDNF("dnf")
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		return nil, assert.AnError
	}

	_, err := manager.ListInstalled(context.Background())
	require.Error(t, err)
	assert.Equal(t, 2, calls)
}

func TestDNF_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewDNF("dnf")
	// Plain error (not exit 100) → wrapped error
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestDNF_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewDNF("dnf")
	// dnf check-update exits 100 when updates exist
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Contains(t, args, "check-update")
		return []byte("htop.x86_64 3.2.1 updates\n"), nil
	}
	updates, err := manager.CheckUpdate(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestDNF_ListRepos_ThroughAdapter(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "custom.repo"), []byte(
		"[custom-repo]\n"+
			"name=Custom Repo\n"+
			"baseurl=https://custom.example.com/repo\n"+
			"enabled=1\n",
	), 0o644))

	manager := NewDNF("dnf")
	repos, err := manager.ListRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "custom-repo", repos[0].Name)
	assert.Equal(t, "https://custom.example.com/repo", repos[0].URL)
}

func TestDNF_AddRepo_URL_ExecError(t *testing.T) {
	t.Parallel()
	manager := NewDNF("dnf")
	manager.exec = mockExecutorHelper("", assert.AnError)

	err := manager.AddRepo(context.Background(), "testrepo", "https://example.com/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add repo")
}

func TestSudoCmd_NonTTY(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_ = w.Close()

	oldStdin := stdIn
	stdIn = r
	defer func() {
		stdIn = oldStdin
		_ = r.Close()
	}()

	result := sudoCmd("install", "-y", "htop")
	assert.Equal(t, []string{"sudo", "-n", "install", "-y", "htop"}, result)
}

func TestSudoCmd_StatError(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_ = r.Close()
	_ = w.Close()

	oldStdin := stdIn
	stdIn = r
	defer func() { stdIn = oldStdin }()

	result := sudoCmd("update")
	// Stat returns error on closed pipe → original behavior: no -n (interactive assumed)
	assert.Equal(t, []string{"sudo", "update"}, result)
}

func TestSudoCmd_WithPassword(t *testing.T) {
	defer ClearSudoPassword()
	SetSudoPassword([]byte("secret"))

	result := sudoCmd("install", "-y", "htop")
	assert.Equal(t, []string{"sudo", "-S", "install", "-y", "htop"}, result)
}

func TestSudoCmd_PasswordOverridesNonTTY(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_ = w.Close()
	defer func() { _ = r.Close() }()

	oldStdin := stdIn
	stdIn = r
	defer func() { stdIn = oldStdin }()

	defer ClearSudoPassword()
	SetSudoPassword([]byte("secret"))

	// Password present → -S, even in non-TTY (not -n)
	result := sudoCmd("update")
	assert.Equal(t, []string{"sudo", "-S", "update"}, result)
}

func TestDNF_ProvidesError(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Provides(context.Background(), "/usr/bin/htop")
	require.Error(t, err)
}

func TestDNF_CheckUpdate_ExitCodeNot100(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("exit status 1")
	}
	_, err := m.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestDNF_CheckUpdate_PkgScoped(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("htop.x86_64 3.2.1 updates\n", nil)
	updates, err := m.CheckUpdate(context.Background(), "htop")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestDNF_Reinstall_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Reinstall(context.Background(), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reinstall")
}

func TestDNF_Clean_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Clean(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clean dnf cache")
}

func TestDNF_AutoRemove_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.AutoRemove(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to autoremove")
}

func TestDNF_AutoRemove(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", nil)
	_, err := m.AutoRemove(context.Background(), false)
	require.NoError(t, err)
}

func TestDNF_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	pkgs, err := m.AutoRemove(context.Background(), true)
	require.NoError(t, err)
	assert.Nil(t, pkgs)
}

func TestDNF_Clean(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", nil)
	_, err := m.Clean(context.Background(), false)
	require.NoError(t, err)
}

func TestDNF_CleanDryRun(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	result, err := m.Clean(context.Background(), true)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDNF_Hold(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", nil)
	err := m.Hold(context.Background(), "nginx")
	require.NoError(t, err)
}

func TestDNF_Unhold(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", nil)
	err := m.Unhold(context.Background(), "nginx")
	require.NoError(t, err)
}

func TestDNF_ListHeld(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("nginx\nredis\n", nil)
	pkgs, err := m.ListHeld(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"nginx", "redis"}, pkgs)
}

func TestDNF_ListHeld_Empty(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", nil)
	pkgs, err := m.ListHeld(context.Background())
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestDNF_Hold_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Hold(context.Background(), "nginx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hold")
}

func TestDNF_Hold_InvalidName(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	err := m.Hold(context.Background(), "-invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestDNF_Unhold_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Unhold(context.Background(), "nginx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unhold")
}

func TestDNF_ListHeld_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.ListHeld(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list held packages")
}

func TestDNF_GroupInstall(t *testing.T) {
	t.Parallel()
	var args []string
	m := NewDNF("dnf")
	m.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := m.Install(WithGroup(context.Background()), "Development Tools")
	require.NoError(t, err)
	assert.Contains(t, args, "group")
	assert.Contains(t, args, "install")
	assert.Contains(t, args, "Development Tools")
}

func TestDNF_GroupInstall_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Install(WithGroup(context.Background()), "Development Tools")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install group")
}

func TestDNF_GroupRemove(t *testing.T) {
	t.Parallel()
	var args []string
	m := NewDNF("dnf")
	m.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := m.Remove(WithGroup(context.Background()), "Development Tools")
	require.NoError(t, err)
	assert.Contains(t, args, "group")
	assert.Contains(t, args, "remove")
	assert.Contains(t, args, "Development Tools")
}

func TestDNF_GroupRemove_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Remove(WithGroup(context.Background()), "Development Tools")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove group")
}

func TestDNF_GroupSearch(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("   Development Tools\n   C Development Tools\n   Backup Client\n", nil)
	pkgs, err := m.Search(WithGroup(context.Background()), "Development")
	require.NoError(t, err)
	assert.Contains(t, pkgs, "Development Tools")
	assert.Contains(t, pkgs, "C Development Tools")
	assert.NotContains(t, pkgs, "Backup Client")
}

func TestDNF_GroupSearch_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Search(WithGroup(context.Background()), "Development")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list groups")
}

func TestDNF_GroupInfo(t *testing.T) {
	t.Parallel()
	var args []string
	m := NewDNF("dnf")
	m.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte("Group: Development Tools\n"), nil
	}
	info, err := m.Info(WithGroup(context.Background()), "Development Tools")
	require.NoError(t, err)
	assert.Contains(t, args, "group")
	assert.Contains(t, args, "info")
	assert.Contains(t, info, "Development Tools")
}

func TestDNF_GroupInfo_Error(t *testing.T) {
	t.Parallel()
	m := NewDNF("dnf")
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Info(WithGroup(context.Background()), "Development Tools")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get group info")
}

func TestParseDNFGroupList(t *testing.T) {
	t.Parallel()
	input := []byte("Installed Environment Groups:\n   Development Tools\nInstalled Groups:\n   C Development Tools\nAvailable Groups:\n   Backup Client\n   LibreOffice\n")
	groups := parseDNFGroupList(input)
	assert.Contains(t, groups, "Development Tools")
	assert.Contains(t, groups, "C Development Tools")
	assert.Contains(t, groups, "Backup Client")
	assert.Contains(t, groups, "LibreOffice")
}
