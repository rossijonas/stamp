package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

func TestCountCasks(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]bool
		want  int
	}{
		{"nil map", nil, 0},
		{"empty", map[string]bool{}, 0},
		{"none", map[string]bool{"a": false, "b": false}, 0},
		{"some", map[string]bool{"a": true, "b": false, "c": true}, 2},
		{"all", map[string]bool{"a": true}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countCasks(tt.input))
		})
	}
}

func TestValidateGroupSupport(t *testing.T) {
	tests := []struct {
		name    string
		adapter manager.Adapter
		group   bool
		wantErr bool
	}{
		{"no group passes", &manager.Mock{ManagerName: "brew"}, false, false},
		{"dnf group ok", &manager.Mock{ManagerName: "dnf"}, true, false},
		{"yum group ok", &manager.Mock{ManagerName: "yum"}, true, false},
		{"brew group rejected", &manager.Mock{ManagerName: "brew"}, true, true},
		{"apt group rejected", &manager.Mock{ManagerName: "apt"}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGroupSupport(tt.adapter, tt.group)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, "--group is only supported for dnf", err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDetectBrewCask(t *testing.T) {
	// fakeCask implements caskDetector (IsCask) so detectBrewCask can hit the
	// true branch without a real *manager.Brew.
	fakeCask := func(cask bool) *fakeCaskDetector {
		return &fakeCaskDetector{Mock: &manager.Mock{ManagerName: "brew"}, cask: cask}
	}
	plainMock := &manager.Mock{ManagerName: "brew"}

	assert.True(t, detectBrewCask(context.Background(), fakeCask(true), "cask-app"),
		"a cask-aware adapter reporting true must resolve as cask")
	assert.False(t, detectBrewCask(context.Background(), fakeCask(false), "formula"),
		"a cask-aware adapter reporting false is not a cask")
	assert.False(t, detectBrewCask(context.Background(), plainMock, "x"),
		"an adapter without IsCask is not a cask detector")
}

type fakeCaskDetector struct {
	*manager.Mock
	cask bool
}

func (f *fakeCaskDetector) IsCask(_ context.Context, _ string) (bool, error) {
	return f.cask, nil
}

func TestFindTrackedAdapter(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew"},
		&manager.Mock{ManagerName: "dnf"},
	}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew", Cask: true},
	}

	a, cask, found := findTrackedAdapter(pkgs, adapters, "htop")
	require.True(t, found)
	assert.Equal(t, "brew", a.Name())
	assert.True(t, cask)

	_, _, found = findTrackedAdapter(pkgs, adapters, "missing")
	assert.False(t, found)

	// Tracked but its manager adapter is absent: not found (fallback applies).
	other := []manager.Adapter{&manager.Mock{ManagerName: "dnf"}}
	_, _, found = findTrackedAdapter(pkgs, other, "htop")
	assert.False(t, found)
}

func TestResolveRemoveTarget(t *testing.T) {
	brew := &manager.Mock{ManagerName: "brew"}
	dnf := &manager.Mock{ManagerName: "dnf"}

	tests := []struct {
		name        string
		app         *AppContext
		pkg         string
		managerFlag string
		wantMgr     string
		wantCask    bool
		wantErr     string
	}{
		{
			name:    "tracked manager wins",
			app:     &AppContext{adapters: []manager.Adapter{brew, dnf}, manifest: &manifest.Manifest{Packages: []manifest.Package{{Name: "htop", Manager: "dnf"}}}},
			pkg:     "htop",
			wantMgr: "dnf",
		},
		{
			name:    "tracked missing manager falls back to first adapter",
			app:     &AppContext{adapters: []manager.Adapter{brew}, manifest: &manifest.Manifest{Packages: []manifest.Package{{Name: "htop", Manager: "flatpak"}}}},
			pkg:     "htop",
			wantMgr: "brew",
		},
		{
			name:        "flag path resolves manager",
			app:         &AppContext{adapters: []manager.Adapter{brew, dnf}, manifest: &manifest.Manifest{}},
			pkg:         "x",
			managerFlag: "dnf",
			wantMgr:     "dnf",
		},
		{
			name:        "flag path unknown manager errors",
			app:         &AppContext{adapters: []manager.Adapter{brew}, manifest: &manifest.Manifest{}},
			pkg:         "x",
			managerFlag: "nope",
			wantErr:     `unknown manager "nope"`,
		},
		{
			name:    "no adapters errors unavailable",
			app:     &AppContext{adapters: nil, manifest: &manifest.Manifest{}},
			pkg:     "x",
			wantErr: "no package managers available",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, cask, err := resolveRemoveTarget(tt.app, tt.pkg, tt.managerFlag)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMgr, a.Name())
			assert.Equal(t, tt.wantCask, cask)
		})
	}
}

func TestValidateBatchPackages(t *testing.T) {
	adapter := &manager.Mock{ManagerName: "brew"}
	require.NoError(t, validateBatchPackages(adapter, []string{"htop", "git"}))

	err := validateBatchPackages(adapter, []string{"htop", "invalid name"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestSelectSearchTargets(t *testing.T) {
	brew := &manager.Mock{ManagerName: "brew"}
	dnf := &manager.Mock{ManagerName: "dnf"}
	all := []manager.Adapter{brew, dnf}

	got, err := selectSearchTargets(all, "")
	require.NoError(t, err)
	assert.Len(t, got, 2)

	got, err = selectSearchTargets(all, "dnf")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "dnf", got[0].Name())

	_, err = selectSearchTargets(all, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown manager "nope"`)
}

func TestValidateGroupSearch(t *testing.T) {
	brew := &manager.Mock{ManagerName: "brew"}
	dnf := &manager.Mock{ManagerName: "dnf"}

	require.Error(t, validateGroupSearch([]manager.Adapter{brew}, "", true),
		"--group without --manager")
	require.Error(t, validateGroupSearch([]manager.Adapter{brew}, "brew", true),
		"--group on non-dnf")
	assert.NoError(t, validateGroupSearch([]manager.Adapter{dnf}, "dnf", true))
	assert.NoError(t, validateGroupSearch([]manager.Adapter{brew}, "brew", false))
}

func TestSearchManagers(t *testing.T) {
	brew := &manager.Mock{ManagerName: "brew", AvailablePkgs: []string{"htop", "htcons"}}
	dnf := &manager.Mock{ManagerName: "dnf", AvailablePkgs: []string{"htop"}}

	var warn bytes.Buffer
	results := searchManagers(context.Background(), []manager.Adapter{brew, dnf}, "hto", &warn, false)
	assert.Empty(t, warn.String())
	assert.ElementsMatch(t, []string{"htop (brew)", "htop (dnf)"}, results)

	failing := &manager.Mock{ManagerName: "apt", SearchErr: assertErr("boom")}
	results = searchManagers(context.Background(), []manager.Adapter{failing}, "x", &warn, false)
	assert.Empty(t, results)
	assert.Contains(t, warn.String(), "warning: apt search")
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
