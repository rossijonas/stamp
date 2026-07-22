package manager

import (
	"bytes"
	"context"
	"fmt"
)

// MacPorts implements the Adapter interface for MacPorts on macOS.
type MacPorts struct {
	exec Executor
}

// NewMacPorts creates a new MacPorts adapter.
func NewMacPorts() *MacPorts {
	return &MacPorts{
		exec: defaultExecutor,
	}
}

// Name returns "macports".
func (m *MacPorts) Name() string {
	return "macports"
}

func parsePortInstalled(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if bytes.HasPrefix(trimmed, []byte("The following")) {
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

func parsePortSearch(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
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

// ListInstalled returns a list of installed ports.
func (m *MacPorts) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "port", "installed")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed ports: %w", err)
	}
	return parsePortInstalled(out), nil
}

// Install installs a port via MacPorts.
func (m *MacPorts) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("port", "install", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall reinstalls a port via MacPorts.
func (m *MacPorts) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("port", "install", "--force", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove removes a port via MacPorts.
func (m *MacPorts) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("port", "uninstall", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search searches for ports via MacPorts.
func (m *MacPorts) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "port", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parsePortSearch(out), nil
}

// Info returns details about a port.
func (m *MacPorts) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "port", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since MacPorts has no native diagnostic command.
func (m *MacPorts) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for macports")
}

// Update runs selfupdate then upgrades outdated ports.
func (m *MacPorts) Update(ctx context.Context) error {
	args := sudoCmd("port", "selfupdate")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	upgradeArgs := sudoCmd("port", "upgrade", "outdated")
	_, err = m.exec(WithStreamIO(ctx), upgradeArgs[0], upgradeArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to upgrade outdated ports: %w", err)
	}
	return nil
}

// AddRepo returns an error since repo management is not supported for MacPorts.
func (m *MacPorts) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for macports")
}

// RemoveRepo returns an error since repo management is not supported for MacPorts.
func (m *MacPorts) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for macports")
}

// ListRepos returns an error since repo management is not supported for MacPorts.
func (m *MacPorts) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, fmt.Errorf("not supported for macports")
}

var _ Adapter = (*MacPorts)(nil)
