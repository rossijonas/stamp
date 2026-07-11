package manager

import (
	"context"
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
		operation   string // "list", "install", "remove", "search"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewDNF()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "dnf", manager.Name()) // hit the name method

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
			mockErr:     nil, // The method itself throws the error before calling exec
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
					// We only check ErrorIs if we passed a mock error, flatpak throws a native error for empty URLs
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

	// Test Name
	assert.Equal(t, "mock", mock.Name())

	// Test ListInstalled
	installed, err := mock.ListInstalled(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"git", "curl"}, installed)

	// Test Install
	err = mock.Install(ctx, "jq")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.Contains(t, installed, "jq")

	// Test Install Duplicate
	err = mock.Install(ctx, "jq")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	// should still be 3 items
	assert.Len(t, installed, 3)

	// Test Remove
	err = mock.Remove(ctx, "curl")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.NotContains(t, installed, "curl")
	assert.Contains(t, installed, "jq")
	assert.Contains(t, installed, "git")

	// Test Search
	results, err := mock.Search(ctx, "to")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop"}, results)

	// Test Add Repo
	err = mock.AddRepo(ctx, "test-repo", "url")
	require.NoError(t, err)
	assert.Contains(t, mock.TrackedRepos, "test-repo")

	// Test Remove Repo
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
