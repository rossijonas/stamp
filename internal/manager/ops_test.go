package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunBatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkgs    []string
		mockErr error
		wantErr bool
	}{
		{"success", []string{"htop", "ripgrep"}, nil, false},
		{"exec error", []string{"htop"}, assert.AnError, true},
		{"validation error", []string{"-invalid"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exec := mockExecutorHelper("", tt.mockErr)
			err := runBatch(WithYes(context.Background()), exec, []string{"pacman", "-R", "--noconfirm"}, "remove", tt.pkgs)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunBatch_NoConsent(t *testing.T) {
	t.Parallel()
	exec := mockExecutorHelper("", nil)
	err := runBatch(context.Background(), exec, []string{"pacman", "-R"}, "remove", []string{"htop"})
	require.Error(t, err)
}

func TestRunSingle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "htop", nil, false},
		{"exec error", "htop", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exec := mockExecutorHelper("", tt.mockErr)
			err := runSingle(WithYes(context.Background()), exec, []string{"pacman", "-R", "--noconfirm", tt.pkg}, "remove", tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunSingle_NoConsent(t *testing.T) {
	t.Parallel()
	exec := mockExecutorHelper("", nil)
	err := runSingle(context.Background(), exec, []string{"pacman", "-R"}, "remove", "htop")
	require.Error(t, err)
}

func TestSudoExec(t *testing.T) {
	t.Parallel()
	exec := mockExecutorHelper("", nil)
	err := sudoExec(WithYes(context.Background()), exec, []string{"pacman", "-R", "htop"}, "failed to remove htop")
	require.NoError(t, err)
}

func TestSudoExec_Error(t *testing.T) {
	t.Parallel()
	exec := mockExecutorHelper("", assert.AnError)
	err := sudoExec(WithYes(context.Background()), exec, []string{"pacman", "-R", "htop"}, "failed to remove htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove htop")
}
