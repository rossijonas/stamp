package manager

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlatpak_Install(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "com.spotify.Client", nil, false},
		{"error", "com.spotify.Client", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Install(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlatpak_Remove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		mockErr error
		wantErr bool
	}{
		{"success", "com.spotify.Client", nil, false},
		{"error", "com.spotify.Client", assert.AnError, true},
		{"validation error", "-invalid", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Remove(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlatpak_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
		wantRes    []string
	}{
		{"success", "com.spotify.Client\n", nil, false, []string{"com.spotify.Client"}},
		{"error", "", assert.AnError, true, nil},
		{"validation error", "", nil, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)
			var pkg string
			if tt.wantErr && tt.mockErr == nil {
				pkg = "-invalid"
			} else {
				pkg = "spotify"
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

func TestFlatpak_AddRepo_NoURL(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	mgr.exec = mockExecutorHelper("", nil)
	err := mgr.AddRepo(WithYes(context.Background()), "flathub", "")
	require.Error(t, err)
}

func TestFlatpak_AddRepo_URL(t *testing.T) {
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
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.AddRepo(WithYes(context.Background()), "flathub", "https://dl.flathub.org/repo/flathub.flatpakrepo")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlatpak_RemoveRepo(t *testing.T) {
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
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.RemoveRepo(WithYes(context.Background()), "flathub")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlatpak_Info(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockOut string
		mockErr error
		wantErr bool
	}{
		{"success", "Name: com.spotify.Client\nVersion: 1.0.0\n", nil, false},
		{"error", "", assert.AnError, true},
		{"validation error", "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper(tt.mockOut, tt.mockErr)
			var pkg string
			if tt.wantErr && tt.mockErr == nil {
				pkg = "-invalid"
			} else {
				pkg = "com.spotify.Client"
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

func TestFlatpak_Reinstall(t *testing.T) {
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
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Reinstall(WithYes(context.Background()), "com.spotify.Client")
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlatpak_Update(t *testing.T) {
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
			mgr := NewFlatpak()
			mgr.exec = mockExecutorHelper("", tt.mockErr)
			err := mgr.Update(WithYes(context.Background()), tt.pkg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.mockErr != nil {
					require.ErrorIs(t, err, tt.mockErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlatpak_Doctor(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	_, err := mgr.Doctor(WithYes(context.Background()))
	require.Error(t, err)
}

func TestFlatpak_ListRepos(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = mockExecutorHelper(
		"Name\tURL\nflathub\thttps://dl.flathub.org/repo/flathub.flatpakrepo\nflathub-beta\thttps://dl.flathub.org/beta-repo/flathub-beta.flatpakrepo\n",
		nil,
	)

	repos, err := manager.ListRepos(WithYes(context.Background()))
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "flathub", repos[0].Name)
	assert.Contains(t, repos[0].URL, "dl.flathub.org")
	assert.Equal(t, "flathub-beta", repos[1].Name)
}

func TestFlatpak_ListReposError(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = mockExecutorHelper("", assert.AnError)

	_, err := manager.ListRepos(WithYes(context.Background()))
	require.Error(t, err)
}

func TestFlatpak_ListInstalledMerged(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = func(_ context.Context, cmd string, args ...string) ([]byte, error) {
		switch {
		case cmd == "flatpak" && slices.Contains(args, "--user") && slices.Contains(args, "--app"):
			return []byte("com.spotify.Client\ncom.obsproject.Studio\n"), nil
		case cmd == "flatpak" && slices.Contains(args, "--user"):
			return []byte("com.spotify.Client\ncom.obsproject.Studio\ncom.obsproject.Studio.Plugin.BackgroundRemoval\norg.gnome.Platform\n"), nil
		case cmd == "flatpak" && slices.Contains(args, "--system") && slices.Contains(args, "--app"):
			return []byte("org.mozilla.firefox\n"), nil
		case cmd == "flatpak" && slices.Contains(args, "--system"):
			return []byte("org.mozilla.firefox\norg.gnome.Platform\norg.freedesktop.Platform\n"), nil
		default:
			return nil, assert.AnError
		}
	}

	pkgs, err := manager.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"com.spotify.Client",
		"com.obsproject.Studio",
		"com.obsproject.Studio.Plugin.BackgroundRemoval",
		"org.mozilla.firefox",
	}, pkgs)
	assert.NotContains(t, pkgs, "org.gnome.Platform")
	assert.NotContains(t, pkgs, "org.freedesktop.Platform")
}

func TestFlatpak_ListInstalledSkipsHeader(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = func(_ context.Context, cmd string, args ...string) ([]byte, error) {
		switch {
		case cmd == "flatpak" && slices.Contains(args, "--system") && slices.Contains(args, "--app"):
			return []byte("Application ID\ncom.github.fabiocolacio.marker\ncom.slack.Slack\n"), nil
		case cmd == "flatpak" && slices.Contains(args, "--system"):
			return []byte("Application ID\ncom.github.fabiocolacio.marker\ncom.slack.Slack\norg.gnome.Platform\n"), nil
		default:
			return []byte{}, nil
		}
	}

	pkgs, err := manager.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.NotContains(t, pkgs, "Application ID")
	assert.ElementsMatch(t, []string{"com.github.fabiocolacio.marker", "com.slack.Slack"}, pkgs)
	assert.NotContains(t, pkgs, "org.gnome.Platform")
}

func TestClassifyFlatpakRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		all  []string
		apps []string
		want []string
	}{
		{
			name: "apps and app plugins kept, runtimes and runtime extensions dropped",
			all: []string{
				"com.slack.Slack",
				"org.gnome.Platform",
				"org.freedesktop.Platform.GL.default",
				"com.obsproject.Studio",
				"com.obsproject.Studio.Plugin.BackgroundRemoval",
			},
			apps: []string{"com.slack.Slack", "com.obsproject.Studio"},
			want: []string{
				"com.slack.Slack",
				"com.obsproject.Studio",
				"com.obsproject.Studio.Plugin.BackgroundRemoval",
			},
		},
		{
			name: "deep extension chain resolved to app parent",
			all:  []string{"com.example.App", "com.example.App.Plugin.Sub"},
			apps: []string{"com.example.App"},
			want: []string{"com.example.App", "com.example.App.Plugin.Sub"},
		},
		{
			name: "runtime extensions dropped",
			all:  []string{"org.gnome.Platform", "org.gnome.Platform.Locale", "org.freedesktop.Platform.GL.default"},
			apps: nil,
			want: nil,
		},
		{
			name: "orphan extension without installed parent dropped",
			all:  []string{"com.orphan.Plugin.X"},
			apps: nil,
			want: nil,
		},
		{
			name: "apps sharing a prefix are not treated as extensions",
			all:  []string{"me.proton.Mail", "me.proton.vpn"},
			apps: []string{"me.proton.Mail", "me.proton.vpn"},
			want: []string{"me.proton.Mail", "me.proton.vpn"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyFlatpakRefs(toSet(tt.all), toSet(tt.apps))
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func toSet(items []string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, s := range items {
		m[s] = struct{}{}
	}
	return m
}

func TestParseFlatpakApplicationIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "application ids",
			output: "com.spotify.Client\ncom.obsproject.Studio.Plugin.BackgroundRemoval\n",
			want:   []string{"com.spotify.Client", "com.obsproject.Studio.Plugin.BackgroundRemoval"},
		},
		{
			name:   "header and empty lines skipped",
			output: "Application ID\n\ncom.slack.Slack\n\n",
			want:   []string{"com.slack.Slack"},
		},
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseFlatpakApplicationIDs([]byte(tt.output))
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestFlatpak_ListInstalledBothFail(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = mockExecutorHelper("", assert.AnError)

	_, err := manager.ListInstalled(WithYes(context.Background()))
	require.Error(t, err)
}

func TestFlatpak_ListInstalledOneScopeFails(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = func(_ context.Context, cmd string, args ...string) ([]byte, error) {
		if cmd == "flatpak" && slices.Contains(args, "--user") {
			return []byte(""), nil
		}
		return nil, assert.AnError
	}

	pkgs, err := manager.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestFlatpak_PreviewInstall(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mockErr  error
		wantNoop bool
		wantOut  string
	}{
		{name: "already installed", mockErr: nil, wantNoop: true, wantOut: "already installed via flatpak"},
		{name: "not installed", mockErr: assert.AnError, wantNoop: false, wantOut: "will install com.spotify.Client via flatpak"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewFlatpak()
			var args []string
			mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
				args = a
				return []byte(""), tt.mockErr
			}

			pv, err := mgr.PreviewInstall(WithYes(context.Background()), "com.spotify.Client")
			require.NoError(t, err)
			assert.Equal(t, []string{"info", "com.spotify.Client"}, args)
			assert.NotContains(t, args, "--dry-run")
			assert.Equal(t, tt.wantNoop, pv.Noop)
			assert.Equal(t, tt.wantOut, pv.Output)
		})
	}
}

func TestFlatpak_PreviewInstallValidation(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	_, err := mgr.PreviewInstall(WithYes(context.Background()), "-invalid")
	require.Error(t, err)
}

func TestFlatpak_PreviewRemove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mockErr  error
		wantNoop bool
		wantOut  string
	}{
		{name: "installed", mockErr: nil, wantNoop: false, wantOut: "will remove com.spotify.Client via flatpak"},
		{name: "not installed", mockErr: assert.AnError, wantNoop: true, wantOut: "not installed via flatpak"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewFlatpak()
			var args []string
			mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
				args = a
				return []byte(""), tt.mockErr
			}

			pv, err := mgr.PreviewRemove(WithYes(context.Background()), "com.spotify.Client")
			require.NoError(t, err)
			assert.Equal(t, []string{"info", "com.spotify.Client"}, args)
			assert.NotContains(t, args, "--dry-run")
			assert.Equal(t, tt.wantNoop, pv.Noop)
			assert.Equal(t, tt.wantOut, pv.Output)
		})
	}
}

func TestFlatpak_PreviewRemoveValidation(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	_, err := mgr.PreviewRemove(WithYes(context.Background()), "-invalid")
	require.Error(t, err)
}

func TestFlatpak_PreviewReinstallDelegatesToInstall(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(""), nil
	}

	pv, err := mgr.PreviewReinstall(WithYes(context.Background()), "com.spotify.Client")
	require.NoError(t, err)
	assert.True(t, pv.Noop)
	assert.Equal(t, "already installed via flatpak", pv.Output)
}

func TestFlatpak_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	out := "Looking for updates…\nUpdates for 'com.spotify.Client' in remote 'flathub'\nUpdates for 'org.gimp.GIMP' in remote 'flathub'\nNothing to update.\n"
	manager.exec = mockExecutorHelper(out, nil)

	updates, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.NoError(t, err)
	require.Len(t, updates, 2)
	assert.Equal(t, "com.spotify.Client", updates[0].Package)
	assert.Equal(t, "org.gimp.GIMP", updates[1].Package)
}

func TestFlatpak_CheckUpdate_Unsupported(t *testing.T) {
	t.Parallel()
	manager := NewFlatpak()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCheckUnsupported)
}

func TestParseFlatpakDryRun(t *testing.T) {
	t.Parallel()
	input := []byte("Looking for updates…\nUpdates for 'com.spotify.Client' in remote 'flathub'\nUpdates for 'org.gimp.GIMP' in remote 'flathub'\nNothing to update.\n")
	updates := parseFlatpakDryRun(input)
	require.Len(t, updates, 2)
	assert.Equal(t, "com.spotify.Client", updates[0].Package)
	assert.Equal(t, "org.gimp.GIMP", updates[1].Package)
}

func TestFlatpak_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	_, err := mgr.Provides(WithYes(context.Background()), "firefox")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestFlatpak_AutoRemove(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	mgr.exec = mockExecutorHelper("", nil)
	_, err := mgr.AutoRemove(WithYes(context.Background()), false)
	require.NoError(t, err)
}

func TestFlatpak_AutoRemoveDryRun(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	pkgs, err := mgr.AutoRemove(WithYes(context.Background()), true)
	require.NoError(t, err)
	assert.Nil(t, pkgs)
}

func TestFlatpak_CleanNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	_, err := mgr.Clean(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestFlatpak_Hold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	err := mgr.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestFlatpak_Unhold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	err := mgr.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestFlatpak_ListHeld_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestFlatpak_Override_Filesystem(t *testing.T) {
	t.Parallel()
	var args []string
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Filesystem: []string{"host", "home"},
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--user")
	assert.Contains(t, args, "--filesystem=host")
	assert.Contains(t, args, "--filesystem=home")
	assert.Contains(t, args, "firefox")
}

func TestFlatpak_Override_Socket(t *testing.T) {
	t.Parallel()
	var args []string
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Socket: []string{"wayland", "pulseaudio"},
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--socket=wayland")
	assert.Contains(t, args, "--socket=pulseaudio")
}

func TestFlatpak_Override_Device(t *testing.T) {
	t.Parallel()
	var args []string
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Device: []string{"dri", "all"},
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--device=dri")
	assert.Contains(t, args, "--device=all")
}

func TestFlatpak_Override_Env(t *testing.T) {
	t.Parallel()
	var args []string
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Env: []string{"MY_VAR=value", "OTHER=val"},
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--env=MY_VAR=value")
	assert.Contains(t, args, "--env=OTHER=val")
}

func TestFlatpak_Override_Reset(t *testing.T) {
	t.Parallel()
	var args []string
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Reset: true,
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--reset")
	assert.Contains(t, args, "firefox")
}

func TestFlatpak_Override_Show(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	mgr.exec = mockExecutorHelper("filesystem=host\nsocket=wayland\n", nil)
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Show: true,
	})
	require.NoError(t, err)
}

func TestFlatpak_Override_System(t *testing.T) {
	t.Parallel()
	var args []string
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		System: true,
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--system")
}

func TestFlatpak_Override_Error(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Filesystem: []string{"host"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set overrides")
}

func TestFlatpak_Override_InvalidName(t *testing.T) {
	t.Parallel()
	mgr := NewFlatpak()
	err := mgr.Override(WithYes(context.Background()), "-invalid", OverrideFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestFlatpak_Override_AllFlags(t *testing.T) {
	t.Parallel()
	var args []string
	mgr := NewFlatpak()
	mgr.exec = func(_ context.Context, _ string, a ...string) ([]byte, error) {
		args = a
		return []byte(""), nil
	}
	err := mgr.Override(WithYes(context.Background()), "firefox", OverrideFlags{
		Filesystem: []string{"host"},
		Socket:     []string{"wayland"},
		Device:     []string{"dri"},
		Env:        []string{"VAR=val"},
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--filesystem=host")
	assert.Contains(t, args, "--socket=wayland")
	assert.Contains(t, args, "--device=dri")
	assert.Contains(t, args, "--env=VAR=val")
	assert.Contains(t, args, "firefox")
}
