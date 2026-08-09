package manager

import (
	"context"
	"fmt"
	"strings"
)

// Paru implements the Adapter interface for Arch Linux's Paru (AUR helper).
type Paru struct {
	exec Executor
}

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
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("paru", "-S", "--noconfirm", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall reinstalls a package via paru.
func (m *Paru) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("paru", "-S", "--noconfirm", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove removes a package and its unneeded dependencies via paru.
func (m *Paru) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	// -Rs removes the package and its unneeded dependencies.
	args := sudoCmd("paru", "-Rs", "--noconfirm", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// InstallMany installs multiple packages in one paru invocation.
func (m *Paru) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("paru", "-S", "--noconfirm"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// ReinstallMany reinstalls multiple packages in one paru invocation.
func (m *Paru) ReinstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("paru", "-S", "--noconfirm"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall packages: %w", err)
	}
	return nil
}

// RemoveMany removes multiple packages in one paru invocation.
func (m *Paru) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("paru", "-Rs", "--noconfirm"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
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
		args = sudoCmd("paru", "-S", "--noconfirm", pkg)
	} else {
		args = sudoCmd("paru", "-Syu", "--noconfirm")
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
	args := sudoCmd("pacman", "-Rs", "--noconfirm")
	args = append(args, orphans...)
	_, err = m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to remove orphans: %w", err)
	}
	return orphans, nil
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
	args := sudoCmd("paru", "-Sc", "--noconfirm")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to clean paru cache: %w", err)
	}
	return nil, nil
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
	modified := false

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
			value := strings.TrimSpace(trimmed[eqIdx+1:])
			pkgs := strings.Fields(value)
			for _, p := range pkgs {
				if p == pkg {
					return nil
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
				lines = append(lines[:i+1], append([]string{"IgnorePkg = " + pkg}, lines[i+1:]...)...)
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
func (m *Paru) Unhold(ctx context.Context, pkg string) error {
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
func (m *Paru) ListHeld(ctx context.Context) ([]string, error) {
	return pacmanIgnorePkg(ctx, m.exec)
}

var _ Adapter = (*Paru)(nil)
