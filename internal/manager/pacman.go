package manager

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
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

// ReconcileReliability reports OverInclusive: `pacman -Qq` returns all
// installed packages including dependencies. Consistent run-to-run.
func (m *Pacman) ReconcileReliability() ReconcileReliability {
	return ReliabilityOverInclusive
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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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

// InstallMany installs multiple packages in one pacman invocation.
func (m *Pacman) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("pacman", "-S", "--noconfirm"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// ReinstallMany reinstalls multiple packages in one pacman invocation.
func (m *Pacman) ReinstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("pacman", "-S", "--noconfirm"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall packages: %w", err)
	}
	return nil
}

// RemoveMany removes multiple packages in one pacman invocation.
func (m *Pacman) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("pacman", "-Rs", "--noconfirm"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
}

// PreviewInstall previews installing pkg.
// -S --print prints the resolved targets without modifying the system and
// without requiring root.
func (m *Pacman) PreviewInstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, "pacman", "-S", "--print", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview install %s: %w", pkg, err)
	}
	return Preview{Output: string(out)}, nil
}

// PreviewRemove previews removing pkg.
func (m *Pacman) PreviewRemove(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, "pacman", "-R", "--print", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview remove %s: %w", pkg, err)
	}
	return Preview{Output: string(out), Noop: strings.Contains(string(out), "was not found")}, nil
}

// PreviewReinstall previews reinstalling pkg.
// pacman reinstalls via the same -S operation as install.
func (m *Pacman) PreviewReinstall(ctx context.Context, pkg string) (Preview, error) {
	return m.PreviewInstall(ctx, pkg)
}

var _ Previewer = (*Pacman)(nil)

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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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

// Provides runs pacman -F <query> to find which package owns a file.
func (m *Pacman) Provides(ctx context.Context, query string) ([]string, error) {
	out, err := m.exec(ctx, "pacman", "-F", query)
	if err != nil {
		return nil, fmt.Errorf("failed to find provides for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AutoRemove lists orphans and removes them.
func (m *Pacman) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	// List orphans first
	out, err := m.exec(ctx, "pacman", "-Qdtq")
	if err != nil {
		return nil, fmt.Errorf("failed to list orphans: %w", err)
	}
	orphans := parseLines(out)
	if len(orphans) == 0 {
		return nil, nil
	}
	if dryRun {
		return orphans, nil
	}
	args := sudoCmd("pacman", "-Rs", "--noconfirm")
	args = append(args, orphans...)
	_, err = m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to remove orphans: %w", err)
	}
	return orphans, nil
}

// Clean runs pacman -Sc to clear the package cache.
func (m *Pacman) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	args := sudoCmd("pacman", "-Sc", "--noconfirm")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to clean pacman cache: %w", err)
	}
	return nil, nil
}

// Refresh syncs pacman databases via pacman -Sy.
func (m *Pacman) Refresh(ctx context.Context) error {
	args := sudoCmd("pacman", "-Sy")
	_, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to sync databases: %w", err)
	}
	return nil
}

// CheckUpdate runs pacman -Qu to list available updates.
// pacman -Qu exits 1 when no updates are available (success path).
func (m *Pacman) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
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
	result := parsePacmanQu(out)
	// Surface unrecognized output instead of silently reporting no updates.
	if len(result) == 0 && len(bytes.TrimSpace(out)) > 0 {
		return nil, fmt.Errorf("unrecognized pacman -Qu output (parser may be outdated)")
	}
	return result, nil
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

const pacmanConfPath = "/etc/pacman.conf"

// pacmanConfRead reads /etc/pacman.conf via the executor and returns lines.
func pacmanConfRead(ctx context.Context, exec Executor) ([]string, error) {
	out, err := exec(ctx, "cat", pacmanConfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pacman.conf: %w", err)
	}
	return strings.Split(string(out), "\n"), nil
}

// pacmanConfWrite writes config lines to /etc/pacman.conf using temp file + sudo cp.
func pacmanConfWrite(ctx context.Context, exec Executor, lines []string) error {
	tmpPath := fmt.Sprintf("/tmp/stamp-pacman-conf.%d", time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()
	args := sudoCmd("cp", tmpPath, pacmanConfPath)
	_, err := exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to write pacman.conf: %w", err)
	}
	return nil
}

// pacmanIgnorePkg reads the IgnorePkg line from pacman.conf via an executor.
func pacmanIgnorePkg(ctx context.Context, exec Executor) ([]string, error) {
	lines, err := pacmanConfRead(ctx, exec)
	if err != nil {
		return nil, err
	}
	inOptions := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[options]" {
			inOptions = true
			continue
		}
		if inOptions && strings.HasPrefix(trimmed, "[") && trimmed != "[options]" {
			inOptions = false
			continue
		}
		if inOptions && strings.HasPrefix(trimmed, "IgnorePkg") {
			eqIdx := strings.Index(trimmed, "=")
			if eqIdx < 0 {
				continue
			}
			value := strings.TrimSpace(trimmed[eqIdx+1:])
			return strings.Fields(value), nil
		}
	}
	return nil, nil
}

// Hold adds a package to IgnorePkg in pacman.conf to prevent upgrades.
func (m *Pacman) Hold(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}

	lines, err := pacmanConfRead(ctx, m.exec)
	if err != nil {
		return err
	}
	inOptions := false
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[options]" {
			inOptions = true
			continue
		}
		if inOptions && strings.HasPrefix(trimmed, "[") && trimmed != "[options]" {
			break
		}
		if inOptions && strings.HasPrefix(trimmed, "IgnorePkg") {
			eqIdx := strings.Index(trimmed, "=")
			if eqIdx < 0 {
				continue
			}
			value := strings.TrimSpace(trimmed[eqIdx+1:])
			pkgs := strings.Fields(value)
			for _, p := range pkgs {
				if p == pkg {
					return nil // already held
				}
			}
			lines[i] = line + " " + pkg
			modified = true
			break
		}
	}

	if !modified {
		for i, line := range lines {
			if strings.TrimSpace(line) == "[options]" {
				insert := "IgnorePkg = " + pkg
				lines = append(lines[:i+1], append([]string{insert}, lines[i+1:]...)...)
				modified = true
				break
			}
		}
	}

	if !modified {
		return fmt.Errorf("could not find [options] section in pacman.conf")
	}

	return pacmanConfWrite(ctx, m.exec, lines)
}

// Unhold removes a package from IgnorePkg in pacman.conf.
func (m *Pacman) Unhold(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}

	lines, err := pacmanConfRead(ctx, m.exec)
	if err != nil {
		return err
	}
	inOptions := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[options]" {
			inOptions = true
			continue
		}
		if inOptions && strings.HasPrefix(trimmed, "[") && trimmed != "[options]" {
			break
		}
		if inOptions && strings.HasPrefix(trimmed, "IgnorePkg") {
			eqIdx := strings.Index(trimmed, "=")
			if eqIdx < 0 {
				continue
			}
			before := strings.TrimSpace(trimmed[:eqIdx])
			value := strings.TrimSpace(trimmed[eqIdx+1:])
			pkgs := strings.Fields(value)
			newPkgs := make([]string, 0, len(pkgs))
			found := false
			for _, p := range pkgs {
				if p == pkg {
					found = true
				} else {
					newPkgs = append(newPkgs, p)
				}
			}
			if !found {
				return fmt.Errorf("package %s is not held", pkg)
			}
			if len(newPkgs) == 0 {
				lines[i] = before + " ="
			} else {
				lines[i] = before + " = " + strings.Join(newPkgs, " ")
			}
			return pacmanConfWrite(ctx, m.exec, lines)
		}
	}

	return fmt.Errorf("package %s is not held", pkg)
}

// ListHeld returns the list of packages in IgnorePkg from pacman.conf.
func (m *Pacman) ListHeld(ctx context.Context) ([]string, error) {
	return pacmanIgnorePkg(ctx, m.exec)
}

var _ Adapter = (*Pacman)(nil)
