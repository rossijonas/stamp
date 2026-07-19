package manager

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// APT implements the Adapter interface for Debian/Ubuntu's APT (or apt-get).
type APT struct {
	exec Executor
	cmd  string
}

// NewAPT creates a new APT adapter for the given command ("apt" or "apt-get").
func NewAPT(cmd string) *APT {
	return &APT{
		exec: defaultExecutor,
		cmd:  cmd,
	}
}

// Name returns the package manager identifier.
func (m *APT) Name() string {
	return "apt"
}

// ListInstalled returns a list of packages currently installed.
func (m *APT) ListInstalled(ctx context.Context) ([]string, error) {
	// Primary: apt list --installed
	out, err := m.exec(ctx, m.cmd, "list", "--installed")
	if err == nil {
		return parseAPTListInstalled(out), nil
	}

	// Fallback: dpkg-query with state filtering (excludes "rc" packages)
	out, err = m.exec(ctx, "dpkg-query", "-f", "${db:Status-Status} ${Package}\n", "-W")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	return parseDPKGQueryInstalled(out), nil
}

// parseAPTListInstalled parses the output of 'apt list --installed'.
// Format: "htop/stable,now 3.2.1 amd64 [installed]"
func parseAPTListInstalled(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("Listing")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		pkgField := fields[0]
		// Package name is the text before '/' or ','
		idx := bytes.IndexAny(pkgField, "/,")
		if idx > 0 {
			result = append(result, string(pkgField[:idx]))
		}
	}
	return result
}

// parseDPKGQueryInstalled parses the output of 'dpkg-query -f "${db:Status-Status} ${Package}\n" -W'.
// Filters only lines with "installed" status to exclude "rc" (removed + config) packages.
func parseDPKGQueryInstalled(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) >= 2 && string(fields[0]) == "installed" {
			result = append(result, string(fields[1]))
		}
	}
	return result
}

// Install executes the native installation command.
func (m *APT) Install(ctx context.Context, pkg string) error {
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

// Reinstall executes the native reinstallation command.
func (m *APT) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "install", "--reinstall", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *APT) Remove(ctx context.Context, pkg string) error {
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

// Search queries the native package manager for the given package name.
func (m *APT) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "apt-cache", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	// Output: "pkgname - description"
	lines := parseLines(out)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			result = append(result, fields[0])
		}
	}
	return result, nil
}

// Info queries apt info metadata.
func (m *APT) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	var out []byte
	var err error
	if m.cmd == "apt" {
		out, err = m.exec(ctx, "apt", "show", pkg)
	} else {
		out, err = m.exec(ctx, "apt-cache", "show", pkg)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since apt has no native diagnostic command.
func (m *APT) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for apt")
}

// Update runs apt update then apt upgrade (two-phase).
func (m *APT) Update(ctx context.Context) error {
	args := sudoCmd(m.cmd, "update")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}

	args = sudoCmd(m.cmd, "upgrade", "-y")
	_, err = m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}

	return nil
}
