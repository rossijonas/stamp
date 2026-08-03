package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnap_Operations(t *testing.T) {
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
			mockOutput:  "Name    Version   Rev   Tracking         Publisher   Notes\nhtop    3.2.2     123   latest/stable    canonical✓  -\ncore    16-2.59   456   latest/stable    canonical✓  core\n",
			expectedRes: []string{"htop", "core"},
		},
		{
			name:        "list installed error",
			operation:   "list",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "list installed no snaps",
			operation:   "list",
			mockOutput:  "Name    Version   Rev   Tracking         Publisher   Notes\n",
			expectedRes: []string{},
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
			mockOutput:  "Name               Version   Publisher   Notes    Summary\nhtop               3.2.2     canonical   -        Interactive process viewer\n",
			expectedRes: []string{"htop"},
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
			mockOutput: "name: htop\nversion: 3.2.2\n",
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
			manager := NewSnap()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "snap", manager.Name())

			var err error
			ctx := WithYes(context.Background())

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

func TestParseSnapTabular(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "standard snap list output",
			input: "Name    Version   Rev   Tracking         Publisher   Notes\n" +
				"htop    3.2.2     123   latest/stable    canonical✓  -\n" +
				"core    16-2.59   456   latest/stable    canonical✓  core\n",
			expected: []string{"htop", "core"},
		},
		{
			name:     "no snaps installed",
			input:    "Name    Version   Rev   Tracking         Publisher   Notes\n",
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name: "single snap",
			input: "Name   Version  Rev   Tracking  Publisher  Notes\n" +
				"hello  2.12     123   stable    canonical  -\n",
			expected: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseSnapTabular([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestSnap_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewSnap()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestSnap_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewSnap()
	manager.exec = mockExecutorHelper("htop                 3.2.2     123   stable\ngit                  2.43.2    456   stable\n", nil)

	updates, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestParseSnapRefreshList(t *testing.T) {
	t.Parallel()
	input := []byte("Name                 Version   Rev   Latest\nhtop                 3.2.2     123   stable\ngit                  2.43.2    456   stable\n")
	updates := parseSnapRefreshList(input)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "git", updates[1].Package)
}

func TestSnap_Clean(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewSnap()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			// snap list (active revisions)
			return []byte("Name\tVersion\tRev\tTracking\nhtop\t1.1\t6\tstable\n"), nil
		case 2:
			// snap list --all (all revisions)
			return []byte("Name\tVersion\tRev\tTracking\nhtop\t1.0\t5\tstable\nhtop\t1.1\t6\tstable\n"), nil
		case 3:
			// snap remove --revision 5 (inactive)
			return []byte(""), nil
		default:
			return nil, assert.AnError
		}
	}
	result, err := m.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result[0], "rev 5")
	assert.Equal(t, 3, call)
}

func TestSnap_CheckUpdate_Error(t *testing.T) {
	t.Parallel()
	m := NewSnap()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestSnap_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	m := NewSnap()
	_, err := m.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestSnap_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	m := NewSnap()
	_, err := m.AutoRemove(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestSnap_Hold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	err := mgr.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestSnap_Unhold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	err := mgr.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestSnap_ListHeld_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestSnap_CleanDryRun(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewSnap()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			// snap list (active)
			return []byte("Name\tVersion\tRev\tTracking\nhtop\t1.1\t6\tstable\n"), nil
		case 2:
			// snap list --all
			return []byte("Name\tVersion\tRev\tTracking\nhtop\t1.0\t5\tstable\nhtop\t1.1\t6\tstable\n"), nil
		default:
			return nil, assert.AnError
		}
	}
	result, err := m.Clean(WithYes(context.Background()), true)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Contains(t, result[0], "rev 5")
	assert.Equal(t, 2, call)
}

func TestSnap_Clean_NoInactive(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewSnap()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		switch call {
		case 1:
			return []byte("Name\tVersion\tRev\tTracking\nhtop\t1.1\t6\tstable\n"), nil
		case 2:
			return []byte("Name\tVersion\tRev\tTracking\nhtop\t1.1\t6\tstable\n"), nil
		default:
			return nil, assert.AnError
		}
	}
	result, err := m.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 2, call)
}

func TestSnap_Clean_ActiveListError(t *testing.T) {
	t.Parallel()
	m := NewSnap()
	m.exec = mockExecutorHelper("", assert.AnError)
	_, err := m.Clean(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list active snaps")
}

func TestSnap_Clean_AllListError(t *testing.T) {
	t.Parallel()
	call := 0
	m := NewSnap()
	m.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("Name\tVersion\tRev\tTracking\nhtop\t1.1\t6\tstable\n"), nil
		}
		return nil, assert.AnError
	}
	_, err := m.Clean(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list all snap revisions")
}

func TestParseSnapRevisions(t *testing.T) {
	t.Parallel()
	input := []byte("Name\tVersion\tRev\tTracking\nhtop\t1.0\t5\tstable\ncore\t2.0\t10\tstable\n")
	result := parseSnapRevisions(input)
	assert.Len(t, result, 2)
	assert.Equal(t, "5", result["htop"])
	assert.Equal(t, "10", result["core"])
}

func TestParseSnapRevisions_Empty(t *testing.T) {
	t.Parallel()
	result := parseSnapRevisions([]byte("Name\tVersion\tRev\tTracking\n"))
	assert.Empty(t, result)
}

func TestParseSnapRevisions_SkipsHeader(t *testing.T) {
	t.Parallel()
	result := parseSnapRevisions([]byte("Name\tVersion\tRev\tTracking\nhtop\t1.0\t5\tstable\n"))
	assert.Equal(t, "5", result["htop"])
}
