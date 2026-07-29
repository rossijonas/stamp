package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPacman_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		operation   string
		pkgName     string
		mockOutput  string
		mockErr     error
		expectedErr bool
		expectedRes []string
	}{
		{
			name:        "list installed success",
			operation:   "list",
			mockOutput:  "htop\nripgrep\n",
			expectedRes: []string{"htop", "ripgrep"},
		},
		{
			name:        "list installed error",
			operation:   "list",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "install success",
			operation: "install",
			pkgName:   "htop",
		},
		{
			name:        "install error",
			operation:   "install",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "htop",
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "htop",
		},
		{
			name:        "remove error",
			operation:   "remove",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "htop",
			mockOutput:  "core/htop 3.2.2-1\ncore/htop-debug 1.0.0-1\n",
			expectedRes: []string{"htop", "htop-debug"},
		},
		{
			name:        "search error",
			operation:   "search",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "install validation error",
			operation:   "install",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:        "remove validation error",
			operation:   "remove",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:        "search validation error",
			operation:   "search",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:       "info success",
			operation:  "info",
			pkgName:    "htop",
			mockOutput: "Name: htop\nVersion: 3.2.2\n",
		},
		{
			name:        "info error",
			operation:   "info",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "info validation error",
			operation:   "info",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:      "update success",
			operation: "update",
		},
		{
			name:        "update error",
			operation:   "update",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "update single package",
			operation: "update_single",
			pkgName:   "htop",
		},
		{
			name:        "doctor not supported",
			operation:   "doctor",
			expectedErr: true,
		},
		{
			name:        "add repo not supported",
			operation:   "addrepo",
			expectedErr: true,
		},
		{
			name:        "remove repo not supported",
			operation:   "removerepo",
			expectedErr: true,
		},
		{
			name:        "list repos not supported",
			operation:   "listrepos",
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewPacman()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "pacman", manager.Name())

			var err error
			ctx := context.Background()

			switch tt.operation {
			case "list":
				res, err := manager.ListInstalled(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedRes, res)
				}
			case "install":
				err = manager.Install(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "reinstall":
				err = manager.Reinstall(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "remove":
				err = manager.Remove(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "search":
				res, err := manager.Search(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedRes, res)
				}
			case "info":
				res, err := manager.Info(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.mockOutput, res)
				}
			case "doctor":
				_, err = manager.Doctor(ctx)
				require.Error(t, err)
			case "update":
				err = manager.Update(ctx, "")
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "update_single":
				err = manager.Update(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "addrepo":
				err = manager.AddRepo(ctx, "repo", "")
				require.Error(t, err)
			case "removerepo":
				err = manager.RemoveRepo(ctx, "repo")
				require.Error(t, err)
			case "listrepos":
				res, err := manager.ListRepos(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Empty(t, res)
				}
			}
		})
	}
}

func TestParsePacmanSearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "standard search output",
			input:    "core/htop 3.2.2-1\ncore/htop-debug 1.0.0-1\n",
			expected: []string{"htop", "htop-debug"},
		},
		{
			name:     "with extra repo",
			input:    "core/htop 3.2.2-1\ncommunity/ripgrep 13.0.0-4\n",
			expected: []string{"htop", "ripgrep"},
		},
		{
			name:     "no results",
			input:    "",
			expected: []string{},
		},
		{
			name:     "with description",
			input:    "extra/htop 3.2.2-1  Interactive process viewer\n",
			expected: []string{"htop"},
		},
		{
			name:     "multiline with description containing slash",
			input:    "core/grep 3.11-1\n    support/use of regular expressions\ncore/sed 4.9-1\n    A stream editor\n",
			expected: []string{"grep", "sed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parsePacmanSearch([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestPacman_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewPacman()
	// Plain error (not exit 1) → wrapped error
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync databases")
}

func TestPacman_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewPacman()
	manager.exec = mockExecutorHelper("htop 3.2.1 -> 3.2.2\ngit 2.43.0 -> 2.43.2\n", nil)

	updates, err := manager.CheckUpdate(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "3.2.2", updates[0].AvailableVersion)
}

func TestPacman_CheckUpdate_RefreshSucceeds_CheckFails(t *testing.T) {
	t.Parallel()
	call := 0
	manager := NewPacman()
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte(""), nil // pacman -Sy succeeds
		}
		return nil, assert.AnError // pacman -Qu fails
	}

	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestParsePacmanQu(t *testing.T) {
	t.Parallel()
	input := []byte("htop 3.2.1 -> 3.2.2\ngit 2.43.0 -> 2.43.2\n")
	updates := parsePacmanQu(input)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "3.2.1", updates[0].CurrentVersion)
	assert.Equal(t, "3.2.2", updates[0].AvailableVersion)
}

func TestPacman_ProvidesError(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Provides(context.Background(), "/usr/bin/htop")
	require.Error(t, err)
}

func TestPacman_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("", nil)
	pkgs, err := m.AutoRemove(context.Background(), true)
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestPacman_AutoRemove_WithOrphans(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewPacman()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte("libfoo\nlibbar\n"), nil // pacman -Qdtq lists orphans
		case 2:
			return []byte(""), nil // sudo pacman -Rs --noconfirm succeeds
		default:
			return nil, assert.AnError
		}
	}
	pkgs, err := m.AutoRemove(context.Background(), false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"libfoo", "libbar"}, pkgs)
	assert.Equal(t, 2, call)
}

func TestPacman_AutoRemove_ExecError(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.AutoRemove(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list orphans")
}

func TestPacman_Clean(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("", nil)
	_, err := m.Clean(context.Background(), false)
	require.NoError(t, err)
}

func TestPacman_CleanDryRun(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	result, err := m.Clean(context.Background(), true)
	require.NoError(t, err)
	assert.Nil(t, result)
}

const testPacmanConf = `[options]
Architecture = auto
IgnorePkg = nginx redis
SigLevel = Required DatabaseOptional

[core]
Include = /etc/pacman.d/mirrorlist
`

func TestPacman_Hold_AddsToIgnorePkg(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewPacman()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte(testPacmanConf), nil // cat /etc/pacman.conf
		case 2:
			return []byte(""), nil // sudo cp
		default:
			return nil, assert.AnError
		}
	}
	err := m.Hold(context.Background(), "htop")
	require.NoError(t, err)
	assert.Equal(t, 2, call)
}

func TestPacman_Hold_AlreadyHeld(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper(testPacmanConf, nil)
	err := m.Hold(context.Background(), "nginx")
	require.NoError(t, err)
}

func TestPacman_Hold_AddsNewSection(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewPacman()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte("[options]\nArchitecture = auto\n"), nil
		case 2:
			return []byte(""), nil // sudo cp
		default:
			return nil, assert.AnError
		}
	}
	err := m.Hold(context.Background(), "htop")
	require.NoError(t, err)
	assert.Equal(t, 2, call)
}

func TestPacman_Unhold_RemovesFromIgnorePkg(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewPacman()
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
	err := m.Unhold(context.Background(), "nginx")
	require.NoError(t, err)
	assert.Equal(t, 2, call)
}

func TestPacman_Unhold_NotHeld(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper(testPacmanConf, nil)
	err := m.Unhold(context.Background(), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not held")
}

func TestPacman_ListHeld(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper(testPacmanConf, nil)
	pkgs, err := m.ListHeld(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"nginx", "redis"}, pkgs)
}

func TestPacman_ListHeld_Empty(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("[options]\nArchitecture = auto\n", nil)
	pkgs, err := m.ListHeld(context.Background())
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestPacman_Hold_InvalidName(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	err := m.Hold(context.Background(), "-invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestPacman_Unhold_InvalidName(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	err := m.Unhold(context.Background(), "-invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestPacman_Hold_ReadError(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Hold(context.Background(), "nginx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read pacman.conf")
}

func TestPacman_Unhold_ReadError(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("", assert.AnError)
	err := m.Unhold(context.Background(), "nginx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read pacman.conf")
}

func TestPacman_ListHeld_ReadError(t *testing.T) {
	t.Parallel()
	m := NewPacman()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.ListHeld(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read pacman.conf")
}

func TestPacman_Hold_WriteError(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewPacman()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte("[options]\nArchitecture = auto\n"), nil
		case 2:
			return nil, assert.AnError // sudo cp fails
		default:
			return nil, assert.AnError
		}
	}
	err := m.Hold(context.Background(), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write pacman.conf")
}
