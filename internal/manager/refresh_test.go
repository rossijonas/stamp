package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdapter_Refresh_NoOp covers managers whose Refresh is a no-op: they must
// return nil without invoking the executor.
func TestAdapter_Refresh_NoOp(t *testing.T) {
	tests := []struct {
		name  string
		build func(e Executor) Adapter
	}{
		{name: "go", build: func(e Executor) Adapter { a := NewGo(); a.exec = e; return a }},
		{name: "macports", build: func(e Executor) Adapter { a := NewMacPorts(); a.exec = e; return a }},
		{name: "npm", build: func(e Executor) Adapter { a := NewNpm(); a.exec = e; return a }},
		{name: "flatpak", build: func(e Executor) Adapter { a := NewFlatpak(); a.exec = e; return a }},
		{name: "zypper", build: func(e Executor) Adapter { a := NewZypper(); a.exec = e; return a }},
		{name: "snap", build: func(e Executor) Adapter { a := NewSnap(); a.exec = e; return a }},
		{name: "cargo", build: func(e Executor) Adapter { a := NewCargo(); a.exec = e; return a }},
		{name: "pipx", build: func(e Executor) Adapter { a := NewPipx(); a.exec = e; return a }},
		{name: "uv", build: func(e Executor) Adapter { a := NewUv(); a.exec = e; return a }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				calls++
				return nil, nil
			})
			require.NoError(t, a.Refresh(context.Background()))
			assert.Zero(t, calls, "expected no-op refresh for %s", tt.name)
		})
	}
}

// TestAdapter_Refresh_Exec covers managers whose Refresh runs a real command:
// success returns nil, failure is wrapped.
func TestAdapter_Refresh_Exec(t *testing.T) {
	refreshErr := errors.New("offline")
	tests := []struct {
		name  string
		build func(e Executor) Adapter
	}{
		{name: "dnf", build: func(e Executor) Adapter { a := NewDNF("dnf"); a.exec = e; return a }},
		{name: "apt", build: func(e Executor) Adapter { a := NewAPT("apt"); a.exec = e; return a }},
		{name: "brew", build: func(e Executor) Adapter { a := NewBrew(); a.exec = e; return a }},
		{name: "pacman", build: func(e Executor) Adapter { a := NewPacman(); a.exec = e; return a }},
		{name: "paru", build: func(e Executor) Adapter { a := NewParu(); a.exec = e; return a }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, nil
			})
			require.NoError(t, a.Refresh(context.Background()))

			a = tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, refreshErr
			})
			err := a.Refresh(context.Background())
			require.Error(t, err)
			assert.ErrorIs(t, err, refreshErr)
		})
	}
}

func TestCheckUpdate_UnrecognizedOutput(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		check   func(exec Executor) ([]UpdateInfo, error)
		goodOut string
	}{
		{name: "dnf", check: func(e Executor) ([]UpdateInfo, error) { a := NewDNF("dnf"); a.exec = e; return a.CheckUpdate(ctx, "") }, goodOut: ""},
		{name: "pacman", check: func(e Executor) ([]UpdateInfo, error) { a := NewPacman(); a.exec = e; return a.CheckUpdate(ctx, "") }, goodOut: ""},
		{name: "apt", check: func(e Executor) ([]UpdateInfo, error) { a := NewAPT("apt"); a.exec = e; return a.CheckUpdate(ctx, "") }, goodOut: "Listing...\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name+" unrecognized output errors", func(t *testing.T) {
			_, err := tt.check(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte("garbage\n"), nil
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "parser may be outdated")
		})
		t.Run(tt.name+" empty or header is clean", func(t *testing.T) {
			_, err := tt.check(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte(tt.goodOut), nil
			})
			require.NoError(t, err)
		})
	}
}

func TestMock_Preview(t *testing.T) {
	m := &Mock{ManagerName: "dnf"}

	pv, err := m.PreviewInstall(context.Background(), "htop")
	require.NoError(t, err)
	assert.Contains(t, pv.Output, "Install: htop")

	pv, err = m.PreviewRemove(context.Background(), "htop")
	require.NoError(t, err)
	assert.Contains(t, pv.Output, "Remove: htop")

	m.PreviewResult = "custom preview"
	pv, err = m.PreviewInstall(context.Background(), "htop")
	require.NoError(t, err)
	assert.Equal(t, "custom preview", pv.Output)

	m.PreviewInstallErr = errors.New("no dry run")
	_, err = m.PreviewInstall(context.Background(), "htop")
	require.Error(t, err)

	_, err = m.PreviewInstall(context.Background(), "-bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")

	m.PreviewResult = ""
	m.PreviewRemoveErr = errors.New("no dry run")
	_, err = m.PreviewRemove(context.Background(), "htop")
	require.Error(t, err)

	_, err = m.PreviewRemove(context.Background(), "-bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")

	pv, err = m.PreviewReinstall(context.Background(), "htop")
	require.NoError(t, err)
	assert.Contains(t, pv.Output, "Reinstall: htop")

	m.PreviewReinstallErr = errors.New("no dry run")
	_, err = m.PreviewReinstall(context.Background(), "htop")
	require.Error(t, err)

	_, err = m.PreviewReinstall(context.Background(), "-bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestMock_Refresh(t *testing.T) {
	refreshErr := errors.New("offline")
	m := &Mock{ManagerName: "dnf"}
	require.NoError(t, m.Refresh(context.Background()))

	m.RefreshErr = refreshErr
	err := m.Refresh(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, refreshErr)
}

func TestMock_Override(t *testing.T) {
	m := &Mock{ManagerName: "flatpak"}
	err := m.Override(context.Background(), "com.example.App", OverrideFlags{Show: true})
	require.ErrorIs(t, err, ErrNotSupported)

	got := ""
	m.OverrideFunc = func(_ context.Context, appID string, _ OverrideFlags) error {
		got = appID
		return nil
	}
	err = m.Override(context.Background(), "com.example.App", OverrideFlags{Show: true})
	require.NoError(t, err)
	assert.Equal(t, "com.example.App", got)
}
