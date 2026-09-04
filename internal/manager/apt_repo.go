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

// addAptRepoCmd is the PPA management command for Debian/Ubuntu.
const addAptRepoCmd = "add-apt-repository"

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

	addFile := func(path string) {
		repos = appendUniqueRepos(repos, parseAPTFile(path, systemDomains))
	}

	addFile(aptSourcesFile)

	if entries, err := os.ReadDir(aptSourcesDir); err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".list" {
				addFile(filepath.Join(aptSourcesDir, entry.Name()))
			}
		}
	}

	return repos, nil
}

// parseAPTFile reads a single APT source file and returns its non-system
// repositories. Unreadable files yield no repos.
func parseAPTFile(path string, systemDomains map[string]bool) []RepositoryInfo {
	//nolint:gosec // path is a controlled path to /etc/apt/sources*
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var repos []RepositoryInfo
	for _, line := range strings.Split(string(data), "\n") {
		if repo, ok := parseAPTLine(line, systemDomains); ok {
			repos = append(repos, repo)
		}
	}
	return repos
}

// parseAPTLine converts one line of an APT source file into a repository, or
// returns ok=false when the line is not a non-system deb source entry.
func parseAPTLine(line string, systemDomains map[string]bool) (RepositoryInfo, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return RepositoryInfo{}, false
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return RepositoryInfo{}, false
	}
	if fields[0] != "deb" && fields[0] != "deb-src" {
		return RepositoryInfo{}, false
	}
	repoURL, ok := aptLineURL(fields)
	if !ok {
		return RepositoryInfo{}, false
	}
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return RepositoryInfo{}, false
	}
	if systemDomains[parsed.Host] {
		return RepositoryInfo{}, false
	}
	return RepositoryInfo{Name: repoName(parsed), URL: repoURL}, true
}

// aptLineURL returns the repository URL from a parsed "deb" source line,
// skipping leading option flags such as [arch=amd64].
func aptLineURL(fields []string) (string, bool) {
	urlIdx := 1
	for urlIdx < len(fields) && strings.HasPrefix(fields[urlIdx], "[") {
		urlIdx++
	}
	if urlIdx >= len(fields) {
		return "", false
	}
	return fields[urlIdx], true
}

// repoName derives a repository display name from a parsed URL, stripping a
// trailing slash and a leading www. prefix.
func repoName(parsed *url.URL) string {
	name := parsed.Host + parsed.Path
	name = strings.TrimSuffix(name, "/")
	name = strings.TrimPrefix(name, "www.")
	return name
}

// appendUniqueRepos appends repos whose URL is not already present.
func appendUniqueRepos(repos []RepositoryInfo, newRepos []RepositoryInfo) []RepositoryInfo {
	for _, nr := range newRepos {
		found := false
		for _, r := range repos {
			if r.URL == nr.URL {
				found = true
				break
			}
		}
		if !found {
			repos = append(repos, nr)
		}
	}
	return repos
}

// AddRepo enables a third-party repository.
func (m *APT) AddRepo(ctx context.Context, name, url string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if url == "" {
		if _, err := lookPath(addAptRepoCmd); err != nil {
			return fmt.Errorf("add-apt-repository not found: install 'software-properties-common' to use PPAs")
		}
		args := sudoCmd(addAptRepoCmd, "-y", name)
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
	if err := requireConsent(ctx); err != nil {
		return err
	}
	listPath := filepath.Join(aptSourcesDir, fmt.Sprintf("%s.list", name))
	if _, err := os.Stat(listPath); err == nil {
		args := sudoCmd("rm", "-f", listPath)
		_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to remove repository %s: %w", name, err)
		}
		return nil
	}

	if _, err := lookPath(addAptRepoCmd); err != nil {
		return fmt.Errorf("add-apt-repository not found: install 'software-properties-common' to remove PPAs")
	}
	args := sudoCmd(addAptRepoCmd, "-y", "--remove", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to remove repository %s: %w", name, err)
	}
	return nil
}
