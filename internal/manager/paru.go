package manager

import (
	"context"
	"fmt"
)

// Paru implements the Adapter interface for Arch Linux's Paru (AUR helper).
type Paru struct {
	exec Executor
}

const paruFlagNoConfirm = "--noconfirm"

// NewParu creates a new Paru adapter.
func NewParu() *Paru {
	return &Paru{
		exec: defaultExecutor,
	}
}

// Name returns "paru".
func (m *Paru) Name() string {
	return "paru"
}

// ReconcileReliability reports OverInclusive: `paru -Qq` returns all installed
// packages including dependencies and AUR deps. Consistent run-to-run.
func (m *Paru) ReconcileReliability() ReconcileReliability {
	return ReliabilityOverInclusive
}

// ListInstalled returns a list of installed packages via paru.
func (m *Paru) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "paru", "-Qq")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseLines(out), nil
}

// Install installs a package via paru.
func (m *Paru) Install(ctx context.Context, pkg string) error {
	return runSingle(ctx, m.exec, sudoCmd("paru", "-S", paruFlagNoConfirm, pkg), "install", pkg)
}

// Reinstall reinstalls a package via paru.
func (m *Paru) Reinstall(ctx context.Context, pkg string) error {
	return runSingle(ctx, m.exec, sudoCmd("paru", "-S", paruFlagNoConfirm, pkg), "reinstall", pkg)
}

// Remove removes a package and its unneeded dependencies via paru.
func (m *Paru) Remove(ctx context.Context, pkg string) error {
	return runSingle(ctx, m.exec, sudoCmd("paru", "-Rs", paruFlagNoConfirm, pkg), "remove", pkg)
}

// InstallMany installs multiple packages in one paru invocation.
func (m *Paru) InstallMany(ctx context.Context, pkgs ...string) error {
	return runBatch(ctx, m.exec, batchArgs(sudoCmd("paru", "-S", paruFlagNoConfirm), pkgs), "install", pkgs)
}

// ReinstallMany reinstalls multiple packages in one paru invocation.
func (m *Paru) ReinstallMany(ctx context.Context, pkgs ...string) error {
	return runBatch(ctx, m.exec, batchArgs(sudoCmd("paru", "-S", paruFlagNoConfirm), pkgs), "reinstall", pkgs)
}

// RemoveMany removes multiple packages in one paru invocation.
func (m *Paru) RemoveMany(ctx context.Context, pkgs ...string) error {
	return runBatch(ctx, m.exec, batchArgs(sudoCmd("paru", "-Rs", paruFlagNoConfirm), pkgs), "remove", pkgs)
}

// Search searches for packages via paru (official repos + AUR).
func (m *Paru) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "paru", "-Ss", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parsePacmanSearch(out), nil
}

// Info returns details about a package.
func (m *Paru) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "paru", "-Qi", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since paru has no native diagnostic command.
func (m *Paru) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for paru")
}

// Update syncs and upgrades all packages via paru (official + AUR).
func (m *Paru) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	var args []string
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args = sudoCmd("paru", "-S", paruFlagNoConfirm, pkg)
	} else {
		args = sudoCmd("paru", "-Syu", paruFlagNoConfirm)
	}
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	return nil
}

// AddRepo returns an error since repo management is not supported for paru.
func (m *Paru) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for paru")
}

// RemoveRepo returns an error since repo management is not supported for paru.
func (m *Paru) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for paru")
}

// ListRepos returns an empty list since paru has no concept of repositories.
func (m *Paru) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Provides runs pacman -F <query> (via paru) to find which package owns a file.
func (m *Paru) Provides(ctx context.Context, query string) ([]string, error) {
	out, err := m.exec(ctx, "pacman", "-F", query)
	if err != nil {
		return nil, fmt.Errorf("failed to find provides for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// AutoRemove lists orphans and removes them via pacman.
func (m *Paru) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
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
	args := sudoCmd("pacman", "-Rs", paruFlagNoConfirm)
	args = append(args, orphans...)
	return orphans, sudoExec(ctx, m.exec, args, "failed to remove orphans")
}

// Clean runs paru -Sc to clear the package cache.
func (m *Paru) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	return nil, sudoExec(ctx, m.exec, sudoCmd("paru", "-Sc", paruFlagNoConfirm), "failed to clean paru cache")
}

// Refresh syncs paru databases via paru -Sy.
func (m *Paru) Refresh(ctx context.Context) error {
	args := sudoCmd("paru", "-Sy")
	_, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to sync databases: %w", err)
	}
	return nil
}

// CheckUpdate runs paru -Qu to list available updates.
// paru -Qu exits 1 when no updates are available (success path).
func (m *Paru) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	args := []string{"paru", "-Qu"}
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
	return parsePacmanQu(out), nil
}

// Hold adds a package to IgnorePkg in pacman.conf (shared with pacman).
func (m *Paru) Hold(ctx context.Context, pkg string) error {
	return pacmanHold(ctx, m.exec, pkg)
}

// Unhold removes a package from IgnorePkg in pacman.conf.
func (m *Paru) Unhold(ctx context.Context, pkg string) error {
	return pacmanUnhold(ctx, m.exec, pkg)
}

// ListHeld returns the list of packages in IgnorePkg from pacman.conf.
func (m *Paru) ListHeld(ctx context.Context) ([]string, error) {
	return pacmanIgnorePkg(ctx, m.exec)
}

var _ Adapter = (*Paru)(nil)
