package manager

import (
	"context"
	"fmt"
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
			name:      "update single package",
			operation: "update_single",
			pkgName:   "htop",
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

func TestParseBrewOutdatedJSON(t *testing.T) {
	t.Parallel()
	input := []byte(`{"formulae":[{"name":"htop","installed_versions":["3.2.1"],"current_version":"3.2.2"}]}`)
	updates, err := parseBrewOutdatedJSON(input)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "3.2.2", updates[0].AvailableVersion)
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

func TestBrew_CleanError(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Clean(context.Background(), false)
	require.Error(t, err)
}

func TestBrew_Doctor_Error(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brew doctor failed")
}

func TestBrew_IsCask_ExecError(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.IsCask(context.Background(), "firefox")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check cask status")
}

func TestBrew_AutoRemove_Error(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.AutoRemove(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to autoremove")
}

func TestBrew_CheckUpdate_OutdatedFails_AfterRefresh(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestBrew_CheckUpdate_Pkg_NotFound(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper(`{"formulae":[]}`, nil)
	updates, err := mgr.CheckUpdate(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, updates)
}

func TestBrew_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestBrew_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper(`{"formulae":[{"name":"htop","installed_versions":["3.2.1"],"current_version":"3.2.2"}]}`, nil)

	updates, err := manager.CheckUpdate(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "3.2.2", updates[0].AvailableVersion)
}

func TestBrew_CheckUpdate_RefreshSucceeds_CheckFails(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestBrew_ListInstalled_WithCasks(t *testing.T) {
	t.Parallel()
	call := 0
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			// brew leaves --installed-on-request
			return []byte("htop\njq\n"), nil
		}
		// brew list --cask
		return []byte("firefox\niterm2\n"), nil
	}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop", "jq", "firefox", "iterm2"}, pkgs)
}

func TestBrew_ListInstalled_CaskFail_ReturnsFormulas(t *testing.T) {
	t.Parallel()
	call := 0
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("htop\njq\n"), nil
		}
		// brew list --cask fails — no casks installed
		return nil, assert.AnError
	}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop", "jq"}, pkgs)
}

func TestBrew_CaskInstall(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"install", "--cask", "firefox"}, args)
		return []byte(""), nil
	}

	err := manager.Install(WithCask(context.Background()), "firefox")
	require.NoError(t, err)
}

func TestBrew_CaskRemove(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"uninstall", "--cask", "firefox"}, args)
		return []byte(""), nil
	}

	err := manager.Remove(WithCask(context.Background()), "firefox")
	require.NoError(t, err)
}

func TestBrew_CaskReinstall(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"reinstall", "--cask", "firefox"}, args)
		return []byte(""), nil
	}

	err := manager.Reinstall(WithCask(context.Background()), "firefox")
	require.NoError(t, err)
}

func TestBrew_IsCask_True(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("firefox: 1.0\n"), nil
	}

	result, err := manager.IsCask(context.Background(), "firefox")
	require.NoError(t, err)
	assert.True(t, result)
}

func TestBrew_IsCask_False(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("Error: No available Cask")
	}

	result, err := manager.IsCask(context.Background(), "htop")
	require.NoError(t, err)
	assert.False(t, result)
}

func TestBrew_Update_IncludesCasks(t *testing.T) {
	t.Parallel()
	call := 0
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			assert.Equal(t, []string{"update"}, args)
			return []byte(""), nil
		case 2:
			assert.Equal(t, []string{"upgrade"}, args)
			return []byte(""), nil
		case 3:
			assert.Equal(t, []string{"upgrade", "--cask"}, args)
			return []byte(""), nil
		default:
			return nil, assert.AnError
		}
	}

	err := manager.Update(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 3, call)
}

func TestBrew_Update_CaskFailureIsBestEffort(t *testing.T) {
	t.Parallel()
	call := 0
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte(""), nil // brew update
		case 2:
			return []byte(""), nil // brew upgrade
		case 3:
			return nil, assert.AnError // brew upgrade --cask fails — best-effort
		default:
			return nil, assert.AnError
		}
	}
	err := mgr.Update(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 3, call)
}

func TestBrew_Update_FormulaFailureIsError(t *testing.T) {
	t.Parallel()
	call := 0
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte(""), nil // brew update
		case 2:
			return nil, assert.AnError // brew upgrade fails — should error
		default:
			return nil, assert.AnError
		}
	}
	err := mgr.Update(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upgrade packages")
	assert.Equal(t, 2, call)
}

func TestBrew_ListReposError(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper("", assert.AnError)

	_, err := manager.ListRepos(context.Background())
	require.Error(t, err)
}

func TestBrew_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	_, err := mgr.Provides(context.Background(), "htop")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestBrew_AutoRemove(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", nil)
	_, err := mgr.AutoRemove(context.Background(), false)
	require.NoError(t, err)
}

func TestBrew_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	pkgs, err := mgr.AutoRemove(context.Background(), true)
	require.NoError(t, err)
	assert.Nil(t, pkgs)
}

func TestBrew_CleanDryRun(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"cleanup", "--dry-run"}, args)
		return []byte("Would remove: 1 old version\n"), nil
	}
	result, err := mgr.Clean(context.Background(), true)
	require.NoError(t, err)
	assert.Contains(t, result, "Would remove: 1 old version")
}

func TestBrew_Clean(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"cleanup"}, args)
		return []byte("Removed: 1 old version\n"), nil
	}
	result, err := mgr.Clean(context.Background(), false)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestBrew_Hold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	err := mgr.Hold(context.Background(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestBrew_Unhold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	err := mgr.Unhold(context.Background(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestBrew_ListHeld_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	_, err := mgr.ListHeld(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}
