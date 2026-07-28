package manager

import (
	"bytes"
	"context"
	"fmt"
)

// Cargo implements the Adapter interface for cargo (Rust crate installer).
type Cargo struct {
	exec Executor
}

// NewCargo creates a new Cargo adapter with the default system executor.
func NewCargo() *Cargo {
	return &Cargo{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Cargo) Name() string {
	return "cargo"
}

// ListInstalled returns a list of crates installed via cargo install.
func (m *Cargo) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "cargo", "install", "--list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseCargoList(out), nil
}

// parseCargoList parses the output of cargo install --list.
// Output format:
//
//	bat v0.25.0:
//	    bat, batcat
//	ripgrep v14.1.0:
//	    rg
func parseCargoList(output []byte) []string {
	var result []string
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == ' ' || trimmed[0] == '\t' || bytes.HasPrefix(trimmed, []byte("    ")) {
			continue
		}
		// Header line: "name vX.Y.Z:" or "name X.Y.Z:"
		colonIdx := bytes.IndexByte(trimmed, ':')
		if colonIdx <= 0 {
			continue
		}
		before := bytes.TrimSpace(trimmed[:colonIdx])
		// Find the space before the version
		spaceIdx := bytes.LastIndexByte(before, ' ')
		if spaceIdx <= 0 {
			continue
		}
		name := string(before[:spaceIdx])
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

// Install runs cargo install <pkg>.
func (m *Cargo) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "cargo", "install", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall runs cargo install <pkg> --force to upgrade to the latest version.
func (m *Cargo) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "cargo", "install", "--force", pkg)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove runs cargo uninstall <pkg>.
func (m *Cargo) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "cargo", "uninstall", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search searches crates.io for packages via cargo search.
func (m *Cargo) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "cargo", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// Info returns details about a crate from the crates.io registry.
func (m *Cargo) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "cargo", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since cargo has no native diagnostic command.
func (m *Cargo) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for cargo")
}

// Update runs cargo install --force for batch or a single package.
func (m *Cargo) Update(ctx context.Context, pkg string) error {
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		_, err := m.exec(WithStreamIO(ctx), "cargo", "install", "--force", pkg)
		if err != nil {
			return fmt.Errorf("failed to update %s: %w", pkg, err)
		}
		return nil
	}

	// Batch: list installed, reinstall each with --force
	pkgs, err := m.ListInstalled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list installed packages for update: %w", err)
	}
	for _, name := range pkgs {
		if _, err := m.exec(WithStreamIO(ctx), "cargo", "install", "--force", name); err != nil {
			return fmt.Errorf("failed to update %s: %w", name, err)
		}
	}
	return nil
}

// CheckUpdate returns an error since cargo has no check-update command.
func (m *Cargo) CheckUpdate(_ context.Context, _ string) ([]UpdateInfo, error) {
	return nil, fmt.Errorf("%w", ErrCheckUnsupported)
}

// AddRepo returns an error since cargo has no concept of repositories.
func (m *Cargo) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for cargo")
}

// RemoveRepo returns an error since cargo has no concept of repositories.
func (m *Cargo) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for cargo")
}

// ListRepos returns an empty list since cargo has no concept of repositories.
func (m *Cargo) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Provides returns an error since cargo has no provides command.
func (m *Cargo) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for cargo", ErrNotSupported)
}

// AutoRemove returns an error since cargo has no autoremove command.
func (m *Cargo) AutoRemove(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: autoremove not supported for cargo", ErrNotSupported)
}

// Clean returns an error since cargo has no cache clean command.
func (m *Cargo) Clean(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: clean not supported for cargo", ErrNotSupported)
}
