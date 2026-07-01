// Package manager implements the adapters for the various package managers
// supported by stamp (e.g., dnf, brew, flatpak).
package manager

import (
	"context"
)

// BrewManager implements the PackageManager interface for Homebrew.
type BrewManager struct {
	exec Executor
}

// NewBrewManager creates a new BrewManager with the default system executor.
func NewBrewManager() *BrewManager {
	return &BrewManager{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *BrewManager) Name() string {
	return "brew"
}

// ListInstalled returns a list of packages currently installed.
func (m *BrewManager) ListInstalled(ctx context.Context) ([]string, error) {
	// 'brew leaves' returns packages that are not dependencies of another package.
	out, err := m.exec(ctx, "brew", "leaves")
	if err != nil {
		return nil, err
	}
	return parseLines(out), nil
}

// Install executes the native installation command.
func (m *BrewManager) Install(ctx context.Context, pkg string) error {
	_, err := m.exec(ctx, "brew", "install", pkg)
	return err
}

// Remove executes the native removal command.
func (m *BrewManager) Remove(ctx context.Context, pkg string) error {
	_, err := m.exec(ctx, "brew", "uninstall", pkg)
	return err
}

// Search queries the native package manager for the given package name.
func (m *BrewManager) Search(ctx context.Context, query string) ([]string, error) {
	// 'brew search' can be slow, but is the standard way.
	out, err := m.exec(ctx, "brew", "search", query)
	if err != nil {
		return nil, err
	}
	return parseLines(out), nil
}
