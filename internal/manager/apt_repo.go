package manager

import (
	"context"
	"fmt"
	"net/url"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
)

// aptSourcesDir and aptSourcesFile are overridable in tests.
var aptSourcesDir = "/etc/apt/sources.list.d"
var aptSourcesFile = "/etc/apt/sources.list"

// lookPath is overridable in tests to simulate missing add-apt-repository.
var lookPath = osexec.LookPath

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

// AddRepo enables a third-party repository.
func (m *APT) AddRepo(ctx context.Context, name, url string) error {
	if url == "" {
		if _, err := lookPath("add-apt-repository"); err != nil {
			return fmt.Errorf("add-apt-repository not found: install 'software-properties-common' to use PPAs")
		}
		args := sudoCmd("add-apt-repository", "-y", name)
		_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to add repository %s: %w", name, err)
		}
		return nil
	}

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

	if _, err := lookPath("add-apt-repository"); err != nil {
		return fmt.Errorf("add-apt-repository not found: install 'software-properties-common' to remove PPAs")
	}
	args := sudoCmd("add-apt-repository", "-y", "--remove", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove repository %s: %w", name, err)
	}
	return nil
}
