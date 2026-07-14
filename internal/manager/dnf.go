package manager

import (
	"context"
	"fmt"
	"os"
)

// DNF implements the Adapter interface for Fedora's DNF (or RHEL 7's yum).
type DNF struct {
	exec Executor
	cmd  string
}

// NewDNF creates a new DNF adapter for the given command ("dnf" or "yum").
func NewDNF(cmd string) *DNF {
	return &DNF{
		exec: defaultExecutor,
		cmd:  cmd,
	}
}

// sudoCmd builds a sudo command that is TTY-aware.
// In non-interactive environments (CI/pipes), adds -n to fail fast.
// In interactive terminals, omits -n so sudo can prompt for a password.
func sudoCmd(args ...string) []string {
	cmd := []string{"sudo"}
	stat, err := os.Stdin.Stat()
	if err == nil && stat.Mode()&os.ModeCharDevice == 0 {
		cmd = append(cmd, "-n")
	}
	return append(cmd, args...)
}

// Name returns the package manager identifier.
func (m *DNF) Name() string {
	return "dnf"
}

// ListInstalled returns a list of packages currently installed.
func (m *DNF) ListInstalled(ctx context.Context) ([]string, error) {
	var out []byte
	var err error
	if m.cmd == "yum" {
		out, err = m.exec(ctx, "repoquery", "--userinstalled", "--qf", "%{name}")
	} else {
		out, err = m.exec(ctx, "dnf", "repoquery", "--userinstalled", "--qf", "%{name}")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	return parseLines(out), nil
}

// Install executes the native installation command.
func (m *DNF) Install(ctx context.Context, pkg string) error {
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

// Remove executes the native removal command.
func (m *DNF) Remove(ctx context.Context, pkg string) error {
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
func (m *DNF) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, m.cmd, "search", "-q", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AddRepo enables a third-party repository.
func (m *DNF) AddRepo(ctx context.Context, name, url string) error {
	if url != "" {
		args := sudoCmd(m.cmd, "config-manager", "--add-repo", url)
		_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to add repo %s: %w", url, err)
		}
		return nil
	}
	args := sudoCmd(m.cmd, "copr", "enable", "-y", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to enable copr %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party repository.
func (m *DNF) RemoveRepo(ctx context.Context, name string) error {
	args := sudoCmd(m.cmd, "copr", "disable", "-y", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to disable copr %s: %w", name, err)
	}
	return nil
}

// Info queries dnf info metadata.
func (m *DNF) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, m.cmd, "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since dnf has no native diagnostic command.
func (m *DNF) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for dnf")
}
