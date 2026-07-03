package manager

import (
	"context"
	"fmt"
)

// Flatpak implements the Adapter interface for Flatpak.
type Flatpak struct {
	exec Executor
}

// NewFlatpak creates a new Flatpak with the default system executor.
func NewFlatpak() *Flatpak {
	return &Flatpak{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Flatpak) Name() string {
	return "flatpak"
}

// ListInstalled returns a list of packages currently installed.
func (m *Flatpak) ListInstalled(ctx context.Context) ([]string, error) {
	// Restrict to apps (excluding runtimes) and print only the application ID.
	out, err := m.exec(ctx, "flatpak", "list", "--app", "--columns=application")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseLines(out), nil
}

// Install executes the native installation command.
func (m *Flatpak) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	// -y auto-answers yes to prompts.
	_, err := m.exec(ctx, "flatpak", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *Flatpak) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(ctx, "flatpak", "uninstall", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search queries the native package manager for the given package name.
func (m *Flatpak) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	// Search and return application IDs
	out, err := m.exec(ctx, "flatpak", "search", "--columns=application", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}
