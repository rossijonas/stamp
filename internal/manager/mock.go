package manager

import (
	"context"
	"strings"
)

// MockManager is a dummy implementation of PackageManager for testing.
type MockManager struct {
	ManagerName   string
	InstalledPkgs []string
	AvailablePkgs []string
	ListErr       error
	InstallErr    error
	RemoveErr     error
	SearchErr     error
}

// Name returns the package manager identifier.
func (m *MockManager) Name() string {
	return m.ManagerName
}

// ListInstalled returns a list of packages currently installed.
func (m *MockManager) ListInstalled(_ context.Context) ([]string, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	// Return a copy to avoid accidental mutation
	pkgs := make([]string, len(m.InstalledPkgs))
	copy(pkgs, m.InstalledPkgs)
	return pkgs, nil
}

// Install executes the native installation command.
func (m *MockManager) Install(_ context.Context, pkg string) error {
	if m.InstallErr != nil {
		return m.InstallErr
	}
	// Don't add if it's already there
	for _, p := range m.InstalledPkgs {
		if p == pkg {
			return nil
		}
	}
	m.InstalledPkgs = append(m.InstalledPkgs, pkg)
	return nil
}

// Remove executes the native removal command.
func (m *MockManager) Remove(_ context.Context, pkg string) error {
	if m.RemoveErr != nil {
		return m.RemoveErr
	}
	for i, p := range m.InstalledPkgs {
		if p == pkg {
			m.InstalledPkgs = append(m.InstalledPkgs[:i], m.InstalledPkgs[i+1:]...)
			return nil
		}
	}
	return nil
}

// Search queries the native package manager for the given package name.
func (m *MockManager) Search(_ context.Context, query string) ([]string, error) {
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
