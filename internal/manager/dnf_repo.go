package manager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// dnfReposDir is the directory containing .repo files. Declared as a variable
// so tests can override it with a temporary directory.
var dnfReposDir = "/etc/yum.repos.d"

// maxRepoFileBytes caps downloads of .repo files to prevent unbounded memory use.
const maxRepoFileBytes = 1 << 20

// dnfRepoFetcher fetches a .repo file over HTTP(S). Declared as a variable so
// tests can inject a stub without network access.
var dnfRepoFetcher = func(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", rawURL, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: unexpected status %d", rawURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRepoFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %w", rawURL, err)
	}
	if len(body) > maxRepoFileBytes {
		return nil, fmt.Errorf("repo file at %s exceeds %d bytes", rawURL, maxRepoFileBytes)
	}
	return body, nil
}

// isRepofileURL reports whether the URL points at a .repo configuration file
// (case-insensitive suffix on the URL path).
func isRepofileURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.HasSuffix(strings.ToLower(parsed.Path), ".repo")
}

// ListRepos returns a list of configured third-party repositories by parsing
// .repo files directly. No shell exec, no cache, no sudo.
func (m *DNF) ListRepos(_ context.Context) ([]RepositoryInfo, error) {
	return parseDNFSources()
}

// exactSystemRepos is the set of known base OS repository IDs for
// Fedora/RHEL/CentOS and their derivatives.
var exactSystemRepos = map[string]bool{
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

// systemRepoSuffixes match repository IDs that are debug/source/testing
// variants of system repos (e.g. fedora-debuginfo, updates-source).
var systemRepoSuffixes = []string{
	"-debuginfo",
	"-debugsource",
	"-source",
	"-testing-debuginfo",
	"-testing-source",
}

// systemRepoPrefixes match vendor repo IDs that ship system-level
// configurations rather than user-tracked third-party repos.
// Note: copr: repos are intentionally NOT filtered — they are always
// user-initiated via `dnf copr enable` and should surface as drift.
var systemRepoPrefixes = []string{
	"docker-ce-",
	"rpmfusion-",
	"google-chrome",
	"google-cloud-sdk",
	"protonvpn-",
}

// systemRepoBasePrefixes identify OS repo families whose testing variants
// (e.g. fedora-updates-testing, epel-testing) are still system repos, while a
// third-party repo like enpass-testing must NOT be filtered.
var systemRepoBasePrefixes = []string{
	"fedora",
	"updates",
	"epel",
	"baseos",
	"appstream",
	"extras",
}

// isSystemRepo reports whether a repository ID belongs to the OS or a
// vendor system repo, and should therefore be hidden from reconcile drift.
func isSystemRepo(repoID string) bool {
	if exactSystemRepos[repoID] {
		return true
	}
	for _, suffix := range systemRepoSuffixes {
		if strings.HasSuffix(repoID, suffix) {
			return true
		}
	}
	for _, base := range systemRepoBasePrefixes {
		if strings.HasPrefix(repoID, base) {
			if strings.HasSuffix(repoID, "-testing") {
				return true
			}
		}
	}
	for _, prefix := range systemRepoPrefixes {
		if strings.HasPrefix(repoID, prefix) {
			return true
		}
	}
	return false
}

// parseDNFSources reads .repo files from dnfReposDir and extracts
// custom/third-party repository identifiers with their base URLs.
func parseDNFSources() ([]RepositoryInfo, error) {
	entries, err := os.ReadDir(dnfReposDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", dnfReposDir, err)
	}

	var repos []RepositoryInfo
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".repo" {
			continue
		}
		fileRepos, err := parseRepoFile(filepath.Join(dnfReposDir, entry.Name()))
		if err != nil {
			continue
		}
		repos = append(repos, fileRepos...)
	}
	return repos, nil
}

// parseRepoFile reads a single .repo file and extracts its custom/third-party
// repository identifiers, filtering out known system repos.
func parseRepoFile(path string) ([]RepositoryInfo, error) {
	//nolint:gosec // path is a controlled path under dnfReposDir (e.g. /etc/yum.repos.d)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseRepoContent(data), nil
}

// parseRepoContent walks the lines of a .repo file, tracking the current
// section (repository ID) and its base URL. Each section is flushed when the
// next section header is seen or the file ends, unless it is a system repo.
func parseRepoContent(data []byte) []RepositoryInfo {
	lines := strings.Split(string(data), "\n")

	var repos []RepositoryInfo
	var currentID string
	var currentURL string
	flush := func() {
		if currentID != "" && !isSystemRepo(currentID) {
			repos = append(repos, RepositoryInfo{Name: currentID, URL: currentURL})
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			flush()
			currentID, currentURL = repoSectionID(line)
			continue
		}
		currentURL = repoLineURL(line, currentURL)
	}
	flush()
	return repos
}

// repoSectionID extracts the repository ID from a "[section]" header line.
// The returned URL is empty because a fresh section starts with no base URL.
func repoSectionID(line string) (id, url string) {
	end := strings.IndexByte(line, ']')
	if end > 1 {
		return line[1:end], ""
	}
	return "", ""
}

// repoLineURL updates the running base URL from a key=value line. The first
// baseurl always wins; metalink and mirrorlist only fill an empty URL.
func repoLineURL(line, currentURL string) string {
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return currentURL
	}
	key := strings.TrimSpace(line[:eq])
	val := strings.TrimSpace(line[eq+1:])
	switch {
	case key == "baseurl":
		return val
	case currentURL == "" && key == "metalink":
		return val
	case currentURL == "" && key == "mirrorlist":
		return val
	}
	return currentURL
}

// parseDNFRepos parses the output of 'dnf repolist' and extracts
// custom/third-party repository identifiers, filtering out known system repos.
func parseDNFRepos(output []byte) []string {
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
		if isSystemRepo(id) {
			continue
		}
		repos = append(repos, id)
	}
	return repos
}

// validateRepoFileContent rejects fetched content that is not a plausible
// .repo file: it must contain at least one [section] and a baseurl, metalink,
// or mirrorlist key.
func validateRepoFileContent(content []byte) error {
	lines := strings.Split(string(content), "\n")
	hasSection := false
	hasURL := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			hasSection = true
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		switch key {
		case "baseurl", "metalink", "mirrorlist":
			hasURL = true
		}
	}
	if !hasSection {
		return errors.New("no repo sections found in fetched repo file")
	}
	if !hasURL {
		return errors.New("no baseurl, metalink, or mirrorlist found in fetched repo file")
	}
	return nil
}

// writeRepoFile writes content to a temp file and sudo-moves it into dnfReposDir.
func writeRepoFile(ctx context.Context, exec Executor, name, content string) error {
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

	destPath := filepath.Join(dnfReposDir, filepath.Base(name)+".repo")
	args := sudoCmd("mv", tmpPath, destPath)
	_, err = exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to add repo %s: %w", name, err)
	}
	return nil
}

// AddRepo enables a third-party repository.
func (m *DNF) AddRepo(ctx context.Context, name, url string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	if url != "" {
		var content string
		if isRepofileURL(url) {
			fetched, err := dnfRepoFetcher(ctx, url)
			if err != nil {
				return fmt.Errorf("failed to fetch repo file: %w", err)
			}
			if err := validateRepoFileContent(fetched); err != nil {
				return fmt.Errorf("invalid repo file from %s: %w", url, err)
			}
			content = string(fetched)
		} else {
			content = fmt.Sprintf("[%s]\nname=%s\nbaseurl=%s\nenabled=1\ngpgcheck=0\n", name, name, url)
		}
		return writeRepoFile(ctx, m.exec, name, content)
	}
	args := sudoCmd(m.cmd, "copr", "enable", "-y", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to enable copr %s: %w", name, err)
	}
	return nil
}

// RemoveRepo removes a third-party repository. Repos added by URL (bare
// baseurl or .repo file) are removed by deleting their .repo file; name-only
// COPR repos fall back to dnf copr disable.
func (m *DNF) RemoveRepo(ctx context.Context, name string) error {
	if err := requireConsent(ctx); err != nil {
		return err
	}
	repoPath := filepath.Join(dnfReposDir, filepath.Base(name)+".repo")
	if _, err := os.Stat(repoPath); err == nil {
		args := sudoCmd("rm", "-f", repoPath)
		_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
		if err != nil {
			return fmt.Errorf("failed to remove repo file %s: %w", repoPath, err)
		}
		return nil
	}
	args := sudoCmd(m.cmd, "copr", "disable", "-y", name)
	_, err := m.exec(WithStreamIO(ctx), args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("failed to disable copr %s: %w", name, err)
	}
	return nil
}
