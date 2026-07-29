package manager

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// Zypper implements the Adapter interface for the openSUSE/SLES Zypper package manager.
type Zypper struct {
	exec Executor
	cmd  string
}

// NewZypper creates a new Zypper adapter.
func NewZypper() *Zypper {
	return &Zypper{
		exec: defaultExecutor,
		cmd:  "zypper",
	}
}

// Name returns "zypper".
func (m *Zypper) Name() string {
	return "zypper"
}

func parseZypperSearch(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if trimmed[0] == 'S' || bytes.HasPrefix(trimmed, []byte("--+")) {
			continue
		}
		parts := bytes.Split(trimmed, []byte("|"))
		if len(parts) < 3 {
			continue
		}
		name := string(bytes.TrimSpace(parts[1]))
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	return result
}

// ListInstalled returns a list of installed packages via zypper.
func (m *Zypper) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, m.cmd, "search", "--installed-only", "--type", "package")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseZypperSearch(out), nil
}

// Install installs a package via zypper.
func (m *Zypper) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "install", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall reinstalls a package via zypper.
func (m *Zypper) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "install", "--force", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove removes a package via zypper.
func (m *Zypper) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "remove", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search searches for packages via zypper.
func (m *Zypper) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, m.cmd, "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseZypperSearch(out), nil
}

// Info returns details about a package via zypper.
func (m *Zypper) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, m.cmd, "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since zypper has no native diagnostic command.
func (m *Zypper) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for zypper")
}

// Update runs a full system upgrade via zypper.
func (m *Zypper) Update(ctx context.Context, pkg string) error {
	args := sudoCmd(m.cmd, "update", "-y")
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args = append(args, pkg)
	}
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	return nil
}

// AddRepo returns an error since repo management is not supported for zypper.
func (m *Zypper) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for zypper")
}

// RemoveRepo returns an error since repo management is not supported for zypper.
func (m *Zypper) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for zypper")
}

// ListRepos returns an empty list until native zypper repo listing is implemented (ADR-008).
func (m *Zypper) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Clean runs zypper clean to clear the package cache.
func (m *Zypper) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if dryRun {
		return nil, nil
	}
	args := sudoCmd("zypper", "clean", "--non-interactive")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to clean zypper cache: %w", err)
	}
	return nil, nil
}

// Provides runs zypper what-provides to find which package owns a file.
func (m *Zypper) Provides(ctx context.Context, query string) ([]string, error) {
	out, err := m.exec(ctx, "zypper", "what-provides", query)
	if err != nil {
		return nil, fmt.Errorf("failed to find provides for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AutoRemove removes unneeded dependencies via zypper rm -u.
func (m *Zypper) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if dryRun {
		return nil, nil
	}
	args := sudoCmd("zypper", "rm", "-u", "--non-interactive")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to autoremove: %w", err)
	}
	return nil, nil
}

// CheckUpdate runs zypper list-updates to show available updates.
func (m *Zypper) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	refreshArgs := sudoCmd("zypper", "refresh")
	_, err := m.exec(ctx, refreshArgs[0], refreshArgs[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh repositories: %w", err)
	}
	args := []string{"zypper", "list-updates"}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return nil, err
		}
		args = append(args, pkg)
	}
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to check updates: %w", err)
	}
	return parseZypperListUpdates(out), nil
}

func parseZypperListUpdates(output []byte) []UpdateInfo {
	var result []UpdateInfo
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "S") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 5 {
			continue
		}
		// Format: S | Repository | Name | Current | Available
		// fields: ["v", "|", "main", "|", "htop", "|", "3.2.1", "|", "3.2.2"]
		// Name is at index 4
		result = append(result, UpdateInfo{Package: fields[4]})
	}
	return result
}

// Hold returns an error since zypper has no hold command.
func (m *Zypper) Hold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: hold not supported for zypper", ErrNotSupported)
}

// Unhold returns an error since zypper has no unhold command.
func (m *Zypper) Unhold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: unhold not supported for zypper", ErrNotSupported)
}

// ListHeld returns an error since zypper has no hold command.
func (m *Zypper) ListHeld(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("%w: hold not supported for zypper", ErrNotSupported)
}

var _ Adapter = (*Zypper)(nil)
