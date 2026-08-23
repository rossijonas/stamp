package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordExec captures the command name and args passed to the executor.
type recordExec struct {
	name string
	args []string
}

func newRecordExec(t *testing.T, output string, err error) (*recordExec, Executor) {
	t.Helper()
	rec := &recordExec{}
	exec := func(_ context.Context, name string, args ...string) ([]byte, error) {
		rec.name = name
		rec.args = append([]string(nil), args...)
		if err != nil {
			return nil, err
		}
		return []byte(output), nil
	}
	return rec, exec
}

func TestPreview_InterfaceCompileChecks(t *testing.T) {
	// The compile-time assertions live on each adapter (var _ Previewer).
	// This test guards against accidental interface regressions via reflection.
	for _, a := range []any{NewDNF("dnf"), NewAPT("apt"), NewPacman(), NewBrew(), NewFlatpak(), NewZypper(), NewNpm()} {
		_, ok := a.(Previewer)
		require.True(t, ok, "expected %T to implement Previewer", a)
	}
	// Fallback-only adapters must NOT implement Previewer.
	for _, a := range []any{NewSnap(), NewCargo(), NewGo(), NewMacPorts(), NewParu(), NewPipx(), NewUv()} {
		_, ok := a.(Previewer)
		require.False(t, ok, "expected %T to NOT implement Previewer", a)
	}
}

func TestPreview_Adapters(t *testing.T) {
	ctx := WithYes(context.Background())
	installOutput := "preview install output\n"

	tests := []struct {
		name     string
		adapter  string
		build    func(exec Executor) Previewer
		wantName string
		wantArgs []string // must all appear in the recorded args
	}{
		{
			name:     "apt preview install",
			build:    func(exec Executor) Previewer { a := NewAPT("apt"); a.exec = exec; return a },
			wantName: "apt",
			wantArgs: []string{"install", "--assume-no", "htop"},
		},
		{
			name:     "apt preview remove",
			build:    func(exec Executor) Previewer { a := NewAPT("apt"); a.exec = exec; return a },
			wantName: "apt",
			wantArgs: []string{"remove", "--assume-no", "htop"},
		},
		{
			name:     "pacman preview install",
			build:    func(exec Executor) Previewer { a := NewPacman(); a.exec = exec; return a },
			wantName: "pacman",
			wantArgs: []string{"-S", "--print", "htop"},
		},
		{
			name:     "pacman preview remove",
			build:    func(exec Executor) Previewer { a := NewPacman(); a.exec = exec; return a },
			wantName: "pacman",
			wantArgs: []string{"-R", "--print", "htop"},
		},
		{
			name:     "brew preview install",
			build:    func(exec Executor) Previewer { a := NewBrew(); a.exec = exec; return a },
			wantName: "brew",
			wantArgs: []string{"install", "--dry-run", "htop"},
		},
		{
			name:     "brew preview remove",
			build:    func(exec Executor) Previewer { a := NewBrew(); a.exec = exec; return a },
			wantName: "brew",
			wantArgs: []string{"uninstall", "--dry-run", "htop"},
		},
		{
			name:     "brew preview cask install",
			build:    func(exec Executor) Previewer { a := NewBrew(); a.exec = exec; return a },
			wantName: "brew",
			wantArgs: []string{"install", "--dry-run", "--cask", "spotify"},
		},
		{
			name:     "flatpak preview install",
			build:    func(exec Executor) Previewer { a := NewFlatpak(); a.exec = exec; return a },
			wantName: "flatpak",
			wantArgs: []string{"install", "--dry-run", "com.spotify.Client"},
		},
		{
			name:     "flatpak preview remove",
			build:    func(exec Executor) Previewer { a := NewFlatpak(); a.exec = exec; return a },
			wantName: "flatpak",
			wantArgs: []string{"uninstall", "--dry-run", "com.spotify.Client"},
		},
		{
			name:     "zypper preview install",
			build:    func(exec Executor) Previewer { a := NewZypper(); a.exec = exec; return a },
			wantName: "zypper",
			wantArgs: []string{"install", "--dry-run", "htop"},
		},
		{
			name:     "zypper preview remove",
			build:    func(exec Executor) Previewer { a := NewZypper(); a.exec = exec; return a },
			wantName: "zypper",
			wantArgs: []string{"remove", "--dry-run", "htop"},
		},
		{
			name:     "npm preview install",
			build:    func(exec Executor) Previewer { a := NewNpm(); a.exec = exec; return a },
			wantName: "npm",
			wantArgs: []string{"install", "--dry-run", "-g", "htop"},
		},
		{
			name:     "npm preview remove",
			build:    func(exec Executor) Previewer { a := NewNpm(); a.exec = exec; return a },
			wantName: "npm",
			wantArgs: []string{"uninstall", "--dry-run", "-g", "htop"},
		},
		{
			name:     "dnf preview install via sudo",
			build:    func(exec Executor) Previewer { a := NewDNF("dnf"); a.exec = exec; return a },
			wantName: "sudo",
			wantArgs: []string{"install", "--assumeno", "htop"},
		},
		{
			name:     "dnf preview remove via sudo",
			build:    func(exec Executor) Previewer { a := NewDNF("dnf"); a.exec = exec; return a },
			wantName: "sudo",
			wantArgs: []string{"remove", "--assumeno", "htop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, exec := newRecordExec(t, installOutput, nil)
			a := tt.build(exec)

			// Exercise both directions and check the matched operation args.
			useRemove := contains(tt.wantArgs, "remove") || contains(tt.wantArgs, "uninstall") || contains(tt.wantArgs, "-R")
			callCtx := ctx
			if contains(tt.wantArgs, "--cask") {
				callCtx = WithCask(ctx)
			}
			rec.args = nil
			var pv Preview
			var err error
			if useRemove {
				pv, err = a.PreviewRemove(callCtx, tt.wantArgs[len(tt.wantArgs)-1])
			} else {
				pv, err = a.PreviewInstall(callCtx, tt.wantArgs[len(tt.wantArgs)-1])
			}
			require.NoError(t, err)
			assert.Equal(t, installOutput, pv.Output)
			assert.Equal(t, tt.wantName, rec.name)
			for _, want := range tt.wantArgs {
				assert.Contains(t, rec.args, want, "expected arg %q in %v", want, rec.args)
			}
		})
	}
}

func TestPreview_Reinstall(t *testing.T) {
	ctx := context.Background()
	output := "reinstall preview\n"
	tests := []struct {
		name     string
		adapter  string
		build    func(exec Executor) Previewer
		wantName string
		wantArgs []string // must all appear in the recorded args
	}{
		{
			name:     "apt preview reinstall",
			build:    func(exec Executor) Previewer { a := NewAPT("apt"); a.exec = exec; return a },
			wantName: "apt",
			wantArgs: []string{"install", "--reinstall", "--assume-no", "htop"},
		},
		{
			name:     "pacman preview reinstall",
			build:    func(exec Executor) Previewer { a := NewPacman(); a.exec = exec; return a },
			wantName: "pacman",
			wantArgs: []string{"-S", "--print", "htop"},
		},
		{
			name:     "flatpak preview reinstall",
			build:    func(exec Executor) Previewer { a := NewFlatpak(); a.exec = exec; return a },
			wantName: "flatpak",
			wantArgs: []string{"install", "--dry-run", "com.spotify.Client"},
		},
		{
			name:     "zypper preview reinstall",
			build:    func(exec Executor) Previewer { a := NewZypper(); a.exec = exec; return a },
			wantName: "zypper",
			wantArgs: []string{"install", "--force", "--dry-run", "htop"},
		},
		{
			name:     "npm preview reinstall",
			build:    func(exec Executor) Previewer { a := NewNpm(); a.exec = exec; return a },
			wantName: "npm",
			wantArgs: []string{"install", "--dry-run", "-g", "htop"},
		},
		{
			name:     "dnf preview reinstall via sudo",
			build:    func(exec Executor) Previewer { a := NewDNF("dnf"); a.exec = exec; return a },
			wantName: "sudo",
			wantArgs: []string{"reinstall", "--assumeno", "htop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, exec := newRecordExec(t, output, nil)
			a := tt.build(exec)
			callCtx := ctx
			if contains(tt.wantArgs, "--cask") {
				callCtx = WithCask(ctx)
			}
			pv, err := a.PreviewReinstall(callCtx, tt.wantArgs[len(tt.wantArgs)-1])
			require.NoError(t, err)
			assert.Equal(t, output, pv.Output)
			assert.Equal(t, tt.wantName, rec.name)
			for _, want := range tt.wantArgs {
				assert.Contains(t, rec.args, want, "expected arg %q in %v", want, rec.args)
			}
		})
	}
}

func TestPreview_NoopDetection(t *testing.T) {
	ctx := WithYes(context.Background())
	tests := []struct {
		name    string
		build   func(e Executor) Previewer
		noopOut string
		realOut string
	}{
		{name: "dnf", build: func(e Executor) Previewer { a := NewDNF("dnf"); a.exec = e; return a },
			noopOut: "Nothing to do.\n", realOut: "Package htop will be installed\n"},
		{name: "apt", build: func(e Executor) Previewer { a := NewAPT("apt"); a.exec = e; return a },
			noopOut: "htop is already the newest version (3.4.1).\n0 upgraded, 0 newly installed, 0 to remove\n",
			realOut: "The following NEW packages will be installed:\n  htop\n"},
		{name: "brew", build: func(e Executor) Previewer { a := NewBrew(); a.exec = e; return a },
			noopOut: "Warning: htop 3.4.1 is already installed\n", realOut: "Would install htop 3.4.1\n"},
		{name: "flatpak", build: func(e Executor) Previewer { a := NewFlatpak(); a.exec = e; return a },
			noopOut: "Nothing to do.\n", realOut: "Installation of com.spotify.Client\n"},
		{name: "zypper", build: func(e Executor) Previewer { a := NewZypper(); a.exec = e; return a },
			noopOut: "Nothing to do.\n", realOut: "The following NEW package is going to be installed:\n  htop\n"},
		{name: "npm", build: func(e Executor) Previewer { a := NewNpm(); a.exec = e; return a },
			noopOut: "up to date, nothing changed\n", realOut: "added 1 package\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" noop", func(t *testing.T) {
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte(tt.noopOut), nil
			})
			pv, err := a.PreviewInstall(ctx, "htop")
			require.NoError(t, err)
			require.True(t, pv.Noop, "expected no-op preview for %s", tt.name)
			assert.Equal(t, tt.noopOut, pv.Output)
		})
		t.Run(tt.name+" real transaction", func(t *testing.T) {
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte(tt.realOut), nil
			})
			pv, err := a.PreviewInstall(ctx, "htop")
			require.NoError(t, err)
			require.False(t, pv.Noop, "expected a real transaction for %s", tt.name)
			assert.Equal(t, tt.realOut, pv.Output)
		})
	}
}

func TestPreview_NonZeroExitStillReturnsOutput(t *testing.T) {
	// Some native dry-runs (dnf --assumeno) display the full transaction, then
	// abort with a non-zero exit. The preview must surface that output rather
	// than discard it (which would fall back to Info and show the wrong thing).
	previewExitErr := errors.New("exit status 1")
	ctx := WithYes(context.Background())
	tests := []struct {
		name   string
		build  func(exec Executor) Previewer
		method string
	}{
		{name: "dnf", build: func(e Executor) Previewer { a := NewDNF("dnf"); a.exec = e; return a }, method: "remove"},
		{name: "apt", build: func(e Executor) Previewer { a := NewAPT("apt"); a.exec = e; return a }, method: "install"},
		{name: "pacman", build: func(e Executor) Previewer { a := NewPacman(); a.exec = e; return a }, method: "remove"},
		{name: "flatpak", build: func(e Executor) Previewer { a := NewFlatpak(); a.exec = e; return a }, method: "install"},
		{name: "zypper", build: func(e Executor) Previewer { a := NewZypper(); a.exec = e; return a }, method: "reinstall"},
		{name: "npm", build: func(e Executor) Previewer { a := NewNpm(); a.exec = e; return a }, method: "remove"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" returns output on non-zero exit", func(t *testing.T) {
			txOut := []byte("Removing: htop\nTransaction Summary:\n  Removing: 1 package\n")
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return txOut, previewExitErr
			})
			var pv Preview
			var err error
			switch tt.method {
			case "install":
				pv, err = a.PreviewInstall(ctx, "htop")
			case "remove":
				pv, err = a.PreviewRemove(ctx, "htop")
			case "reinstall":
				pv, err = a.PreviewReinstall(ctx, "htop")
			}
			require.NoError(t, err)
			assert.Equal(t, string(txOut), pv.Output)
		})
		t.Run(tt.name+" errors on empty output", func(t *testing.T) {
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, previewExitErr
			})
			var err error
			switch tt.method {
			case "install":
				_, err = a.PreviewInstall(ctx, "htop")
			case "remove":
				_, err = a.PreviewRemove(ctx, "htop")
			case "reinstall":
				_, err = a.PreviewReinstall(ctx, "htop")
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to preview")
			assert.ErrorIs(t, err, previewExitErr)
		})
	}
}

func TestPreview_RemoveAbsentSignals(t *testing.T) {
	ctx := WithYes(context.Background())
	tests := []struct {
		name   string
		build  func(e Executor) Previewer
		absent string
		real   string
	}{
		{name: "dnf", build: func(e Executor) Previewer { a := NewDNF("dnf"); a.exec = e; return a },
			absent: "Error: No match for argument: htop\n", real: "Removing:\n htop\nTransaction Summary:\n"},
		{name: "apt", build: func(e Executor) Previewer { a := NewAPT("apt"); a.exec = e; return a },
			absent: "Package 'htop' is not installed, so not removed\n0 upgraded, 0 newly installed, 0 to remove\n",
			real:   "Removing:\n htop\n0 upgraded, 0 newly installed, 1 to remove\n"},
		{name: "pacman", build: func(e Executor) Previewer { a := NewPacman(); a.exec = e; return a },
			absent: "error: package 'htop' was not found\n", real: "htop 3.4.1-1\n"},
		{name: "brew", build: func(e Executor) Previewer { a := NewBrew(); a.exec = e; return a },
			absent: "Error: No such keg: /usr/local/Cellar/htop\n", real: "Would remove htop 3.4.1\n"},
		{name: "flatpak", build: func(e Executor) Previewer { a := NewFlatpak(); a.exec = e; return a },
			absent: "error: No such ref\n", real: "Uninstall com.spotify.Client\n"},
		{name: "zypper", build: func(e Executor) Previewer { a := NewZypper(); a.exec = e; return a },
			absent: "Nothing to do.\n", real: "The following package is going to be removed:\n  htop\n"},
		{name: "npm", build: func(e Executor) Previewer { a := NewNpm(); a.exec = e; return a },
			absent: "up to date\n", real: "removed 1 package\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name+" absent is noop", func(t *testing.T) {
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte(tt.absent), nil
			})
			pv, err := a.PreviewRemove(ctx, "htop")
			require.NoError(t, err)
			require.True(t, pv.Noop, "expected absent remove to be a no-op for %s", tt.name)
		})
		t.Run(tt.name+" real removal", func(t *testing.T) {
			a := tt.build(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte(tt.real), nil
			})
			pv, err := a.PreviewRemove(ctx, "htop")
			require.NoError(t, err)
			require.False(t, pv.Noop, "expected a real removal for %s", tt.name)
		})
	}
}

func TestPreview_ReinstallNoop(t *testing.T) {
	ctx := WithYes(context.Background())
	// dnf reinstall of an absent package is a no-op; of an installed package it
	// is a real operation even when the version matches.
	t.Run("dnf absent is noop", func(t *testing.T) {
		a := NewDNF("dnf")
		a.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("Error: package htop is not installed\n"), nil
		}
		pv, err := a.PreviewReinstall(ctx, "htop")
		require.NoError(t, err)
		require.True(t, pv.Noop)
	})
	t.Run("dnf installed reinstall is real", func(t *testing.T) {
		a := NewDNF("dnf")
		a.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("Reinstalling:\n htop 3.4.1-3\nTransaction Summary:\n Reinstalling: 1 package\n"), nil
		}
		pv, err := a.PreviewReinstall(ctx, "htop")
		require.NoError(t, err)
		require.False(t, pv.Noop, "reinstall of an installed package must never be a no-op")
	})
	t.Run("brew reinstall has no preview", func(t *testing.T) {
		a := NewBrew()
		a.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			t.Fatal("exec must not be called: brew reinstall has no --dry-run")
			return nil, nil
		}
		_, err := a.PreviewReinstall(ctx, "htop")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support a dry-run preview")
	})
}

func TestPreview_Adapters_ValidationAndErrors(t *testing.T) {
	previewErr := errors.New("boom")
	tests := []struct {
		name    string
		adapter func(exec Executor) Previewer
	}{
		{name: "dnf", adapter: func(e Executor) Previewer { a := NewDNF("dnf"); a.exec = e; return a }},
		{name: "apt", adapter: func(e Executor) Previewer { a := NewAPT("apt"); a.exec = e; return a }},
		{name: "pacman", adapter: func(e Executor) Previewer { a := NewPacman(); a.exec = e; return a }},
		{name: "brew", adapter: func(e Executor) Previewer { a := NewBrew(); a.exec = e; return a }},
		{name: "flatpak", adapter: func(e Executor) Previewer { a := NewFlatpak(); a.exec = e; return a }},
		{name: "zypper", adapter: func(e Executor) Previewer { a := NewZypper(); a.exec = e; return a }},
		{name: "npm", adapter: func(e Executor) Previewer { a := NewNpm(); a.exec = e; return a }},
	}

	ctx := WithYes(context.Background())
	for _, tt := range tests {
		for _, method := range []string{"install", "remove", "reinstall"} {
			t.Run(tt.name+" "+method+" invalid name", func(t *testing.T) {
				a := tt.adapter(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
					t.Fatal("executor should not be called for invalid package name")
					return nil, nil
				})
				var err error
				switch method {
				case "install":
					_, err = a.PreviewInstall(ctx, "-bad")
				case "remove":
					_, err = a.PreviewRemove(ctx, "-bad")
				case "reinstall":
					_, err = a.PreviewReinstall(ctx, "-bad")
				}
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid package name")
			})
			t.Run(tt.name+" "+method+" exec error wrapped", func(t *testing.T) {
				a := tt.adapter(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
					return nil, previewErr
				})
				var err error
				switch method {
				case "install":
					_, err = a.PreviewInstall(ctx, "htop")
				case "remove":
					_, err = a.PreviewRemove(ctx, "htop")
				case "reinstall":
					_, err = a.PreviewReinstall(ctx, "htop")
				}
				require.Error(t, err)
				if tt.name == "brew" && method == "reinstall" {
					// brew reinstall short-circuits: no exec, so no wrapping
					assert.Contains(t, err.Error(), "does not support a dry-run preview")
					return
				}
				assert.Contains(t, err.Error(), "failed to preview")
				assert.ErrorIs(t, err, previewErr)
			})
		}
	}
}

func TestDNF_PreviewGroup(t *testing.T) {
	rec, exec := newRecordExec(t, "group preview\n", nil)
	d := NewDNF("dnf")
	d.exec = exec

	_, err := d.PreviewInstall(WithGroup(WithYes(context.Background())), "Development Tools")
	require.NoError(t, err)
	assert.Equal(t, "sudo", rec.name)
	assert.Contains(t, rec.args, "group")
	assert.Contains(t, rec.args, "install")
	assert.Contains(t, rec.args, "--assumeno")

	_, err = d.PreviewRemove(WithGroup(WithYes(context.Background())), "Development Tools")
	require.NoError(t, err)
	assert.Contains(t, rec.args, "group")
	assert.Contains(t, rec.args, "remove")
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}
