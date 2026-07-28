package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Go implements the Adapter interface for the Go toolchain (go install).
// Binaries are resolved via go env GOBIN or go env GOPATH with a fallback
// to $HOME/go/bin. Module paths must contain at least one "/"
// (e.g., github.com/example/tool).
type Go struct {
	exec   Executor
	binDir string // cached resolved bin directory
}

// NewGo creates a new Go adapter.
func NewGo() *Go {
	return &Go{
		exec: defaultExecutor,
	}
}

// Name returns the package manager identifier.
func (m *Go) Name() string {
	return "go"
}

// goBinDir resolves the directory where go install places binaries.
// Precedence: GOBIN env → first entry of GOPATH/bin → $HOME/go/bin.
// The result is cached for the lifetime of the adapter.
func (m *Go) goBinDir(ctx context.Context) (string, error) {
	if m.binDir != "" {
		return m.binDir, nil
	}

	out, err := m.exec(ctx, "go", "env", "GOBIN")
	if err == nil {
		dir := strings.TrimSpace(string(out))
		if dir != "" {
			m.binDir = dir
			return m.binDir, nil
		}
	}

	out, err = m.exec(ctx, "go", "env", "GOPATH")
	if err == nil {
		gopath := strings.TrimSpace(string(out))
		entries := filepath.SplitList(gopath)
		for _, entry := range entries {
			if entry != "" {
				m.binDir = filepath.Join(entry, "bin")
				return m.binDir, nil
			}
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine go bin directory: %w", err)
	}
	m.binDir = filepath.Join(home, "go", "bin")
	return m.binDir, nil
}

// binNameFromModule extracts the binary name from a Go module path.
// For "github.com/example/tool/cmd/tool" it returns "tool".
func binNameFromModule(pkg string) (string, error) {
	binName := filepath.Base(pkg)
	if binName == "." || binName == ".." || binName == "" || strings.ContainsRune(binName, '/') {
		return "", fmt.Errorf("invalid module path %q: binary name must be a simple filename", pkg)
	}
	return binName, nil
}

// recoverModulePath tries to recover the go module path from a binary's
// embedded metadata (from "go version -m"). Falls back to the binary name
// if the module path cannot be read.
func recoverModulePath(ctx context.Context, exec Executor, binPath, fallback string) string {
	out, err := exec(ctx, "go", "version", "-m", binPath)
	if err != nil {
		return fallback
	}
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "path\t") {
			if mod := strings.TrimSpace(strings.TrimPrefix(trimmed, "path\t")); mod != "" {
				return mod
			}
		}
	}
	return fallback
}

// ListInstalled returns a list of binaries installed via go install.
// Each entry is the module path when recoverable from the binary metadata,
// falling back to the binary name.
func (m *Go) ListInstalled(ctx context.Context) ([]string, error) {
	dir, err := m.goBinDir(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list go binaries: %w", err)
	}

	var result []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}

		binPath := filepath.Join(dir, e.Name())
		modulePath := recoverModulePath(ctx, m.exec, binPath, e.Name())
		result = append(result, modulePath)
	}
	return result, nil
}

// Install runs go install <pkg>@latest.
func (m *Go) Install(ctx context.Context, pkg string) error {
	if err := ValidateModulePath(pkg); err != nil {
		return err
	}
	_, err := m.exec(WithStreamIO(ctx), "go", "install", pkg+"@latest")
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall is the same as Install (go install is idempotent).
func (m *Go) Reinstall(ctx context.Context, pkg string) error {
	return m.Install(ctx, pkg)
}

// Remove removes a binary installed via go install.
func (m *Go) Remove(ctx context.Context, pkg string) error {
	if err := ValidateModulePath(pkg); err != nil {
		return err
	}
	binName, err := binNameFromModule(pkg)
	if err != nil {
		return err
	}
	dir, err := m.goBinDir(ctx)
	if err != nil {
		return err
	}
	binPath := filepath.Join(dir, binName)
	if _, err := os.Stat(binPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found in go bin directory", binName)
		}
		return fmt.Errorf("failed to check %s: %w", binPath, err)
	}
	if err := os.Remove(binPath); err != nil {
		return fmt.Errorf("failed to remove %s: %w", binPath, err)
	}
	return nil
}

// Search is not supported for go. Returns an error so the CLI
// can print a warning to stderr instead of producing fake results.
func (m *Go) Search(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("go: search not supported (go has no package registry CLI)")
}

// Info returns module metadata from the installed binary.
func (m *Go) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidateModulePath(pkg); err != nil {
		return "", err
	}
	binName, err := binNameFromModule(pkg)
	if err != nil {
		return "", err
	}
	dir, err := m.goBinDir(ctx)
	if err != nil {
		return "", err
	}
	binPath := filepath.Join(dir, binName)
	out, err := m.exec(ctx, "go", "version", "-m", binPath)
	if err != nil {
		return "", fmt.Errorf("%s not found in go bin directory", binName)
	}
	return string(out), nil
}

// Doctor returns an error since go has no native diagnostic command.
func (m *Go) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for go")
}

// Update runs go install <pkg>@latest for a single package, or
// reinstalls all tools with recoverable module paths for batch.
func (m *Go) Update(ctx context.Context, pkg string) error {
	if pkg != "" {
		return m.Install(ctx, pkg)
	}

	// Batch: list installed and reinstall each recoverable module path
	pkgs, err := m.ListInstalled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list go binaries for update: %w", err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for _, mod := range pkgs {
		if !strings.Contains(mod, "/") {
			continue
		}
		mod := mod
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Install(ctx, mod); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// CheckUpdate returns an error since go has no check-update command.
func (m *Go) CheckUpdate(_ context.Context, _ string) ([]UpdateInfo, error) {
	return nil, fmt.Errorf("%w", ErrCheckUnsupported)
}

// AddRepo returns an error since go has no concept of repositories.
func (m *Go) AddRepo(_ context.Context, _, _ string) error {
	return fmt.Errorf("not supported for go")
}

// RemoveRepo returns an error since go has no concept of repositories.
func (m *Go) RemoveRepo(_ context.Context, _ string) error {
	return fmt.Errorf("not supported for go")
}

// ListRepos returns an empty list since go has no concept of repositories.
func (m *Go) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return nil, nil
}

// Provides returns an error since go has no provides command.
func (m *Go) Provides(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("%w: provides not supported for go", ErrNotSupported)
}

// AutoRemove returns an error since go has no autoremove command.
func (m *Go) AutoRemove(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: autoremove not supported for go", ErrNotSupported)
}

// Clean returns an error since go has no cache clean command.
func (m *Go) Clean(_ context.Context, _ bool) ([]string, error) {
	return nil, fmt.Errorf("%w: clean not supported for go", ErrNotSupported)
}
