package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

func TestMissingFromSystem(t *testing.T) {
	manifestWith := func(pkgs ...manifest.Package) *manifest.Manifest {
		return &manifest.Manifest{Packages: pkgs}
	}

	tests := []struct {
		name      string
		adapters  []manager.Adapter
		manifest  *manifest.Manifest
		wantNames []string
	}{
		{
			name: "missing package detected",
			adapters: []manager.Adapter{
				&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"lazygit"}},
			},
			manifest: manifestWith(
				manifest.Package{Name: "htop", Manager: "brew"},
				manifest.Package{Name: "lazygit", Manager: "brew"},
			),
			wantNames: []string{"htop"},
		},
		{
			name: "all installed yields none",
			adapters: []manager.Adapter{
				&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"htop", "lazygit"}},
			},
			manifest: manifestWith(
				manifest.Package{Name: "htop", Manager: "brew"},
				manifest.Package{Name: "lazygit", Manager: "brew"},
			),
			wantNames: nil,
		},
		{
			name: "list error skips manager",
			adapters: []manager.Adapter{
				&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"lazygit"}, ListErr: errors.New("boom")},
			},
			manifest: manifestWith(
				manifest.Package{Name: "htop", Manager: "brew"},
			),
			wantNames: nil,
		},
		{
			name: "inactive manager skipped",
			adapters: []manager.Adapter{
				&manager.Mock{ManagerName: "flatpak", InstalledPkgs: []string{"org.foo.Bar"}},
			},
			manifest: manifestWith(
				manifest.Package{Name: "htop", Manager: "brew"},
			),
			wantNames: nil,
		},
		{
			name: "groups and casks excluded",
			adapters: []manager.Adapter{
				&manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"vim"}},
			},
			manifest: manifestWith(
				manifest.Package{Name: "development-tools", Manager: "dnf", Group: true},
				manifest.Package{Name: "obsidian", Manager: "brew", Cask: true},
				manifest.Package{Name: "vim", Manager: "dnf"},
			),
			wantNames: nil,
		},
		{
			name:      "nil manifest yields none",
			adapters:  []manager.Adapter{&manager.Mock{ManagerName: "brew"}},
			manifest:  nil,
			wantNames: nil,
		},
		{
			name:      "no adapters yields none",
			adapters:  nil,
			manifest:  manifestWith(manifest.Package{Name: "htop", Manager: "brew"}),
			wantNames: nil,
		},
		{
			name: "sorted and deduplicated across managers",
			adapters: []manager.Adapter{
				&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{"jq"}},
				&manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}},
			},
			manifest: manifestWith(
				manifest.Package{Name: "jq", Manager: "brew"},
				manifest.Package{Name: "ripgrep", Manager: "brew"},
				manifest.Package{Name: "ripgrep", Manager: "brew"},
				manifest.Package{Name: "vim", Manager: "dnf"},
			),
			wantNames: []string{"ripgrep", "vim"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missingFromSystem(context.Background(), tt.adapters, tt.manifest)
			require.Len(t, got, len(tt.wantNames), "unexpected missing count")
			for i, want := range tt.wantNames {
				assert.Equal(t, want, got[i].Name, "missing entry %d", i)
			}
		})
	}
}

// checkerMock wraps manager.Mock and answers presence checks through
// InstalledChecker instead of ListInstalled, mirroring the DNF adapter.
type checkerMock struct {
	manager.Mock
	absent map[string]bool
	err    error
	calls  int
}

func (c *checkerMock) CheckInstalled(_ context.Context, pkgs []string) ([]string, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	var out []string
	for _, p := range pkgs {
		if c.absent[p] {
			out = append(out, p)
		}
	}
	return out, nil
}

var _ manager.InstalledChecker = (*checkerMock)(nil)

func TestMissingFromSystem_InstalledChecker(t *testing.T) {
	manifestWith := func(pkgs ...manifest.Package) *manifest.Manifest {
		return &manifest.Manifest{Packages: pkgs}
	}

	t.Run("checker preferred over ListInstalled", func(t *testing.T) {
		// nodejs is absent from the raw listing (alias for nodejs22) but the
		// checker resolves it as present — no false Missing may be reported.
		checker := &checkerMock{
			Mock:   manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"nodejs22"}},
			absent: map[string]bool{"ghost": true},
		}
		got := missingFromSystem(context.Background(), []manager.Adapter{checker}, manifestWith(
			manifest.Package{Name: "nodejs", Manager: "dnf"},
			manifest.Package{Name: "ghost", Manager: "dnf"},
		))
		require.Len(t, got, 1)
		assert.Equal(t, "ghost", got[0].Name)
		assert.Equal(t, 1, checker.calls)
	})

	t.Run("all present via checker yields none", func(t *testing.T) {
		checker := &checkerMock{Mock: manager.Mock{ManagerName: "dnf"}}
		got := missingFromSystem(context.Background(), []manager.Adapter{checker}, manifestWith(
			manifest.Package{Name: "vim", Manager: "dnf"},
		))
		assert.Empty(t, got)
		assert.Equal(t, 1, checker.calls)
	})

	t.Run("checker error degrades to ListInstalled matching", func(t *testing.T) {
		checker := &checkerMock{
			Mock: manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}},
			err:  errors.New("repoquery boom"),
		}
		got := missingFromSystem(context.Background(), []manager.Adapter{checker}, manifestWith(
			manifest.Package{Name: "htop", Manager: "dnf"},
			manifest.Package{Name: "ghost", Manager: "dnf"},
		))
		// Legacy exact-match path: htop present, ghost absent.
		require.Len(t, got, 1)
		assert.Equal(t, "ghost", got[0].Name)
	})
}

func TestMissingFromDeltas(t *testing.T) {
	manifestWith := func(pkgs ...manifest.Package) *manifest.Manifest {
		return &manifest.Manifest{Packages: pkgs}
	}
	deltasWith := func(ds ...state.Delta) []state.Delta {
		return ds
	}

	tests := []struct {
		name      string
		deltas    []state.Delta
		manifest  *manifest.Manifest
		wantNames []string
	}{
		{
			name:   "removed package in manifest detected",
			deltas: deltasWith(state.Delta{Manager: "brew", Removed: []string{"htop"}}),
			manifest: manifestWith(
				manifest.Package{Name: "htop", Manager: "brew"},
				manifest.Package{Name: "lazygit", Manager: "brew"},
			),
			wantNames: []string{"htop"},
		},
		{
			name:   "removed package not in manifest yields none",
			deltas: deltasWith(state.Delta{Manager: "brew", Removed: []string{"htop"}}),
			manifest: manifestWith(
				manifest.Package{Name: "lazygit", Manager: "brew"},
			),
			wantNames: nil,
		},
		{
			name:   "group and cask excluded",
			deltas: deltasWith(state.Delta{Manager: "dnf", Removed: []string{"development-tools"}}),
			manifest: manifestWith(
				manifest.Package{Name: "development-tools", Manager: "dnf", Group: true},
			),
			wantNames: nil,
		},
		{
			name:      "no deltas yields none",
			deltas:    nil,
			manifest:  manifestWith(manifest.Package{Name: "htop", Manager: "brew"}),
			wantNames: nil,
		},
		{
			name:      "nil manifest yields none",
			deltas:    deltasWith(state.Delta{Manager: "brew", Removed: []string{"htop"}}),
			manifest:  nil,
			wantNames: nil,
		},
		{
			name: "sorted and deduplicated across managers",
			deltas: deltasWith(
				state.Delta{Manager: "brew", Removed: []string{"ripgrep"}},
				state.Delta{Manager: "dnf", Removed: []string{"vim", "htop"}},
			),
			manifest: manifestWith(
				manifest.Package{Name: "vim", Manager: "dnf"},
				manifest.Package{Name: "htop", Manager: "dnf"},
				manifest.Package{Name: "htop", Manager: "dnf"},
				manifest.Package{Name: "ripgrep", Manager: "brew"},
			),
			wantNames: []string{"ripgrep", "htop", "vim"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missingFromDeltas(tt.deltas, tt.manifest)
			require.Len(t, got, len(tt.wantNames), "unexpected missing count")
			for i, want := range tt.wantNames {
				assert.Equal(t, want, got[i].Name, "missing entry %d", i)
			}
		})
	}
}

func TestDedupeAndSortMissing(t *testing.T) {
	t.Run("sorts by manager then name", func(t *testing.T) {
		got := dedupeAndSortMissing([]manifest.Package{
			{Name: "zsh", Manager: "brew"},
			{Name: "vim", Manager: "dnf"},
			{Name: "htop", Manager: "dnf"},
		})
		require.Len(t, got, 3)
		assert.Equal(t, "zsh", got[0].Name)
		assert.Equal(t, "htop", got[1].Name)
		assert.Equal(t, "vim", got[2].Name)
	})

	t.Run("deduplicates manager name pairs", func(t *testing.T) {
		got := dedupeAndSortMissing([]manifest.Package{
			{Name: "vim", Manager: "dnf"},
			{Name: "vim", Manager: "dnf"},
			{Name: "vim", Manager: "brew"},
		})
		require.Len(t, got, 2)
		assert.Equal(t, "brew", got[0].Manager)
		assert.Equal(t, "dnf", got[1].Manager)
	})

	t.Run("empty input yields nil", func(t *testing.T) {
		assert.Nil(t, dedupeAndSortMissing(nil))
	})

	t.Run("does not mutate the caller slice", func(t *testing.T) {
		in := []manifest.Package{{Name: "zsh", Manager: "brew"}, {Name: "vim", Manager: "dnf"}}
		_ = dedupeAndSortMissing(in)
		assert.Equal(t, "zsh", in[0].Name)
		assert.Equal(t, "brew", in[0].Manager)
	})
}
