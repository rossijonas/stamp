// Package manager provides adapters for various package managers.
package manager

import (
	"bytes"
	"context"
	"fmt"
)

// Snap implements the Adapter interface for the snap package manager.
type Snap struct {
	exec Executor
}

// NewSnap creates a new Snap adapter.
func NewSnap() *Snap {
	return &Snap{
		exec: defaultExecutor,
	}
}

// Name returns "snap".
func (m *Snap) Name() string {
	return "snap"
}

// parseSnapTabular extracts the first column from snap tabular output.
func parseSnapTabular(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if bytes.HasPrefix(trimmed, []byte("Name")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		name := string(fields[0])
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	return result
}

// ListInstalled returns a list of installed snap packages.
func (m *Snap) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "snap", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed snaps: %w", err)
	}
	return parseSnapTabular(out), nil
}

// Install installs a snap package.
func (m *Snap) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("snap", "install", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall reinstalls a snap package via remove + install.
func (m *Snap) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	removeArgs := sudoCmd("snap", "remove", pkg)
	_, err := m.exec(WithStreamIO(ctx), removeArgs[0], removeArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: remove step failed: %w", pkg, err)
	}
	installArgs := sudoCmd("snap", "install", pkg)
	_, err = m.exec(WithStreamIO(ctx), installArgs[0], installArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: install step failed: %w", pkg, err)
	}
	return nil
}

// Remove removes a snap package.
func (m *Snap) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("snap", "remove", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search searches for snap packages matching the query.
func (m *Snap) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "snap", "find", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseSnapTabular(out), nil
}

// Info returns details about a snap package.
func (m *Snap) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "snap", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since snap has no native diagnostic command.
func (m *Snap) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for snap")
}

// Update refreshes all installed snaps.
func (m *Snap) Update(ctx context.Context) error {
	args := sudoCmd("snap", "refresh")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update snaps: %w", err)
	}
	return nil
}

// AddRepo returns an error since snap has no concept of repositories.
func (m *Snap) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for snap")
}

// RemoveRepo returns an error since snap has no concept of repositories.
func (m *Snap) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for snap")
}

// ListRepos returns an error since snap has no concept of repositories.
func (m *Snap) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, fmt.Errorf("not supported for snap")
}

// Compile-time interface check.
var _ Adapter = (*Snap)(nil)
