package manager

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// Pacman implements the Adapter interface for Arch Linux's Pacman.
type Pacman struct {
	exec Executor
}

// NewPacman creates a new Pacman adapter.
func NewPacman() *Pacman {
	return &Pacman{
		exec: defaultExecutor,
	}
}

// Name returns "pacman".
func (m *Pacman) Name() string {
	return "pacman"
}

func parsePacmanSearch(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		fields := bytes.Fields(line)
		if len(fields) == 0 {
			continue
		}
		full := string(fields[0])
		parts := strings.SplitN(full, "/", 2)
		if len(parts) == 2 {
			result = append(result, parts[1])
		}
	}
	return result
}

// ListInstalled returns a list of installed packages via pacman.
func (m *Pacman) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "pacman", "-Qq")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseLines(out), nil
}

// Install installs a package via pacman.
func (m *Pacman) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("pacman", "-S", "--noconfirm", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall reinstalls a package via pacman.
func (m *Pacman) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("pacman", "-S", "--noconfirm", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove removes a package and its unneeded dependencies via pacman.
func (m *Pacman) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	// -Rs removes the package and its unneeded dependencies.
	args := sudoCmd("pacman", "-Rs", "--noconfirm", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search searches for packages via pacman.
func (m *Pacman) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "pacman", "-Ss", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parsePacmanSearch(out), nil
}

// Info returns details about a package.
func (m *Pacman) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "pacman", "-Qi", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since pacman has no native diagnostic command.
func (m *Pacman) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for pacman")
}

// Update syncs and upgrades all packages via pacman.
func (m *Pacman) Update(ctx context.Context) error {
	args := sudoCmd("pacman", "-Syu", "--noconfirm")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	return nil
}

// AddRepo returns an error since repo management is not supported for pacman.
func (m *Pacman) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for pacman")
}

// RemoveRepo returns an error since repo management is not supported for pacman.
func (m *Pacman) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for pacman")
}

// ListRepos returns an error since repo management is not supported for pacman.
func (m *Pacman) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, fmt.Errorf("not supported for pacman")
}

var _ Adapter = (*Pacman)(nil)
