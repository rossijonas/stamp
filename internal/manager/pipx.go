package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Pipx implements the Adapter interface for pipx (pip-installable CLI tools).
type Pipx struct {
	exec Executor
}

// NewPipx creates a new Pipx adapter with the default system executor.
func NewPipx() *Pipx {
	return &Pipx{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Pipx) Name() string {
	return "pipx"
}

// ListInstalled returns a list of packages installed via pipx.
func (m *Pipx) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "pipx", "list", "--json")
	if err == nil {
		return parsePipxJSON(out)
	}
	out, err = m.exec(ctx, "pipx", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parsePipxText(out), nil
}

// pipxJSON represents the top-level structure of pipx list --json.
type pipxJSON struct {
	Venvs map[string]json.RawMessage `json:"venvs"`
}

func parsePipxJSON(output []byte) ([]string, error) {
	var data pipxJSON
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}
	var result []string
	for name := range data.Venvs {
		result = append(result, name)
	}
	return result, nil
}

func parsePipxText(output []byte) []string {
	var result []string
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := string(bytes.TrimSpace(line))
		if strings.HasPrefix(trimmed, "package ") {
			parts := strings.SplitN(trimmed, " ", 3)
			if len(parts) >= 2 {
				result = append(result, parts[1])
			}
		}
	}
	return result
}

// Install executes the native installation command.
func (m *Pipx) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "pipx", "install", "--yes", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall re-executes the native installation command with force.
func (m *Pipx) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "pipx", "install", "--yes", "--force", pkg)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *Pipx) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "pipx", "uninstall", "--yes", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search is not supported for pipx.
func (m *Pipx) Search(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("pipx: search not supported (no package registry CLI)")
}

// Info returns details about an installed pipx package.
func (m *Pipx) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "pipx", "list", "--json")
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return filterPipxInfo(out, pkg)
}

func filterPipxInfo(output []byte, pkg string) (string, error) {
	var data pipxJSON
	if err := json.Unmarshal(output, &data); err != nil {
		return "", err
	}
	entry, ok := data.Venvs[pkg]
	if !ok {
		return "", fmt.Errorf("%s not found", pkg)
	}
	return string(entry), nil
}

// Doctor returns an error since pipx has no native diagnostic command.
func (m *Pipx) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for pipx")
}

// Update runs pipx upgrade for a single package or upgrade-all for batch.
func (m *Pipx) Update(ctx context.Context, pkg string) error {
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		_, err := m.exec(WithStreamIO(ctx), "pipx", "upgrade", pkg)
		if err != nil {
			return fmt.Errorf("failed to upgrade %s: %w", pkg, err)
		}
		return nil
	}
	_, err := m.exec(WithStreamIO(ctx), "pipx", "upgrade-all")
	if err != nil {
		return fmt.Errorf("failed to upgrade all packages: %w", err)
	}
	return nil
}

// CheckUpdate returns an error since pipx has no check-update command.
func (m *Pipx) CheckUpdate(_ context.Context, _ string) ([]UpdateInfo, error) {
	return nil, fmt.Errorf("%w", ErrCheckUnsupported)
}

// Refresh is a no-op for pipx.
func (m *Pipx) Refresh(_ context.Context) error {
	return nil
}

// AddRepo returns an error since pipx has no concept of repositories.
func (m *Pipx) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for pipx")
}

// RemoveRepo returns an error since pipx has no concept of repositories.
func (m *Pipx) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for pipx")
}

// ListRepos returns an empty list since pipx has no concept of repositories.
func (m *Pipx) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Provides returns an error since pipx has no provides command.
func (m *Pipx) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for pipx", ErrNotSupported)
}

// AutoRemove returns an error since pipx has no autoremove command.
func (m *Pipx) AutoRemove(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: autoremove not supported for pipx", ErrNotSupported)
}

// Clean returns an error since pipx has no cache clean command.
func (m *Pipx) Clean(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: clean not supported for pipx", ErrNotSupported)
}

// Hold returns an error since pipx has no hold command.
func (m *Pipx) Hold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: hold not supported for pipx", ErrNotSupported)
}

// Unhold returns an error since pipx has no unhold command.
func (m *Pipx) Unhold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: unhold not supported for pipx", ErrNotSupported)
}

// ListHeld returns an error since pipx has no hold command.
func (m *Pipx) ListHeld(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("%w: hold not supported for pipx", ErrNotSupported)
}
