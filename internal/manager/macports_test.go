package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMacPorts_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "The following ports are installed:\n  htop @3.2.2 (active)\n  ripgrep @13.0.0 (active)\n", nil, false, []string{"htop", "ripgrep"}},
		{"error", "", assert.AnError, true, nil},
		{"no ports", "The following ports are installed:\n", nil, false, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewMacPorts()
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

func TestMacPorts_Install(t *testing.T) {
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
			mgr := NewMacPorts()
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

func TestMacPorts_Reinstall(t *testing.T) {
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
			mgr := NewMacPorts()
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

func TestMacPorts_Remove(t *testing.T) {
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
			mgr := NewMacPorts()
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

func TestMacPorts_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "htop @3.2.2 (sysutils, interactive process viewer)\n", nil, false, []string{"htop"}},
		{"error", "", assert.AnError, true, nil},
		{"validation error", "", nil, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewMacPorts()
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

func TestMacPorts_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "htop @3.2.2 (sysutils)\n", nil, false},
		{"error", "", assert.AnError, true},
		{"validation error", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewMacPorts()
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

func TestMacPorts_Update(t *testing.T) {
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
			mgr := NewMacPorts()
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

func TestMacPorts_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewMacPorts()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestMacPorts_AddRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewMacPorts()
	err := mgr.AddRepo(WithYes(context.Background()), "repo", "")
	require.Error(t, err)
}

func TestMacPorts_RemoveRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewMacPorts()
	err := mgr.RemoveRepo(WithYes(context.Background()), "repo")
	require.Error(t, err)
}

func TestMacPorts_ListReposNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewMacPorts()
	res, err := mgr.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestMacPorts_AutoRemove(t *testing.T) {
	t.Parallel()
	m := NewMacPorts()
	m.exec = mockExecutorHelper("", nil)
	_, err := m.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
}

func TestMacPorts_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	m := NewMacPorts()
	m.exec = mockExecutorHelper("", nil)
	pkgs, err := m.AutoRemove(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Nil(t, pkgs)
}

func TestMacPorts_Clean(t *testing.T) {
	t.Parallel()
	m := NewMacPorts()
	m.exec = mockExecutorHelper("", nil)
	_, err := m.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
}

func TestMacPorts_CleanDryRun(t *testing.T) {
	t.Parallel()
	m := NewMacPorts()
	result, err := m.Clean(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMacPorts_ProvidesError(t *testing.T) {
	t.Parallel()
	m := NewMacPorts()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Provides(WithYes(context.Background()), "/usr/bin/htop")
	require.Error(t, err)
}

func TestMacPorts_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewMacPorts()
	manager.exec = mockExecutorHelper("htop @3.2.1 < 3.2.2\ngit @2.43.0 < 2.43.2\n", nil)
	updates, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestMacPorts_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewMacPorts()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync ports tree")
}

func TestMacPorts_CheckUpdateRefreshSucceedsCheckFails(t *testing.T) {
	t.Parallel()
	call := 0
	manager := NewMacPorts()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte(""), nil
		}
		return nil, assert.AnError
	}
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestParsePortOutdated(t *testing.T) {
	t.Parallel()
	input := []byte("htop @3.2.1 < 3.2.2\ngit @2.43.0 < 2.43.2\n")
	updates := parsePortOutdated(input)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "3.2.2", updates[0].AvailableVersion)
}

func TestMacPorts_HoldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewMacPorts()
	err := mgr.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestMacPorts_UnholdNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewMacPorts()
	err := mgr.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestMacPorts_ListHeldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewMacPorts()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}
