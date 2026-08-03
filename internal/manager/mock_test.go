package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMock(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "mock",
		InstalledPkgs: []string{"git", "curl"},
		AvailablePkgs: []string{"git", "curl", "htop", "jq", "docker"},
	}

	ctx := WithYes(context.Background())

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

	// Test Doctor
	result, err := mock.Doctor(ctx)
	require.NoError(t, err)
	assert.Contains(t, result, "mock doctor")
}

func TestMockErrors(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("simulated error")
	mock := &Mock{
		ListErr:       expectedErr,
		InstallErr:    expectedErr,
		ReinstallErr:  expectedErr,
		RemoveErr:     expectedErr,
		SearchErr:     expectedErr,
		AddRepoErr:    expectedErr,
		RemoveRepoErr: expectedErr,
		UpdateErr:     expectedErr,
		DoctorErr:     expectedErr,
	}

	ctx := WithYes(context.Background())

	_, err := mock.ListInstalled(ctx)
	require.ErrorIs(t, err, expectedErr)

	err = mock.Install(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	err = mock.Reinstall(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	err = mock.Remove(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	_, err = mock.Search(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	err = mock.AddRepo(ctx, "repo", "url")
	require.ErrorIs(t, err, expectedErr)

	err = mock.RemoveRepo(ctx, "repo")
	require.ErrorIs(t, err, expectedErr)

	err = mock.Update(ctx, "")
	require.ErrorIs(t, err, expectedErr)

	_, err = mock.Doctor(ctx)
	require.ErrorIs(t, err, expectedErr)
}

func TestMockRemove_Nonexistent(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "test",
		InstalledPkgs: []string{"git"},
	}
	ctx := WithYes(context.Background())
	err := mock.Remove(ctx, "nonexistent")
	require.NoError(t, err) // removing uninstalled package doesn't fail
	assert.Len(t, mock.InstalledPkgs, 1)
}

func TestMock_InstallInvalidName(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "test"}
	ctx := WithYes(context.Background())
	err := mock.Install(ctx, "-invalid")
	require.Error(t, err)
}

func TestMock_Provides(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:    "test",
		ProvidesResult: []string{"htop (test)"},
	}

	results, err := mock.Provides(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop (test)"}, results)
}

func TestMock_ProvidesError(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "test",
		ProvidesErr: assert.AnError,
	}

	_, err := mock.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestMock_AutoRemove(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:      "test",
		AutoRemoveResult: []string{"orphan1", "orphan2"},
	}

	results, err := mock.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"orphan1", "orphan2"}, results)
}

func TestMock_AutoRemoveError(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "test",
		AutoRemoveErr: assert.AnError,
	}

	_, err := mock.AutoRemove(WithYes(context.Background()), false)
	require.Error(t, err)
}

func TestMock_Provides_FilePaths(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "test"}
	// Provides queries accept file paths, not just package names
	result, err := mock.Provides(WithYes(context.Background()), "/usr/bin/htop")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestMock_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:      "test",
		AutoRemoveResult: []string{"orphan1"},
	}

	results, err := mock.AutoRemove(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Equal(t, []string{"orphan1"}, results)
}

func TestMock_Clean(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "test",
		CleanResult: []string{"cleaned 1.2MB"},
	}

	results, err := mock.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"cleaned 1.2MB"}, results)
}

func TestMock_CleanError(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "test",
		CleanErr:    assert.AnError,
	}

	_, err := mock.Clean(WithYes(context.Background()), false)
	require.Error(t, err)
}

func TestMock_CheckUpdate(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "brew",
		CheckUpdates: []UpdateInfo{
			{Package: "htop", CurrentVersion: "3.2.1", AvailableVersion: "3.2.2"},
		},
	}
	updates, err := mock.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "3.2.2", updates[0].AvailableVersion)

	// Scoped to a package
	updates, err = mock.CheckUpdate(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	require.Len(t, updates, 1)

	// Non-existent package
	updates, err = mock.CheckUpdate(WithYes(context.Background()), "missing")
	require.NoError(t, err)
	assert.Empty(t, updates)

	// Error case
	mock.CheckUpdateErr = assert.AnError
	_, err = mock.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestMock_SearchInvalidName(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "test"}
	ctx := WithYes(context.Background())
	_, err := mock.Search(ctx, "-invalid")
	require.Error(t, err)
}

func TestMock_RemoveRepoNonexistent(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "test"}
	ctx := WithYes(context.Background())
	err := mock.RemoveRepo(ctx, "nonexistent")
	require.NoError(t, err) // removing uninstalled repo doesn't fail
}

func TestMockInfo_Result(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "brew",
		InfoResult:  "Name: htop\nVersion: 3.4.1\n",
	}
	res, err := mock.Info(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	assert.Equal(t, "Name: htop\nVersion: 3.4.1\n", res)
}

func TestMockInfo_Error(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("not found")
	mock := &Mock{
		ManagerName: "brew",
		InfoErr:     expectedErr,
	}
	_, err := mock.Info(WithYes(context.Background()), "htop")
	require.ErrorIs(t, err, expectedErr)
}

func TestMockInfo_Fallback(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "brew"}
	res, err := mock.Info(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	assert.Contains(t, res, "Name: htop")
	assert.Contains(t, res, "Version: 1.0.0")
	assert.Contains(t, res, "mock details")
}

func TestMockInfo_InvalidName(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "brew"}
	_, err := mock.Info(WithYes(context.Background()), "-invalid")
	require.Error(t, err)
}

func TestMockRemove_Found(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"htop", "git", "curl"},
	}
	ctx := WithYes(context.Background())
	err := mock.Remove(ctx, "htop")
	require.NoError(t, err)
	assert.NotContains(t, mock.InstalledPkgs, "htop")
	assert.Contains(t, mock.InstalledPkgs, "git")
	assert.Contains(t, mock.InstalledPkgs, "curl")
}

func TestMockListRepos_ReturnsInstalledRepos(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "brew",
		InstalledRepos: []RepositoryInfo{
			{Name: "aovestdipaperino/tap"},
			{Name: "yvgude/lean-ctx"},
		},
	}
	repos, err := mock.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "aovestdipaperino/tap", repos[0].Name)
	assert.Equal(t, "yvgude/lean-ctx", repos[1].Name)
}

func TestMockListRepos_ReturnsTrackedRepos(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:  "brew",
		TrackedRepos: []string{"old-tap"},
	}
	repos, err := mock.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "old-tap", repos[0].Name)
}

func TestMockListRepos_Error(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:  "brew",
		ListReposErr: assert.AnError,
	}
	_, err := mock.ListRepos(WithYes(context.Background()))
	require.Error(t, err)
}

func TestMockReinstall_Success(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"htop"},
	}
	err := mock.Reinstall(WithYes(context.Background()), "htop")
	require.NoError(t, err)
}

func TestMockReinstall_AddsPackage(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "brew",
		InstalledPkgs: []string{"htop", "git"},
	}
	err := mock.Reinstall(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	assert.Contains(t, mock.InstalledPkgs, "htop")
	// Should have removed and re-added, but same package name so length unchanged
	assert.Len(t, mock.InstalledPkgs, 2)
}

func TestMockReinstall_Error(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:  "brew",
		ReinstallErr: assert.AnError,
	}
	err := mock.Reinstall(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestMockReinstall_InvalidName(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "brew"}
	err := mock.Reinstall(WithYes(context.Background()), "-invalid")
	require.Error(t, err)
}

func TestMock_Hold_Success(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "apt"}
	err := mock.Hold(WithYes(context.Background()), "nginx")
	require.NoError(t, err)
	assert.Contains(t, mock.HeldPkgs, "nginx")
}

func TestMock_Hold_AlreadyHeld(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "apt",
		HeldPkgs:    []string{"nginx"},
	}
	err := mock.Hold(WithYes(context.Background()), "nginx")
	require.NoError(t, err)
	assert.Len(t, mock.HeldPkgs, 1)
}

func TestMock_Hold_Error(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "apt",
		HoldErr:     assert.AnError,
	}
	err := mock.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
}

func TestMock_Unhold_Success(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "apt",
		HeldPkgs:    []string{"nginx", "redis"},
	}
	err := mock.Unhold(WithYes(context.Background()), "nginx")
	require.NoError(t, err)
	assert.NotContains(t, mock.HeldPkgs, "nginx")
	assert.Contains(t, mock.HeldPkgs, "redis")
}

func TestMock_Unhold_NotHeld(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "apt"}
	err := mock.Unhold(WithYes(context.Background()), "nginx")
	require.NoError(t, err) // not an error — nothing to remove
}

func TestMock_Unhold_Error(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "apt",
		UnholdErr:   assert.AnError,
	}
	err := mock.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
}

func TestMock_ListHeld_WithResults(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "apt",
		HeldPkgs:    []string{"nginx", "redis"},
	}
	pkgs, err := mock.ListHeld(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"nginx", "redis"}, pkgs)
}

func TestMock_ListHeld_Empty(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "apt"}
	pkgs, err := mock.ListHeld(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}
func TestMock_ListHeld_Error(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "apt",
		ListHeldErr: assert.AnError,
	}

	_, err := mock.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
}

func TestMock_Update_Error(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName: "brew",
		UpdateErr:   assert.AnError,
	}
	err := mock.Update(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestMock_Doctor_Result(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:  "brew",
		DoctorResult: "mock doctor: healthy",
	}
	res, err := mock.Doctor(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Contains(t, res, "healthy")
}

func TestMock_Provides_AvailablePkgs(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "brew",
		AvailablePkgs: []string{"htop", "htop-debug", "jq"},
	}
	results, err := mock.Provides(WithYes(context.Background()), "htop")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop", "htop-debug"}, results)
}

func TestMock_AutoRemove_WithResult(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:      "brew",
		AutoRemoveResult: []string{"libfoo"},
	}
	results, err := mock.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"libfoo"}, results)
}

func TestMock_EmptyResults(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "brew"}
	results, err := mock.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.Nil(t, results)

	clean, err := mock.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.Nil(t, clean)

	held, err := mock.ListHeld(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Nil(t, held)
}
