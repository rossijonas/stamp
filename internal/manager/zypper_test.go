package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZypper_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "S | Name | Summary | Type\n--+------+---------+--------\ni | htop | Interactive process viewer | package\ni | git  | git    | package\n", nil, false, []string{"htop", "git"}},
		{"error", "", assert.AnError, true, nil},
		{"no packages", "S | Name | Summary | Type\n--+------+---------+--------\n", nil, false, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewZypper()
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

func TestZypper_Install(t *testing.T) {
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
			mgr := NewZypper()
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

func TestZypper_Reinstall(t *testing.T) {
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
			mgr := NewZypper()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Reinstall(WithYes(context.Background()), "htop")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestZypper_Remove(t *testing.T) {
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
			mgr := NewZypper()
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

func TestZypper_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "S | Name | Summary | Type\n--+------+---------+--------\n  | htop | Interactive process viewer | package\n", nil, false, []string{"htop"}},
		{"error", "", assert.AnError, true, nil},
		{"validation error", "", nil, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewZypper()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)
			var pkg string
			if tt.wantErr && tt.mockErr == nil {
				pkg = "-invalid"
			} else {
				pkg = "htop"
			}
			res, err := mgr.Search(WithYes(context.Background()), pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.wantRes, res)
			}
		})
	}
}

func TestZypper_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "Name: htop\nVersion: 3.2.2\n", nil, false},
		{"error", "", assert.AnError, true},
		{"validation error", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewZypper()
			mgr.exec = mockExecutorHelper(tt.mockOut, tt.mockErr)
			var pkg string
			if tt.wantErr && tt.mockErr == nil {
				pkg = "-invalid"
			} else {
				pkg = "htop"
			}
			_, err := mgr.Info(WithYes(context.Background()), pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestZypper_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "", nil, false},
		{"error", "", assert.AnError, true},
		{"single package", "htop", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewZypper()
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

func TestZypper_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewZypper()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestZypper_AddRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewZypper()
	err := mgr.AddRepo(WithYes(context.Background()), "repo", "")
	require.Error(t, err)
}

func TestZypper_RemoveRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewZypper()
	err := mgr.RemoveRepo(WithYes(context.Background()), "repo")
	require.Error(t, err)
}

func TestZypper_ListReposNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewZypper()
	res, err := mgr.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestZypper_AutoRemove(t *testing.T) {
	t.Parallel()
	m := NewZypper()
	m.exec = mockExecutorHelper("", nil)
	_, err := m.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
}

func TestZypper_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	m := NewZypper()
	m.exec = mockExecutorHelper("", nil)
	pkgs, err := m.AutoRemove(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Nil(t, pkgs)
}

func TestZypper_AutoRemoveError(t *testing.T) {
	t.Parallel()
	m := NewZypper()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.AutoRemove(WithYes(context.Background()), false)
	require.Error(t, err)
}

func TestZypper_Clean(t *testing.T) {
	t.Parallel()
	m := NewZypper()
	m.exec = mockExecutorHelper("", nil)
	_, err := m.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
}

func TestZypper_CleanDryRun(t *testing.T) {
	t.Parallel()
	m := NewZypper()
	result, err := m.Clean(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestZypper_ProvidesError(t *testing.T) {
	t.Parallel()
	m := NewZypper()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Provides(WithYes(context.Background()), "/usr/bin/htop")
	require.Error(t, err)
}

func TestZypper_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewZypper()
	manager.exec = mockExecutorHelper("S | Repository | Name | Current | Available\n--+------------+------+---------+-----------\nv | main | htop | 3.2.1 | 3.2.2\nv | main | git | 2.43.0 | 2.43.2\n", nil)
	updates, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestZypper_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewZypper()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestZypper_CheckUpdateRefreshFails(t *testing.T) {
	t.Parallel()
	call := 0
	manager := NewZypper()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte(""), nil
		}
		return nil, assert.AnError
	}
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestParseZypperListUpdates(t *testing.T) {
	t.Parallel()
	input := []byte("S | Repository | Name | Current | Available\n--+------------+------+---------+-----------\nv | main | htop | 3.2.1 | 3.2.2\nv | main | git | 2.43.0 | 2.43.2\n")
	updates := parseZypperListUpdates(input)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "git", updates[1].Package)
}

func TestZypper_HoldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewZypper()
	err := mgr.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestZypper_UnholdNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewZypper()
	err := mgr.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestZypper_ListHeldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewZypper()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}
