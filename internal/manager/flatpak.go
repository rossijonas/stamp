package manager

import (
	"context"
)

// FlatpakManager implements the PackageManager interface for Flatpak.
type FlatpakManager struct {
	exec Executor
}

// NewFlatpakManager creates a new FlatpakManager with the default system executor.
func NewFlatpakManager() *FlatpakManager {
	return &FlatpakManager{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *FlatpakManager) Name() string {
	return "flatpak"
}

// ListInstalled returns a list of packages currently installed.
func (m *FlatpakManager) ListInstalled(ctx context.Context) ([]string, error) {
	// Restrict to apps (excluding runtimes) and print only the application ID.
	out, err := m.exec(ctx, "flatpak", "list", "--app", "--columns=application")
	if err != nil {
		return nil, err
	}
	return parseLines(out), nil
}

// Install executes the native installation command.
func (m *FlatpakManager) Install(ctx context.Context, pkg string) error {
	// -y auto-answers yes to prompts.
	_, err := m.exec(ctx, "flatpak", "install", "-y", pkg)
	return err
}

// Remove executes the native removal command.
func (m *FlatpakManager) Remove(ctx context.Context, pkg string) error {
	_, err := m.exec(ctx, "flatpak", "uninstall", "-y", pkg)
	return err
}

// Search queries the native package manager for the given package name.
func (m *FlatpakManager) Search(ctx context.Context, query string) ([]string, error) {
	// Search and return application IDs
	out, err := m.exec(ctx, "flatpak", "search", "--columns=application", query)
	if err != nil {
		return nil, err
	}
	return parseLines(out), nil
}
