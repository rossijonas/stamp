package manager

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Mock is a dummy implementation of Adapter for testing.
type Mock struct {
	ManagerName         string
	InstalledPkgs       []string
	InstalledRepos      []RepositoryInfo
	AvailablePkgs       []string
	TrackedRepos        []string
	ListErr             error
	ListReposErr        error
	InstallErr          error
	ReinstallErr        error
	RemoveErr           error
	SearchErr           error
	AddRepoErr          error
	RemoveRepoErr       error
	InfoErr             error
	InfoResult          string
	DoctorResult        string
	DoctorErr           error
	UpdateErr           error
	CheckUpdateErr      error
	RefreshErr          error
	CheckUpdates        []UpdateInfo
	ProvidesErr         error
	ProvidesResult      []string
	AutoRemoveErr       error
	AutoRemoveResult    []string
	CleanErr            error
	CleanResult         []string
	OverrideFunc        func(ctx context.Context, appID string, flags OverrideFlags) error
	HoldErr             error
	UnholdErr           error
	ListHeldErr         error
	HeldPkgs            []string
	PreviewInstallErr   error
	PreviewRemoveErr    error
	PreviewReinstallErr error
	PreviewResult       string
	PreviewNoop         bool
}

// Name returns the package manager identifier.
func (m *Mock) Name() string {
	return m.ManagerName
}

// ListInstalled returns a list of packages currently installed.
func (m *Mock) ListInstalled(_ context.Context) ([]string, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	// Return a copy to avoid accidental mutation
	return slices.Clone(m.InstalledPkgs), nil
}

// ListRepos returns a list of repositories currently configured.
func (m *Mock) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	if m.ListReposErr != nil {
		return nil, m.ListReposErr
	}
	if m.InstalledRepos != nil {
		return slices.Clone(m.InstalledRepos), nil
	}
	if m.TrackedRepos != nil {
		result := make([]RepositoryInfo, len(m.TrackedRepos))
		for i, r := range m.TrackedRepos {
			result[i] = RepositoryInfo{Name: r}
		}
		return result, nil
	}
	return nil, nil
}

// Install executes the native installation command.
func (m *Mock) Install(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if m.InstallErr != nil {
		return m.InstallErr
	}
	for _, p := range m.InstalledPkgs {
		if p == pkg {
			return nil
		}
	}
	m.InstalledPkgs = append(m.InstalledPkgs, pkg)
	return nil
}

// Reinstall executes the native reinstallation command.
func (m *Mock) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if m.ReinstallErr != nil {
		return m.ReinstallErr
	}
	// Remove then re-add to simulate reinstall
	for i, p := range m.InstalledPkgs {
		if p == pkg {
			m.InstalledPkgs = slices.Delete(m.InstalledPkgs, i, i+1)
			break
		}
	}
	m.InstalledPkgs = append(m.InstalledPkgs, pkg)
	return nil
}

// Remove executes the native removal command.
func (m *Mock) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if m.RemoveErr != nil {
		return m.RemoveErr
	}
	for i, p := range m.InstalledPkgs {
		if p == pkg {
			m.InstalledPkgs = slices.Delete(m.InstalledPkgs, i, i+1)
			return nil
		}
	}
	return nil
}

// Search queries the native package manager for the given package name.
func (m *Mock) Search(_ context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	if m.SearchErr != nil {
		return nil, m.SearchErr
	}

	var results []string
	for _, p := range m.AvailablePkgs {
		if strings.Contains(p, query) {
			results = append(results, p)
		}
	}
	return results, nil
}

// AddRepo adds a repository to the mock.
func (m *Mock) AddRepo(ctx context.Context, name, _ string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if m.AddRepoErr != nil {
		return m.AddRepoErr
	}
	m.TrackedRepos = append(m.TrackedRepos, name)
	return nil
}

// RemoveRepo removes a repository from the mock.
func (m *Mock) RemoveRepo(ctx context.Context, name string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if m.RemoveRepoErr != nil {
		return m.RemoveRepoErr
	}
	for i, r := range m.TrackedRepos {
		if r == name {
			m.TrackedRepos = slices.Delete(m.TrackedRepos, i, i+1)
			return nil
		}
	}
	return nil
}

// Info queries mock info metadata.
func (m *Mock) Info(_ context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	if m.InfoErr != nil {
		return "", m.InfoErr
	}
	if m.InfoResult != "" {
		return m.InfoResult, nil
	}
	// Fallback mock output
	return fmt.Sprintf("Name: %s\nVersion: 1.0.0\nDescription: mock details", pkg), nil
}

// PreviewInstall returns mock dry-run output for installing a package.
func (m *Mock) PreviewInstall(_ context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	if m.PreviewInstallErr != nil {
		return Preview{}, m.PreviewInstallErr
	}
	if m.PreviewNoop {
		return Preview{Output: "Nothing to do.", Noop: true}, nil
	}
	if m.PreviewResult != "" {
		return Preview{Output: m.PreviewResult}, nil
	}
	return Preview{Output: fmt.Sprintf("Install: %s@1.0.0", pkg)}, nil
}

// PreviewRemove returns mock dry-run output for removing a package.
func (m *Mock) PreviewRemove(_ context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	if m.PreviewRemoveErr != nil {
		return Preview{}, m.PreviewRemoveErr
	}
	if m.PreviewNoop {
		return Preview{Output: "Nothing to remove.", Noop: true}, nil
	}
	if m.PreviewResult != "" {
		return Preview{Output: m.PreviewResult}, nil
	}
	return Preview{Output: fmt.Sprintf("Remove: %s", pkg)}, nil
}

// PreviewReinstall returns mock dry-run output for reinstalling a package.
func (m *Mock) PreviewReinstall(_ context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	if m.PreviewReinstallErr != nil {
		return Preview{}, m.PreviewReinstallErr
	}
	if m.PreviewNoop {
		return Preview{Output: "Nothing to do.", Noop: true}, nil
	}
	if m.PreviewResult != "" {
		return Preview{Output: m.PreviewResult}, nil
	}
	return Preview{Output: fmt.Sprintf("Reinstall: %s@1.0.0", pkg)}, nil
}

// Doctor runs mock doctor diagnostic.
func (m *Mock) Doctor(_ context.Context) (string, error) {
	if m.DoctorErr != nil {
		return "", m.DoctorErr
	}
	if m.DoctorResult != "" {
		return m.DoctorResult, nil
	}
	return "mock doctor: all good", nil
}

// Update runs mock update — succeeds unless UpdateErr is set.
func (m *Mock) Update(ctx context.Context, _ string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	return nil
}

// CheckUpdate returns mock update info or error.
func (m *Mock) CheckUpdate(_ context.Context, pkg string) ([]UpdateInfo, error) {
	if m.CheckUpdateErr != nil {
		return nil, m.CheckUpdateErr
	}
	if pkg != "" {
		for _, u := range m.CheckUpdates {
			if u.Package == pkg {
				return []UpdateInfo{u}, nil
			}
		}
		return nil, nil
	}
	if m.CheckUpdates != nil {
		return slices.Clone(m.CheckUpdates), nil
	}
	return nil, nil
}

// Refresh returns mock refresh error if set.
func (m *Mock) Refresh(_ context.Context) error {
	if m.RefreshErr != nil {
		return m.RefreshErr
	}
	return nil
}

// Provides returns mock provides result or error.
func (m *Mock) Provides(_ context.Context, query string) ([]string, error) {
	// No package name validation — provides queries are file paths, not package names
	if m.ProvidesErr != nil {
		return nil, m.ProvidesErr
	}
	if m.ProvidesResult != nil {
		return slices.Clone(m.ProvidesResult), nil
	}
	if m.AvailablePkgs != nil {
		var results []string
		for _, p := range m.AvailablePkgs {
			if strings.Contains(p, query) {
				results = append(results, p)
			}
		}
		return results, nil
	}
	return nil, nil
}

// AutoRemove returns mock autoremove result or error.
func (m *Mock) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if m.AutoRemoveErr != nil {
		return nil, m.AutoRemoveErr
	}
	if m.AutoRemoveResult != nil {
		return slices.Clone(m.AutoRemoveResult), nil
	}
	return nil, nil
}

// Override delegates to OverrideFunc if set, otherwise returns ErrNotSupported.
func (m *Mock) Override(ctx context.Context, appID string, flags OverrideFlags) error {
	if m.OverrideFunc != nil {
		return m.OverrideFunc(ctx, appID, flags)
	}
	return fmt.Errorf("%w: override not supported", ErrNotSupported)
}

// Clean returns mock clean result or error.
func (m *Mock) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if m.CleanErr != nil {
		return nil, m.CleanErr
	}
	if m.CleanResult != nil {
		return slices.Clone(m.CleanResult), nil
	}
	return nil, nil
}

// Hold mocks pinning a package at its current version.
func (m *Mock) Hold(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if m.HoldErr != nil {
		return m.HoldErr
	}
	if !slices.Contains(m.HeldPkgs, pkg) {
		m.HeldPkgs = append(m.HeldPkgs, pkg)
	}
	return nil
}

// Unhold mocks removing a version pin.
func (m *Mock) Unhold(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if m.UnholdErr != nil {
		return m.UnholdErr
	}
	for i, p := range m.HeldPkgs {
		if p == pkg {
			m.HeldPkgs = slices.Delete(m.HeldPkgs, i, i+1)
			return nil
		}
	}
	return nil
}

// ListHeld returns mock held packages or error.
func (m *Mock) ListHeld(_ context.Context) ([]string, error) {
	if m.ListHeldErr != nil {
		return nil, m.ListHeldErr
	}
	if m.HeldPkgs != nil {
		return slices.Clone(m.HeldPkgs), nil
	}
	return nil, nil
}
