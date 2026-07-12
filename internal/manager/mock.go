package manager

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Mock is a dummy implementation of Adapter for testing.
type Mock struct {
	ManagerName   string
	InstalledPkgs []string
	AvailablePkgs []string
	TrackedRepos  []string
	ListErr       error
	InstallErr    error
	RemoveErr     error
	SearchErr     error
	AddRepoErr    error
	RemoveRepoErr error
	InfoErr       error
	InfoResult    string
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

// Install executes the native installation command.
func (m *Mock) Install(_ context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
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
