package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// failingHoldAdapter wraps a working Mock with a Hold that always fails with
// the configured error.
type failingHoldAdapter struct {
	manager.Adapter
	name string
	err  error
}

func (f *failingHoldAdapter) Name() string {
	return f.name
}

func (f *failingHoldAdapter) Hold(_ context.Context, _ string) error {
	return f.err
}

// failingListHeldAdapter wraps a working Mock with a ListHeld that always
// fails with the configured error.
type failingListHeldAdapter struct {
	manager.Adapter
	name string
	err  error
}

func (f *failingListHeldAdapter) Name() string {
	return f.name
}

func (f *failingListHeldAdapter) ListHeld(_ context.Context) ([]string, error) {
	return nil, f.err
}

func TestResolveTargets_EmptyFlagReturnsAll(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
		&manager.Mock{ManagerName: "brew"},
	}
	got, err := resolveTargets(adapters, "")
	require.NoError(t, err)
	assert.Equal(t, adapters, got)
}

func TestResolveTargets_ScopesToSingleMatch(t *testing.T) {
	t.Parallel()
	apt := &manager.Mock{ManagerName: "apt"}
	brew := &manager.Mock{ManagerName: "brew"}
	got, err := resolveTargets([]manager.Adapter{apt, brew}, "apt")
	require.NoError(t, err)
	assert.Equal(t, []manager.Adapter{apt}, got)
}

func TestResolveTargets_UnknownManagerErrors(t *testing.T) {
	t.Parallel()
	_, err := resolveTargets([]manager.Adapter{&manager.Mock{ManagerName: "apt"}}, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestHoldCmd_FallthroughToSecondManager(t *testing.T) {
	buf, err := execCmd(t, []string{"hold", "nginx", "-y"}, []manager.Adapter{
		&mockAdapter{name: "dnf"},             // Hold → ErrNotSupported, skipped
		&manager.Mock{ManagerName: "flatpak"}, // succeeds
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "held nginx via flatpak")
}

func TestHoldCmd_GenericErrorWrapped(t *testing.T) {
	_, err := execCmd(t, []string{"hold", "nginx", "-y"}, []manager.Adapter{
		&failingHoldAdapter{Adapter: &manager.Mock{}, name: "apt", err: errors.New("dpkg locked")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hold nginx via apt")
	assert.Contains(t, err.Error(), "dpkg locked")
}

func TestHeldCmd_WarnsOnListFailureAndContinues(t *testing.T) {
	buf, err := execCmd(t, []string{"held"}, []manager.Adapter{
		&failingListHeldAdapter{Adapter: &manager.Mock{}, name: "apt", err: errors.New("boom")},
		&manager.Mock{ManagerName: "brew", HeldPkgs: []string{"nginx"}},
	})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "warning: apt held failed: boom")
	assert.Contains(t, out, "nginx (brew)")
}

func TestHoldCmd_Success(t *testing.T) {
	buf, err := execCmd(t, []string{"hold", "nginx", "-m", "apt", "-y"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "held nginx via apt")
}

func TestHoldCmd_UnsupportedManager(t *testing.T) {
	// mockAdapter.Hold returns ErrNotSupported → CLI skips it and falls through
	_, err := execCmd(t, []string{"hold", "nginx", "-m", "dnf", "-y"}, []manager.Adapter{
		&mockAdapter{name: "dnf"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no manager supports hold")
}

func TestHoldCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"hold", "nginx", "-m", "nonexistent"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestUnholdCmd_Success(t *testing.T) {
	buf, err := execCmd(t, []string{"unhold", "nginx", "-m", "apt", "-y"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "unheld nginx via apt")
}

func TestUnholdCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"unhold", "nginx", "-m", "nonexistent"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestHeldCmd_WithResults(t *testing.T) {
	buf, err := execCmd(t, []string{"held"}, []manager.Adapter{
		&manager.Mock{
			ManagerName: "apt",
			HeldPkgs:    []string{"nginx", "redis"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nginx (apt)")
	assert.Contains(t, buf.String(), "redis (apt)")
}

func TestHeldCmd_NoResults(t *testing.T) {
	buf, err := execCmd(t, []string{"held"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no packages held")
}

func TestHeldCmd_WithManagerFlag(t *testing.T) {
	buf, err := execCmd(t, []string{"held", "-m", "apt"}, []manager.Adapter{
		&manager.Mock{
			ManagerName: "apt",
			HeldPkgs:    []string{"nginx"},
		},
		&manager.Mock{ManagerName: "brew"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nginx (apt)")
	assert.NotContains(t, buf.String(), "brew")
}

func TestHeldCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"held", "-m", "nonexistent"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}
