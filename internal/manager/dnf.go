package manager

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
)

type groupKey struct{}

// WithGroup returns a context that signals group operations for dnf.
func WithGroup(ctx context.Context) context.Context {
	return context.WithValue(ctx, groupKey{}, true)
}

func isGroup(ctx context.Context) bool {
	v, _ := ctx.Value(groupKey{}).(bool)
	return v
}

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

// ReconcileReliability reports OverInclusive: `repoquery --userinstalled` is
// the documented dnf5 method but returns base OS packages (kernel, glibc,
// bash) that Anaconda marks as "User". Output is consistent run-to-run, so
// baseline diffing stays safe — reconcile may list system packages but will
// not produce false-positive drift.
func (m *DNF) ReconcileReliability() ReconcileReliability {
	return ReliabilityOverInclusive
}

// ListInstalled returns a list of packages currently installed.
//
// Strategy (behavior-probed, never version-sniffed):
//  1. `m.cmd history userinstalled` — transaction-based, precise on dnf4/legacy
//     (RHEL/CentOS/Rocky) and any build where the subcommand exists.
//  2. `m.cmd repoquery --userinstalled` — documented fallback for dnf5, where
//     `history userinstalled` is absent. Over-reports on dnf5 (Anaconda marks
//     system packages as user-installed), but output is consistent run-to-run.
//  3. error otherwise.
//
// `yum` (RHEL 7) is special-cased: it has neither a `history userinstalled`
// subcommand nor a `repoquery` subcommand — repoquery is a standalone binary
// from yum-utils, so it is invoked without a manager prefix.
func (m *DNF) ListInstalled(ctx context.Context) ([]string, error) {
	if m.cmd == "yum" {
		out, err := m.exec(ctx, "repoquery", "--userinstalled", "--qf", "%{name}\n")
		if err != nil {
			return nil, fmt.Errorf("failed to list installed packages: %w", err)
		}
		return parseLines(out), nil
	}

	out, err := m.exec(ctx, m.cmd, "history", "userinstalled")
	if err == nil {
		return parseDNFHistoryUserInstalled(out), nil
	}

	out, err = m.exec(ctx, m.cmd, "repoquery", "--userinstalled", "--qf", "%{name}\n")
	if err == nil {
		return parseLines(out), nil
	}

	return nil, fmt.Errorf("failed to list installed packages: %w", err)
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

// CheckInstalled reports which of pkgs are absent from the system. Presence
// resolves against the full installed set rather than userinstalled history,
// so dependency- and preseed-installed packages count as present, and intent
// names resolve through provides to their concrete packages (nodejs →
// nodejs22) — mirroring what `dnf install <pkg>` would satisfy. Read-only;
// never needs root.
//
// Names that fail ValidatePackageName are reported absent without executing
// anything. A failed full-set or provides lookup aborts the whole check with
// an error so callers can degrade to legacy ListInstalled matching instead of
// reporting false absences from partial data.
func (m *DNF) CheckInstalled(ctx context.Context, pkgs []string) ([]string, error) {
	bin := m.cmd
	if m.cmd == "yum" {
		bin = "repoquery"
	}

	valid := make([]string, 0, len(pkgs))
	var absent []string
	for _, p := range pkgs {
		if err := ValidatePackageName(p); err != nil {
			absent = append(absent, p)
			continue
		}
		valid = append(valid, p)
	}
	if len(valid) == 0 {
		return absent, nil
	}

	out, err := m.exec(ctx, bin, "repoquery", "--installed", "--qf", "%{name}\n")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	system := make(map[string]struct{}, 128)
	for _, n := range parseLines(out) {
		system[n] = struct{}{}
	}

	for _, p := range valid {
		if _, ok := system[p]; ok {
			continue
		}
		// Scope to installed packages: a bare --whatprovides also matches
		// available repo providers, which would report genuinely-missing
		// packages as present.
		out, err := m.exec(ctx, bin, "repoquery", "--installed", "--whatprovides", p, "--qf", "%{name}\n")
		if err != nil {
			return nil, fmt.Errorf("failed to resolve provides for %s: %w", p, err)
		}
		if len(bytes.TrimSpace(out)) == 0 {
			absent = append(absent, p)
		}
	}
	return absent, nil
}

var _ InstalledChecker = (*DNF)(nil)

// Install executes the native installation command.
func (m *DNF) Install(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if isGroup(ctx) {
		return sudoExec(ctx, m.exec, sudoCmd(m.cmd, "group", "install", "-y", pkg), fmt.Sprintf("failed to install group %s", pkg))
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	return sudoExec(ctx, m.exec, sudoCmd(m.cmd, "install", "-y", pkg), fmt.Sprintf("failed to install %s", pkg))
}

// Reinstall executes the native reinstallation command.
func (m *DNF) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if isGroup(ctx) {
		return m.Install(ctx, pkg)
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	return sudoExec(ctx, m.exec, sudoCmd(m.cmd, "reinstall", "-y", pkg), fmt.Sprintf("failed to reinstall %s", pkg))
}

// Remove executes the native removal command.
func (m *DNF) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if isGroup(ctx) {
		return sudoExec(ctx, m.exec, sudoCmd(m.cmd, "group", "remove", "-y", pkg), fmt.Sprintf("failed to remove group %s", pkg))
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	return sudoExec(ctx, m.exec, sudoCmd(m.cmd, "remove", "-y", pkg), fmt.Sprintf("failed to remove %s", pkg))
}

// InstallMany installs multiple packages in one dnf invocation. Groups are
// single-package only (the CLI rejects --group with multiple packages).
func (m *DNF) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if isGroup(ctx) {
		return fmt.Errorf("batch install is not supported for groups")
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	return sudoExec(ctx, m.exec, batchArgs(sudoCmd(m.cmd, "install", "-y"), pkgs), "failed to install packages")
}

// ReinstallMany reinstalls multiple packages in one dnf invocation.
func (m *DNF) ReinstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	return sudoExec(ctx, m.exec, batchArgs(sudoCmd(m.cmd, "reinstall", "-y"), pkgs), "failed to reinstall packages")
}

// RemoveMany removes multiple packages in one dnf invocation.
func (m *DNF) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	return sudoExec(ctx, m.exec, batchArgs(sudoCmd(m.cmd, "remove", "-y"), pkgs), "failed to remove packages")
}

// PreviewInstall previews installing pkg.
// dnf resolves the transaction only when run with privileges, so this runs
// under sudo; --assumeno answers "no" to the confirmation prompt, so no
// system change occurs.
func (m *DNF) PreviewInstall(ctx context.Context, pkg string) (Preview, error) {
	ctx = WithCombinedOutput(ctx)
	if isGroup(ctx) {
		args := sudoCmd(m.cmd, "group", "install", "--assumeno", pkg)
		out, err := m.exec(ctx, args[0], args[1:]...)
		if err != nil && len(bytes.TrimSpace(out)) == 0 {
			return Preview{}, fmt.Errorf("failed to preview group install %s: %w", pkg, err)
		}
		return Preview{Output: string(out), Noop: strings.Contains(string(out), "Nothing to do.")}, nil
	}
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	args := sudoCmd(m.cmd, "install", "--assumeno", pkg)
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview install %s: %w", pkg, err)
	}
	return Preview{Output: string(out), Noop: strings.Contains(string(out), "Nothing to do.")}, nil
}

// PreviewRemove previews removing pkg.
// See PreviewInstall for the privilege/--assumeno notes.
func (m *DNF) PreviewRemove(ctx context.Context, pkg string) (Preview, error) {
	ctx = WithCombinedOutput(ctx)
	if isGroup(ctx) {
		// Groups may contain spaces, so skip package-name validation.
		args := sudoCmd(m.cmd, "group", "remove", "--assumeno", pkg)
		out, err := m.exec(ctx, args[0], args[1:]...)
		if err != nil && len(bytes.TrimSpace(out)) == 0 {
			return Preview{}, fmt.Errorf("failed to preview group remove %s: %w", pkg, err)
		}
		return Preview{Output: string(out), Noop: strings.Contains(string(out), "Nothing to do.")}, nil
	}
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	args := sudoCmd(m.cmd, "remove", "--assumeno", pkg)
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview remove %s: %w", pkg, err)
	}
	// dnf remove of an absent package errors "No match for argument".
	return Preview{Output: string(out), Noop: strings.Contains(string(out), "No match for argument")}, nil
}

// PreviewReinstall previews reinstalling pkg.
// See PreviewInstall for the privilege/--assumeno notes.
func (m *DNF) PreviewReinstall(ctx context.Context, pkg string) (Preview, error) {
	if isGroup(ctx) {
		return m.PreviewInstall(ctx, pkg)
	}
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	args := sudoCmd(m.cmd, "reinstall", "--assumeno", pkg)
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview reinstall %s: %w", pkg, err)
	}
	// Reinstalling an installed package is always a real operation; only an
	// absent package is a no-op (dnf errors "No match for argument"/"not
	// installed"). Reinstall is never a no-op just because a version matches.
	s := string(out)
	return Preview{Output: s, Noop: strings.Contains(s, "No match for argument") || strings.Contains(s, "not installed")}, nil
}

var _ Previewer = (*DNF)(nil)

// parseDNFGroupList parses the output of 'dnf group list' and returns group names.
// Format:
//
//	Installed Environment Groups:
//	   Development Tools
//	Installed Groups:
//	   C Development Tools
//	Available Groups:
//	   Backup Client
func parseDNFGroupList(output []byte) []string {
	var result []string
	// Group names are indented with 3+ spaces in 'dnf group list' output.
	// Headers (e.g. "Installed Groups:") have no leading whitespace.
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		// Group names are indented with leading whitespace; headers are not.
		// Heuristic: if the line has leading spaces and the trimmed content
		// doesn't contain a colon, it's a group name.
		if bytes.HasPrefix([]byte(line), []byte("   ")) && !bytes.Contains(trimmed, []byte(":")) {
			result = append(result, string(trimmed))
		}
	}
	return result
}

// Search queries the native package manager for the given package name.
func (m *DNF) Search(ctx context.Context, query string) ([]string, error) {
	if isGroup(ctx) {
		out, err := m.exec(ctx, m.cmd, "group", "list")
		if err != nil {
			return nil, fmt.Errorf("failed to list groups: %w", err)
		}
		groups := parseDNFGroupList(out)
		var result []string
		lowerQuery := strings.ToLower(query)
		for _, g := range groups {
			if strings.Contains(strings.ToLower(g), lowerQuery) {
				result = append(result, g)
			}
		}
		return result, nil
	}
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
	if isGroup(ctx) {
		out, err := m.exec(ctx, m.cmd, "group", "info", pkg)
		if err != nil {
			return "", fmt.Errorf("failed to get group info for %s: %w", pkg, err)
		}
		return string(out), nil
	}
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
	if err := requireConsent(ctx); err != nil {
		return err
	}
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
	result := parseDNFCheckUpdate(out)
	// A non-empty check-update output that the parser could not recognize means
	// the vendor changed the format; surface it instead of silently reporting
	// "all up to date".
	if len(result) == 0 && len(bytes.TrimSpace(out)) > 0 {
		return nil, fmt.Errorf("unrecognized %s check-update output (parser may be outdated)", m.cmd)
	}
	return result, nil
}

// Refresh syncs dnf metadata via makecache so subsequent checks are fresh.
func (m *DNF) Refresh(ctx context.Context) error {
	args := sudoCmd(m.cmd, "makecache", "-q")
	_, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to refresh metadata: %w", err)
	}
	return nil
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
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	return nil, sudoExec(ctx, m.exec, sudoCmd(m.cmd, "autoremove", "-y"), "failed to autoremove")
}

// Clean runs dnf clean all to clear the package cache.
func (m *DNF) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	return nil, sudoExec(ctx, m.exec, sudoCmd(m.cmd, "clean", "all"), "failed to clean dnf cache")
}

// Hold pins a package via dnf versionlock add.
func (m *DNF) Hold(ctx context.Context, pkg string) error {
	return runSingle(ctx, m.exec, sudoCmd(m.cmd, "versionlock", "add", pkg), "hold", pkg)
}

// Unhold removes a version pin via dnf versionlock delete.
func (m *DNF) Unhold(ctx context.Context, pkg string) error {
	return runSingle(ctx, m.exec, sudoCmd(m.cmd, "versionlock", "delete", pkg), "unhold", pkg)
}

// ListHeld returns held packages via dnf versionlock list.
func (m *DNF) ListHeld(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, m.cmd, "versionlock", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list held packages: %w", err)
	}
	return parseLines(out), nil
}
