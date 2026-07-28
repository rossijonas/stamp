// Package manager implements the adapters for the various package managers
// supported by stamp (e.g., dnf, brew, flatpak).
package manager

import (
	"context"
	"encoding/json"
	"fmt"
)

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

// ListInstalled returns a list of packages currently installed.
func (m *Brew) ListInstalled(ctx context.Context) ([]string, error) {
	// 'brew leaves --installed-on-request' returns packages the user explicitly installed.
	out, err := m.exec(ctx, "brew", "leaves", "--installed-on-request")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseLines(out), nil
}

// Install executes the native installation command.
func (m *Brew) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "brew", "install", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall executes the native reinstallation command.
func (m *Brew) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "brew", "reinstall", pkg)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *Brew) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "brew", "uninstall", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

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
// If pkg is non-empty, upgrades only that package via brew upgrade <pkg>.
func (m *Brew) Update(ctx context.Context, pkg string) error {
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		_, err := m.exec(WithStreamIO(ctx), "brew", "upgrade", pkg)
		if err != nil {
			return fmt.Errorf("failed to upgrade %s: %w", pkg, err)
		}
		return nil
	}

	_, err := m.exec(WithStreamIO(ctx), "brew", "update")
	if err != nil {
		return fmt.Errorf("failed to update homebrew: %w", err)
	}
	_, err = m.exec(WithStreamIO(ctx), "brew", "upgrade")
	if err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}
	return nil
}

type brewOutdatedJSON struct {
	Formulae []brewFormula `json:"formulae"`
}

type brewFormula struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

// CheckUpdate runs brew outdated --json to list available updates.
func (m *Brew) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	if pkg == "" {
		_, err := m.exec(ctx, "brew", "update")
		if err != nil {
			return nil, fmt.Errorf("failed to update homebrew: %w", err)
		}
	}
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
