package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpm_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "/usr/lib\n├── corepack@0.29.4\n├── typescript@5.6.3\n", nil, false, []string{"corepack", "typescript"}},
		{"error", "", assert.AnError, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewNpm()
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

func TestNpm_Install(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "typescript", nil, false},
		{"error", "typescript", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewNpm()
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

func TestNpm_Reinstall(t *testing.T) {
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
			mgr := NewNpm()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Reinstall(WithYes(context.Background()), "typescript")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNpm_Remove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "typescript", nil, false},
		{"error", "typescript", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewNpm()
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

func TestNpm_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "Name: typescript\nVersion: 5.6.3\n", nil, false},
		{"error", "", assert.AnError, true},
		{"validation error", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewNpm()
			mgr.exec = mockExecutorHelper(tt.mockOut, tt.mockErr)
			var pkg string
			if tt.wantErr && tt.mockErr == nil {
				pkg = "-invalid"
			} else {
				pkg = "typescript"
			}
			res, err := mgr.Info(WithYes(context.Background()), pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mockOut, res)
			}
		})
	}
}

func TestNpm_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "", nil, false},
		{"error", "", assert.AnError, true},
		{"single package", "typescript", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewNpm()
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

func TestNpm_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestNpm_AddRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	err := mgr.AddRepo(WithYes(context.Background()), "repo", "")
	require.Error(t, err)
}

func TestNpm_RemoveRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	err := mgr.RemoveRepo(WithYes(context.Background()), "repo")
	require.Error(t, err)
}

func TestNpm_ListReposNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	res, err := mgr.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestNpm_SearchNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.Search(WithYes(context.Background()), "typescript")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestNpm_CheckUpdateNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestNpm_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestNpm_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.AutoRemove(WithYes(context.Background()), true)
	require.Error(t, err)
}

func TestNpm_CleanNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.Clean(WithYes(context.Background()), true)
	require.Error(t, err)
}

func TestNpm_HoldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	err := mgr.Hold(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestNpm_UnholdNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	err := mgr.Unhold(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestNpm_ListHeldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
}
