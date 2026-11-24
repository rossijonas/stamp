package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// syncBuffer wraps bytes.Buffer with a mutex for concurrent-safe writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.b.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.b.String()
}

// execUpdateCmd is like execCmd but uses syncBuffer for concurrent-safe output.
func execUpdateCmd(t *testing.T, args []string, adapters []manager.Adapter) (*syncBuffer, error) {
	t.Helper()
	return execUpdateCmdSudo(t, args, adapters, func(context.Context) bool { return true }, nil, false)
}

// execUpdateCmdSudo runs the update command with sudo preflight seams injected,
// so tests never invoke sudo. terminal controls the isTerminal gate.
func execUpdateCmdSudo(t *testing.T, args []string, adapters []manager.Adapter,
	ready func(context.Context) bool, ensureErr error, terminal bool) (*syncBuffer, error) {
	t.Helper()
	oldReady, oldEnsure, oldTerm := sudoReady, sudoEnsure, isTerminal
	sudoReady = ready
	sudoEnsure = func(context.Context) error { return ensureErr }
	isTerminal = func(io.Reader) bool { return terminal }
	t.Cleanup(func() { sudoReady, sudoEnsure, isTerminal = oldReady, oldEnsure, oldTerm })

	buf := new(syncBuffer)
	tmpDir := t.TempDir()
	cPath := filepath.Join(tmpDir, "config.toml")
	mPath := filepath.Join(tmpDir, "manifest.toml")
	root := NewRootCmd(WithAdapters(adapters), WithConfigPath(cPath), WithManifestPath(mPath))
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	root.SetIn(r)
	_ = w.Close()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf, err
}

func TestUpdateCmd_GuardFalseRunsSerial(t *testing.T) {
	probe := &inflight{}
	newAdapter := func(name string) manager.Adapter {
		return &mockAdapter{name: name, UpdateFunc: func(context.Context, string) error { return probe.run() }}
	}
	adapters := []manager.Adapter{newAdapter("dnf"), newAdapter("apt")}

	buf, err := execUpdateCmdSudo(t, []string{"update", "-y"}, adapters,
		func(context.Context) bool { return false }, assert.AnError, true)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "sudo validation failed")
	assert.Equal(t, 1, probe.peak(), "guard false must serialize updates")
}

func TestUpdateCmd_GuardTrueRunsParallel(t *testing.T) {
	probe := &inflight{}
	newAdapter := func(name string) manager.Adapter {
		return &mockAdapter{name: name, UpdateFunc: func(context.Context, string) error { return probe.run() }}
	}
	adapters := []manager.Adapter{newAdapter("dnf"), newAdapter("apt")}

	_, err := execUpdateCmdSudo(t, []string{"update", "-y"}, adapters,
		func(context.Context) bool { return true }, nil, false)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, probe.peak(), 2, "ready sudo must run updates in parallel")
}

func TestUpdateCmd_AllManagers(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf"},
		&mockAdapter{name: "flatpak"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via brew")
	assert.Contains(t, output, "updated packages via dnf")
	assert.Contains(t, output, "updated packages via flatpak")
}

func TestUpdateCmd_ManagerFlag(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf"},
		&mockAdapter{name: "flatpak"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-m", "dnf", "-y"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via dnf")
	assert.NotContains(t, output, "brew")
	assert.NotContains(t, output, "flatpak")
}

func TestUpdateCmd_UnknownManager(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	_, err := execUpdateCmd(t, []string{"update", "-m", "nonexistent"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestUpdateCmd_OneFails(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf", err: assert.AnError},
		&mockAdapter{name: "flatpak"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.Error(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via brew")
	assert.Contains(t, output, "⚠ update failed for dnf")
	assert.Contains(t, output, "updated packages via flatpak")
}

func TestUpdateCmd_NoAdapters(t *testing.T) {
	_, err := execUpdateCmd(t, []string{"update"}, []manager.Adapter{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no package managers available")
	assert.Equal(t, ExitUnavailable, exitCodeFor(err))
}

func TestUpdateCmd_AllFail(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew", err: assert.AnError},
		&mockAdapter{name: "dnf", err: assert.AnError},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.Error(t, err)
	output := buf.String()
	assert.Contains(t, output, "⚠ update failed for brew")
	assert.Contains(t, output, "⚠ update failed for dnf")
	assert.Contains(t, err.Error(), "one or more managers failed to update")
}

func TestUpdateCmd_UsesAdapterUpdate(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated packages via brew")
}

func TestUpdateCmd_UpgradeAlias(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	buf, err := execUpdateCmd(t, []string{"upgrade", "-y"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated packages via brew")
}

func TestUpdateCmd_WithManifestNotRequired(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated packages via brew")
}

func TestUpdateCmd_SinglePackage(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-p", "htop", "-m", "brew", "-y"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated htop via brew")
	assert.NotContains(t, buf.String(), "dnf")
}

func TestUpdateCmd_SinglePackage_NoManager(t *testing.T) {
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}}
	_, err := execUpdateCmd(t, []string{"update", "-p", "htop"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify --manager")
	assert.Equal(t, ExitUsage, exitCodeFor(err))
}

func TestUpdateCmd_GoSinglePackage(t *testing.T) {
	adapters := []manager.Adapter{&mockAdapter{name: "go"}}
	buf, err := execUpdateCmd(t, []string{"update", "-p", "github.com/example/tool", "-m", "go", "-y"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "updated github.com/example/tool via go")
}

func TestUpdateCmd_SinglePackage_InvalidName(t *testing.T) {
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}}
	_, err := execUpdateCmd(t, []string{"update", "-p", "-htop", "-m", "brew"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestUpdateCmd_Serial(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "--serial", "-y"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "updating via brew")
	assert.NotContains(t, output, "updating via dnf")
	assert.Contains(t, output, "updated packages via brew")
	assert.Contains(t, output, "updated packages via dnf")
}

func TestUpdateCmd_Serial_OneFails(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "brew"},
		&mockAdapter{name: "dnf", err: assert.AnError},
		&mockAdapter{name: "flatpak"},
	}
	buf, err := execUpdateCmd(t, []string{"update", "--serial", "-y"}, adapters)
	require.Error(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via brew")
	assert.Contains(t, output, "⚠ update failed for dnf")
	assert.Contains(t, output, "updated packages via flatpak")
}

func TestUpdateCmd_WithUpdatesNonTerminalAborts(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{
			name: "brew",
			checkUpdates: []manager.UpdateInfo{
				{Package: "htop", CurrentVersion: "3.2.1", AvailableVersion: "3.2.2"},
			},
		},
	}
	// No -y, updates available, stdin is a pipe → fail closed (no run phase)
	buf, err := execUpdateCmd(t, []string{"update"}, adapters)
	require.Error(t, err)
	output := buf.String()
	assert.NotContains(t, output, "Checking for updates")
	assert.Contains(t, output, "brew: htop 3.2.1 → 3.2.2")
	assert.Contains(t, err.Error(), "non-interactive mode")
	assert.NotContains(t, output, "updated packages via brew")
}

func TestUpdateCmd_WithUpdatesYesProceeds(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{
			name: "brew",
			checkUpdates: []manager.UpdateInfo{
				{Package: "htop", CurrentVersion: "3.2.1", AvailableVersion: "3.2.2"},
			},
		},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "Checking for updates")
	assert.Contains(t, output, "updated packages via brew")
}

func TestUpdateCmd_CheckOnly(t *testing.T) {
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}}
	buf, err := execUpdateCmd(t, []string{"update", "--check"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "Checking for updates")
	assert.Contains(t, output, "brew: No updates available")
}

func TestUpdateCmd_CheckOnly_Unsupported(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "pipx", checkUpdatesErr: manager.ErrCheckUnsupported},
	}
	buf, err := execUpdateCmd(t, []string{"update", "--check"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "pipx: cannot preview updates")
}

func TestUpdateCmd_CheckYConflict(t *testing.T) {
	_, err := execUpdateCmd(t, []string{"update", "--check", "-y"}, []manager.Adapter{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestUpdateCmd_YesSkipsCheck(t *testing.T) {
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "Checking for updates")
	assert.Contains(t, output, "updated packages via brew")
}

func TestUpdateCmd_CheckOnly_UpdatesAvailable(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{
			name: "brew",
			checkUpdates: []manager.UpdateInfo{
				{Package: "htop", CurrentVersion: "3.2.1", AvailableVersion: "3.2.2"},
			},
		},
	}
	buf, err := execUpdateCmd(t, []string{"update", "--check"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "Checking for updates")
	assert.Contains(t, output, "brew: htop 3.2.1 → 3.2.2")
}

func TestUpdateCmd_UnsupportedStillRuns(t *testing.T) {
	adapters := []manager.Adapter{
		&mockAdapter{name: "pipx", checkUpdatesErr: manager.ErrCheckUnsupported},
	}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "updated packages via pipx")
}

func TestUpdateCmd_NoUpdates(t *testing.T) {
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}}
	buf, err := execUpdateCmd(t, []string{"update"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "Checking for updates")
	assert.NotContains(t, output, "\n\n", "piped output must not contain blank lines from suppressed progress")
	assert.Contains(t, output, "brew: No updates available")
	assert.Contains(t, output, "All up to date")
}

func TestUpdateCheck_TTYGlyphs(t *testing.T) {
	forceOutputTTY(t)
	adapters := []manager.Adapter{&mockAdapter{name: "brew"}}
	buf, err := execUpdateCmd(t, []string{"update", "--check"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "▪ Refreshing package metadata...")
	assert.Contains(t, output, "▪ Checking for updates...")
}

// mockPhaser is an adapter with split user/system update phases.
type mockPhaser struct {
	mockAdapter

	mu      sync.Mutex
	events  []string
	noStdin map[string]bool
	refs    bool
	refsErr error
	userUpd []manager.UpdateInfo
	sysUpd  []manager.UpdateInfo
	userErr error
	sysErr  error
}

func (m *mockPhaser) record(ctx context.Context, event string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.noStdin == nil {
		m.noStdin = make(map[string]bool)
	}
	m.events = append(m.events, event)
	m.noStdin[event] = manager.NoStdin(ctx)
}

func (m *mockPhaser) UpdateUser(ctx context.Context, _ string) error {
	m.record(ctx, "user")
	return m.userErr
}

func (m *mockPhaser) UpdateSystem(ctx context.Context, _ string) error {
	m.record(ctx, "system")
	return m.sysErr
}

func (m *mockPhaser) CheckUser(context.Context, string) ([]manager.UpdateInfo, error) {
	return m.userUpd, m.userErr
}

func (m *mockPhaser) CheckSystem(context.Context, string) ([]manager.UpdateInfo, error) {
	return m.sysUpd, m.sysErr
}

func (m *mockPhaser) HasSystemRefs(context.Context, string) (bool, error) {
	return m.refs, m.refsErr
}

func TestUpdateCmd_PhaserUserPhaseHasNoStdin(t *testing.T) {
	phaser := &mockPhaser{mockAdapter: mockAdapter{name: "flatpak"}}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, []manager.Adapter{phaser})
	require.NoError(t, err)
	require.Equal(t, []string{"user"}, phaser.events)
	assert.True(t, phaser.noStdin["user"], "user phase must not read the terminal")
	assert.NotContains(t, buf.String(), "system update")
}

func TestUpdateCmd_PhaserSystemPhaseSerialWithStdin(t *testing.T) {
	phaser := &mockPhaser{mockAdapter: mockAdapter{name: "flatpak"}, refs: true}
	probe := &inflight{}
	regular := &mockAdapter{name: "dnf", UpdateFunc: func(_ context.Context, _ string) error {
		probe.enter()
		defer probe.exit()
		return nil
	}}
	buf, err := execUpdateCmdSudo(t, []string{"update", "-y"}, []manager.Adapter{regular, phaser},
		func(context.Context) bool { return true }, nil, true)
	require.NoError(t, err)
	require.Equal(t, []string{"user", "system"}, phaser.events)
	assert.True(t, phaser.noStdin["user"])
	assert.False(t, phaser.noStdin["system"], "auth phase keeps the terminal")
	output := buf.String()
	assert.Contains(t, output, "flatpak system update — authentication may be required (polkit)")
	assert.Contains(t, output, "updated packages via flatpak system")
}

func TestUpdateCmd_PhaserNoSystemRefsSkipsSystem(t *testing.T) {
	phaser := &mockPhaser{mockAdapter: mockAdapter{name: "flatpak"}}
	buf, err := execUpdateCmd(t, []string{"update", "-y"}, []manager.Adapter{phaser})
	require.NoError(t, err)
	assert.Equal(t, []string{"user"}, phaser.events)
	assert.NotContains(t, buf.String(), "polkit")
}

func TestUpdateCmd_FlatpakUserOnlySkipsSystem(t *testing.T) {
	phaser := &mockPhaser{mockAdapter: mockAdapter{name: "flatpak"}, refs: true}
	buf, err := execUpdateCmd(t, []string{"update", "-y", "--flatpak-user-only"}, []manager.Adapter{phaser})
	require.NoError(t, err)
	assert.Equal(t, []string{"user"}, phaser.events)
	assert.NotContains(t, buf.String(), "system update")
}

func TestUpdateCmd_PhaserNonTTYSystemFailsAfterUser(t *testing.T) {
	phaser := &mockPhaser{mockAdapter: mockAdapter{name: "flatpak"}, refs: true}
	_, err := execUpdateCmdSudo(t, []string{"update", "-y"}, []manager.Adapter{phaser},
		func(context.Context) bool { return true }, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "need a terminal")
	assert.Equal(t, []string{"user"}, phaser.events, "user phase still runs; system never starts")
}

func TestUpdateCmd_PhaserPolkitWarningOnce(t *testing.T) {
	old := polkitChallenge
	polkitChallenge = func(context.Context, string) (bool, error) { return true, nil }
	t.Cleanup(func() { polkitChallenge = old })

	phaser := &mockPhaser{mockAdapter: mockAdapter{name: "flatpak"}, refs: true}
	buf, err := execUpdateCmdSudo(t, []string{"update", "-y"}, []manager.Adapter{phaser},
		func(context.Context) bool { return true }, nil, true)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "polkit authentication may be required")
	assert.Equal(t, 1, strings.Count(output, "polkit authentication may be required"))
}

func TestUpdateCmd_CheckScopesSystemWhenTTY(t *testing.T) {
	phaser := &mockPhaser{
		mockAdapter: mockAdapter{name: "flatpak"},
		refs:        true,
		userUpd:     []manager.UpdateInfo{{Package: "com.spotify.Client"}},
		sysUpd:      []manager.UpdateInfo{{Package: "org.gimp.GIMP"}},
	}
	buf, err := execUpdateCmdSudo(t, []string{"update", "--check"}, []manager.Adapter{phaser},
		func(context.Context) bool { return true }, nil, true)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "flatpak: com.spotify.Client")
	assert.Contains(t, output, "flatpak (system): org.gimp.GIMP")
	assert.Contains(t, output, "flatpak system check — authentication may be required (polkit)")
}

func TestUpdateCmd_CheckNonTTYSystemSkipped(t *testing.T) {
	phaser := &mockPhaser{
		mockAdapter: mockAdapter{name: "flatpak"},
		refs:        true,
		userUpd:     []manager.UpdateInfo{{Package: "com.spotify.Client"}},
	}
	buf, err := execUpdateCmdSudo(t, []string{"update", "--check"}, []manager.Adapter{phaser},
		func(context.Context) bool { return true }, nil, false)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "flatpak: system check skipped (authentication required)")
	assert.NotContains(t, output, "flatpak (system)")
}

func TestUpdateCmd_CheckFlatpakUserOnlySkipsSystem(t *testing.T) {
	phaser := &mockPhaser{mockAdapter: mockAdapter{name: "flatpak"}, refs: true}
	buf, err := execUpdateCmdSudo(t, []string{"update", "--check", "--flatpak-user-only"}, []manager.Adapter{phaser},
		func(context.Context) bool { return true }, nil, true)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "system check")
}
