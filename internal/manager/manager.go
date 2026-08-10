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

// BatchInstaller is implemented by adapters whose native install command
// accepts multiple packages in one invocation (e.g. `dnf install a b`). The
// CLI type-asserts this interface for `stamp install <a> <b> -m <manager>`;
// adapters without it reject multi-package installs. See issue #185.
type BatchInstaller interface {
	InstallMany(ctx context.Context, pkgs ...string) error
}

// BatchRemover is the remove-side counterpart of BatchInstaller.
type BatchRemover interface {
	RemoveMany(ctx context.Context, pkgs ...string) error
}

// BatchReinstaller is the reinstall-side counterpart of BatchInstaller. Some
// adapters omit it even when they batch install/remove: snap has no native
// batch reinstall (reinstall = remove + install), so multi-package reinstall
// stays single-only there.
type BatchReinstaller interface {
	ReinstallMany(ctx context.Context, pkgs ...string) error
}

// validatePackages validates every package name in a batch before any native
// command runs, so a bad name aborts the whole operation upfront.
func validatePackages(pkgs []string) error {
	for _, p := range pkgs {
		if err := ValidatePackageName(p); err != nil {
			return err
		}
	}
	return nil
}

// batchArgs appends pkgs to a fixed arg prefix, preserving the caller's slice.
func batchArgs(args []string, pkgs []string) []string {
	out := make([]string, 0, len(args)+len(pkgs))
	out = append(out, args...)
	return append(out, pkgs...)
}

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

// ErrConfirmationRequired is returned by destructive adapter methods when the
// context carries no explicit operator consent (see WithYes). The CLI is the
// only layer that sets consent, so any direct adapter invocation without it
// fails closed instead of mutating the system.
var ErrConfirmationRequired = errors.New("confirmation required")

// exitCodeFromError attempts to extract a process exit code from an error.
// Returns -1 if the error is not an *exec.ExitError or has no process state.
func exitCodeFromError(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// ReconcileReliability describes how trustworthy a manager's ListInstalled
// output is for drift detection. Adapters implement ReliabilityReporter to
// annotate themselves; reconcile surfaces this to the user.
type ReconcileReliability int

const (
	// ReliabilityReliable means ListInstalled returns only user-installed packages
	// (e.g. brew leaves --installed-on-request, pipx/uv/go tool listings).
	ReliabilityReliable ReconcileReliability = iota
	// ReliabilityOverInclusive means ListInstalled returns ALL installed packages
	// including the base OS and dependencies. Output is consistent run-to-run,
	// so baseline diffing stays safe, but reconcile may list system packages.
	ReliabilityOverInclusive
	// ReliabilityKnownUnreliable means ListInstalled output is inconsistent between
	// runs, producing false-positive drift. Must be fixed, not just documented.
	ReliabilityKnownUnreliable
)

// ReliabilityReporter is implemented by adapters whose ListInstalled output
// needs a reliability annotation for reconcile. Optional — absent means
// ReliabilityReliable.
type ReliabilityReporter interface {
	ReconcileReliability() ReconcileReliability
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

	// Refresh syncs the package manager's metadata (e.g., apt update, pacman -Sy).
	// Called once before CheckUpdate to ensure fresh results.
	// Returns nil for managers that do not require explicit refresh.
	Refresh(ctx context.Context) error

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

	// Hold pins a package at its current version to prevent upgrades.
	// Returns ErrNotSupported if the manager has no hold command.
	Hold(ctx context.Context, pkg string) error

	// Unhold removes a version pin, allowing the package to be upgraded.
	// Returns ErrNotSupported if the manager has no unhold command.
	Unhold(ctx context.Context, pkg string) error

	// ListHeld returns the list of currently held/pinned packages.
	// Returns ErrNotSupported if the manager has no hold command.
	ListHeld(ctx context.Context) ([]string, error)
}

// Preview describes what a destructive operation would do, rendered by the
// adapter from its native dry-run. Interpretation is fully adapter-owned — the
// CLI renders Output verbatim and never parses vendor output.
type Preview struct {
	// Output is the verbatim combined stdout+stderr of the native dry-run,
	// including the transaction display a manager like dnf prints before an
	// assume-no abort.
	Output string
	// Noop reports that no transaction would occur (e.g. the package is
	// already installed and up to date). The CLI fails fast without prompting.
	Noop bool
}

// Previewer is an optional capability interface implemented by adapters that
// can render a native transaction preview (dry-run) before a destructive
// operation. Preview methods MUST NOT modify system state — they only display
// what a real operation would do.
type Previewer interface {
	// PreviewInstall previews installing pkg.
	PreviewInstall(ctx context.Context, pkg string) (Preview, error)
	// PreviewRemove previews removing pkg.
	PreviewRemove(ctx context.Context, pkg string) (Preview, error)
	// PreviewReinstall previews reinstalling pkg.
	PreviewReinstall(ctx context.Context, pkg string) (Preview, error)
}

// yesKey is the context key for operator consent (see WithYes).
type yesKey struct{}

// WithYes returns a context that marks explicit operator consent for a
// destructive operation. Only the CLI confirmation gate sets this, and only
// after the user confirmed or passed -y/--yes.
func WithYes(ctx context.Context) context.Context {
	return context.WithValue(ctx, yesKey{}, true)
}

// isYes reports whether the context carries explicit operator consent.
func isYes(ctx context.Context) bool {
	v, _ := ctx.Value(yesKey{}).(bool)
	return v
}

// requireConsent returns ErrConfirmationRequired when the context carries no
// explicit operator consent. Destructive adapter methods call it first.
func requireConsent(ctx context.Context) error {
	if !isYes(ctx) {
		return ErrConfirmationRequired
	}
	return nil
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
