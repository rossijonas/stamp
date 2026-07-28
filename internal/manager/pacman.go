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
func (m *Pacman) Update(ctx context.Context, pkg string) error {
	var args []string
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args = sudoCmd("pacman", "-S", "--noconfirm", pkg)
	} else {
		args = sudoCmd("pacman", "-Syu", "--noconfirm")
	}
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

// ListRepos returns an empty list since pacman has no concept of repositories.
func (m *Pacman) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// CheckUpdate runs pacman -Qu to list available updates.
// pacman -Qu exits 1 when no updates are available (success path).
func (m *Pacman) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	syncArgs := sudoCmd("pacman", "-Sy")
	_, err := m.exec(ctx, syncArgs[0], syncArgs[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to sync databases: %w", err)
	}
	args := []string{"pacman", "-Qu"}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return nil, err
		}
		args = append(args, pkg)
	}
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		if exitCodeFromError(err) == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check updates: %w", err)
	}
	return parsePacmanQu(out), nil
}

func parsePacmanQu(output []byte) []UpdateInfo {
	var result []UpdateInfo
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		// Format: "pkg oldversion -> newversion"
		parts := bytes.SplitN(trimmed, []byte(" -> "), 2)
		if len(parts) != 2 {
			continue
		}
		nameAndCurrent := bytes.Fields(parts[0])
		if len(nameAndCurrent) == 0 {
			continue
		}
		name := string(nameAndCurrent[0])
		currentVer := ""
		if len(nameAndCurrent) > 1 {
			currentVer = string(nameAndCurrent[1])
		}
		avail := strings.TrimSpace(string(parts[1]))
		result = append(result, UpdateInfo{Package: name, CurrentVersion: currentVer, AvailableVersion: avail})
	}
	return result
}

var _ Adapter = (*Pacman)(nil)
