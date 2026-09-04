package cli

import (
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var osExecutable = os.Executable

// resolveRealExecutable returns the resolved path of the running binary.
func resolveRealExecutable() (string, error) {
	exe, err := osExecutable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	return realExe, nil
}

// verifyWritePermission ensures the install directory is writable.
func verifyWritePermission(exeDir string) error {
	permCheck, err := os.CreateTemp(exeDir, "stamp-perm-*")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot write to %s\nPlease run 'sudo stamp self-update' to update", exeDir)
		}
		return fmt.Errorf("cannot access install directory: %w", err)
	}
	_ = permCheck.Close()
	_ = os.Remove(permCheck.Name())
	return nil
}

// downloadAndVerify downloads the release tarball and its checksums, verifies
// the tarball, and returns the verified bytes.
func downloadAndVerify(rel *release, targetName string) ([]byte, error) {
	tarballAsset := findAsset(rel.Assets, targetName)
	if tarballAsset == nil {
		return nil, fmt.Errorf("release asset %s not found", targetName)
	}

	checksumAsset := findAsset(rel.Assets, "checksums.txt")
	if checksumAsset == nil {
		return nil, fmt.Errorf("checksums.txt not found in release")
	}

	tarballData, err := downloadFile(tarballAsset.BrowserDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download update: %w", err)
	}

	checksumData, err := downloadFile(checksumAsset.BrowserDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums: %w", err)
	}

	expectedHex, err := checksumFor(targetName, strings.NewReader(string(checksumData)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse checksums: %w", err)
	}
	if err := verifyChecksum(tarballData, expectedHex); err != nil {
		return nil, fmt.Errorf("integrity check failed: %w", err)
	}
	return tarballData, nil
}

// extractToTemp writes the verified binary to a temp file in exeDir and
// returns its path. The caller owns cleanup and the final rename.
func extractToTemp(tarballData []byte, exeDir string) (string, error) {
	tmpFile, err := os.CreateTemp(exeDir, "stamp-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	if err := extractBinary(tarballData, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to extract binary: %w", err)
	}
	return tmpPath, nil
}

// reinstallPostUpdate refreshes shell completions and man pages with the new
// binary after a successful self-update.
func reinstallPostUpdate(realExe string, tty bool, errOut io.Writer) {
	_, _ = fmt.Fprintln(errOut, "  Reinstalling shell completions...")
	if err := runNewBinary(realExe, "completion"); err != nil {
		_, _ = fmt.Fprintf(errOut, "  ⚠ completion install failed: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(errOut, "  %s\n", iconLine(tty, "✓", "Completions updated"))
	}

	_, _ = fmt.Fprintln(errOut, "  Reinstalling man pages...")
	if err := runNewBinary(realExe, "man", "install"); err != nil {
		_, _ = fmt.Fprintf(errOut, "  ⚠ man page install failed: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(errOut, "  %s\n", iconLine(tty, "✓", "Man pages updated"))
	}
}

// renderUpdateStatus prints the check-mode or up-to-date status. It returns
// an error when a check found a new version (so the CLI exits non-zero).
func renderUpdateStatus(w io.Writer, rel *release, currentVersion, latestVersion string, checkOnly bool) error {
	if checkOnly {
		_, _ = fmt.Fprintf(w, "  Current version: v%s\n", currentVersion)
		_, _ = fmt.Fprintf(w, "  Latest version:  %s\n", rel.TagName)
		if currentVersion == latestVersion {
			_, _ = fmt.Fprintln(w, "  Already up to date.")
		} else {
			_, _ = fmt.Fprintln(w, "  A new version is available.")
			return fmt.Errorf("update available: %s", rel.TagName)
		}
		return nil
	}
	if currentVersion == latestVersion {
		_, _ = fmt.Fprintln(w, "  Already up to date.")
	}
	return nil
}

// performUpdate downloads, verifies, and atomically replaces the binary.
// The temp-file cleanup and rename ownership stay inside so an interrupted
// write can never leave a partial binary in place.
func performUpdate(rel *release, targetName, realExe, exeDir string) error {
	tarballData, err := downloadAndVerify(rel, targetName)
	if err != nil {
		return err
	}

	tmpPath, err := extractToTemp(tarballData, exeDir)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if info, statErr := os.Stat(realExe); statErr == nil {
		_ = os.Chmod(tmpPath, info.Mode())
	}

	if err := os.Rename(tmpPath, realExe); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	return nil
}

func newSelfUpdateCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:     "self-update",
		Aliases: []string{"self-upgrade"},
		Short:   "Update stamp to the latest version",
		Example: `  # update stamp to the latest release
  stamp self-update

  # check for a newer release without downloading
  stamp self-update --check

  # alias form
  stamp self-upgrade`,
		Long: `Check for and apply updates to the stamp binary.

Downloads the latest release from GitHub, verifies its SHA-256 checksum,
replaces the current binary atomically, and re-installs shell completions
and man pages automatically. Use --check to query without downloading.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			errOut := cmd.ErrOrStderr()
			tty := isOutputTerminal(errOut)
			_, _ = fmt.Fprintln(errOut, iconLine(tty, "▪", "Self-Update"))

			rel, err := fetchLatestRelease()
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			latestVersion := strings.TrimPrefix(rel.TagName, "v")
			currentVersion := Version

			if err := renderUpdateStatus(errOut, rel, currentVersion, latestVersion, checkOnly); err != nil {
				return err
			}
			if checkOnly || currentVersion == latestVersion {
				return nil
			}

			realExe, err := resolveRealExecutable()
			if err != nil {
				return err
			}
			exeDir := filepath.Dir(realExe)

			if err := verifyWritePermission(exeDir); err != nil {
				return err
			}

			targetName := releaseAssetName(rel.TagName, runtime.GOOS, runtime.GOARCH)
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Downloading %s...\n", targetName)

			if err := performUpdate(rel, targetName, realExe, exeDir); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(errOut, "  %s\n", iconLine(tty, "✓", fmt.Sprintf("Updated to %s", rel.TagName)))

			reinstallPostUpdate(realExe, tty, errOut)
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
