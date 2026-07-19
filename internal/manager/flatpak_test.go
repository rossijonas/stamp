package manager

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "com.spotify.Client",
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "com.spotify.Client",
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
			name:        "doctor not supported",
			operation:   "doctor",
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
