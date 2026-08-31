package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUv_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "black v24.8.0\n- black\nruff v0.6.1\n- ruff\n", nil, false, []string{"black", "ruff"}},
		{"error", "", assert.AnError, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewUv()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)
			res, err := mgr.ListInstalled(WithYes(context.Background()))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.wantRes, res)
			}
		})
	}
}

func TestUv_Install(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "black", nil, false},
		{"error", "black", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewUv()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Install(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUv_Reinstall(t *testing.T) {
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
			mgr := NewUv()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Reinstall(WithYes(context.Background()), "black")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUv_Remove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "black", nil, false},
		{"error", "black", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewUv()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Remove(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUv_Search(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Search(WithYes(context.Background()), "anything")
	require.Error(t, err)
}

func TestUv_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "black", "black v24.8.0\n- black\n", nil, false},
		{"not found", "missing", "black v24.8.0\n", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewUv()
			mgr.exec = mockExecutorHelper(tt.mockOut, tt.mockErr)
			res, err := mgr.Info(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, res, tt.pkg)
			}
		})
	}
}

func TestUv_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"single success", "black", nil, false},
		{"single error", "black", assert.AnError, true},
		{"all success", "", nil, false},
		{"all error", "", assert.AnError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewUv()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Update(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUv_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestUv_CheckUpdate_Unsupported(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	_, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCheckUnsupported)
}

func TestUv_UnsupportedOperations(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	ctx := WithYes(context.Background())

	err := mgr.AddRepo(ctx, "repo", "url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	err = mgr.RemoveRepo(ctx, "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	repos, err := mgr.ListRepos(ctx)
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestUv_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	_, err := mgr.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestUv_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	_, err := mgr.AutoRemove(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestUv_CleanNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	_, err := mgr.Clean(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestUv_Hold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	err := mgr.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestUv_Unhold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	err := mgr.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestUv_ListHeld_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestUv_Info_Error(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Info(WithYes(context.Background()), "ruff")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get info")
}
