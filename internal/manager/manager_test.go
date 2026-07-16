package manager

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutorHelper returns an Executor that injects a predefined string output.
func mockExecutorHelper(output string, err error) Executor {
	return func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(output), err
	}
}

func TestDNF_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		operation   string // "list", "install", "reinstall", "remove", "search"
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
			}
		})
	}
}

func TestBrew_Operations(t *testing.T) {
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
			mockOutput:  "jq\nfzf\ntmux\n",
			expectedRes: []string{"jq", "fzf", "tmux"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewBrew()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "brew", manager.Name())

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
			}
		})
	}
}

func TestFlatpak_Operations(t *testing.T) {
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
			mockOutput:  "com.spotify.Client\norg.mozilla.firefox\n",
			expectedRes: []string{"com.spotify.Client", "org.mozilla.firefox"},
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
			pkgName:   "com.spotify.Client",
			mockErr:   nil,
		},
		{
			name:        "install error",
			operation:   "install",
			pkgName:     "com.spotify.Client",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "com.spotify.Client",
			mockErr:   nil,
		},
		{
			name:        "remove error",
			operation:   "remove",
			pkgName:     "com.spotify.Client",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "spotify",
			mockOutput:  "com.spotify.Client\n",
			expectedRes: []string{"com.spotify.Client"},
		},
		{
			name:        "search error",
			operation:   "search",
			pkgName:     "spotify",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "add repo error (no url)",
			operation:   "addrepo",
			pkgName:     "flathub",
			mockErr:     nil,
			expectedErr: true,
		},
		{
			name:      "add repo success (url)",
			operation: "addrepo_url",
			pkgName:   "flathub",
			mockErr:   nil,
		},
		{
			name:        "add repo error",
			operation:   "addrepo_url",
			pkgName:     "flathub",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove repo success",
			operation: "removerepo",
			pkgName:   "flathub",
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
			pkgName:    "com.spotify.Client",
			mockOutput: "Name: com.spotify.Client\nVersion: 1.0.0\n",
		},
		{
			name:        "info error",
			operation:   "info",
			pkgName:     "com.spotify.Client",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "info validation error",
			operation:   "info",
			pkgName:     "-invalid",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewFlatpak()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "flatpak", manager.Name())

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
				err = manager.AddRepo(ctx, tt.pkgName, "https://dl.flathub.org/repo/flathub.flatpakrepo")
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
			}
		})
	}
}

func TestMockManager(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "mock",
		InstalledPkgs: []string{"git", "curl"},
		AvailablePkgs: []string{"git", "curl", "htop", "jq", "docker"},
	}

	ctx := context.Background()

	assert.Equal(t, "mock", mock.Name())

	installed, err := mock.ListInstalled(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"git", "curl"}, installed)

	err = mock.Install(ctx, "jq")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.Contains(t, installed, "jq")

	err = mock.Install(ctx, "jq")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.Len(t, installed, 3)

	err = mock.Remove(ctx, "curl")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.NotContains(t, installed, "curl")
	assert.Contains(t, installed, "jq")
	assert.Contains(t, installed, "git")

	results, err := mock.Search(ctx, "to")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop"}, results)

	err = mock.AddRepo(ctx, "test-repo", "url")
	require.NoError(t, err)
	assert.Contains(t, mock.TrackedRepos, "test-repo")

	err = mock.RemoveRepo(ctx, "test-repo")
	require.NoError(t, err)
	assert.NotContains(t, mock.TrackedRepos, "test-repo")
}

func TestMockManagerErrors(t *testing.T) {
	t.Parallel()
	expectedErr := assert.AnError
	mock := &Mock{
		ListErr:       expectedErr,
		InstallErr:    expectedErr,
		RemoveErr:     expectedErr,
		SearchErr:     expectedErr,
		AddRepoErr:    expectedErr,
		RemoveRepoErr: expectedErr,
		ListReposErr:  expectedErr,
	}

	ctx := context.Background()

	_, err := mock.ListInstalled(ctx)
	require.ErrorIs(t, err, expectedErr)

	err = mock.Install(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	err = mock.Remove(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	_, err = mock.Search(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	err = mock.AddRepo(ctx, "repo", "url")
	require.ErrorIs(t, err, expectedErr)

	err = mock.RemoveRepo(ctx, "repo")
	require.ErrorIs(t, err, expectedErr)

	_, err = mock.ListRepos(ctx)
	require.ErrorIs(t, err, expectedErr)
}

func TestParseLines(t *testing.T) {
	t.Parallel()
	input := []byte(" line1 \nline2\n\n  line3  \n")
	expected := []string{"line1", "line2", "line3"}
	actual := parseLines(input)
	assert.ElementsMatch(t, expected, actual)
}

func TestValidatePackageName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pkg   string
		valid bool
	}{
		{"htop", true},
		{"google-chrome", true},
		{"foo_bar.baz+qux", true},
		{"--remove-all", false},
		{"-y", false},
		{"foo;rm -rf /", false},
		{"curl|bash", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			t.Parallel()
			err := ValidatePackageName(tt.pkg)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
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

	// Create fixture .repo files
	//nolint:gosec // test fixtures
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "enpass-yum.repo"), []byte(
		"[enpass]\n"+
			"name=Enpass\n"+
			"baseurl=https://yum.enpass.io/\n"+
			"enabled=1\n",
	), 0o644))
	//nolint:gosec // test fixtures
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "google-chrome.repo"), []byte(
		"[google-chrome]\n"+
			"name=Google Chrome\n"+
			"baseurl=https://dl.google.com/linux/chrome/rpm/stable/x86_64\n"+
			"enabled=1\n",
	), 0o644))
	//nolint:gosec // test fixtures
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

func TestBrew_ListRepos(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper("aovestdipaperino/tap\nyvgude/lean-ctx\n", nil)

	repos, err := manager.ListRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "aovestdipaperino/tap", repos[0].Name)
	assert.Equal(t, "yvgude/lean-ctx", repos[1].Name)
}

func TestBrew_ListReposError(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper("", assert.AnError)

	_, err := manager.ListRepos(context.Background())
	require.Error(t, err)
}

func TestFlatpak_ListRepos(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = mockExecutorHelper(
		"Name\tURL\nflathub\thttps://dl.flathub.org/repo/flathub.flatpakrepo\nflathub-beta\thttps://dl.flathub.org/beta-repo/flathub-beta.flatpakrepo\n",
		nil,
	)

	repos, err := manager.ListRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "flathub", repos[0].Name)
	assert.Contains(t, repos[0].URL, "dl.flathub.org")
	assert.Equal(t, "flathub-beta", repos[1].Name)
}

func TestFlatpak_ListReposError(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = mockExecutorHelper("", assert.AnError)

	_, err := manager.ListRepos(context.Background())
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

func TestFlatpak_ListInstalledMerged(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = func(_ context.Context, cmd string, args ...string) ([]byte, error) {
		switch {
		case cmd == "flatpak" && slices.Contains(args, "--user"):
			return []byte("com.spotify.Client\norg.mozilla.firefox\n"), nil
		case cmd == "flatpak" && slices.Contains(args, "--system"):
			return []byte("org.mozilla.firefox\norg.gimp.GIMP\n"), nil
		default:
			return nil, assert.AnError
		}
	}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"com.spotify.Client", "org.mozilla.firefox", "org.gimp.GIMP"}, pkgs)
}

func TestFlatpak_ListInstalledSkipsHeader(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = func(_ context.Context, cmd string, args ...string) ([]byte, error) {
		switch {
		case cmd == "flatpak" && slices.Contains(args, "--system"):
			return []byte("Application ID\ncom.github.fabiocolacio.marker\ncom.slack.Slack\n"), nil
		default:
			return []byte{}, nil
		}
	}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.NotContains(t, pkgs, "Application ID")
	assert.ElementsMatch(t, []string{"com.github.fabiocolacio.marker", "com.slack.Slack"}, pkgs)
}

func TestFlatpak_ListInstalledBothFail(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = mockExecutorHelper("", assert.AnError)

	_, err := manager.ListInstalled(context.Background())
	require.Error(t, err)
}
