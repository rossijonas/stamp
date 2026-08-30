package manager

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const flatpakColumnsApplication = "--columns=application"

// Flatpak implements the Adapter interface for Flatpak.
type Flatpak struct {
	exec Executor
}

// NewFlatpak creates a new Flatpak with the default system executor.
func NewFlatpak() *Flatpak {
	return &Flatpak{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Flatpak) Name() string {
	return "flatpak"
}

// ListInstalled returns a list of packages currently installed.
// App-kind extensions and plugins (e.g. OBS Studio plugins) are included;
// runtimes and runtime-kind extensions (locale, debug, GPU drivers) are
// excluded to avoid reconcile noise.
func (m *Flatpak) ListInstalled(ctx context.Context) ([]string, error) {
	all := make(map[string]struct{})
	apps := make(map[string]struct{})
	var firstErr error
	appsKnown := true
	anyScopeSucceeded := false

	for _, scope := range []string{"--user", "--system"} {
		out, err := m.exec(ctx, "flatpak", "list", scope, flatpakColumnsApplication)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		anyScopeSucceeded = true
		for _, id := range parseFlatpakApplicationIDs(out) {
			all[id] = struct{}{}
		}

		out, err = m.exec(ctx, "flatpak", "list", scope, "--app", flatpakColumnsApplication)
		if err != nil {
			appsKnown = false
			continue
		}
		for _, id := range parseFlatpakApplicationIDs(out) {
			apps[id] = struct{}{}
		}
	}

	if !anyScopeSucceeded && firstErr != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", firstErr)
	}

	// Without the --app listing app extensions cannot be isolated; fall back
	// to the full listing (over-inclusive) rather than dropping apps.
	if !appsKnown {
		return slices.Collect(maps.Keys(all)), nil
	}
	return classifyFlatpakRefs(all, apps), nil
}

// classifyFlatpakRefs keeps app-kind refs (pure apps from the --app listing
// plus app extensions) and drops runtimes and runtime-kind extensions.
// flatpak refs carry no kind prefix and its filters cannot isolate app
// extensions, so extensions are identified by their parent ref: extension IDs
// live in their parent's namespace (<parent-id>.<name>).
func classifyFlatpakRefs(all, apps map[string]struct{}) []string {
	result := make([]string, 0, len(all))
	for ref := range all {
		if _, ok := apps[ref]; ok {
			result = append(result, ref)
			continue
		}
		if isFlatpakAppExtension(ref, all, apps) {
			result = append(result, ref)
		}
	}
	return result
}

// isFlatpakAppExtension reports whether ref is an extension whose nearest
// installed parent ref is an app. A runtime parent (or no installed parent)
// marks the ref as runtime noise to be excluded.
func isFlatpakAppExtension(ref string, all, apps map[string]struct{}) bool {
	for {
		dot := strings.LastIndex(ref, ".")
		if dot < 0 {
			return false
		}
		ref = ref[:dot]
		if _, ok := all[ref]; ok {
			_, isApp := apps[ref]
			return isApp
		}
	}
}

// parseFlatpakApplicationIDs parses the output of 'flatpak list
// --columns=application'. Lines are application or runtime IDs with a header
// row "Application ID".
func parseFlatpakApplicationIDs(output []byte) []string {
	var result []string
	for _, line := range bytes.Split(output, []byte("\n")) {
		id := string(bytes.TrimSpace(line))
		if id == "" || id == "Application ID" {
			continue
		}
		result = append(result, id)
	}
	return result
}

// Install executes the native installation command.
func (m *Flatpak) Install(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	// -y auto-answers yes to prompts.
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall executes the native reinstallation command.
func (m *Flatpak) Reinstall(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *Flatpak) Remove(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "uninstall", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// InstallMany installs multiple apps in one flatpak invocation.
func (m *Flatpak) InstallMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := append([]string{"install", "-y"}, pkgs...)
	_, err := m.exec(WithStreamIO(ctx), "flatpak", args...)
	if err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// ReinstallMany is the same as InstallMany (flatpak reinstall ≡ install).
func (m *Flatpak) ReinstallMany(ctx context.Context, pkgs ...string) error {
	return m.InstallMany(ctx, pkgs...)
}

// RemoveMany removes multiple apps in one flatpak invocation.
func (m *Flatpak) RemoveMany(ctx context.Context, pkgs ...string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if err := validatePackages(pkgs); err != nil {
		return err
	}
	args := append([]string{"uninstall", "-y"}, pkgs...)
	_, err := m.exec(WithStreamIO(ctx), "flatpak", args...)
	if err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
}

// PreviewInstall previews installing pkg.
// flatpak has no --dry-run option for install, so installed state is queried
// via 'flatpak info' — a read-only query that never modifies system state.
func (m *Flatpak) PreviewInstall(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	_, err := m.exec(ctx, "flatpak", "info", pkg)
	if err == nil {
		return Preview{Output: "already installed via flatpak", Noop: true}, nil
	}
	return Preview{Output: "will install " + pkg + " via flatpak", Noop: false}, nil
}

// PreviewRemove previews removing pkg.
func (m *Flatpak) PreviewRemove(ctx context.Context, pkg string) (Preview, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return Preview{}, err
	}
	_, err := m.exec(ctx, "flatpak", "info", pkg)
	if err == nil {
		return Preview{Output: "will remove " + pkg + " via flatpak", Noop: false}, nil
	}
	return Preview{Output: "not installed via flatpak", Noop: true}, nil
}

// PreviewReinstall previews reinstalling pkg.
// flatpak reinstalls via the same install operation.
func (m *Flatpak) PreviewReinstall(ctx context.Context, pkg string) (Preview, error) {
	return m.PreviewInstall(ctx, pkg)
}

var _ Previewer = (*Flatpak)(nil)

// Search queries the native package manager for the given package name.
func (m *Flatpak) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	// Search and return application IDs
	out, err := m.exec(ctx, "flatpak", "search", flatpakColumnsApplication, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	return parseLines(out), nil
}

// ListRepos returns a list of remotes currently configured.
func (m *Flatpak) ListRepos(ctx context.Context) ([]RepositoryInfo, error) {
	out, err := m.exec(ctx, "flatpak", "remotes", "--columns=name,url")
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}
	return parseFlatpakRemotes(out), nil
}

// parseFlatpakRemotes parses the output of 'flatpak remotes --columns=name,url'.
// Output is tab-separated with a header line "Name\tURL".
func parseFlatpakRemotes(output []byte) []RepositoryInfo {
	lines := bytes.Split(output, []byte("\n"))
	var repos []RepositoryInfo
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		// Skip header
		if bytes.Equal(bytes.ToLower(trimmed), []byte("name\turl")) ||
			bytes.Equal(bytes.TrimSpace(bytes.ToLower(trimmed)), []byte("name")) {
			continue
		}
		parts := bytes.SplitN(trimmed, []byte("\t"), 2)
		name := string(bytes.TrimSpace(parts[0]))
		if name == "" {
			continue
		}
		info := RepositoryInfo{Name: name}
		if len(parts) > 1 {
			url := string(bytes.TrimSpace(parts[1]))
			if url != "" && url != "(unset)" {
				info.URL = url
			}
		}
		repos = append(repos, info)
	}
	return repos
}

// AddRepo enables a third-party remote.
func (m *Flatpak) AddRepo(ctx context.Context, name, url string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if url == "" {
		return fmt.Errorf("flatpak requires a url to add remote %s", name)
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "remote-add", "--if-not-exists", name, url)
	if err != nil {
		return fmt.Errorf("failed to add remote %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party remote.
func (m *Flatpak) RemoveRepo(ctx context.Context, name string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "remote-delete", name)
	if err != nil {
		return fmt.Errorf("failed to remove remote %s: %w", name, err)
	}
	return nil
}

// Info queries flatpak info metadata.
func (m *Flatpak) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	out, err := m.exec(ctx, "flatpak", "info", pkg)
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since flatpak has no native diagnostic command.
func (m *Flatpak) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for flatpak")
}

// Update runs the native flatpak update command.
func (m *Flatpak) Update(ctx context.Context, pkg string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	args := []string{"update", "-y"}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return err
		}
		args = append(args, pkg)
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", args...)
	if err != nil {
		return fmt.Errorf("failed to update flatpak: %w", err)
	}
	return nil
}

// CheckUpdate runs flatpak update --no-deploy to check available updates.
// Falls back to ErrCheckUnsupported if the flag isn't available (older flatpak).
func (m *Flatpak) CheckUpdate(ctx context.Context, pkg string) ([]UpdateInfo, error) {
	args := []string{"flatpak", "update", "--no-deploy", "-y"}
	if pkg != "" {
		if err := ValidatePackageName(pkg); err != nil {
			return nil, err
		}
		args = append(args, pkg)
	}
	out, err := m.exec(ctx, args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrCheckUnsupported)
	}
	return parseFlatpakDryRun(out), nil
}

// Refresh is a no-op for this manager.
func (m *Flatpak) Refresh(_ context.Context) error {
	return nil
}

func parseFlatpakDryRun(output []byte) []UpdateInfo {
	var result []UpdateInfo
	for _, line := range bytes.Split(output, []byte("\n")) {
		trimmed := string(bytes.TrimSpace(line))
		if !strings.HasPrefix(trimmed, "Updates for '") {
			continue
		}
		// "Updates for 'com.spotify.Client' in remote 'flathub'"
		parts := strings.Split(trimmed, "'")
		if len(parts) >= 2 && parts[1] != "" {
			result = append(result, UpdateInfo{Package: parts[1]})
		}
	}
	return result
}

// Provides returns an error since flatpak has no provides command.
func (m *Flatpak) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for flatpak", ErrNotSupported)
}

// AutoRemove removes unused runtimes via flatpak uninstall --unused.
func (m *Flatpak) AutoRemove(ctx context.Context, dryRun bool) ([]string, error) {
	if !dryRun {
		if err := requireConsent(ctx); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return nil, nil
	}
	_, err := m.exec(WithStreamIO(ctx), "flatpak", "uninstall", "--unused", "-y")
	if err != nil {
		return nil, fmt.Errorf("failed to remove unused flatpaks: %w", err)
	}
	return nil, nil
}

// Clean returns an error since flatpak has no cache clean command.
func (m *Flatpak) Clean(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: clean not supported for flatpak", ErrNotSupported)
}

// OverrideFlags holds flag values for flatpak override permission management.
type OverrideFlags struct {
	Filesystem []string
	Socket     []string
	Device     []string
	Env        []string
	Reset      bool
	Show       bool
	System     bool
}

// Override manages flatpak sandbox permissions for the given application.
// This method is NOT part of the Adapter interface — it's flatpak-specific.
func (m *Flatpak) Override(ctx context.Context, appID string, flags OverrideFlags) error {
	if err := ValidatePackageName(appID); err != nil {
		return err
	}

	args := []string{"override"}
	if flags.System {
		args = append(args, "--system")
	} else {
		args = append(args, "--user")
	}

	if flags.Reset {
		args = append(args, "--reset", appID)
		_, err := m.exec(WithStreamIO(ctx), "flatpak", args...)
		if err != nil {
			return fmt.Errorf("failed to reset overrides for %s: %w", appID, err)
		}
		return nil
	}

	if flags.Show {
		args = append(args, "--show", appID)
		out, err := m.exec(WithStreamIO(ctx), "flatpak", args...)
		if err != nil {
			return fmt.Errorf("failed to show overrides for %s: %w", appID, err)
		}
		fmt.Print(string(out))
		return nil
	}

	for _, fs := range flags.Filesystem {
		args = append(args, "--filesystem="+fs)
	}
	for _, s := range flags.Socket {
		args = append(args, "--socket="+s)
	}
	for _, d := range flags.Device {
		args = append(args, "--device="+d)
	}
	for _, e := range flags.Env {
		args = append(args, "--env="+e)
	}
	args = append(args, appID)

	_, err := m.exec(WithStreamIO(ctx), "flatpak", args...)
	if err != nil {
		return fmt.Errorf("failed to set overrides for %s: %w", appID, err)
	}
	return nil
}

// Hold returns an error since flatpak has no hold command.
func (m *Flatpak) Hold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: hold not supported for flatpak", ErrNotSupported)
}

// Unhold returns an error since flatpak has no unhold command.
func (m *Flatpak) Unhold(_ context.Context, _ string) error {
	return fmt.Errorf("%w: unhold not supported for flatpak", ErrNotSupported)
}

// ListHeld returns an error since flatpak has no hold command.
func (m *Flatpak) ListHeld(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("%w: hold not supported for flatpak", ErrNotSupported)
}
