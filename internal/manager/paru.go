package manager

import (
	"context"
	"fmt"
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

// CheckUpdate runs paru -Qu to list available updates.
// paru -Qu exits 1 when no updates are available (success path).
func (m *Paru) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	syncArgs := sudoCmd("paru", "-Sy")
	_, err := m.exec(ctx, syncArgs[0], syncArgs[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to sync databases: %w", err)
	}
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

var _ Adapter = (*Paru)(nil)
