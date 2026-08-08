package manager

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// Npm implements the Adapter interface for npm (globally installed CLI tools).
type Npm struct {
	exec Executor
}

// NewNpm creates a new Npm adapter with the default system executor.
func NewNpm() *Npm {
	return &Npm{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Npm) Name() string {
	return "npm"
}

// ListInstalled returns a list of globally installed npm packages.
func (m *Npm) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "npm", "ls", "-g", "--depth=0")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}
	return parseNpmLs(out), nil
}

// parseNpmLs parses the output of npm ls -g --depth=0.
// Output format:
//
//	/usr/lib/node_modules
//	├── corepack@0.29.4
//	├── npm@10.8.2
//	└── typescript@5.6.3
func parseNpmLs(output []byte) []string {
	var result []string
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '/' || trimmed[0] == '(' {
			continue
		}
		stripped := bytes.TrimLeft(trimmed, " ├─│└")
		stripped = bytes.TrimSpace(stripped)
		if len(stripped) == 0 {
			continue
		}
		atIdx := bytes.LastIndexByte(stripped, '@')
		name := ""
		if atIdx > 0 {
			name = string(stripped[:atIdx])
		} else {
			name = string(stripped)
		}
		if name != "" && name != "npm" {
			result = append(result, name)
		}
	}
	return result
}

// Install runs npm install -g <pkg>.
func (m *Npm) Install(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	cmdArgs := sudoCmd("npm", "install", "-g", pkg)
	_, err := m.exec(WithStreamIO(ctx), cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall is the same as Install (npm install is idempotent).
func (m *Npm) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	return m.Install(ctx, pkg)
}

// Remove runs npm uninstall -g <pkg>.
func (m *Npm) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	cmdArgs := sudoCmd("npm", "uninstall", "-g", pkg)
	_, err := m.exec(WithStreamIO(ctx), cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// InstallMany installs multiple global packages in one npm invocation.
func (m *Npm) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	cmdArgs := batchArgs(sudoCmd("npm", "install", "-g"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// ReinstallMany is the same as InstallMany (npm install is idempotent).
func (m *Npm) ReinstallMany(ctx context.Context, pkgs ...string) error {
	return m.InstallMany(ctx, pkgs...)
}

// RemoveMany removes multiple global packages in one npm invocation.
func (m *Npm) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	cmdArgs := batchArgs(sudoCmd("npm", "uninstall", "-g"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
}

// PreviewInstall previews installing pkg.
// npm install --dry-run reports what would be installed without changes.
func (m *Npm) PreviewInstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, "npm", "install", "--dry-run", "-g", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview install %s: %w", pkg, err)
	}
	s := string(out)
	return Preview{Output: s, Noop: strings.Contains(s, "up to date")}, nil
}

// PreviewRemove previews removing pkg.
func (m *Npm) PreviewRemove(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	ctx = WithCombinedOutput(ctx)
	out, err := m.exec(ctx, "npm", "uninstall", "--dry-run", "-g", pkg)
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return Preview{}, fmt.Errorf("failed to preview remove %s: %w", pkg, err)
	}
	s := string(out)
	return Preview{Output: s, Noop: strings.Contains(s, "up to date")}, nil
}

// PreviewReinstall previews reinstalling pkg.
// npm reinstalls via the same install operation.
func (m *Npm) PreviewReinstall(ctx context.Context, pkg string) (Preview, error) {
	return m.PreviewInstall(ctx, pkg)
}

var _ Previewer = (*Npm)(nil)

// Search is not supported for npm.
func (m *Npm) Search(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("npm: search not supported")
}

// Info returns details about a package via the npm registry.
func (m *Npm) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "npm", "view", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since npm has no native diagnostic command.
func (m *Npm) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for npm")
}

// Update runs npm update -g for batch or a single package.
func (m *Npm) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	var args []string
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args = []string{"update", "-g", pkg}
	} else {
		args = []string{"update", "-g"}
	}
	cmdArgs := sudoCmd(append([]string{"npm"}, args...)...)
	_, err := m.exec(WithStreamIO(ctx), cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	return nil
}

// CheckUpdate returns an error since npm has no check-update command.
func (m *Npm) CheckUpdate(_ context.Context, _ string) ([]UpdateInfo, error) {
	return nil, fmt.Errorf("%w", ErrCheckUnsupported)
}

// Refresh is a no-op for npm.
func (m *Npm) Refresh(_ context.Context) error {
	return nil
}

// AddRepo returns an error since npm has no concept of repositories.
func (m *Npm) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for npm")
}

// RemoveRepo returns an error since npm has no concept of repositories.
func (m *Npm) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for npm")
}

// ListRepos returns an empty list since npm has no concept of repositories.
func (m *Npm) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Provides returns an error since npm has no provides command.
func (m *Npm) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for npm", ErrNotSupported)
}

// AutoRemove returns an error since npm has no autoremove command.
func (m *Npm) AutoRemove(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: autoremove not supported for npm", ErrNotSupported)
}

// Clean returns an error since npm has no cache clean command.
func (m *Npm) Clean(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: clean not supported for npm", ErrNotSupported)
}

// Hold returns an error since npm has no hold command.
func (m *Npm) Hold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: hold not supported for npm", ErrNotSupported)
}

// Unhold returns an error since npm has no unhold command.
func (m *Npm) Unhold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: unhold not supported for npm", ErrNotSupported)
}

// ListHeld returns an error since npm has no hold command.
func (m *Npm) ListHeld(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("%w: hold not supported for npm", ErrNotSupported)
}
