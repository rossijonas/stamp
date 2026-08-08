// Package manager implements the adapters for the various package managers
// supported by stamp (e.g., dnf, brew, flatpak).
package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type caskKey struct{}

// WithCask returns a context that signals cask operations for brew.
func WithCask(ctx context.Context) context.Context {
	return context.WithValue(ctx, caskKey{}, true)
}

func isCask(ctx context.Context) bool {
	v, _ := ctx.Value(caskKey{}).(bool)
	return v
}

// Brew implements the Adapter interface for Homebrew.
type Brew struct {
	exec Executor
}

// NewBrew creates a new Brew with the default system executor.
func NewBrew() *Brew {
	return &Brew{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Brew) Name() string {
	return "brew"
}

// ListInstalled returns a list of packages currently installed (formulas + casks).
func (m *Brew) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "brew", "leaves", "--installed-on-request")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	formulas := parseLines(out)

	// Also list installed casks and merge
	caskOut, err := m.exec(ctx, "brew", "list", "--cask")
	if err != nil {
		// brew list --cask fails when no casks are installed — ignore
		return formulas, nil
	}
	casks := parseLines(caskOut)
	for _, c := range casks {
		if !slices.Contains(formulas, c) {
			formulas = append(formulas, c)
		}
	}
	return formulas, nil
}

// IsCask returns true if the given package is a Homebrew cask.
func (m *Brew) IsCask(ctx context.Context, pkg string) (bool, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return false, err
	}
	// brew info --cask succeeds only for casks
	_, err := m.exec(ctx, "brew", "info", "--cask", pkg)
	if err != nil {
		if strings.Contains(err.Error(), "No available Cask") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check cask status for %s: %w", pkg, err)
	}
	return true, nil
}

// Install executes the native installation command.
// Adds --cask when the context is marked as cask (via WithCask).
func (m *Brew) Install(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := []string{"install"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkg)
	_, err := m.exec(WithStreamIO(ctx), "brew", args...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall executes the native reinstallation command.
// Adds --cask when the context is marked as cask (via WithCask).
func (m *Brew) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := []string{"reinstall"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkg)
	_, err := m.exec(WithStreamIO(ctx), "brew", args...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
// Adds --cask when the context is marked as cask (via WithCask).
func (m *Brew) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := []string{"uninstall"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkg)
	_, err := m.exec(WithStreamIO(ctx), "brew", args...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// InstallMany installs multiple formulae/casks in one brew invocation. The
// cask flag is batch-wide; the CLI falls back to per-package installs when a
// batch mixes casks and formulae.
func (m *Brew) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := []string{"install"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkgs...)
	_, err := m.exec(WithStreamIO(ctx), "brew", args...)
	if err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// ReinstallMany reinstalls multiple formulae/casks in one brew invocation.
func (m *Brew) ReinstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := []string{"reinstall"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkgs...)
	_, err := m.exec(WithStreamIO(ctx), "brew", args...)
	if err != nil {
		return fmt.Errorf("failed to reinstall packages: %w", err)
	}
	return nil
}

// RemoveMany removes multiple formulae/casks in one brew invocation.
func (m *Brew) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := []string{"uninstall"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkgs...)
	_, err := m.exec(WithStreamIO(ctx), "brew", args...)
	if err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
}

// PreviewInstall previews installing pkg.
// brew install --dry-run prints what would be installed without changing
// anything and without requiring root.
func (m *Brew) PreviewInstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	args := []string{"install", "--dry-run"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkg)
	out, err := m.exec(ctx, "brew", args...)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview install %s: %w", pkg, err)
	}
	s := string(out)
	return Preview{Output: s, Noop: strings.Contains(s, "is already installed")}, nil
}

// PreviewRemove previews removing pkg.
func (m *Brew) PreviewRemove(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	args := []string{"uninstall", "--dry-run"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkg)
	out, err := m.exec(ctx, "brew", args...)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview remove %s: %w", pkg, err)
	}
	s := string(out)
	return Preview{Output: s, Noop: strings.Contains(s, "No such keg") || strings.Contains(s, "is not installed")}, nil
}

// PreviewReinstall previews reinstalling pkg.
func (m *Brew) PreviewReinstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	args := []string{"reinstall", "--dry-run"}
	if isCask(ctx) {
		args = append(args, "--cask")
	}
	args = append(args, pkg)
	out, err := m.exec(ctx, "brew", args...)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview reinstall %s: %w", pkg, err)
	}
	s := string(out)
	return Preview{Output: s, Noop: strings.Contains(s, "No such keg") || strings.Contains(s, "is not installed")}, nil
}

var _ Previewer = (*Brew)(nil)

// Search queries the native package manager for the given package name.
func (m *Brew) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	// 'brew search' can be slow, but is the standard way.
	out, err := m.exec(ctx, "brew", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// ListRepos returns a list of third-party taps currently configured.
func (m *Brew) ListRepos(ctx context.Context) ([]RepositoryInfo, error) {
	out, err := m.exec(ctx, "brew", "tap")
	if err != nil {
		return nil, fmt.Errorf("failed to list taps: %w", err)
	}
	names := parseLines(out)
	result := make([]RepositoryInfo, len(names))
	for i, name := range names {
		result[i] = RepositoryInfo{Name: name}
	}
	return result, nil
}

// AddRepo enables a third-party tap.
func (m *Brew) AddRepo(ctx context.Context, name, url string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	var err error
	if url != "" {
		_, err = m.exec(WithStreamIO(ctx), "brew", "tap", name, url)
	} else {
		_, err = m.exec(WithStreamIO(ctx), "brew", "tap", name)
	}
	if err != nil {
		return fmt.Errorf("failed to tap %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party tap.
func (m *Brew) RemoveRepo(ctx context.Context, name string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "brew", "untap", name)
	if err != nil {
		return fmt.Errorf("failed to untap %s: %w", name, err)
	}
	return nil
}

// Info queries brew info metadata.
func (m *Brew) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "brew", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor runs brew doctor diagnostic.
func (m *Brew) Doctor(ctx context.Context) (string, error) {
	out, err := m.exec(ctx, "brew", "doctor")
	if err != nil {
		return "", fmt.Errorf("brew doctor failed: %w", err)
	}
	return string(out), nil
}

// Update runs brew update then brew upgrade (two-phase).
// For batch, also runs brew upgrade --cask to update casks.
// If pkg is non-empty, upgrades only that package.
func (m *Brew) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args := []string{"upgrade", pkg}
		if isCask(ctx) {
			args = []string{"upgrade", "--cask", pkg}
		}
		_, err := m.exec(WithStreamIO(ctx), "brew", args...)
		if err != nil {
			return fmt.Errorf("failed to upgrade %s: %w", pkg, err)
		}
		return nil
	}

	_, err := m.exec(WithStreamIO(ctx), "brew", "update")
	if err != nil {
		return fmt.Errorf("failed to update homebrew: %w", err)
	}
	if _, err := m.exec(WithStreamIO(ctx), "brew", "upgrade"); err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}
	// Cask upgrade is best-effort — may fail on systems with no casks installed
	_, _ = m.exec(WithStreamIO(ctx), "brew", "upgrade", "--cask")
	return nil
}

// Hold returns an error since brew has no hold command.
func (m *Brew) Hold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: hold not supported for brew", ErrNotSupported)
}

// Unhold returns an error since brew has no unhold command.
func (m *Brew) Unhold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: unhold not supported for brew", ErrNotSupported)
}

// ListHeld returns an error since brew has no hold command.
func (m *Brew) ListHeld(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("%w: hold not supported for brew", ErrNotSupported)
}

type brewOutdatedJSON struct {
	Formulae []brewFormula `json:"formulae"`
}

type brewFormula struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

// Refresh updates homebrew's package metadata.
func (m *Brew) Refresh(ctx context.Context) error {
	_, err := m.exec(ctx, "brew", "update")
	if err != nil {
		return fmt.Errorf("failed to update homebrew: %w", err)
	}
	return nil
}

// CheckUpdate runs brew outdated --json to list available updates.
func (m *Brew) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	args := []string{"brew", "outdated", "--json"}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return nil, err
		}
		args = []string{"brew", "outdated", "--json", pkg}
	}
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to check updates: %w", err)
	}
	return parseBrewOutdatedJSON(out)
}

func parseBrewOutdatedJSON(output []byte) ([]UpdateInfo, error) {
	var data brewOutdatedJSON
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}
	var result []UpdateInfo
	for _, f := range data.Formulae {
		cur := ""
		if len(f.InstalledVersions) > 0 {
			cur = f.InstalledVersions[0]
		}
		result = append(result, UpdateInfo{Package: f.Name, CurrentVersion: cur, AvailableVersion: f.CurrentVersion})
	}
	return result, nil
}

// Provides returns an error since brew has no provides command.
func (m *Brew) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for brew", ErrNotSupported)
}

// AutoRemove removes orphaned formulae via brew autoremove.
func (m *Brew) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	_, err := m.exec(WithStreamIO(ctx), "brew", "autoremove")
	if err != nil {
		return nil, fmt.Errorf("failed to autoremove: %w", err)
	}
	return nil, nil
}

// Clean runs brew cleanup to remove old package versions.
func (m *Brew) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	args := []string{"cleanup"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	out, err := m.exec(ctx, "brew", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to clean brew cache: %w", err)
	}
	return parseLines(out), nil
}
