package manager

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dnfReposDir is the directory containing .repo files. Declared as a variable
// so tests can override it with a temporary directory.
var dnfReposDir = "/etc/yum.repos.d"

// ListRepos returns a list of configured third-party repositories by parsing
// .repo files directly. No shell exec, no cache, no sudo.
func (m *DNF) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return parseDNFSources()
}

// parseDNFSources reads .repo files from dnfReposDir and extracts
// custom/third-party repository identifiers with their base URLs.
func parseDNFSources() ([]RepositoryInfo, error) {
	systemRepos := map[string]bool{
		"fedora":                true,
		"fedora-updates":        true,
		"fedora-modular":        true,
		"updates":               true,
		"updates-modular":       true,
		"updates-testing":       true,
		"baseos":                true,
		"appstream":             true,
		"extras":                true,
		"epel":                  true,
		"epel-next":             true,
		"epel-testing":          true,
		"epel-next-testing":     true,
		"fedora-cisco-openh264": true,
	}

	entries, err := os.ReadDir(dnfReposDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", dnfReposDir, err)
	}

	var repos []RepositoryInfo
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".repo" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dnfReposDir, entry.Name()))
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")

		var currentID string
		var currentURL string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "[") {
				if currentID != "" && !systemRepos[currentID] {
					repos = append(repos, RepositoryInfo{Name: currentID, URL: currentURL})
				}
				end := strings.IndexByte(line, ']')
				if end > 1 {
					currentID = line[1:end]
				} else {
					currentID = ""
				}
				currentURL = ""
				continue
			}
			eq := strings.IndexByte(line, '=')
			if eq < 0 {
				continue
			}
			key := strings.TrimSpace(line[:eq])
			val := strings.TrimSpace(line[eq+1:])
			switch {
			case key == "baseurl":
				currentURL = val
			case currentURL == "" && key == "metalink":
				currentURL = val
			case currentURL == "" && key == "mirrorlist":
				currentURL = val
			}
		}
		if currentID != "" && !systemRepos[currentID] {
			repos = append(repos, RepositoryInfo{Name: currentID, URL: currentURL})
		}
	}
	return repos, nil
}

// parseDNFRepos parses the output of 'dnf repolist' and extracts
// custom/third-party repository identifiers, filtering out known system repos.
func parseDNFRepos(output []byte) []string {
	systemRepos := map[string]bool{
		"fedora":                true,
		"fedora-updates":        true,
		"fedora-modular":        true,
		"updates":               true,
		"updates-modular":       true,
		"updates-testing":       true,
		"baseos":                true,
		"appstream":             true,
		"extras":                true,
		"epel":                  true,
		"epel-next":             true,
		"epel-testing":          true,
		"epel-next-testing":     true,
		"fedora-cisco-openh264": true,
	}

	lines := bytes.Split(output, []byte("\n"))
	repos := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if bytes.HasPrefix(bytes.ToLower(trimmed), []byte("repo id")) ||
			bytes.HasPrefix(bytes.ToLower(trimmed), []byte("id")) {
			continue
		}
		fields := bytes.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		id := string(fields[0])
		if systemRepos[id] {
			continue
		}
		repos = append(repos, id)
	}
	return repos
}

// AddRepo enables a third-party repository.
func (m *DNF) AddRepo(ctx context.Context, name, url string) error {
	if url != "" {
		content := fmt.Sprintf("[%s]\nname=%s\nbaseurl=%s\nenabled=1\ngpgcheck=0\n", name, name, url)
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("stamp-%s-*.repo", name))
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		if _, err := tmpFile.WriteString(content); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to write repo file: %w", err)
		}
		_ = tmpFile.Close()
		defer func() { _ = os.Remove(tmpPath) }()

		destPath := fmt.Sprintf("/etc/yum.repos.d/%s.repo", name)
		args := sudoCmd("mv", tmpPath, destPath)
		_, err = m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to add repo %s: %w", name, err)
		}
		return nil
	}
	args := sudoCmd(m.cmd, "copr", "enable", "-y", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to enable copr %s: %w", name, err)
	}
	return nil
}

// RemoveRepo disables a third-party repository.
func (m *DNF) RemoveRepo(ctx context.Context, name string) error {
	args := sudoCmd(m.cmd, "copr", "disable", "-y", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to disable copr %s: %w", name, err)
	}
	return nil
}
