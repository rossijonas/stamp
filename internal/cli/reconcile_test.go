package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

func TestReconcile_NoDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No drift detected")
	// strategy banner always shown
	assert.Contains(t, output, "reconcile is a fallback")
	assert.Contains(t, output, "prefer 'stamp install")
}

// reliabilityMock wraps manager.Mock and reports a fixed reliability.
type reliabilityMock struct {
	manager.Mock
	reliability manager.ReconcileReliability
}

func (r *reliabilityMock) ReconcileReliability() manager.ReconcileReliability {
	return r.reliability
}

func TestReconcile_ReliabilityNotes_AlwaysShown(t *testing.T) {
	tests := []struct {
		name     string
		level    manager.ReconcileReliability
		wantNote string
	}{
		{"over inclusive", manager.ReliabilityOverInclusive, "lists all installed packages"},
		{"reliable", manager.ReliabilityReliable, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &reliabilityMock{
				Mock:        manager.Mock{ManagerName: "dnf"},
				reliability: tt.level,
			}
			// OverInclusive uses the snap message; keep manager name consistent
			if tt.level == manager.ReliabilityOverInclusive {
				adapter.ManagerName = "snap"
			}

			snapDir := setupSnapshots(t, []state.Snapshot{
				{Manager: adapter.ManagerName, Packages: []string{}},
			})
			t.Setenv("XDG_DATA_HOME", snapDir)

			buf, err := execCmd(t, []string{"reconcile", "-d"}, []manager.Adapter{adapter})
			require.NoError(t, err)
			output := buf.String()
			// banner always present
			assert.Contains(t, output, "reconcile is a fallback")
			if tt.wantNote != "" {
				assert.Contains(t, output, tt.wantNote)
			} else {
				assert.NotContains(t, output, "lists all installed packages")
			}
		})
	}
}

func TestReconcile_ReliabilityNotes_OnlyForReporters(t *testing.T) {
	// manager.Mock does NOT implement ReliabilityReporter → no notes.
	adapter := &manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"lazygit"}}
	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "-d"}, []manager.Adapter{adapter})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "reconcile is a fallback")
	assert.NotContains(t, output, "false positive")
	assert.NotContains(t, output, "lists all installed packages")
}

func TestReconcile_DriftAndAutoTrack(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "ripgrep (brew)")
	assert.Contains(t, output, "Tracked 1 package(s)")
}

func TestReconcile_AutoTrackedEntriesHaveReconciledOrigin(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")

	require.NoError(t, os.MkdirAll(tmpDir, 0700))
	require.NoError(t, os.WriteFile(manifestPath, []byte("version = 1\nsystem = \"linux\"\n"), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())

	loaded, err := manifest.Load(manifestPath)
	require.NoError(t, err)
	require.Len(t, loaded.Packages, 1)
	assert.Equal(t, "ripgrep", loaded.Packages[0].Name)
	assert.Equal(t, manifest.OriginReconciled, loaded.Packages[0].Origin)
}

func TestReconcile_MissingPackageWarnsOnNoDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "htop"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "lazygit"
manager = "brew"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "1 manifest package(s) not installed")
	assert.Contains(t, output, "htop (brew)")
	assert.Contains(t, output, "stamp ls --type missing")
	assert.Contains(t, output, "stamp restore")
	assert.NotContains(t, output, "No drift detected")
}

func TestReconcile_MissingAndDriftBothWarned(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "htop"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "ripgrep (brew)")
	assert.Contains(t, output, "1 manifest package(s) not installed")
	assert.Contains(t, output, "htop (brew)")
	assert.Contains(t, output, "Tracked 1 package(s)")
}

func TestReconcile_DryRun_MissingWarnsWithoutTracking(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "htop"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "htop"
manager = "brew"
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile", "--dry-run"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "1 manifest package(s) not installed")
	assert.Contains(t, output, "stamp restore")
	assert.Contains(t, output, "Use `stamp reconcile` without --dry-run to track")
	assert.NotContains(t, output, "Tracked")
}

func TestReconcile_NoMissingNoWarning(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "lazygit"
manager = "brew"
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())
	output := buf.String()
	assert.Contains(t, output, "No drift detected")
	assert.NotContains(t, output, "not installed")
}

func TestReconcile_MissingGroupsSkipped(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "devtools"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.toml")
	content := `version = 1
system = "linux"

[[packages]]
name = "devtools"
manager = "brew"
group = true
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(tmpDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	require.NoError(t, root.Execute())
	assert.NotContains(t, buf.String(), "not installed")
}

func TestReconcile_MultipleManagers(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "ripgrep"},
		},
		&manager.Mock{
			ManagerName:   "dnf",
			InstalledPkgs: []string{"htop", "tmux", "curl"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
		{Manager: "dnf", Packages: []string{"htop"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 3 new package(s)")
	assert.Contains(t, output, "ripgrep (brew)")
	assert.Contains(t, output, "tmux (dnf)")
	assert.Contains(t, output, "curl (dnf)")
	assert.Contains(t, output, "Tracked 3 package(s)")
}

func TestReconcile_FirstRun(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "initial baseline snapshot taken")
}

func TestReconcile_FirstRunDryRun(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--dry-run"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No baseline snapshot exists")
}

func TestReconcile_AlreadyTracked(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "ripgrep"
manager = "brew"
`

	require.NoError(t, os.MkdirAll(manifestDir, 0700))
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(manifestDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	// A manifest-tracked package is not new drift — reconcile must not report
	// it as discovered or claim to track nothing after discovering it.
	assert.Contains(t, output, "No drift detected")
	assert.NotContains(t, output, "Discovered")
	assert.NotContains(t, output, "Tracked 0")
}

func TestReconcile_RepoAlreadyTracked(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:    "brew",
			InstalledPkgs:  []string{"lazygit"},
			InstalledRepos: []manager.RepositoryInfo{{Name: "anomalyco/tap"}},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "anomalyco/tap"
manager = "brew"
`

	require.NoError(t, os.MkdirAll(manifestDir, 0700))
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(manifestDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No drift detected")
	assert.NotContains(t, output, "Discovered")
	assert.NotContains(t, output, "Tracked")
}

func TestReconcile_RepoAlreadyTracked_DryRun(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:    "brew",
			InstalledPkgs:  []string{"lazygit"},
			InstalledRepos: []manager.RepositoryInfo{{Name: "anomalyco/tap"}},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	manifestContent := `version = 1
system = "linux"

[[repositories]]
name = "anomalyco/tap"
manager = "brew"
`

	require.NoError(t, os.MkdirAll(manifestDir, 0700))
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(manifestDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile", "--dry-run"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No drift detected")
	assert.NotContains(t, output, "Discovered")
	assert.NotContains(t, output, "Use `stamp reconcile` without --dry-run to track")
}

func TestReconcile_MixedTrackedAndUntracked(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "jq"
manager = "brew"
`

	require.NoError(t, os.MkdirAll(manifestDir, 0700))
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(manifestDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	// Only genuinely untracked drift (ripgrep) is discovered and tracked; the
	// manifest-tracked jq must not appear as drift.
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "ripgrep (brew)")
	assert.NotContains(t, output, "jq (brew)")
	assert.Contains(t, output, "Tracked 1 package(s)")

	loaded, err := manifest.Load(manifestPath)
	require.NoError(t, err)
	require.Len(t, loaded.Packages, 2)
}

func TestReconcile_ManagerFlag_Success(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
		&manager.Mock{
			ManagerName:   "dnf",
			InstalledPkgs: []string{"htop"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
		{Manager: "dnf", Packages: []string{"htop"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "-m", "brew"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 2 new package(s)")
	assert.Contains(t, output, "jq")
	assert.Contains(t, output, "ripgrep")
}

func TestReconcile_ManagerFlag_NotFound(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	_, err := execCmd(t, []string{"reconcile", "-m", "nonexistent"}, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available on this system")
	assert.Equal(t, ExitUnavailable, exitCodeFor(err))
}

func TestReconcile_RepoDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:    "brew",
			InstalledPkgs:  []string{"lazygit"},
			InstalledRepos: []manager.RepositoryInfo{{Name: "aovestdipaperino/tap"}, {Name: "yvgude/lean-ctx"}},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 2 new repository(ies)")
	assert.Contains(t, output, "aovestdipaperino/tap (brew)")
	assert.Contains(t, output, "yvgude/lean-ctx (brew)")
	assert.Contains(t, output, "Tracked 0 package(s), 2 repository(ies)")
}

func TestReconcile_RepoAndPackageDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:    "brew",
			InstalledPkgs:  []string{"lazygit", "ripgrep"},
			InstalledRepos: []manager.RepositoryInfo{{Name: "aovestdipaperino/tap"}},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Discovered 1 new repository(ies)")
	assert.Contains(t, output, "Tracked 1 package(s), 1 repository(ies)")
}

func TestReconcile_DryRun(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--dry-run"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Use `stamp reconcile` without --dry-run to track")
	assert.NotContains(t, output, "Tracked")
}

func TestReconcile_DryRun_NoDrift(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--dry-run"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No drift detected")
}

func TestReconcile_YFlag_Compatibility(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "--yes"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Tracked 1 package(s)")
}

func TestReconcile_SnapshotsUpdatedOnNoDrift(t *testing.T) {
	// Simulate: package in baseline → removed externally → reconcile (no drift, save snapshot without it) → reinstalled → reconcile detects drift
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{},
		},
	}

	// Old snapshot has lazygit (which was externally removed)
	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	// First reconcile: no drift (lazygit removed, nothing added), snapshot saved without lazygit
	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No drift detected")

	// Manually verify snapshot no longer has lazygit
	loaded, err := state.Load(filepath.Join(snapDir, "stamp", "snapshots"), "brew")
	require.NoError(t, err)
	assert.NotContains(t, loaded.Packages, "lazygit")

	// Simulate: lazygit reinstalled externally
	adapters[0].(*manager.Mock).InstalledPkgs = []string{"lazygit"}

	// Second reconcile: should now detect lazygit as added
	buf2, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf2.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "lazygit (brew)")
	assert.Contains(t, output, "Tracked 1 package(s)")
}

func TestReconcile_DryRun_ShortFlag(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq", "ripgrep"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "jq"}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile", "-d"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 1 new package(s)")
	assert.Contains(t, output, "Use `stamp reconcile` without --dry-run to track")
	assert.NotContains(t, output, "Tracked")
}

func TestReconcile_GoAdapter(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "go",
			InstalledPkgs: []string{"github.com/golangci/golangci-lint", "github.com/example/tool"},
		},
	}

	// Old snapshot has no go packages → full drift detected
	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "go", Packages: []string{}},
	})

	t.Setenv("XDG_DATA_HOME", snapDir)

	buf, err := execCmd(t, []string{"reconcile"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Discovered 2 new package(s)")
	assert.Contains(t, output, "github.com/golangci/golangci-lint (go)")
	assert.Contains(t, output, "github.com/example/tool (go)")
}

func TestReconcile_FilteredDriftStillWarnsMissing(t *testing.T) {
	// All snapshot drift is manifest-tracked (filtered out), but a tracked
	// package was removed externally — the missing warning must still fire.
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:   "brew",
			InstalledPkgs: []string{"lazygit", "jq"},
		},
	}

	snapDir := setupSnapshots(t, []state.Snapshot{
		{Manager: "brew", Packages: []string{"lazygit", "htop"}},
	})
	t.Setenv("XDG_DATA_HOME", snapDir)

	manifestDir := t.TempDir()
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	manifestContent := `version = 1
system = "linux"

[[packages]]
name = "jq"
manager = "brew"

[[packages]]
name = "htop"
manager = "brew"
`

	require.NoError(t, os.MkdirAll(manifestDir, 0700))
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0600))

	root := NewRootCmd(WithAdapters(adapters), WithManifestPath(manifestPath), WithConfigPath(filepath.Join(manifestDir, "config.toml")))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"reconcile"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	// Drift (jq) is already tracked → nothing discovered, no tracking line.
	// htop is in the manifest but gone from the system → missing warning.
	assert.Contains(t, output, "1 manifest package(s) not installed")
	assert.Contains(t, output, "htop (brew)")
	assert.NotContains(t, output, "Discovered")
	assert.NotContains(t, output, "Tracked")
}

func TestFilterDiscoveredByManifest(t *testing.T) {
	tests := []struct {
		name            string
		discovered      []discoveredPkg
		discoveredRepos []discoveredRepo
		manifest        *manifest.Manifest
		wantPkgs        []discoveredPkg
		wantRepos       []discoveredRepo
	}{
		{
			name:       "empty inputs",
			discovered: nil,
			manifest:   &manifest.Manifest{},
			wantPkgs:   nil,
			wantRepos:  nil,
		},
		{
			name:       "nil manifest returns inputs unchanged",
			discovered: []discoveredPkg{{Name: "jq", Manager: "brew"}},
			manifest:   nil,
			wantPkgs:   []discoveredPkg{{Name: "jq", Manager: "brew"}},
		},
		{
			name:       "tracked package filtered",
			discovered: []discoveredPkg{{Name: "jq", Manager: "brew"}},
			manifest: &manifest.Manifest{
				Packages: []manifest.Package{{Name: "jq", Manager: "brew"}},
			},
			wantPkgs: nil,
		},
		{
			name:       "same name different manager kept",
			discovered: []discoveredPkg{{Name: "jq", Manager: "brew"}},
			manifest: &manifest.Manifest{
				Packages: []manifest.Package{{Name: "jq", Manager: "dnf"}},
			},
			wantPkgs: []discoveredPkg{{Name: "jq", Manager: "brew"}},
		},
		{
			name:       "untracked package kept",
			discovered: []discoveredPkg{{Name: "ripgrep", Manager: "brew"}},
			manifest:   &manifest.Manifest{},
			wantPkgs:   []discoveredPkg{{Name: "ripgrep", Manager: "brew"}},
		},
		{
			name:            "tracked repo filtered",
			discoveredRepos: []discoveredRepo{{Name: "anomalyco/tap", Manager: "brew"}},
			manifest: &manifest.Manifest{
				Repositories: []manifest.Repository{{Name: "anomalyco/tap", Manager: "brew"}},
			},
			wantRepos: nil,
		},
		{
			name:            "untracked repo kept",
			discoveredRepos: []discoveredRepo{{Name: "anomalyco/tap", Manager: "brew"}},
			manifest:        &manifest.Manifest{},
			wantRepos:       []discoveredRepo{{Name: "anomalyco/tap", Manager: "brew"}},
		},
		{
			name: "mixed batch keeps only untracked",
			discovered: []discoveredPkg{
				{Name: "jq", Manager: "brew"},
				{Name: "ripgrep", Manager: "brew"},
			},
			discoveredRepos: []discoveredRepo{
				{Name: "anomalyco/tap", Manager: "brew"},
				{Name: "yvgude/lean-ctx", Manager: "brew"},
			},
			manifest: &manifest.Manifest{
				Packages:     []manifest.Package{{Name: "jq", Manager: "brew"}},
				Repositories: []manifest.Repository{{Name: "anomalyco/tap", Manager: "brew"}},
			},
			wantPkgs:  []discoveredPkg{{Name: "ripgrep", Manager: "brew"}},
			wantRepos: []discoveredRepo{{Name: "yvgude/lean-ctx", Manager: "brew"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgs, repos := filterDiscoveredByManifest(tt.discovered, tt.discoveredRepos, tt.manifest)
			assert.Equal(t, tt.wantPkgs, pkgs)
			assert.Equal(t, tt.wantRepos, repos)
		})
	}
}

// setupSnapshots saves snapshots to {tmpDir}/stamp/snapshots/ and returns tmpDir.
func setupSnapshots(t *testing.T, snaps []state.Snapshot) string {
	t.Helper()
	dir := t.TempDir()
	for _, s := range snaps {
		snapDir := filepath.Join(dir, "stamp", "snapshots")
		require.NoError(t, os.MkdirAll(snapDir, 0700))
		require.NoError(t, state.Save(snapDir, s))
	}
	return dir
}
