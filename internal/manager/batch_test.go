package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingExecutor captures the args of the single exec call for batch tests.
func recordingExecutor(rec *[][]string) Executor {
	return func(_ context.Context, _ string, args ...string) ([]byte, error) {
		*rec = append(*rec, args)
		return nil, nil
	}
}

// assertBatchArgs asserts exactly one exec call happened and that its trailing
// args are the batch packages (order preserved).
func assertBatchArgs(t *testing.T, rec [][]string, pkgs ...string) {
	t.Helper()
	require.Len(t, rec, 1, "expected exactly one native invocation")
	got := rec[0]
	require.GreaterOrEqual(t, len(got), len(pkgs))
	assert.Equal(t, pkgs, got[len(got)-len(pkgs):])
}

func TestBatchInstallMany(t *testing.T) {
	t.Parallel()
	pkgs := []string{"aaa", "bbb", "ccc"}

	tests := []struct {
		name string
		adpt func(rec *[][]string) Adapter
	}{
		{"apt", func(rec *[][]string) Adapter { a := NewAPT("apt"); a.exec = recordingExecutor(rec); return a }},
		{"dnf", func(rec *[][]string) Adapter { a := NewDNF("dnf"); a.exec = recordingExecutor(rec); return a }},
		{"pacman", func(rec *[][]string) Adapter { a := NewPacman(); a.exec = recordingExecutor(rec); return a }},
		{"paru", func(rec *[][]string) Adapter { a := NewParu(); a.exec = recordingExecutor(rec); return a }},
		{"zypper", func(rec *[][]string) Adapter { a := NewZypper(); a.exec = recordingExecutor(rec); return a }},
		{"snap", func(rec *[][]string) Adapter { a := NewSnap(); a.exec = recordingExecutor(rec); return a }},
		{"flatpak", func(rec *[][]string) Adapter { a := NewFlatpak(); a.exec = recordingExecutor(rec); return a }},
		{"brew", func(rec *[][]string) Adapter { a := NewBrew(); a.exec = recordingExecutor(rec); return a }},
		{"macports", func(rec *[][]string) Adapter { a := NewMacPorts(); a.exec = recordingExecutor(rec); return a }},
		{"npm", func(rec *[][]string) Adapter { a := NewNpm(); a.exec = recordingExecutor(rec); return a }},
		{"cargo", func(rec *[][]string) Adapter { a := NewCargo(); a.exec = recordingExecutor(rec); return a }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rec [][]string
			a := tt.adpt(&rec)
			bi, ok := a.(BatchInstaller)
			require.True(t, ok, "adapter must implement BatchInstaller")

			err := bi.InstallMany(WithYes(context.Background()), pkgs...)
			require.NoError(t, err)
			assertBatchArgs(t, rec, pkgs...)
		})
	}
}

func TestBatchRemoveMany(t *testing.T) {
	t.Parallel()
	pkgs := []string{"aaa", "bbb"}

	tests := []struct {
		name string
		adpt func(rec *[][]string) Adapter
	}{
		{"apt", func(rec *[][]string) Adapter { a := NewAPT("apt"); a.exec = recordingExecutor(rec); return a }},
		{"dnf", func(rec *[][]string) Adapter { a := NewDNF("dnf"); a.exec = recordingExecutor(rec); return a }},
		{"pacman", func(rec *[][]string) Adapter { a := NewPacman(); a.exec = recordingExecutor(rec); return a }},
		{"paru", func(rec *[][]string) Adapter { a := NewParu(); a.exec = recordingExecutor(rec); return a }},
		{"zypper", func(rec *[][]string) Adapter { a := NewZypper(); a.exec = recordingExecutor(rec); return a }},
		{"snap", func(rec *[][]string) Adapter { a := NewSnap(); a.exec = recordingExecutor(rec); return a }},
		{"flatpak", func(rec *[][]string) Adapter { a := NewFlatpak(); a.exec = recordingExecutor(rec); return a }},
		{"brew", func(rec *[][]string) Adapter { a := NewBrew(); a.exec = recordingExecutor(rec); return a }},
		{"macports", func(rec *[][]string) Adapter { a := NewMacPorts(); a.exec = recordingExecutor(rec); return a }},
		{"npm", func(rec *[][]string) Adapter { a := NewNpm(); a.exec = recordingExecutor(rec); return a }},
		{"cargo", func(rec *[][]string) Adapter { a := NewCargo(); a.exec = recordingExecutor(rec); return a }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rec [][]string
			a := tt.adpt(&rec)
			br, ok := a.(BatchRemover)
			require.True(t, ok, "adapter must implement BatchRemover")

			err := br.RemoveMany(WithYes(context.Background()), pkgs...)
			require.NoError(t, err)
			assertBatchArgs(t, rec, pkgs...)
		})
	}
}

func TestBatchReinstallMany(t *testing.T) {
	t.Parallel()
	pkgs := []string{"aaa", "bbb"}

	tests := []struct {
		name string
		adpt func(rec *[][]string) Adapter
	}{
		{"apt", func(rec *[][]string) Adapter { a := NewAPT("apt"); a.exec = recordingExecutor(rec); return a }},
		{"dnf", func(rec *[][]string) Adapter { a := NewDNF("dnf"); a.exec = recordingExecutor(rec); return a }},
		{"pacman", func(rec *[][]string) Adapter { a := NewPacman(); a.exec = recordingExecutor(rec); return a }},
		{"paru", func(rec *[][]string) Adapter { a := NewParu(); a.exec = recordingExecutor(rec); return a }},
		{"zypper", func(rec *[][]string) Adapter { a := NewZypper(); a.exec = recordingExecutor(rec); return a }},
		{"flatpak", func(rec *[][]string) Adapter { a := NewFlatpak(); a.exec = recordingExecutor(rec); return a }},
		{"brew", func(rec *[][]string) Adapter { a := NewBrew(); a.exec = recordingExecutor(rec); return a }},
		{"macports", func(rec *[][]string) Adapter { a := NewMacPorts(); a.exec = recordingExecutor(rec); return a }},
		{"npm", func(rec *[][]string) Adapter { a := NewNpm(); a.exec = recordingExecutor(rec); return a }},
		{"cargo", func(rec *[][]string) Adapter { a := NewCargo(); a.exec = recordingExecutor(rec); return a }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rec [][]string
			a := tt.adpt(&rec)
			br, ok := a.(BatchReinstaller)
			require.True(t, ok, "adapter must implement BatchReinstaller")

			err := br.ReinstallMany(WithYes(context.Background()), pkgs...)
			require.NoError(t, err)
			assertBatchArgs(t, rec, pkgs...)
		})
	}
}

func TestBatchCapabilityMatrix(t *testing.T) {
	t.Parallel()

	// snap reinstall is remove+install with no native batch form: it must NOT
	// implement BatchReinstaller.
	_, ok := any(NewSnap()).(BatchReinstaller)
	assert.False(t, ok, "snap must not implement BatchReinstaller")

	// go, pipx, uv have no native multi-package support at all.
	for _, a := range []Adapter{NewGo(), NewPipx(), NewUv()} {
		_, ok := a.(BatchInstaller)
		assert.False(t, ok, "%T must not implement BatchInstaller", a)
		_, ok = a.(BatchRemover)
		assert.False(t, ok, "%T must not implement BatchRemover", a)
	}
}

// failingExecutor returns an error from every exec call, to cover the
// error-wrapping branches of the batch methods.
func failingExecutor() Executor {
	return func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, assert.AnError
	}
}

func TestBatchExecError(t *testing.T) {
	t.Parallel()
	pkgs := []string{"aaa", "bbb"}

	tests := []struct {
		name      string
		adpt      func() Adapter
		install   bool
		remove    bool
		reinstall bool
	}{
		{"apt", func() Adapter { a := NewAPT("apt"); a.exec = failingExecutor(); return a }, true, true, true},
		{"dnf", func() Adapter { a := NewDNF("dnf"); a.exec = failingExecutor(); return a }, true, true, true},
		{"pacman", func() Adapter { a := NewPacman(); a.exec = failingExecutor(); return a }, true, true, true},
		{"paru", func() Adapter { a := NewParu(); a.exec = failingExecutor(); return a }, true, true, true},
		{"zypper", func() Adapter { a := NewZypper(); a.exec = failingExecutor(); return a }, true, true, true},
		{"snap", func() Adapter { a := NewSnap(); a.exec = failingExecutor(); return a }, true, true, false},
		{"flatpak", func() Adapter { a := NewFlatpak(); a.exec = failingExecutor(); return a }, true, true, true},
		{"brew", func() Adapter { a := NewBrew(); a.exec = failingExecutor(); return a }, true, true, true},
		{"macports", func() Adapter { a := NewMacPorts(); a.exec = failingExecutor(); return a }, true, true, true},
		{"npm", func() Adapter { a := NewNpm(); a.exec = failingExecutor(); return a }, true, true, true},
		{"cargo", func() Adapter { a := NewCargo(); a.exec = failingExecutor(); return a }, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := WithYes(context.Background())
			a := tt.adpt()

			if tt.install {
				err := a.(BatchInstaller).InstallMany(ctx, pkgs...)
				require.ErrorIs(t, err, assert.AnError)
			}
			if tt.remove {
				err := a.(BatchRemover).RemoveMany(ctx, pkgs...)
				require.ErrorIs(t, err, assert.AnError)
			}
			if tt.reinstall {
				err := a.(BatchReinstaller).ReinstallMany(ctx, pkgs...)
				require.ErrorIs(t, err, assert.AnError)
			}
		})
	}
}

func TestBatchValidationAndConsent(t *testing.T) {
	t.Parallel()

	t.Run("invalid package name aborts before exec", func(t *testing.T) {
		var rec [][]string
		a := NewDNF("dnf")
		a.exec = recordingExecutor(&rec)
		err := a.InstallMany(WithYes(context.Background()), "htop", "-bad")
		require.Error(t, err)
		assert.Empty(t, rec, "no native call must happen on validation failure")
	})

	t.Run("consent required without -y", func(t *testing.T) {
		var rec [][]string
		a := NewDNF("dnf")
		a.exec = recordingExecutor(&rec)
		err := a.InstallMany(context.Background(), "htop", "atop")
		require.ErrorIs(t, err, ErrConfirmationRequired)
		assert.Empty(t, rec)
	})

	t.Run("dnf group context rejects batch install", func(t *testing.T) {
		var rec [][]string
		a := NewDNF("dnf")
		a.exec = recordingExecutor(&rec)
		err := a.InstallMany(WithGroup(WithYes(context.Background())), "Dev Tools")
		require.Error(t, err)
		assert.Empty(t, rec)
	})
}
