package manager

import (
	"bytes"
	"context"
	"fmt"
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

// RepositoryInfo holds the name and URL of a third-party repository.
type RepositoryInfo struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
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
