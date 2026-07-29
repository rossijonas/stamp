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

// stdIn is overridable in tests to simulate pipe vs TTY for sudo decisions.
var stdIn = os.Stdin

// sudoPassword caches the password for sudo -S when provided by the CLI layer.
var sudoPassword []byte

// SetSudoPassword stores a password for use with sudo -S.
// The password is cleared automatically via ClearSudoPassword after the run phase.
func SetSudoPassword(pw []byte) {
	sudoPassword = pw
}

// ClearSudoPassword zeros and releases the cached sudo password.
func ClearSudoPassword() {
	if sudoPassword != nil {
		clear(sudoPassword)
		sudoPassword = nil
	}
}

// sudoCmd builds a sudo command that is TTY-aware.
// When a password is cached via SetSudoPassword, appends -S and the executor pipes it to stdin.
// In non-interactive environments (CI/pipes) without a cached password, adds -n to fail fast.
// In interactive terminals without a cached password, omits extra flags so sudo prompts normally.
func sudoCmd(args ...string) []string {
	cmd := []string{"sudo"}
	if sudoPassword != nil {
		cmd = append(cmd, "-S")
	} else {
		stat, err := stdIn.Stat()
		if err == nil && stat.Mode()&os.ModeCharDevice == 0 {
			cmd = append(cmd, "-n")
		}
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

// Update runs the native system upgrade command.
func (m *DNF) Update(ctx context.Context, pkg string) error {
	args := sudoCmd(m.cmd, "upgrade", "-y")
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

// CheckUpdate runs dnf check-update to list available updates.
// dnf check-update exits 100 when updates exist — that's success.
func (m *DNF) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	args := []string{m.cmd, "check-update"}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return nil, err
		}
		args = append(args, pkg)
	}
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		if exitCodeFromError(err) != 100 {
			return nil, fmt.Errorf("failed to check updates: %w", err)
		}
	}
	return parseDNFCheckUpdate(out), nil
}

func parseDNFCheckUpdate(output []byte) []UpdateInfo {
	var result []UpdateInfo
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		// Format: "pkg.arch version repo"
		name := string(fields[0])
		if dotIdx := strings.Index(name, "."); dotIdx > 0 {
			name = name[:dotIdx]
		}
		result = append(result, UpdateInfo{Package: name, CurrentVersion: string(fields[1])})
	}
	return result
}

// Provides runs dnf provides to find which package owns a file.
func (m *DNF) Provides(ctx context.Context, query string) ([]string, error) {
	out, err := m.exec(ctx, m.cmd, "provides", query)
	if err != nil {
		return nil, fmt.Errorf("failed to find provides for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AutoRemove removes orphaned packages via dnf autoremove.
func (m *DNF) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if dryRun {
		return nil, nil
	}
	args := sudoCmd(m.cmd, "autoremove", "-y")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to autoremove: %w", err)
	}
	return nil, nil
}

// Clean runs dnf clean all to clear the package cache.
func (m *DNF) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if dryRun {
		return nil, nil
	}
	args := sudoCmd(m.cmd, "clean", "all")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to clean dnf cache: %w", err)
	}
	return nil, nil
}

// Hold pins a package via dnf versionlock add.
func (m *DNF) Hold(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "versionlock", "add", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to hold %s: %w", pkg, err)
	}
	return nil
}

// Unhold removes a version pin via dnf versionlock delete.
func (m *DNF) Unhold(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "versionlock", "delete", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to unhold %s: %w", pkg, err)
	}
	return nil
}

// ListHeld returns held packages via dnf versionlock list.
func (m *DNF) ListHeld(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, m.cmd, "versionlock", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list held packages: %w", err)
	}
	return parseLines(out), nil
}
