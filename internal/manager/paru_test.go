package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParu_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "htop\nripgrep\n", nil, false, []string{"htop", "ripgrep"}},
		{"error", "", assert.AnError, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewParu()
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

func TestParu_Install(t *testing.T) {
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
			mgr := NewParu()
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

func TestParu_Reinstall(t *testing.T) {
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
			mgr := NewParu()
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

func TestParu_Remove(t *testing.T) {
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
			mgr := NewParu()
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

func TestParu_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "aur/htop 3.2.2-1\ncore/htop-debug 1.0.0-1\n", nil, false, []string{"htop", "htop-debug"}},
		{"error", "", assert.AnError, true, nil},
		{"validation error", "", nil, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewParu()
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

func TestParu_Info(t *testing.T) {
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
			mgr := NewParu()
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

func TestParu_Update(t *testing.T) {
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
			mgr := NewParu()
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

func TestParu_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewParu()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestParu_AutoRemove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		dryRun     bool
		mockOutput string
		mockErr    error
		wantErr    bool
		wantCount  int
	}{
		{"dry run with orphans", true, "orphan1\norphan2\n", nil, false, 2},
		{"dry run no orphans", true, "", nil, false, 0},
		{"exec error", true, "", assert.AnError, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewParu()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)
			res, err := mgr.AutoRemove(WithYes(context.Background()), tt.dryRun)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, res, tt.wantCount)
			}
		})
	}
}

func TestParu_AddRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewParu()
	err := mgr.AddRepo(WithYes(context.Background()), "repo", "")
	require.Error(t, err)
}

func TestParu_RemoveRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewParu()
	err := mgr.RemoveRepo(WithYes(context.Background()), "repo")
	require.Error(t, err)
}

func TestParu_ListReposNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewParu()
	res, err := mgr.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestParu_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	m := NewParu()
	m.exec = mockExecutorHelper("", nil)
	pkgs, err := m.AutoRemove(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestParu_AutoRemoveWithOrphans(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewParu()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte("libfoo\n"), nil
		case 2:
			return []byte(""), nil
		default:
			return nil, assert.AnError
		}
	}
	pkgs, err := m.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"libfoo"}, pkgs)
}

func TestParu_AutoRemoveExecError(t *testing.T) {
	t.Parallel()
	m := NewParu()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.AutoRemove(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list orphans")
}

func TestParu_ProvidesError(t *testing.T) {
	t.Parallel()
	m := NewParu()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Provides(WithYes(context.Background()), "/usr/bin/htop")
	require.Error(t, err)
}

func TestParu_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewParu()
	manager.exec = mockExecutorHelper("htop 3.2.1 -> 3.2.2\ngit 2.43.0 -> 2.43.2\n", nil)
	updates, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "3.2.2", updates[0].AvailableVersion)
}

func TestParu_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewParu()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestParu_CheckUpdateFails(t *testing.T) {
	t.Parallel()
	manager := NewParu()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestParu_Clean(t *testing.T) {
	t.Parallel()
	m := NewParu()
	m.exec = mockExecutorHelper("", nil)
	_, err := m.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
}

func TestParu_CleanDryRun(t *testing.T) {
	t.Parallel()
	m := NewParu()
	result, err := m.Clean(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestParu_Hold(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewParu()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte(testPacmanConf), nil
		case 2:
			return []byte(""), nil
		default:
			return nil, assert.AnError
		}
	}
	err := m.Hold(WithYes(context.Background()), "htop")
	require.NoError(t, err)
}

func TestParu_HoldInvalidName(t *testing.T) {
	t.Parallel()
	m := NewParu()
	err := m.Hold(WithYes(context.Background()), "-invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestParu_HoldReadError(t *testing.T) {
	t.Parallel()
	m := NewParu()
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read pacman.conf")
}

func TestParu_ListHeld(t *testing.T) {
	t.Parallel()
	m := NewParu()
	m.exec = mockExecutorHelper(testPacmanConf, nil)
	pkgs, err := m.ListHeld(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"nginx", "redis"}, pkgs)
}

func TestParu_Unhold(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewParu()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte(testPacmanConf), nil
		case 2:
			return []byte(""), nil
		default:
			return nil, assert.AnError
		}
	}
	err := m.Unhold(WithYes(context.Background()), "nginx")
	require.NoError(t, err)
}

func TestParu_UnholdNotHeld(t *testing.T) {
	t.Parallel()
	m := NewParu()
	m.exec = mockExecutorHelper(testPacmanConf, nil)
	err := m.Unhold(WithYes(context.Background()), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not held")
}
