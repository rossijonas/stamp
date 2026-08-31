package manager

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrew_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "jq\nfzf\ntmux\n", nil, false, []string{"jq", "fzf", "tmux"}},
		{"error", "", assert.AnError, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)
			assert.Equal(t, "brew", mgr.Name())
			res, err := mgr.ListInstalled(WithYes(context.Background()))
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.wantRes, res)
			}
		})
	}
}

func TestBrew_Install(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "htop", nil, false},
		{"error", "htop", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Install(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrew_Remove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "htop", nil, false},
		{"error", "htop", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Remove(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrew_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "htop\nhtop-debuginfo\n", nil, false, []string{"htop", "htop-debuginfo"}},
		{"error", "", assert.AnError, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)
			res, err := mgr.Search(WithYes(context.Background()), "htop")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.wantRes, res)
			}
		})
	}
}

func TestBrew_Search_Validation(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", nil)
	_, err := mgr.Search(WithYes(context.Background()), "-invalid")
	require.Error(t, err)
}

func TestBrew_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "Name: htop\nVersion: 3.4.1\n", nil, false},
		{"error", "", assert.AnError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper(tt.mockOut, tt.mockErr)
			_, err := mgr.Info(WithYes(context.Background()), "htop")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrew_Info_Validation(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", nil)
	_, err := mgr.Info(WithYes(context.Background()), "-invalid")
	require.Error(t, err)
}

func TestBrew_Reinstall(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"error", assert.AnError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Reinstall(WithYes(context.Background()), "htop")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrew_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "", nil, false},
		{"error", "", assert.AnError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Update(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrew_Update_Single(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", nil)
	err := mgr.Update(WithYes(context.Background()), "htop")
	require.NoError(t, err)
}

func TestBrew_Doctor_Success(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("mock doctor: all good", nil)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.NoError(t, err)
}

func TestBrew_AddRepo_Copr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"error", assert.AnError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.AddRepo(WithYes(context.Background()), "petersen/cava", "")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrew_AddRepo_URL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"error", assert.AnError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.AddRepo(WithYes(context.Background()), "google-chrome", "http://example.com/repo")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBrew_RemoveRepo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"error", assert.AnError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewBrew()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.RemoveRepo(WithYes(context.Background()), "petersen/cava")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
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

	repos, err := manager.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "aovestdipaperino/tap", repos[0].Name)
	assert.Equal(t, "yvgude/lean-ctx", repos[1].Name)
}

func TestBrew_CleanError(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Clean(WithYes(context.Background()), false)
	require.Error(t, err)
}

func TestBrew_Doctor_Error(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brew doctor failed")
}

func TestBrew_IsCask_ExecError(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.IsCask(WithYes(context.Background()), "firefox")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check cask status")
}

func TestBrew_AutoRemove_Error(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.AutoRemove(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to autoremove")
}

func TestBrew_CheckUpdate_OutdatedFails_AfterRefresh(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestBrew_CheckUpdate_Pkg_NotFound(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper(`{"formulae":[]}`, nil)
	updates, err := mgr.CheckUpdate(WithYes(context.Background()), "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, updates)
}

func TestBrew_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestBrew_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper(`{"formulae":[{"name":"htop","installed_versions":["3.2.1"],"current_version":"3.2.2"}]}`, nil)

	updates, err := manager.CheckUpdate(WithYes(context.Background()), "")
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
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestBrew_ListInstalled_UsesLeavesInstalledOnRequest(t *testing.T) {
	t.Parallel()
	var firstCallArgs []string
	call := 0
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		call++
		if call == 1 {
			firstCallArgs = append([]string{}, args...)
			return []byte("htop\n"), nil
		}
		return nil, assert.AnError // cask list fails — fine
	}

	pkgs, err := manager.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop"}, pkgs)
	// Reliability lock: brew leaves --installed-on-request returns only
	// user-requested formulas (not dependencies), keeping reconcile precise.
	assert.Equal(t, []string{"leaves", "--installed-on-request"}, firstCallArgs)
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

	pkgs, err := manager.ListInstalled(WithYes(context.Background()))
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

	pkgs, err := manager.ListInstalled(WithYes(context.Background()))
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

	err := manager.Install(WithCask(WithYes(context.Background())), "firefox")
	require.NoError(t, err)
}

func TestBrew_CaskRemove(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"uninstall", "--cask", "firefox"}, args)
		return []byte(""), nil
	}

	err := manager.Remove(WithCask(WithYes(context.Background())), "firefox")
	require.NoError(t, err)
}

func TestBrew_CaskReinstall(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"reinstall", "--cask", "firefox"}, args)
		return []byte(""), nil
	}

	err := manager.Reinstall(WithCask(WithYes(context.Background())), "firefox")
	require.NoError(t, err)
}

func TestBrew_IsCask_True(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("firefox: 1.0\n"), nil
	}

	result, err := manager.IsCask(WithYes(context.Background()), "firefox")
	require.NoError(t, err)
	assert.True(t, result)
}

func TestBrew_IsCask_False(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("Error: No available Cask")
	}

	result, err := manager.IsCask(WithYes(context.Background()), "htop")
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

	err := manager.Update(WithYes(context.Background()), "")
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
	err := mgr.Update(WithYes(context.Background()), "")
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
	err := mgr.Update(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upgrade packages")
	assert.Equal(t, 2, call)
}

func TestBrew_Update_BatchPipesYesWhenConsent(t *testing.T) {
	t.Parallel()
	var stdins []string
	mgr := NewBrew()
	mgr.exec = func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		stdins = append(stdins, getStdInString(ctx))
		return []byte(""), nil
	}
	err := mgr.Update(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, stdins, 3, "batch update should run update, upgrade, upgrade --cask")
	for i, s := range stdins {
		assert.Equal(t, "y\n", s, "batch update call %d should pipe yes under consent", i+1)
	}
}

func TestBrew_ListReposError(t *testing.T) {
	t.Parallel()
	manager := NewBrew()
	manager.exec = mockExecutorHelper("", assert.AnError)

	_, err := manager.ListRepos(WithYes(context.Background()))
	require.Error(t, err)
}

func TestBrew_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	_, err := mgr.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestBrew_AutoRemove(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = mockExecutorHelper("", nil)
	_, err := mgr.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
}

func TestBrew_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	pkgs, err := mgr.AutoRemove(WithYes(context.Background()), true)
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
	result, err := mgr.Clean(WithYes(context.Background()), true)
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
	result, err := mgr.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestBrew_Hold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	err := mgr.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestBrew_Unhold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	err := mgr.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestBrew_ListHeld_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestBrew_PreviewReinstall_Unsupported(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		t.Fatal("exec must not be called: brew reinstall has no --dry-run")
		return nil, nil
	}
	_, err := mgr.PreviewReinstall(context.Background(), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support a dry-run preview")
	assert.NotContains(t, err.Error(), "Usage: brew reinstall")
	assert.NotContains(t, err.Error(), "invalid option")
}

func TestBrewExecCtx_WithYesPipesYes(t *testing.T) {
	t.Parallel()
	ctx := brewExecCtx(WithYes(context.Background()))
	require.True(t, isStreamIO(ctx))
	require.Equal(t, "y\n", getStdInString(ctx))
}

func TestBrewExecCtx_WithoutYesNoStdinString(t *testing.T) {
	t.Parallel()
	ctx := brewExecCtx(context.Background())
	require.True(t, isStreamIO(ctx))
	require.Empty(t, getStdInString(ctx))
}

func TestBrew_InstallWithoutConsentFails(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	err := mgr.Install(context.Background(), "htop")
	require.ErrorIs(t, err, ErrConfirmationRequired)
}

func TestBrew_AddRepo_TapsAndTrusts(t *testing.T) {
	t.Parallel()
	var calls [][]string
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	err := mgr.AddRepo(WithYes(context.Background()), "user/repo", "")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"tap", "user/repo"}, {"trust", "user/repo"}}, calls)
}

func TestBrew_AddRepo_TrustFailureWarnsNotFails(t *testing.T) {
	t.Parallel()
	var calls [][]string
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		if args[0] == "trust" {
			return nil, assert.AnError
		}
		return nil, nil
	}
	err := mgr.AddRepo(WithYes(context.Background()), "user/repo", "")
	require.NoError(t, err) // tap succeeded; trust failure is best-effort
	require.Equal(t, [][]string{{"tap", "user/repo"}, {"trust", "user/repo"}}, calls)
}

func TestBrew_RemoveRepo_UntapsAndUntrusts(t *testing.T) {
	t.Parallel()
	var calls [][]string
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	err := mgr.RemoveRepo(WithYes(context.Background()), "user/repo")
	require.NoError(t, err)
	require.Equal(t, [][]string{{"untap", "user/repo"}, {"untrust", "user/repo"}}, calls)
}

func TestBrew_Trust(t *testing.T) {
	t.Parallel()
	var calls [][]string
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	require.NoError(t, mgr.Trust(WithYes(context.Background()), "user/repo"))
	require.Equal(t, [][]string{{"trust", "user/repo"}}, calls)
}

func TestBrew_Untrust(t *testing.T) {
	t.Parallel()
	var calls [][]string
	mgr := NewBrew()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	require.NoError(t, mgr.Untrust(WithYes(context.Background()), "user/repo"))
	require.Equal(t, [][]string{{"untrust", "user/repo"}}, calls)
}

func TestBrew_TrustWithoutConsentFails(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	err := mgr.Trust(context.Background(), "user/repo")
	require.ErrorIs(t, err, ErrConfirmationRequired)
	err = mgr.Untrust(context.Background(), "user/repo")
	require.ErrorIs(t, err, ErrConfirmationRequired)
}

func TestBrew_TrustInvalidName(t *testing.T) {
	t.Parallel()
	mgr := NewBrew()
	err := mgr.Trust(WithYes(context.Background()), "-formula")
	require.Error(t, err)
	err = mgr.Untrust(WithYes(context.Background()), "-formula")
	require.Error(t, err)
}
