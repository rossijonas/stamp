package manager

import "context"

// PackageManager abstracts operations for different underlying package managers
// like dnf, brew, and flatpak.
type PackageManager interface {
	// Name returns the identifier of the package manager (e.g., "dnf", "brew").
	Name() string

	// ListInstalled returns a list of packages currently installed by this manager.
	// For MVP, this just returns the package names.
	ListInstalled(ctx context.Context) ([]string, error)

	// Install executes the native installation command for the given package.
	Install(ctx context.Context, pkg string) error

	// Remove executes the native removal command for the given package.
	Remove(ctx context.Context, pkg string) error

	// Search queries the native package manager for the given package name.
	Search(ctx context.Context, query string) ([]string, error)
}
