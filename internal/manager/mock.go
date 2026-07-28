package manager

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Mock is a dummy implementation of Adapter for testing.
type Mock struct {
	ManagerName      string
	InstalledPkgs    []string
	InstalledRepos   []RepositoryInfo
	AvailablePkgs    []string
	TrackedRepos     []string
	ListErr          error
	ListReposErr     error
	InstallErr       error
	ReinstallErr     error
	RemoveErr        error
	SearchErr        error
	AddRepoErr       error
	RemoveRepoErr    error
	InfoErr          error
	InfoResult       string
	DoctorResult     string
	DoctorErr        error
	UpdateErr        error
	CheckUpdateErr   error
	CheckUpdates     []UpdateInfo
	ProvidesErr      error
	ProvidesResult   []string
	AutoRemoveErr    error
	AutoRemoveResult []string
	CleanErr         error
	CleanResult      []string
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
func (m *Mock) Install(_ context.Context, pkg string) error {
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
func (m *Mock) Reinstall(_ context.Context, pkg string) error {
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
func (m *Mock) Remove(_ context.Context, pkg string) error {
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
func (m *Mock) AddRepo(_ context.Context, name, _ string) error {
	if m.AddRepoErr != nil {
		return m.AddRepoErr
	}
	m.TrackedRepos = append(m.TrackedRepos, name)
	return nil
}

// RemoveRepo removes a repository from the mock.
func (m *Mock) RemoveRepo(_ context.Context, name string) error {
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
func (m *Mock) Update(_ context.Context, _ string) error {
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
func (m *Mock) AutoRemove(_ context.Context, _ bool) ([]string, error) {
	if m.AutoRemoveErr != nil {
		return nil, m.AutoRemoveErr
	}
	if m.AutoRemoveResult != nil {
		return slices.Clone(m.AutoRemoveResult), nil
	}
	return nil, nil
}

// Clean returns mock clean result or error.
func (m *Mock) Clean(_ context.Context, _ bool) ([]string, error) {
	if m.CleanErr != nil {
		return nil, m.CleanErr
	}
	if m.CleanResult != nil {
		return slices.Clone(m.CleanResult), nil
	}
	return nil, nil
}
