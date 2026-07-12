// Package manager implements the adapters for the various package managers
// supported by stamp (e.g., dnf, brew, flatpak).
package manager

import (
	"context"
	"fmt"
)

// Brew implements the Adapter interface for Homebrew.
type Brew struct {
	exec Executor
}

// NewBrew creates a new Brew with the default system executor.
func NewBrew() *Brew {
	return &Brew{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Brew) Name() string {
	return "brew"
}

// ListInstalled returns a list of packages currently installed.
func (m *Brew) ListInstalled(ctx context.Context) ([]string, error) {
	// 'brew leaves' returns packages that are not dependencies of another package.
	out, err := m.exec(ctx, "brew", "leaves")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseLines(out), nil
}

// Install executes the native installation command.
func (m *Brew) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "brew", "install", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *Brew) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "brew", "uninstall", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search queries the native package manager for the given package name.
func (m *Brew) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	// 'brew search' can be slow, but is the standard way.
	out, err := m.exec(ctx, "brew", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AddRepo enables a third-party tap.
func (m *Brew) AddRepo(ctx context.Context, name, url string) error {
	var err error
	if url != "" {
		_, err = m.exec(WithStreamIO(ctx), "brew", "tap", name, url)
	} else {
		_, err = m.exec(WithStreamIO(ctx), "brew", "tap", name)
	}
	if err != nil {
		return fmt.Errorf("failed to tap %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party tap.
func (m *Brew) RemoveRepo(ctx context.Context, name string) error {
	_, err := m.exec(WithStreamIO(ctx), "brew", "untap", name)
	if err != nil {
		return fmt.Errorf("failed to untap %s: %w", name, err)
	}
	return nil
}

// Info queries brew info metadata.
func (m *Brew) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "brew", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}
