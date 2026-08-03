package manager

import (
	"context"
	"fmt"
	"strings"
)

// UvTool implements the Adapter interface for uv (tool subcommand).
type UvTool struct {
	exec Executor
}

// NewUv creates a new UvTool adapter with the default system executor.
func NewUv() *UvTool {
	return &UvTool{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *UvTool) Name() string {
	return "uv"
}

// ListInstalled returns a list of packages installed via uv tool.
func (m *UvTool) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "uv", "tool", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseUvToolList(out), nil
}

// parseUvToolList parses the output of uv tool list.
// Output format:
//
//	black v24.8.0
//	- black
//	ruff v0.6.1
//	- ruff
func parseUvToolList(output []byte) []string {
	var result []string
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "-") {
			continue
		}
		parts := strings.Fields(trimmed)
		if len(parts) > 0 {
			result = append(result, parts[0])
		}
	}
	return result
}

// Install runs uv tool install <pkg>.
func (m *UvTool) Install(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "uv", "tool", "install", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall runs uv tool install --force <pkg>.
func (m *UvTool) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "uv", "tool", "install", "--force", pkg)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove runs uv tool uninstall <pkg>.
func (m *UvTool) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "uv", "tool", "uninstall", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search is not supported for uv.
func (m *UvTool) Search(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("uv: search not supported (no package registry CLI)")
}

// Info returns details about an installed uv tool package.
func (m *UvTool) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "uv", "tool", "list")
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return filterUvToolInfo(out, pkg)
}

func filterUvToolInfo(output []byte, pkg string) (string, error) {
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, pkg+" ") || strings.HasPrefix(trimmed, pkg+"\t") {
			return line, nil
		}
	}
	return "", fmt.Errorf("%s not found", pkg)
}

// Doctor returns an error since uv has no native diagnostic command.
func (m *UvTool) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for uv")
}

// Update runs uv tool upgrade for a single package or --all for batch.
func (m *UvTool) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		_, err := m.exec(WithStreamIO(ctx), "uv", "tool", "upgrade", pkg)
		if err != nil {
			return fmt.Errorf("failed to upgrade %s: %w", pkg, err)
		}
		return nil
	}
	_, err := m.exec(WithStreamIO(ctx), "uv", "tool", "upgrade", "--all")
	if err != nil {
		return fmt.Errorf("failed to upgrade all packages: %w", err)
	}
	return nil
}

// CheckUpdate returns an error since uv has no check-update command.
func (m *UvTool) CheckUpdate(_ context.Context, _ string) ([]UpdateInfo, error) {
	return nil, fmt.Errorf("%w", ErrCheckUnsupported)
}

// Refresh is a no-op for uv.
func (m *UvTool) Refresh(_ context.Context) error {
	return nil
}

// AddRepo returns an error since uv has no concept of repositories.
func (m *UvTool) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for uv")
}

// RemoveRepo returns an error since uv has no concept of repositories.
func (m *UvTool) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for uv")
}

// ListRepos returns an empty list since uv has no concept of repositories.
func (m *UvTool) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Provides returns an error since uv has no provides command.
func (m *UvTool) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for uv", ErrNotSupported)
}

// AutoRemove returns an error since uv has no autoremove command.
func (m *UvTool) AutoRemove(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: autoremove not supported for uv", ErrNotSupported)
}

// Clean returns an error since uv has no cache clean command.
func (m *UvTool) Clean(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: clean not supported for uv", ErrNotSupported)
}

// Hold returns an error since uv has no hold command.
func (m *UvTool) Hold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: hold not supported for uv", ErrNotSupported)
}

// Unhold returns an error since uv has no unhold command.
func (m *UvTool) Unhold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: unhold not supported for uv", ErrNotSupported)
}

// ListHeld returns an error since uv has no hold command.
func (m *UvTool) ListHeld(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("%w: hold not supported for uv", ErrNotSupported)
}
