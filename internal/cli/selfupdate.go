package cli

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

var osExecutable = os.Executable

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var githubAPI = "https://api.github.com/repos/rossijonas/stamp/releases/latest"

func releaseAssetName(version, goos, arch string) string {
	return fmt.Sprintf("stamp_%s_%s_%s.tar.gz", version, goos, arch)
}

func fetchLatestRelease() (*release, error) {
	req, err := http.NewRequest(http.MethodGet, githubAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch release: HTTP %d", resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release has no tag_name")
	}
	return &rel, nil
}

func findAsset(assets []asset, name string) *asset {
	for _, a := range assets {
		if a.Name == name {
			return &a
		}
	}
	return nil
}

func checksumFor(assetName string, body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == assetName {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", assetName)
}

func downloadFile(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	return data, nil
}

func verifyChecksum(data []byte, expectedHex string) error {
	h := sha256.Sum256(data)
	got := fmt.Sprintf("%x", h)
	if got != expectedHex {
		return fmt.Errorf("checksum mismatch: got %s, expected %s", got, expectedHex)
	}
	return nil
}

func extractBinary(data []byte, dest string) error {
	gzr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to open gzip: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		cleanName := filepath.Base(filepath.Clean(header.Name))
		if cleanName != "stamp" {
			continue
		}

		//nolint:gosec // dest is a temp file created by os.CreateTemp in the binary's directory
		out, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("failed to create temp binary: %w", err)
		}

		//nolint:gosec // G110: trusted source — downloaded from official GitHub release and SHA-256 verified
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return fmt.Errorf("failed to extract binary: %w", err)
		}

		if err := out.Close(); err != nil {
			return fmt.Errorf("failed to close temp binary: %w", err)
		}

		return nil
	}

	return fmt.Errorf("binary not found in archive")
}

func newSelfUpdateCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:     "self-update",
		Aliases: []string{"self-upgrade"},
		Short:   "Update stamp to the latest version",
		Long: `Check for and apply updates to the stamp binary.

Downloads the latest release from GitHub, verifies its SHA-256 checksum,
replaces the current binary atomically, and re-installs shell completions
and man pages automatically. Use --check to query without downloading.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "▪ Self-Update")

			rel, err := fetchLatestRelease()
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			latestVersion := strings.TrimPrefix(rel.TagName, "v")
			currentVersion := strings.TrimPrefix(Version, "v")

			if checkOnly {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Current version: v%s\n", currentVersion)
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Latest version:  %s\n", rel.TagName)
				if currentVersion == latestVersion {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  Already up to date.")
				} else {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  A new version is available.")
					return fmt.Errorf("update available: %s", rel.TagName)
				}
				return nil
			}

			if currentVersion == latestVersion {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  Already up to date.")
				return nil
			}

			exe, err := osExecutable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}
			realExe, err := filepath.EvalSymlinks(exe)
			if err != nil {
				return fmt.Errorf("failed to resolve executable path: %w", err)
			}

			exeDir := filepath.Dir(realExe)

			permCheck, err := os.CreateTemp(exeDir, "stamp-perm-*")
			if err != nil {
				if os.IsPermission(err) {
					return fmt.Errorf("permission denied: cannot write to %s\nPlease run 'sudo stamp self-update' to update", exeDir)
				}
				return fmt.Errorf("cannot access install directory: %w", err)
			}
			_ = permCheck.Close()
			_ = os.Remove(permCheck.Name())

			targetName := releaseAssetName(rel.TagName, runtime.GOOS, runtime.GOARCH)

			tarballAsset := findAsset(rel.Assets, targetName)
			if tarballAsset == nil {
				return fmt.Errorf("release asset %s not found", targetName)
			}

			checksumAsset := findAsset(rel.Assets, "checksums.txt")
			if checksumAsset == nil {
				return fmt.Errorf("checksums.txt not found in release")
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Downloading %s...\n", targetName)

			tarballData, err := downloadFile(tarballAsset.BrowserDownloadURL)
			if err != nil {
				return fmt.Errorf("failed to download update: %w", err)
			}

			checksumData, err := downloadFile(checksumAsset.BrowserDownloadURL)
			if err != nil {
				return fmt.Errorf("failed to download checksums: %w", err)
			}

			expectedHex, err := checksumFor(targetName, strings.NewReader(string(checksumData)))
			if err != nil {
				return fmt.Errorf("failed to parse checksums: %w", err)
			}
			if err := verifyChecksum(tarballData, expectedHex); err != nil {
				return fmt.Errorf("integrity check failed: %w", err)
			}

			tmpFile, err := os.CreateTemp(exeDir, "stamp-*")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			_ = tmpFile.Close()
			defer func() { _ = os.Remove(tmpPath) }()

			if err := extractBinary(tarballData, tmpPath); err != nil {
				return fmt.Errorf("failed to extract binary: %w", err)
			}

			if info, statErr := os.Stat(realExe); statErr == nil {
				_ = os.Chmod(tmpPath, info.Mode())
			}

			if err := os.Rename(tmpPath, realExe); err != nil {
				return fmt.Errorf("failed to replace binary: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ✅ Updated to %s\n", rel.TagName)

			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  Reinstalling shell completions...")
			if err := runNewBinary(realExe, "completion"); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ⚠ completion install failed: %v\n", err)
			} else {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  ✅ Completions updated")
			}

			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  Reinstalling man pages...")
			if err := runNewBinary(realExe, "man", "install"); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ⚠ man page install failed: %v\n", err)
			} else {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "  ✅ Man pages updated")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "check for update without downloading")
	return cmd
}

func runNewBinary(bin string, args ...string) error {
	//nolint:gosec // bin is the resolved path to the stamp binary itself, not user input
	cmd := osexec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
