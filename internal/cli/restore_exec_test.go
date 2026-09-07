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

// caskRestoreMock wraps manager.Mock and implements the caskDetector interface
// (defined in install.go) for restore cask detection tests.
type caskRestoreMock struct {
	*manager.Mock
	casks map[string]bool
}

func (c *caskRestoreMock) IsCask(_ context.Context, pkg string) (bool, error) {
	return c.casks[pkg], nil
}

// noBatchAdapter implements manager.Adapter but NOT manager.BatchInstaller.
// Used to test the sequential fallback path in restorePackageGroup.
type noBatchAdapter struct {
	mock *manager.Mock
}

func (a *noBatchAdapter) Name() string { return a.mock.Name() }
func (a *noBatchAdapter) ListInstalled(ctx context.Context) ([]string, error) {
	return a.mock.ListInstalled(ctx)
}
func (a *noBatchAdapter) ListRepos(ctx context.Context) ([]manager.RepositoryInfo, error) {
	return a.mock.ListRepos(ctx)
}
func (a *noBatchAdapter) Install(ctx context.Context, pkg string) error {
	return a.mock.Install(ctx, pkg)
}
func (a *noBatchAdapter) Reinstall(ctx context.Context, pkg string) error {
	return a.mock.Reinstall(ctx, pkg)
}
func (a *noBatchAdapter) Remove(ctx context.Context, pkg string) error {
	return a.mock.Remove(ctx, pkg)
}
func (a *noBatchAdapter) Search(ctx context.Context, q string) ([]string, error) {
	return a.mock.Search(ctx, q)
}
func (a *noBatchAdapter) AddRepo(ctx context.Context, name, url string) error {
	return a.mock.AddRepo(ctx, name, url)
}
func (a *noBatchAdapter) RemoveRepo(ctx context.Context, name string) error {
	return a.mock.RemoveRepo(ctx, name)
}
func (a *noBatchAdapter) Info(ctx context.Context, pkg string) (string, error) {
	return a.mock.Info(ctx, pkg)
}
func (a *noBatchAdapter) Doctor(ctx context.Context) (string, error) { return a.mock.Doctor(ctx) }
func (a *noBatchAdapter) Update(ctx context.Context, pkg string) error {
	return a.mock.Update(ctx, pkg)
}
func (a *noBatchAdapter) CheckUpdate(ctx context.Context, pkg string) ([]manager.UpdateInfo, error) {
	return a.mock.CheckUpdate(ctx, pkg)
}
func (a *noBatchAdapter) Refresh(ctx context.Context) error { return a.mock.Refresh(ctx) }
func (a *noBatchAdapter) Provides(ctx context.Context, q string) ([]string, error) {
	return a.mock.Provides(ctx, q)
}
func (a *noBatchAdapter) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	return a.mock.AutoRemove(ctx, dryRun)
}
func (a *noBatchAdapter) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	return a.mock.Clean(ctx, dryRun)
}
func (a *noBatchAdapter) Hold(ctx context.Context, pkg string) error { return a.mock.Hold(ctx, pkg) }
func (a *noBatchAdapter) Unhold(ctx context.Context, pkg string) error {
	return a.mock.Unhold(ctx, pkg)
}
func (a *noBatchAdapter) ListHeld(ctx context.Context) ([]string, error) {
	return a.mock.ListHeld(ctx)
}

func TestRestoreBatch_Success(t *testing.T) {
	t.Parallel()
	mock := &manager.Mock{ManagerName: "brew"}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew"},
		{Name: "btop", Manager: "brew"},
		{Name: "tmux", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	assert.Equal(t, 1, mock.InstallManyCalls, "one batch call per manager")
	assert.Contains(t, mock.InstalledPkgs, "htop")
	assert.Contains(t, mock.InstalledPkgs, "btop")
	assert.Contains(t, mock.InstalledPkgs, "tmux")
	assert.Contains(t, buf.String(), "restored 3 package(s) via brew")
}

func TestRestoreBatch_BatchFailRetryAllSucceed(t *testing.T) {
	t.Parallel()
	mock := &manager.Mock{
		ManagerName:    "brew",
		InstallManyErr: assert.AnError,
	}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew"},
		{Name: "btop", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs, "all retries succeed → no errors")
	assert.Equal(t, 1, mock.InstallManyCalls)
	assert.Equal(t, []string{"htop", "btop"}, mock.InstallCalls, "each retried individually")
	assert.Equal(t, []string{"htop", "btop"}, mock.InstalledPkgs)
}

func TestRestoreBatch_BatchFailOneRetryFails(t *testing.T) {
	t.Parallel()
	mock := &manager.Mock{
		ManagerName:    "brew",
		InstallManyErr: assert.AnError,
		InstallFunc: func(_ context.Context, pkg string) error {
			if pkg == "btop" {
				return assert.AnError
			}
			return nil
		},
	}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew"},
		{Name: "btop", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Len(t, errs, 1, "only the failing retry is reported")
	require.Equal(t, "btop", errs[0].Pkg)
	require.ErrorIs(t, errs[0].Err, assert.AnError)
	assert.Contains(t, mock.InstalledPkgs, "htop")
	assert.NotContains(t, mock.InstalledPkgs, "btop")
}

func TestRestoreBatch_BatchFailAllRetriesFail(t *testing.T) {
	t.Parallel()
	mock := &manager.Mock{
		ManagerName:    "brew",
		InstallManyErr: assert.AnError,
		InstallErr:     assert.AnError,
	}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew"},
		{Name: "btop", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Len(t, errs, 2)
	assert.Equal(t, "htop", errs[0].Pkg)
	assert.Equal(t, "btop", errs[1].Pkg)
	assert.ErrorIs(t, errs[0].Err, assert.AnError)
}

func TestRestoreBatch_GroupExcluded(t *testing.T) {
	t.Parallel()
	mock := &manager.Mock{ManagerName: "dnf"}
	pkgs := []manifest.Package{
		{Name: "development-tools", Manager: "dnf", Group: true},
		{Name: "htop", Manager: "dnf"},
		{Name: "tmux", Manager: "dnf"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	assert.Equal(t, 1, mock.InstallManyCalls, "non-group packages batched")
	assert.False(t, mock.LastBatchGroup, "batch context has group=false (correct)")
	// Group package installed individually via Install
	groupIdx := -1
	for i, c := range mock.InstallCalls {
		if c == "development-tools" {
			groupIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, groupIdx, 0, "group package installed via Install")
	assert.True(t, mock.InstallGroupCalls[groupIdx], "group package installed with WithGroup")
	// Non-group packages in batch
	assert.Contains(t, mock.InstalledPkgs, "htop")
	assert.Contains(t, mock.InstalledPkgs, "tmux")
}

func TestRestoreBatch_AllCaskBatch(t *testing.T) {
	t.Parallel()
	mock := &caskRestoreMock{
		Mock:  &manager.Mock{ManagerName: "brew"},
		casks: map[string]bool{"cask-app": true, "cask-util": true},
	}
	pkgs := []manifest.Package{
		{Name: "cask-app", Manager: "brew"},
		{Name: "cask-util", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	assert.Equal(t, 1, mock.InstallManyCalls, "all-cask batch")
	assert.True(t, mock.LastBatchCask, "cask flag set on batch context")
	assert.Contains(t, buf.String(), "restored 2 package(s) via brew")
}

func TestRestoreBatch_MixedCaskFormulaFallback(t *testing.T) {
	t.Parallel()
	mock := &caskRestoreMock{
		Mock:  &manager.Mock{ManagerName: "brew"},
		casks: map[string]bool{"cask-app": true},
	}
	pkgs := []manifest.Package{
		{Name: "cask-app", Manager: "brew"},
		{Name: "formula", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	// Mixed → per-package fallback, no InstallMany
	assert.Equal(t, 0, mock.InstallManyCalls, "mixed must not batch")
	assert.Equal(t, []string{"cask-app", "formula"}, mock.InstallCalls, "each installed individually")
	assert.Equal(t, []bool{true, false}, mock.InstallCaskCalls, "cask flag applied per package")
	assert.Contains(t, mock.InstalledPkgs, "cask-app")
	assert.Contains(t, mock.InstalledPkgs, "formula")
}

func TestRestoreBatch_NonBatchInstallerFallback(t *testing.T) {
	t.Parallel()
	nba := &noBatchAdapter{mock: &manager.Mock{ManagerName: "pipx"}}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "pipx"},
		{Name: "tmux", Manager: "pipx"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{nba}, pkgs)
	require.Empty(t, errs)
	// NoBatchInstaller → sequential Install
	assert.Len(t, nba.mock.InstallCalls, 2, "each package installed individually")
	assert.Contains(t, nba.mock.InstalledPkgs, "htop")
	assert.Contains(t, nba.mock.InstalledPkgs, "tmux")
}

func TestRestoreBatch_BatchSuccessNoEcho(t *testing.T) {
	t.Parallel()
	mock := &manager.Mock{ManagerName: "dnf"}
	pkgs := []manifest.Package{
		{Name: "vim", Manager: "dnf"},
		{Name: "git", Manager: "dnf"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	output := buf.String()
	assert.Contains(t, output, "restored 2 package(s) via dnf")
	// Batch success: no per-package echo
	assert.NotContains(t, output, "installed vim via dnf")
	assert.NotContains(t, output, "installed git via dnf")
}

func TestRestoreBatch_RetryKeepsEcho(t *testing.T) {
	t.Parallel()
	mock := &manager.Mock{
		ManagerName:    "brew",
		InstallManyErr: assert.AnError,
	}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	// Retry path echoes on success
	assert.Contains(t, buf.String(), "installed htop via brew")
}

func TestRestoreBatch_RetryAppliesCask(t *testing.T) {
	t.Parallel()
	mock := &caskRestoreMock{
		Mock: &manager.Mock{
			ManagerName:    "brew",
			InstallManyErr: assert.AnError,
		},
		casks: map[string]bool{"firefox": true},
	}
	pkgs := []manifest.Package{
		{Name: "firefox", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	// Batch failed → retry applies live cask detection (WithCask on Install)
	assert.Equal(t, []bool{true}, mock.InstallCaskCalls, "retry stacks WithCask for cask")
	assert.Contains(t, mock.InstalledPkgs, "firefox")
}

func TestRestoreBatch_MultiManager(t *testing.T) {
	t.Parallel()
	mockBrew := &manager.Mock{ManagerName: "brew"}
	mockDNF := &manager.Mock{ManagerName: "dnf"}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew"},
		{Name: "vim", Manager: "dnf"},
		{Name: "git", Manager: "dnf"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mockBrew, mockDNF}, pkgs)
	require.Empty(t, errs)
	assert.Equal(t, 1, mockBrew.InstallManyCalls, "brew gets 1 batch")
	assert.Equal(t, 1, mockDNF.InstallManyCalls, "dnf gets 1 batch")
	assert.Contains(t, buf.String(), "restored 1 package(s) via brew")
	assert.Contains(t, buf.String(), "restored 2 package(s) via dnf")
}

func TestRestoreBatch_BatchSuccessCaskEchoRemoved(t *testing.T) {
	t.Parallel()
	mock := &caskRestoreMock{
		Mock:  &manager.Mock{ManagerName: "brew"},
		casks: map[string]bool{"firefox": true},
	}
	pkgs := []manifest.Package{
		{Name: "firefox", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mock}, pkgs)
	require.Empty(t, errs)
	assert.Contains(t, buf.String(), "restored 1 package(s) via brew")
	assert.NotContains(t, buf.String(), "installed firefox via brew")
}

func TestRestoreBatch_SinglePackageUsesBatch(t *testing.T) {
	t.Parallel()
	mockBrew := &manager.Mock{ManagerName: "brew"}
	pkgs := []manifest.Package{
		{Name: "htop", Manager: "brew"},
	}
	var buf bytes.Buffer
	errs := restorePackages(context.Background(), &buf, []manager.Adapter{mockBrew}, pkgs)
	require.Empty(t, errs)
	assert.Equal(t, 1, mockBrew.InstallManyCalls, "single package still batched")
	assert.Contains(t, buf.String(), "restored 1 package(s) via brew")
}
