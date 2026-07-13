package manager

import (
	"context"
	"fmt"
)

// DNF implements the Adapter interface for Fedora's DNF.
type DNF struct {
	exec Executor
}

// NewDNF creates a new DNF with the default system executor.
func NewDNF() *DNF {
	return &DNF{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *DNF) Name() string {
	return "dnf"
}

// ListInstalled returns a list of packages currently installed.
func (m *DNF) ListInstalled(ctx context.Context) ([]string, error) {
	// Query user-installed packages, formatted to only show the name.
	out, err := m.exec(ctx, "dnf", "repoquery", "--userinstalled", "--qf", "%{name}")
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
	_, err := m.exec(WithStreamIO(ctx), "sudo", "-n", "dnf", "install", "-y", pkg)
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
	_, err := m.exec(WithStreamIO(ctx), "sudo", "-n", "dnf", "remove", "-y", pkg)
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
	out, err := m.exec(ctx, "dnf", "search", "-q", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AddRepo enables a third-party repository.
func (m *DNF) AddRepo(ctx context.Context, name, url string) error {
	if url != "" {
		_, err := m.exec(WithStreamIO(ctx), "sudo", "-n", "dnf", "config-manager", "--add-repo", url)
		if err != nil {
			return fmt.Errorf("failed to add repo %s: %w", url, err)
		}
		return nil
	}
	_, err := m.exec(WithStreamIO(ctx), "sudo", "-n", "dnf", "copr", "enable", "-y", name)
	if err != nil {
		return fmt.Errorf("failed to enable copr %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party repository.
func (m *DNF) RemoveRepo(ctx context.Context, name string) error {
	_, err := m.exec(WithStreamIO(ctx), "sudo", "-n", "dnf", "copr", "disable", "-y", name)
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
	out, err := m.exec(ctx, "dnf", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since dnf has no native diagnostic command.
func (m *DNF) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for dnf")
}
