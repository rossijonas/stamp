package manager

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
)

// Flatpak implements the Adapter interface for Flatpak.
type Flatpak struct {
	exec Executor
}

// NewFlatpak creates a new Flatpak with the default system executor.
func NewFlatpak() *Flatpak {
	return &Flatpak{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Flatpak) Name() string {
	return "flatpak"
}

// ListInstalled returns a list of packages currently installed.
func (m *Flatpak) ListInstalled(ctx context.Context) ([]string, error) {
	// Query both user and system installations, then merge and deduplicate.
	seen := make(map[string]struct{})

	userOut, userErr := m.exec(ctx, "flatpak", "list", "--user", "--app", "--columns=application")
	if userErr == nil {
		for _, pkg := range parseLines(userOut) {
			if pkg == "Application ID" {
				continue
			}
			seen[pkg] = struct{}{}
		}
	}

	sysOut, sysErr := m.exec(ctx, "flatpak", "list", "--system", "--app", "--columns=application")
	if sysErr == nil {
		for _, pkg := range parseLines(sysOut) {
			if pkg == "Application ID" {
				continue
			}
			seen[pkg] = struct{}{}
		}
	}

	if userErr != nil && sysErr != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", userErr)
	}

	result := slices.Collect(maps.Keys(seen))
	return result, nil
}

// Install executes the native installation command.
func (m *Flatpak) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	// -y auto-answers yes to prompts.
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall executes the native reinstallation command.
func (m *Flatpak) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *Flatpak) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "uninstall", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search queries the native package manager for the given package name.
func (m *Flatpak) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	// Search and return application IDs
	out, err := m.exec(ctx, "flatpak", "search", "--columns=application", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// ListRepos returns a list of remotes currently configured.
func (m *Flatpak) ListRepos(ctx context.Context) ([]RepositoryInfo, error) {
	out, err := m.exec(ctx, "flatpak", "remotes", "--columns=name,url")
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}
	return parseFlatpakRemotes(out), nil
}

// parseFlatpakRemotes parses the output of 'flatpak remotes --columns=name,url'.
// Output is tab-separated with a header line "Name\tURL".
func parseFlatpakRemotes(output []byte) []RepositoryInfo {
	lines := bytes.Split(output, []byte("\n"))
	var repos []RepositoryInfo
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		// Skip header
		if bytes.Equal(bytes.ToLower(trimmed), []byte("name\turl")) ||
			bytes.Equal(bytes.TrimSpace(bytes.ToLower(trimmed)), []byte("name")) {
			continue
		}
		parts := bytes.SplitN(trimmed, []byte("\t"), 2)
		name := string(bytes.TrimSpace(parts[0]))
		if name == "" {
			continue
		}
		info := RepositoryInfo{Name: name}
		if len(parts) > 1 {
			url := string(bytes.TrimSpace(parts[1]))
			if url != "" && url != "(unset)" {
				info.URL = url
			}
		}
		repos = append(repos, info)
	}
	return repos
}

// AddRepo enables a third-party remote.
func (m *Flatpak) AddRepo(ctx context.Context, name, url string) error {
	if url == "" {
		return fmt.Errorf("flatpak requires a url to add remote %s", name)
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "remote-add", "--if-not-exists", name, url)
	if err != nil {
		return fmt.Errorf("failed to add remote %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party remote.
func (m *Flatpak) RemoveRepo(ctx context.Context, name string) error {
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "remote-delete", name)
	if err != nil {
		return fmt.Errorf("failed to remove remote %s: %w", name, err)
	}
	return nil
}

// Info queries flatpak info metadata.
func (m *Flatpak) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "flatpak", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since flatpak has no native diagnostic command.
func (m *Flatpak) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for flatpak")
}

// Update runs the native flatpak update command.
func (m *Flatpak) Update(ctx context.Context) error {
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "update", "-y")
	if err != nil {
		return fmt.Errorf("failed to update flatpak: %w", err)
	}
	return nil
}
