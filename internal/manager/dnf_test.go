package manager

import (
	"context"
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
				err = manager.Update(ctx)
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
