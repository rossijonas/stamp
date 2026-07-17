package manager

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
)

// APT implements the Adapter interface for Debian/Ubuntu's APT (or apt-get).
type APT struct {
	exec Executor
	cmd  string
}

// NewAPT creates a new APT adapter for the given command ("apt" or "apt-get").
func NewAPT(cmd string) *APT {
	return &APT{
		exec: defaultExecutor,
		cmd:  cmd,
	}
}

// Name returns the package manager identifier.
func (m *APT) Name() string {
	return "apt"
}

// ListInstalled returns a list of packages currently installed.
func (m *APT) ListInstalled(ctx context.Context) ([]string, error) {
	// Primary: apt list --installed
	out, err := m.exec(ctx, m.cmd, "list", "--installed")
	if err == nil {
		return parseAPTListInstalled(out), nil
	}

	// Fallback: dpkg-query with state filtering (excludes "rc" packages)
	out, err = m.exec(ctx, "dpkg-query", "-f", "${db:Status-Status} ${Package}\n", "-W")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	return parseDPKGQueryInstalled(out), nil
}

// parseAPTListInstalled parses the output of 'apt list --installed'.
// Format: "htop/stable,now 3.2.1 amd64 [installed]"
func parseAPTListInstalled(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || bytes.HasPrefix(trimmed, []byte("Listing")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		pkgField := fields[0]
		// Package name is the text before '/' or ','
		idx := bytes.IndexAny(pkgField, "/,")
		if idx > 0 {
			result = append(result, string(pkgField[:idx]))
		}
	}
	return result
}

// parseDPKGQueryInstalled parses the output of 'dpkg-query -f "${db:Status-Status} ${Package}\n" -W'.
// Filters only lines with "installed" status to exclude "rc" (removed + config) packages.
func parseDPKGQueryInstalled(output []byte) []string {
	lines := bytes.Split(output, []byte("\n"))
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) >= 2 && string(fields[0]) == "installed" {
			result = append(result, string(fields[1]))
		}
	}
	return result
}

// Install executes the native installation command.
func (m *APT) Install(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "install", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to install %s: %w", pkg, err)
	}
	return nil
}

// Reinstall executes the native reinstallation command.
func (m *APT) Reinstall(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "install", "--reinstall", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
	}
	return nil
}

// Remove executes the native removal command.
func (m *APT) Remove(ctx context.Context, pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	args := sudoCmd(m.cmd, "remove", "-y", pkg)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove %s: %w", pkg, err)
	}
	return nil
}

// Search queries the native package manager for the given package name.
func (m *APT) Search(ctx context.Context, query string) ([]string, error) {
	if err := ValidatePackageName(query); err != nil {
		return nil, err
	}
	out, err := m.exec(ctx, "apt-cache", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %s: %w", query, err)
	}
	// Output: "pkgname - description"
	lines := parseLines(out)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			result = append(result, fields[0])
		}
	}
	return result, nil
}

// ListRepos returns a list of third-party repositories by parsing
// /etc/apt/sources.list and files in /etc/apt/sources.list.d/.
func (m *APT) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return parseAPTSources()
}

// parseAPTSources reads APT source files and extracts non-system repositories.
func parseAPTSources() ([]RepositoryInfo, error) {
	systemDomains := map[string]bool{
		"archive.ubuntu.com":  true,
		"security.ubuntu.com": true,
		"ports.ubuntu.com":    true,
		"deb.debian.org":      true,
		"security.debian.org": true,
	}

	var repos []RepositoryInfo

	parseFile := func(path string) {
		//nolint:gosec // path is a controlled path to /etc/apt/sources*
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			if fields[0] != "deb" && fields[0] != "deb-src" {
				continue
			}
			// Skip bracketed options (e.g. [arch=amd64]) and find the URL
			urlIdx := 1
			for urlIdx < len(fields) && strings.HasPrefix(fields[urlIdx], "[") {
				urlIdx++
			}
			if urlIdx >= len(fields) {
				continue
			}
			repoURL := fields[urlIdx]
			parsed, err := url.Parse(repoURL)
			if err != nil {
				continue
			}
			if systemDomains[parsed.Host] {
				continue
			}
			name := parsed.Host + parsed.Path
			name = strings.TrimSuffix(name, "/")
			name = strings.TrimPrefix(name, "www.")

			found := false
			for _, r := range repos {
				if r.URL == repoURL {
					found = true
					break
				}
			}
			if !found {
				repos = append(repos, RepositoryInfo{Name: name, URL: repoURL})
			}
		}
	}

	parseFile(aptSourcesFile)

	if entries, err := os.ReadDir(aptSourcesDir); err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".list" {
				parseFile(filepath.Join(aptSourcesDir, entry.Name()))
			}
		}
	}

	return repos, nil
}

// aptSourcesDir and aptSourcesFile are overridable in tests.
var aptSourcesDir = "/etc/apt/sources.list.d"
var aptSourcesFile = "/etc/apt/sources.list"

// AddRepo enables a third-party repository.
func (m *APT) AddRepo(ctx context.Context, name, url string) error {
	if url == "" {
		// PPA: requires add-apt-repository
		if _, err := osexec.LookPath("add-apt-repository"); err != nil {
			return fmt.Errorf("add-apt-repository not found: install 'software-properties-common' to use PPAs")
		}
		args := sudoCmd("add-apt-repository", "-y", name)
		_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to add repository %s: %w", name, err)
		}
		return nil
	}

	// Custom URL: write .list file directly (no add-apt-repository needed)
	content := fmt.Sprintf("deb [trusted=yes] %s ./\n", url)
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("stamp-%s-*.list", name))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write list file: %w", err)
	}
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	destPath := filepath.Join(aptSourcesDir, fmt.Sprintf("%s.list", name))
	args := sudoCmd("mv", tmpPath, destPath)
	_, err = m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to add repository %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party repository.
func (m *APT) RemoveRepo(ctx context.Context, name string) error {
	listPath := filepath.Join(aptSourcesDir, fmt.Sprintf("%s.list", name))
	if _, err := os.Stat(listPath); err == nil {
		args := sudoCmd("rm", "-f", listPath)
		_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to remove repository %s: %w", name, err)
		}
		return nil
	}

	if _, err := osexec.LookPath("add-apt-repository"); err != nil {
		return fmt.Errorf("add-apt-repository not found: install 'software-properties-common' to remove PPAs")
	}
	args := sudoCmd("add-apt-repository", "-y", "--remove", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove repository %s: %w", name, err)
	}
	return nil
}

// Info queries apt info metadata.
func (m *APT) Info(ctx context.Context, pkg string) (string, error) {
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	var out []byte
	var err error
	if m.cmd == "apt" {
		out, err = m.exec(ctx, "apt", "show", pkg)
	} else {
		out, err = m.exec(ctx, "apt-cache", "show", pkg)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get info for %s: %w", pkg, err)
	}
	return string(out), nil
}

// Doctor returns an error since apt has no native diagnostic command.
func (m *APT) Doctor(_ context.Context) (string, error) {
	return "", fmt.Errorf("doctor not supported for apt")
}

// Update runs apt update then apt upgrade (two-phase).
func (m *APT) Update(ctx context.Context) error {
	args := sudoCmd(m.cmd, "update")
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}

	args = sudoCmd(m.cmd, "upgrade", "-y")
	_, err = m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}

	return nil
}
