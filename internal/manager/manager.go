package manager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var validPkgNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_\-\.\+]*$`)

// ValidatePackageName ensures the package name is safe to pass to a shell command.
// It prevents arguments that start with '-' and restricts characters to a safe set.
func ValidatePackageName(pkg string) error {
	if strings.HasPrefix(pkg, "-") {
		return fmt.Errorf("invalid package name %q: cannot start with '-'", pkg)
	}
	if !validPkgNameRegex.MatchString(pkg) {
		return fmt.Errorf("invalid package name %q: contains invalid characters", pkg)
	}
	return nil
}

var validModulePathRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-\.\+/]*$`)

// ValidateModulePath ensures the module path is safe for go install.
// Allows full module paths (e.g., github.com/example/tool) but blocks
// shell metacharacters via the regex character class.
func ValidateModulePath(pkg string) error {
	if len(pkg) == 0 {
		return fmt.Errorf("invalid module path: cannot be empty")
	}
	if !strings.Contains(pkg, "/") {
		return fmt.Errorf("go install requires a full module path (e.g., github.com/example/tool)")
	}
	if !validModulePathRegex.MatchString(pkg) {
		return fmt.Errorf("invalid module path %q: contains invalid characters", pkg)
	}
	return nil
}

// ValidatePackageForManager dispatches to the correct validation function
// for the given manager name. Go adapters use ValidateModulePath; all
// others use ValidatePackageName.
func ValidatePackageForManager(managerName, pkg string) error {
	if managerName == "go" {
		return ValidateModulePath(pkg)
	}
	return ValidatePackageName(pkg)
}

// RepositoryInfo holds the name and URL of a third-party repository.
type RepositoryInfo struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// UpdateInfo holds the package name and version details for available updates.
type UpdateInfo struct {
	Package          string `json:"package"`
	CurrentVersion   string `json:"current_version,omitempty"`
	AvailableVersion string `json:"available_version,omitempty"`
}

// ErrCheckUnsupported is returned by CheckUpdate when the adapter has no native
// "check for updates" command (e.g. pipx, uv, go).
var ErrCheckUnsupported = errors.New("check not supported")

// ErrNotSupported is returned by adapter methods that are not supported
// by a particular package manager (e.g. provides, autoremove).
var ErrNotSupported = errors.New("not supported")

// exitCodeFromError attempts to extract a process exit code from an error.
// Returns -1 if the error is not an *exec.ExitError or has no process state.
func exitCodeFromError(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// Adapter abstracts operations for different underlying package managers
// like dnf, brew, and flatpak.
type Adapter interface {
	// Name returns the identifier of the package manager (e.g., "dnf", "brew").
	Name() string

	// ListInstalled returns a list of packages currently installed by this manager.
	// For MVP, this just returns the package names.
	ListInstalled(ctx context.Context) ([]string, error)

	// ListRepos returns a list of third-party repositories or taps currently configured.
	ListRepos(ctx context.Context) ([]RepositoryInfo, error)

	// Install executes the native installation command for the given package.
	Install(ctx context.Context, pkg string) error

	// Reinstall executes the native reinstallation command for the given package.
	Reinstall(ctx context.Context, pkg string) error

	// Remove executes the native removal command for the given package.
	Remove(ctx context.Context, pkg string) error

	// Search queries the native package manager for the given package name.
	Search(ctx context.Context, query string) ([]string, error)

	// AddRepo adds a third-party repository or tap.
	AddRepo(ctx context.Context, name, url string) error

	// RemoveRepo removes a tracked repository.
	RemoveRepo(ctx context.Context, name string) error

	// Info queries the native package manager for details on the given package.
	Info(ctx context.Context, pkg string) (string, error)

	// Doctor runs the native diagnostic command for the package manager.
	Doctor(ctx context.Context) (string, error)

	// Update runs the native system upgrade command for this package manager.
	// If pkg is non-empty, updates only that package instead of all packages.
	Update(ctx context.Context, pkg string) error

	// CheckUpdate returns a list of available updates for this manager.
	// If pkg is non-empty, scopes the check to that package.
	// Returns err = nil + empty slice if up to date.
	// Returns ErrCheckUnsupported if the manager has no native check command.
	CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error)

	// Provides searches for which package provides the given file.
	// Returns the raw output lines from the native command.
	// Returns ErrNotSupported if the manager has no provides command.
	Provides(ctx context.Context, query string) ([]string, error)

	// AutoRemove removes orphaned/unused dependencies.
	// If dryRun is true, returns the list of packages that would be removed
	// without actually removing them.
	// Returns ErrNotSupported if the manager has no autoremove command.
	AutoRemove(ctx context.Context, dryRun bool) ([]string, error)

	// Clean removes locally cached package files (e.g. download cache).
	// If dryRun is true, returns what would be cleaned without deleting.
	// Returns ErrNotSupported if the manager has no cache clean command.
	Clean(ctx context.Context, dryRun bool) ([]string, error)
}

func init() {
	managerAliases = map[string]string{
		"yum": "dnf",
	}
}

var managerAliases map[string]string

// ResolveManager resolves the manager name, including aliases (e.g. "yum" → "dnf").
func ResolveManager(name string) string {
	if resolved, ok := managerAliases[name]; ok {
		return resolved
	}
	return name
}

// parseLines splits byte output by newline and removes empty strings.
func parseLines(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			result = append(result, string(trimmed))
		}
	}
	return result
}
