package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipx_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"json output", `{"venvs":{"black":{"metadata":{}},"httpie":{"metadata":{}}}}`, nil, false, []string{"black", "httpie"}},
		{"error", "", assert.AnError, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewPipx()
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

func TestPipx_ListInstalled_TextFallback(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	first := true
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		if first {
			first = false
			return nil, assert.AnError
		}
		return []byte("   package black 23.1.0, installed using Python 3.11\n   package httpie 3.2.2, installed using Python 3.11\n"), nil
	}
	res, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"black", "httpie"}, res)
}

func TestPipx_Install(t *testing.T) {
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
			mgr := NewPipx()
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

func TestPipx_Reinstall(t *testing.T) {
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
			mgr := NewPipx()
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

func TestPipx_Remove(t *testing.T) {
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
			mgr := NewPipx()
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

func TestPipx_Search(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Search(WithYes(context.Background()), "anything")
	require.Error(t, err)
}

func TestPipx_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "black", `{"venvs":{"black":{"metadata":{"version":"23.1.0"}}}}`, nil, false},
		{"not found", "missing", `{"venvs":{"black":{}}}`, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewPipx()
			mgr.exec = mockExecutorHelper(tt.mockOut, tt.mockErr)
			_, err := mgr.Info(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPipx_InfoError(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Info(WithYes(context.Background()), "black")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get info")
}

func TestPipx_Update(t *testing.T) {
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
			mgr := NewPipx()
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

func TestPipx_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestPipx_CheckUpdateUnsupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	_, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestPipx_ListReposUnsupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	res, err := mgr.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Nil(t, res)
}

func TestPipx_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	_, err := mgr.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestPipx_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	_, err := mgr.AutoRemove(WithYes(context.Background()), true)
	require.Error(t, err)
}

func TestPipx_CleanNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	_, err := mgr.Clean(WithYes(context.Background()), true)
	require.Error(t, err)
}

func TestPipx_HoldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	err := mgr.Hold(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestPipx_UnholdNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	err := mgr.Unhold(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestPipx_ListHeldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
}
