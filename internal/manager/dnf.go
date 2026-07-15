package manager

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
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
		out, err = m.exec(ctx, "repoquery", "--userinstalled", "--qf", "%{name}\n")
	} else {
		out, err = m.exec(ctx, "dnf", "repoquery", "--userinstalled", "--qf", "%{name}\n")
	}
	if err != nil {
		// Fallback: try dnf history userinstalled on repoquery failure.
		out, err = m.exec(ctx, m.cmd, "history", "userinstalled")
		if err != nil {
			return nil, fmt.Errorf("failed to list installed packages: %w", err)
		}
		return parseDNFHistoryUserInstalled(out), nil
	}

	return parseLines(out), nil
}

// parseDNFHistoryUserInstalled parses the output of 'dnf history userinstalled'.
// Lines are in NEVRA format (name-version-release.arch). Extracts the package name
// by taking everything before the second-to-last hyphen.
func parseDNFHistoryUserInstalled(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		// Skip header lines that don't look like NEVRA
		s := string(trimmed)
		parts := strings.Split(s, "-")
		if len(parts) < 3 {
			continue
		}
		// Everything before the second-to-last hyphen is the package name.
		name := strings.Join(parts[:len(parts)-2], "-")
		result = append(result, name)
	}
	return result
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

// Reinstall executes the native reinstallation command.
func (m *DNF) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "reinstall", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
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

// ListRepos returns a list of configured third-party repositories.
func (m *DNF) ListRepos(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, m.cmd, "repolist")
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	return parseDNFRepos(out), nil
}

// parseDNFRepos parses the output of 'dnf repolist' and extracts
// custom/third-party repository identifiers, filtering out known system repos.
func parseDNFRepos(output []byte) []string {
	systemRepos := map[string]bool{
		"fedora":                true,
		"fedora-updates":        true,
		"fedora-modular":        true,
		"updates":               true,
		"updates-modular":       true,
		"updates-testing":       true,
		"baseos":                true,
		"appstream":             true,
		"extras":                true,
		"epel":                  true,
		"epel-next":             true,
		"epel-testing":          true,
		"epel-next-testing":     true,
		"fedora-cisco-openh264": true,
	}

	lines := bytes.Split(output, []byte("\n"))
	repos := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		// Skip header lines
		if bytes.HasPrefix(bytes.ToLower(trimmed), []byte("repo id")) ||
			bytes.HasPrefix(bytes.ToLower(trimmed), []byte("id")) {
			continue
		}
		// First field is repo ID
		fields := bytes.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		id := string(fields[0])
		if systemRepos[id] {
			continue
		}
		repos = append(repos, id)
	}
	return repos
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
