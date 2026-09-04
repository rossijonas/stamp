package cli

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manVersionRegex = regexp.MustCompile(`\.TH "STAMP" "1" "[^"]*" "stamp ([^"]+)"`)

type manPageStatus struct {
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	Matches   bool   `json:"matches"`
}

type manCheckResult struct {
	version string
	matches bool
}

var manPageCandidates []string

func init() {
	manPageCandidates = defaultManPageCandidates()
}

func defaultManPageCandidates() []string {
	return []string{
		filepath.Join(os.Getenv("HOME"), ".local", "share", "man", "man1", "stamp.1"),
		"/usr/local/share/man/man1/stamp.1",
		"/usr/share/man/man1/stamp.1",
		"/opt/homebrew/share/man/man1/stamp.1",
	}
}

// manExecFunc runs a command and returns its stdout. Injectable in tests,
// mirroring the manager.Executor pattern.
type manExecFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// manExec resolves installed man page paths via `man -w`.
var manExec manExecFunc = defaultManExec

func defaultManExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	//nolint:gosec // name/args are hardcoded ("man", "-w", "1", "stamp")
	return exec.CommandContext(ctx, name, args...).Output()
}

// resolveInstalledManPage returns the path to the installed stamp man page.
// It prefers `man -w 1 stamp`, which honors MANPATH and the system man search
// path (so pages installed to a custom --prefix are found), and falls back to
// the hardcoded candidate list when man is unavailable or finds nothing.
func resolveInstalledManPage() string {
	out, err := manExec(context.Background(), "man", "-w", "1", "stamp")
	if err == nil {
		if p := firstManPagePath(out); p != "" {
			return p
		}
	}
	return installedManPagePath()
}

// firstManPagePath returns the first .1/.1.gz path token in man -w output.
// man-db prints paths space/newline-separated; BSD prints a single path.
func firstManPagePath(out []byte) string {
	for _, tok := range strings.Fields(string(out)) {
		if strings.HasSuffix(tok, ".1") || strings.HasSuffix(tok, ".1.gz") {
			return tok
		}
	}
	return ""
}

// installedManPagePath returns the first existing hardcoded candidate path.
func installedManPagePath() string {
	for _, p := range manPageCandidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// readManPage reads a man page, decompressing gzip-compressed pages (system
// pages found via MANPATH are commonly .gz) before version matching.
func readManPage(path string) ([]byte, error) {
	//nolint:gosec // path is resolved via man -w or the candidate list
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer func() { _ = gz.Close() }()
		return io.ReadAll(gz)
	}
	return data, nil
}

func checkInstalledManVersion() (*manCheckResult, string, error) {
	path := resolveInstalledManPage()
	if path == "" {
		return nil, "", nil
	}

	data, err := readManPage(path)
	if err != nil {
		return nil, "", err
	}

	// Parse version string via regex
	match := manVersionRegex.FindSubmatch(data)
	if len(match) < 2 {
		return &manCheckResult{version: "unknown", matches: false}, path, nil
	}

	ver := string(match[1])
	return &manCheckResult{
		version: ver,
		matches: strings.TrimPrefix(ver, "v") == Version,
	}, path, nil
}

func newManCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "man",
		Short: "Manage stamp troff man pages",
		Example: `  # install the man page to the default (user) location
  stamp man install

  # check whether the installed man page matches this version
  stamp man check`,
		Long: `Command group to generate, install, and check stamp man pages.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newManInstallCmd())
	cmd.AddCommand(newManCheckCmd())
	return cmd
}

// copyManPages copies .1 files from tmpDir to manDir. Returns the count.
func copyManPages(tmpDir, manDir string) (int, error) {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read generated man pages: %w", err)
	}
	installed := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".1" {
			continue
		}
		srcPath := filepath.Join(tmpDir, entry.Name())
		dstPath := filepath.Join(manDir, entry.Name())
		//nolint:gosec // paths are controlled by --prefix flag
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s: %w", srcPath, err)
		}
		//nolint:gosec // man pages must be world-readable
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return 0, fmt.Errorf("failed to write %s: %w", dstPath, err)
		}
		installed++
	}
	return installed, nil
}

// installManPages generates man pages into a temp dir, copies them to manDir,
// and reports the count.
func installManPages(cmd *cobra.Command, prefix string) error {
	header := &doc.GenManHeader{
		Title:   "STAMP",
		Section: "1",
		Source:  fmt.Sprintf("stamp v%s", Version),
		Manual:  "Stamp Manual",
	}
	if prefix == "" {
		prefix = defaultManPrefix()
	}
	manDir := filepath.Join(prefix, "share", "man", "man1")
	if err := os.MkdirAll(manDir, 0750); err != nil {
		return fmt.Errorf("failed to create man directory %s: %w", manDir, err)
	}
	tmpDir, err := os.MkdirTemp("", "stamp-man-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := doc.GenManTree(cmd.Root(), header, tmpDir); err != nil {
		return fmt.Errorf("failed to generate man pages: %w", err)
	}
	installed, err := copyManPages(tmpDir, manDir)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "installed %d man page(s) to %s\n", installed, manDir)
	return nil
}

func newManInstallCmd() *cobra.Command {
	var prefix string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the stamp man page to system or user path",
		Example: `  # install to the default user path (~/.local/share/man)
  stamp man install

  # install under a custom prefix
  stamp man install --prefix /usr/local`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return installManPages(cmd, prefix)
		},
	}

	cmd.Flags().StringVar(&prefix, "prefix", "", "install prefix (default: ~/.local)")
	return cmd
}

// runManCheck checks the installed man page version and reports the result.
func runManCheck(cmd *cobra.Command) error {
	app := appFromCtx(cmd)
	tty := isOutputTerminal(cmd.ErrOrStderr())

	status, installedPath, err := checkInstalledManVersion()

	if app != nil && app.json {
		type jsonReport struct {
			Installed     bool   `json:"installed"`
			ManVersion    string `json:"man_version,omitempty"`
			BinaryVersion string `json:"binary_version"`
			Match         bool   `json:"match"`
			Error         string `json:"error,omitempty"`
		}
		report := jsonReport{BinaryVersion: Version}
		switch {
		case err != nil:
			report.Error = err.Error()
		case installedPath == "":
			report.Error = "not found"
		default:
			report.Installed = true
			report.ManVersion = status.version
			report.Match = status.matches
		}
		data, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	if err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", iconLine(tty, "✗", fmt.Sprintf("Error checking man page: %v", err)))
		return nil
	}
	if installedPath == "" {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), iconLine(tty, "✗", "Man page not installed. Run 'stamp man install' to install."))
		return nil
	}
	if status.matches {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", iconLine(tty, "✓", fmt.Sprintf("Man page is up to date (%s)", status.version)))
	} else {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Man page is outdated (installed %s, current v%s). Run 'stamp man install' to update.\n", status.version, Version)
	}
	return nil
}

func newManCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Verify installed man page version matches current stamp version",
		Example: `  # check the installed man page version
  stamp man check

  # machine-readable check result
  stamp man check --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runManCheck(cmd)
		},
	}
	return cmd
}

func defaultManPrefix() string {
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".local")
	}
	return "/usr/local"
}
