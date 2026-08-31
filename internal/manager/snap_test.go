package manager

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnap_ListInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "Name    Version   Rev   Tracking         Publisher   Notes\nhtop    3.2.2     123   latest/stable    canonical✓  -\ncore    16-2.59   456   latest/stable    canonical✓  core\n", nil, false, []string{"htop"}},
		{"error", "", assert.AnError, true, nil},
		{"no snaps", "Name    Version   Rev   Tracking         Publisher   Notes\n", nil, false, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewSnap()
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

func TestSnap_Install(t *testing.T) {
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
			mgr := NewSnap()
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

func TestSnap_Reinstall(t *testing.T) {
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
			mgr := NewSnap()
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

func TestSnap_Remove(t *testing.T) {
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
			mgr := NewSnap()
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

func TestSnap_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "Name               Version   Publisher   Notes    Summary\nhtop               3.2.2     canonical   -        Interactive process viewer\n", nil, false, []string{"htop"}},
		{"error", "", assert.AnError, true, nil},
		{"validation error", "", nil, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewSnap()
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

func TestSnap_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "name: htop\nversion: 3.2.2\n", nil, false},
		{"error", "", assert.AnError, true},
		{"validation error", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewSnap()
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

func TestSnap_Update(t *testing.T) {
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
			mgr := NewSnap()
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

func TestSnap_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestSnap_AddRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	err := mgr.AddRepo(WithYes(context.Background()), "repo", "")
	require.Error(t, err)
}

func TestSnap_RemoveRepoNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	err := mgr.RemoveRepo(WithYes(context.Background()), "repo")
	require.Error(t, err)
}

func TestSnap_ListReposNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	res, err := mgr.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, res)
}

func TestSnap_Clean_DryRun(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	active := "Name    Version   Rev   Tracking\nhtop    3.2.2     123   latest/stable\n"
	all := "Name    Version   Rev   Tracking\nhtop    3.2.2     123   latest/stable\nhtop    3.2.1     122   latest/stable\n"
	mgr.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "snap" && slices.Contains(args, "--all") {
			return []byte(all), nil
		}
		return []byte(active), nil
	}
	removed, err := mgr.Clean(WithYes(context.Background()), true)
	require.NoError(t, err)
	require.Len(t, removed, 1)
	assert.Contains(t, removed[0], "htop rev 122")
}

func TestSnap_Clean_Actual(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	active := "Name    Version   Rev   Tracking\nhtop    3.2.2     123   latest/stable\n"
	all := "Name    Version   Rev   Tracking\nhtop    3.2.2     123   latest/stable\nhtop    3.2.1     122   latest/stable\n"
	removed := []string{}
	mgr.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "sudo" && slices.Contains(args, "remove") {
			removed = append(removed, strings.Join(args, " "))
			return nil, nil
		}
		if name == "snap" && slices.Contains(args, "--all") {
			return []byte(all), nil
		}
		return []byte(active), nil
	}
	result, err := mgr.Clean(WithYes(context.Background()), false)
	require.NoError(t, err)
	require.Len(t, removed, 1)
	assert.Contains(t, removed[0], "htop")
	assert.Contains(t, removed[0], "--revision")
	require.Len(t, result, 1)
	assert.Contains(t, result[0], "htop")
}

func TestSnap_Clean_NothingToRemove(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	active := "Name    Version   Rev   Tracking\nhtop    3.2.2     123   latest/stable\n"
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(active), nil
	}
	removed, err := mgr.Clean(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Empty(t, removed)
}

func TestSnap_Clean_Error(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Clean(WithYes(context.Background()), true)
	require.Error(t, err)
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

func TestParseSnapRevisions_HeaderOnly(t *testing.T) {
	t.Parallel()
	input := []byte("Name    Version   Rev   Tracking\n")
	result := parseSnapRevisions(input)
	assert.Empty(t, result)
}

func TestParseSnapRevisions_Empty(t *testing.T) {
	t.Parallel()
	result := parseSnapRevisions([]byte(""))
	assert.Empty(t, result)
}

func TestParseSnapRevisions_ShortFields(t *testing.T) {
	t.Parallel()
	input := []byte("Name\nhtop\n")
	result := parseSnapRevisions(input)
	assert.Empty(t, result)
}

func TestAllDigits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		s    string
		want bool
	}{
		{"123", true},
		{"20", true},
		{"3", true},
		{"", false},
		{"abc", false},
		{"12a3", false},
		{"20a", false},
		{"a", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			assert.Equal(t, tt.want, allDigits(tt.s))
		})
	}
}

func TestIsSystemSnap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want bool
	}{
		// core runtimes
		{"core", true},
		{"core18", true},
		{"core20", true},
		{"core22", true},
		{"core24", true},
		{"core26", true},
		// gnome platform runtimes (digit-heavy shapes)
		{"gnome-3-38-2004", true},
		{"gnome-42-2204", true},
		{"gnome-46-2404", true},
		// known system snaps (exact)
		{"snapd", true},
		{"gtk-common-themes", true},
		{"snap-store", true},
		{"firmware-updater", true},
		{"bare", true},
		// genuine user apps — must NOT be filtered
		{"firefox", false},
		{"spotify", false},
		{"gnome-calculator", false},
		{"gnome-terminal", false},
		{"vscode", false},
		{"htop", false},
		// user apps whose names merely start with "core" — must NOT be filtered
		{"corebird", false},
		{"coreutils", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSystemSnap(tt.name))
		})
	}
}

func TestParseSnapRefreshList(t *testing.T) {
	t.Parallel()
	input := []byte("Name  Version  Rev  Tracking\nhtop  3.2.2     123  latest/stable\ngit   2.43.0    456  latest/stable\n")
	result := parseSnapRefreshList(input)
	require.Len(t, result, 2)
	assert.Equal(t, "htop", result[0].Package)
	assert.Equal(t, "git", result[1].Package)
}

func TestParseSnapRefreshListEmpty(t *testing.T) {
	t.Parallel()
	input := []byte("Name  Version  Rev  Tracking\n")
	result := parseSnapRefreshList(input)
	assert.Empty(t, result)
}

func TestSnap_CheckUpdate(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	mgr.exec = mockExecutorHelper("Name  Version  Rev  Tracking\nhtop  3.2.2     123  latest/stable\n", nil)
	updates, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 1)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestSnap_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestSnap_CheckUpdateRefreshFails(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
}

func TestSnap_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	_, err := mgr.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestSnap_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	_, err := mgr.AutoRemove(WithYes(context.Background()), true)
	require.Error(t, err)
}

func TestSnap_HoldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	err := mgr.Hold(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestSnap_UnholdNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	err := mgr.Unhold(WithYes(context.Background()), "htop")
	require.Error(t, err)
}

func TestSnap_ListHeldNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewSnap()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
}
