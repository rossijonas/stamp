package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "htop",
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "htop",
			mockErr:     assert.AnError,
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
			name:       "doctor success",
			operation:  "doctor",
			mockOutput: "mock doctor: all good",
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
