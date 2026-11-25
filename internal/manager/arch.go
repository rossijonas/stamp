package manager

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
)

// geteuid is overridable in tests to exercise the AUR root-refusal path.
var geteuid = os.Geteuid

// archHelper shares pacman-compatible behavior across pacman, paru, and yay.
//
// flags are extra non-interactive flags for install/update operations (yay's
// --answer* set). upgradeFlags apply only to a full -Syu. Methods always add
// --noconfirm themselves; flags must not repeat it.
type archHelper struct {
	exec         Executor
	name         string
	bin          string
	flags        []string
	upgradeFlags []string
	escalate     bool // wrap commands with sudo (pacman); false for AUR helpers
	strictQu     bool // reject unrecognized -Qu output (pacman only)
}

func (m *archHelper) Name() string   { return m.name }
func (m *archHelper) Binary() string { return m.bin }

func (m *archHelper) ReconcileReliability() ReconcileReliability {
	return ReliabilityOverInclusive
}

// requireUser refuses helper operations when stamp runs as root. AUR helpers
// build as the invoking user and escalate internally.
func (m *archHelper) requireUser() error {
	if !m.escalate && geteuid() == 0 {
		return fmt.Errorf("%s must run as your user; run stamp without sudo", m.name)
	}
	return nil
}

// argv prepends the binary and wraps with sudo when the adapter escalates.
func (m *archHelper) argv(args ...string) []string {
	full := append([]string{m.bin}, args...)
	if m.escalate {
		return sudoCmd(full...)
	}
	return full
}

func (m *archHelper) run(ctx context.Context, args ...string) ([]byte, error) {
	argv := m.argv(args...)
	return m.exec(ctx, argv[0], argv[1:]...)
}

func (m *archHelper) stream(ctx context.Context, args ...string) error {
	argv := m.argv(args...)
	_, err := m.exec(WithStreamIO(ctx), argv[0], argv[1:]...)
	return err
}

func (m *archHelper) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.run(ctx, "-Qq")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseLines(out), nil
}

// installArgs builds args for -S operations, adding helper flags.
func (m *archHelper) installArgs(pkgs ...string) []string {
	args := []string{"-S", pacmanFlagNoConfirm}
	args = append(args, m.flags...)
	return append(args, pkgs...)
}

func (m *archHelper) Install(ctx context.Context, pkg string) error {
	if err := m.requireUser(); err != nil {
		return err
	}
	return runSingle(ctx, m.exec, m.argv(m.installArgs(pkg)...), "install", m.name, pkg)
}

func (m *archHelper) Reinstall(ctx context.Context, pkg string) error {
	if err := m.requireUser(); err != nil {
		return err
	}
	return runSingle(ctx, m.exec, m.argv(m.installArgs(pkg)...), "reinstall", m.name, pkg)
}

func (m *archHelper) Remove(ctx context.Context, pkg string) error {
	if err := m.requireUser(); err != nil {
		return err
	}
	return runSingle(ctx, m.exec, m.argv("-Rs", pacmanFlagNoConfirm, pkg), "remove", m.name, pkg)
}

func (m *archHelper) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := m.requireUser(); err != nil {
		return err
	}
	return runBatch(ctx, m.exec, batchArgs(m.argv(m.installArgs()...), pkgs), "install", m.name, pkgs)
}

func (m *archHelper) ReinstallMany(ctx context.Context, pkgs ...string) error {
	if err := m.requireUser(); err != nil {
		return err
	}
	return runBatch(ctx, m.exec, batchArgs(m.argv(m.installArgs()...), pkgs), "reinstall", m.name, pkgs)
}

func (m *archHelper) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := m.requireUser(); err != nil {
		return err
	}
	return runBatch(ctx, m.exec, batchArgs(m.argv("-Rs", pacmanFlagNoConfirm), pkgs), "remove", m.name, pkgs)
}

func (m *archHelper) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidateSpecForManager(m.name, query); err != nil {
		return nil, err
	}
	out, err := m.run(ctx, "-Ss", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parsePacmanSearch(out), nil
}

func (m *archHelper) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidateSpecForManager(m.name, pkg); err != nil {
		return "", err
	}
	out, err := m.run(ctx, "-Qi", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

func (m *archHelper) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for %s", m.name)
}

func (m *archHelper) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := m.requireUser(); err != nil {
		return err
	}

	var args []string
	if pkg != "" {
		if err := ValidateSpecForManager(m.name, pkg); err != nil {
			return err
		}
		args = m.installArgs(pkg)
	} else {
		args = []string{"-Syu", pacmanFlagNoConfirm}
		args = append(args, m.flags...)
		args = append(args, m.upgradeFlags...)
	}
	if err := m.stream(ctx, args...); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	return nil
}

func (m *archHelper) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for %s", m.name)
}

func (m *archHelper) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for %s", m.name)
}

func (m *archHelper) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Provides delegates to pacman for AUR helpers; pacman uses its own binary.
func (m *archHelper) Provides(ctx context.Context, query string) ([]string, error) {
	bin := m.bin
	if !m.escalate {
		bin = "pacman"
	}
	out, err := m.exec(ctx, bin, "-F", query)
	if err != nil {
		return nil, fmt.Errorf("failed to find provides for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AutoRemove lists orphans and removes them with pacman. Removal always uses
// sudo pacman: AUR helpers cannot remove packages themselves.
func (m *archHelper) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	listBin := m.bin
	if !m.escalate {
		listBin = "pacman"
	}
	out, err := m.exec(ctx, listBin, "-Qdtq")
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
	args := append([]string{"pacman", "-Rs", pacmanFlagNoConfirm}, orphans...)
	return orphans, sudoExec(ctx, m.exec, sudoCmd(args...), "failed to remove orphans")
}

// Clean clears the manager's package cache. pacman needs sudo; helpers clean
// their user cache directly.
func (m *archHelper) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
		if err := m.requireUser(); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	if err := m.stream(ctx, "-Sc", pacmanFlagNoConfirm); err != nil {
		return nil, fmt.Errorf("failed to clean %s cache: %w", m.name, err)
	}
	return nil, nil
}

func (m *archHelper) Refresh(ctx context.Context) error {
	if err := m.requireUser(); err != nil {
		return err
	}
	if _, err := m.run(ctx, "-Sy"); err != nil {
		return fmt.Errorf("failed to sync databases: %w", err)
	}
	return nil
}

func (m *archHelper) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	args := []string{"-Qu"}
	if pkg != "" {
		if err := ValidateSpecForManager(m.name, pkg); err != nil {
			return nil, err
		}
		args = append(args, pkg)
	}
	out, err := m.run(ctx, args...)
	if err != nil {
		if exitCodeFromError(err) == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check updates: %w", err)
	}
	result := parsePacmanQu(out)
	if m.strictQu && len(result) == 0 && len(bytes.TrimSpace(out)) > 0 {
		return nil, fmt.Errorf("unrecognized pacman -Qu output (parser may be outdated)")
	}
	return result, nil
}

func (m *archHelper) Hold(ctx context.Context, pkg string) error {
	return pacmanHold(ctx, m.exec, pkg)
}

func (m *archHelper) Unhold(ctx context.Context, pkg string) error {
	return pacmanUnhold(ctx, m.exec, pkg)
}

func (m *archHelper) ListHeld(ctx context.Context) ([]string, error) {
	return pacmanIgnorePkg(ctx, m.exec)
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

const pacmanFlagNoConfirm = "--noconfirm"

var _ Adapter = (*archHelper)(nil)
