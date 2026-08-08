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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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

// InstallMany installs multiple packages in one apt invocation.
func (m *APT) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd(m.cmd, "install", "-y"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// ReinstallMany reinstalls multiple packages in one apt invocation.
func (m *APT) ReinstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd(m.cmd, "install", "--reinstall", "-y"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall packages: %w", err)
	}
	return nil
}

// RemoveMany removes multiple packages in one apt invocation.
func (m *APT) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd(m.cmd, "remove", "-y"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
}

// PreviewInstall previews installing pkg.
// --assume-no implies --simulate: no root, no locks, no system change.
func (m *APT) PreviewInstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, m.cmd, "install", "--assume-no", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview install %s: %w", pkg, err)
	}
	s := string(out)
	noop := strings.Contains(s, "is already the newest version") || strings.Contains(s, "0 newly installed")
	return Preview{Output: s, Noop: noop}, nil
}

// PreviewRemove previews removing pkg.
// See PreviewInstall for the --assume-no semantics.
func (m *APT) PreviewRemove(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, m.cmd, "remove", "--assume-no", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview remove %s: %w", pkg, err)
	}
	s := string(out)
	return Preview{Output: s, Noop: strings.Contains(s, "is not installed") || strings.Contains(s, "0 to remove")}, nil
}

// PreviewReinstall previews reinstalling pkg.
func (m *APT) PreviewReinstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, m.cmd, "install", "--reinstall", "--assume-no", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview reinstall %s: %w", pkg, err)
	}
	s := string(out)
	noop := strings.Contains(s, "is already the newest version") || strings.Contains(s, "0 newly installed")
	return Preview{Output: s, Noop: noop}, nil
}

var _ Previewer = (*APT)(nil)

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
// If pkg is non-empty, updates only that package via apt install --only-upgrade.
func (m *APT) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args := sudoCmd(m.cmd, "install", "--only-upgrade", "-y", pkg)
		_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to update %s: %w", pkg, err)
		}
		return nil
	}

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

// Refresh syncs apt package lists via apt update.
func (m *APT) Refresh(ctx context.Context) error {
	args := sudoCmd(m.cmd, "update")
	_, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}
	return nil
}

// CheckUpdate returns a list of upgradable packages for apt.
// Uses "apt list --upgradable" even for apt-get (apt-get has no list subcommand).
func (m *APT) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	args := []string{"apt", "list", "--upgradable"}
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
	result := parseAPTUpgradable(out)
	// A bare "Listing..." header means no upgrades; anything else that did not
	// parse signals the vendor changed the output format.
	if len(result) == 0 && !aptOnlyUpgradableHeader(out) {
		return nil, fmt.Errorf("unrecognized apt list --upgradable output (parser may be outdated)")
	}
	return result, nil
}

// aptOnlyUpgradableHeader reports whether the output consists solely of the
// "Listing..." header (i.e. there are no upgrades).
func aptOnlyUpgradableHeader(output []byte) bool {
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("Listing")) {
			continue
		}
		return false
	}
	return true
}

func parseAPTUpgradable(output []byte) []UpdateInfo {
	var result []UpdateInfo
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("Listing")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		// Format: "htop/stable 1.2.3 amd64 [upgradable from: 1.2.2]"
		pkgField := fields[0]
		idx := bytes.IndexByte(pkgField, '/')
		if idx <= 0 {
			continue
		}
		name := string(pkgField[:idx])
		availVer := string(fields[1])
		currentVer := ""
		for i, f := range fields {
			if string(f) == "from:" && i+1 < len(fields) {
				currentVer = strings.TrimRight(string(fields[i+1]), "]")
			}
		}
		result = append(result, UpdateInfo{Package: name, CurrentVersion: currentVer, AvailableVersion: availVer})
	}
	return result
}

// Provides runs dpkg -S to find which package owns a file.
func (m *APT) Provides(ctx context.Context, query string) ([]string, error) {
	out, err := m.exec(ctx, "dpkg", "-S", query)
	if err != nil {
		return nil, fmt.Errorf("failed to find provides for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AutoRemove removes orphaned packages via apt autoremove.
func (m *APT) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	args := sudoCmd("apt", "autoremove", "-y")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to autoremove: %w", err)
	}
	return nil, nil
}

// Clean runs apt clean to clear the package cache.
func (m *APT) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	args := sudoCmd("apt", "clean")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to clean apt cache: %w", err)
	}
	return nil, nil
}

// Hold pins a package via apt-mark hold.
func (m *APT) Hold(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("apt-mark", "hold", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to hold %s: %w", pkg, err)
	}
	return nil
}

// Unhold removes a version pin via apt-mark unhold.
func (m *APT) Unhold(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("apt-mark", "unhold", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to unhold %s: %w", pkg, err)
	}
	return nil
}

// ListHeld returns held packages via apt-mark showhold.
func (m *APT) ListHeld(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "apt-mark", "showhold")
	if err != nil {
		return nil, fmt.Errorf("failed to list held packages: %w", err)
	}
	return parseLines(out), nil
}
