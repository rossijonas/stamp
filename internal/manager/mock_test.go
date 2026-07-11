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

func TestMockErrors(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("simulated error")
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

func TestMockRemove_Nonexistent(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "test",
		InstalledPkgs: []string{"git"},
	}
	ctx := context.Background()
	err := mock.Remove(ctx, "nonexistent")
	require.NoError(t, err) // removing uninstalled package doesn't fail
	assert.Len(t, mock.InstalledPkgs, 1)
}

func TestMock_InstallInvalidName(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "test"}
	ctx := context.Background()
	err := mock.Install(ctx, "-invalid")
	require.Error(t, err)
}

func TestMock_SearchInvalidName(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "test"}
	ctx := context.Background()
	_, err := mock.Search(ctx, "-invalid")
	require.Error(t, err)
}

func TestMock_RemoveRepoNonexistent(t *testing.T) {
	t.Parallel()
	mock := &Mock{ManagerName: "test"}
	ctx := context.Background()
	err := mock.RemoveRepo(ctx, "nonexistent")
	require.NoError(t, err) // removing uninstalled repo doesn't fail
}
