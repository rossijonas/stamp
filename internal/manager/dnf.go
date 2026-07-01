package manager

import (
	"context"
	"strings"
)

// DnfManager implements the PackageManager interface for Fedora's DNF.
type DnfManager struct {
	exec Executor
}

// NewDnfManager creates a new DnfManager with the default system executor.
func NewDnfManager() *DnfManager {
	return &DnfManager{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *DnfManager) Name() string {
	return "dnf"
}

// ListInstalled returns a list of packages currently installed.
func (m *DnfManager) ListInstalled(ctx context.Context) ([]string, error) {
	// Query user-installed packages, formatted to only show the name.
	out, err := m.exec(ctx, "dnf", "repoquery", "--userinstalled", "--qf", "%{name}")
	if err != nil {
		return nil, err
	}

	return parseLines(out), nil
}

// Install executes the native installation command.
func (m *DnfManager) Install(ctx context.Context, pkg string) error {
	_, err := m.exec(ctx, "sudo", "dnf", "install", "-y", pkg)
	return err
}

// Remove executes the native removal command.
func (m *DnfManager) Remove(ctx context.Context, pkg string) error {
	_, err := m.exec(ctx, "sudo", "dnf", "remove", "-y", pkg)
	return err
}

// Search queries the native package manager for the given package name.
func (m *DnfManager) Search(ctx context.Context, query string) ([]string, error) {
	out, err := m.exec(ctx, "dnf", "search", "-q", query)
	if err != nil {
		return nil, err
	}
	return parseLines(out), nil
}

// parseLines splits byte output by newline and removes empty strings.
func parseLines(output []byte) []string {
	var result []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
