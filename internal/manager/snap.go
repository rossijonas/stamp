// Package manager provides adapters for various package managers.
package manager

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// Snap implements the Adapter interface for the snap package manager.
type Snap struct {
	exec Executor
}

// NewSnap creates a new Snap adapter.
func NewSnap() *Snap {
	return &Snap{
		exec: defaultExecutor,
	}
}

// Name returns "snap".
func (m *Snap) Name() string {
	return "snap"
}

// parseSnapTabular extracts the first column from snap tabular output.
func parseSnapTabular(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if bytes.HasPrefix(trimmed, []byte("Name")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		name := string(fields[0])
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	return result
}

// ListInstalled returns a list of installed snap packages.
func (m *Snap) ListInstalled(ctx context.Context) ([]string, error) {
	out, err := m.exec(ctx, "snap", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed snaps: %w", err)
	}
	return parseSnapTabular(out), nil
}

// Install installs a snap package.
func (m *Snap) Install(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("snap", "install", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall reinstalls a snap package via remove + install.
func (m *Snap) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	removeArgs := sudoCmd("snap", "remove", pkg)
	_, err := m.exec(WithStreamIO(ctx), removeArgs[0], removeArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: remove step failed: %w", pkg, err)
	}
	installArgs := sudoCmd("snap", "install", pkg)
	_, err = m.exec(WithStreamIO(ctx), installArgs[0], installArgs[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: install step failed: %w", pkg, err)
	}
	return nil
}

// Remove removes a snap package.
func (m *Snap) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd("snap", "remove", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// InstallMany installs multiple snaps in one snap invocation.
func (m *Snap) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("snap", "install"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// RemoveMany removes multiple snaps in one snap invocation. Note: snap has no
// BatchReinstaller — reinstall is remove + install, with no native batch form.
func (m *Snap) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := batchArgs(sudoCmd("snap", "remove"), pkgs)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
}

// Search searches for snap packages matching the query.
func (m *Snap) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "snap", "find", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseSnapTabular(out), nil
}

// Info returns details about a snap package.
func (m *Snap) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "snap", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since snap has no native diagnostic command.
func (m *Snap) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for snap")
}

// Update refreshes all installed snaps.
func (m *Snap) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	args := sudoCmd("snap", "refresh")
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args = append(args, pkg)
	}
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update snaps: %w", err)
	}
	return nil
}

// AddRepo returns an error since snap has no concept of repositories.
func (m *Snap) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for snap")
}

// RemoveRepo returns an error since snap has no concept of repositories.
func (m *Snap) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for snap")
}

// ListRepos returns an empty list since snap has no concept of repositories.
func (m *Snap) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// CheckUpdate runs snap refresh --list to show available updates.
func (m *Snap) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	args := []string{"snap", "refresh", "--list"}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return nil, err
		}
		args = append(args, pkg)
	}
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to check updates: %w", err)
	}
	return parseSnapRefreshList(out), nil
}

// Refresh is a no-op for this manager.
func (m *Snap) Refresh(_ context.Context) error {
	return nil
}

func parseSnapRefreshList(output []byte) []UpdateInfo {
	var result []UpdateInfo
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("Name")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		name := string(fields[0])
		result = append(result, UpdateInfo{Package: name})
	}
	return result
}

// Provides returns an error since snap has no provides command.
func (m *Snap) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for snap", ErrNotSupported)
}

// AutoRemove returns an error since snap has no autoremove command.
func (m *Snap) AutoRemove(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: autoremove not supported for snap", ErrNotSupported)
}

// parseSnapRevisions extracts active (name→rev) mapping from snap list output.
func parseSnapRevisions(output []byte) map[string]string {
	result := make(map[string]string)
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("Name")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) < 3 {
			continue
		}
		name := string(fields[0])
		rev := string(fields[2])
		result[name] = rev
	}
	return result
}

// Clean removes old snap revisions to free disk space.
// Keeps active revisions; removes all inactive (old) revisions.
func (m *Snap) Clean(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	activeOut, err := m.exec(ctx, "snap", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list active snaps: %w", err)
	}
	activeRevs := parseSnapRevisions(activeOut)

	allOut, err := m.exec(ctx, "snap", "list", "--all")
	if err != nil {
		return nil, fmt.Errorf("failed to list all snap revisions: %w", err)
	}
	allLines := parseLines(allOut)

	var removed []string
	for _, line := range allLines {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] == "Name" {
			continue
		}
		name := fields[0]
		rev := fields[2]
		if activeRevs[name] == rev {
			continue
		}
		removed = append(removed, fmt.Sprintf("%s rev %s", name, rev))
	}

	if dryRun {
		return removed, nil
	}

	for _, line := range allLines {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] == "Name" {
			continue
		}
		name := fields[0]
		rev := fields[2]
		if activeRevs[name] == rev {
			continue
		}
		args := sudoCmd("snap", "remove", name, "--revision", rev)
		if _, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...); err != nil {
			return nil, fmt.Errorf("failed to remove snap %s revision %s: %w", name, rev, err)
		}
	}
	return removed, nil
}

// Hold returns an error since snap has no hold command.
func (m *Snap) Hold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: hold not supported for snap", ErrNotSupported)
}

// Unhold returns an error since snap has no unhold command.
func (m *Snap) Unhold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: unhold not supported for snap", ErrNotSupported)
}

// ListHeld returns an error since snap has no hold command.
func (m *Snap) ListHeld(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("%w: hold not supported for snap", ErrNotSupported)
}

// Compile-time interface check.
var _ Adapter = (*Snap)(nil)
