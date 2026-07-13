package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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

func installedManPagePath() string {
	for _, p := range manPageCandidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func checkInstalledManVersion() (*manCheckResult, string, error) {
	path := installedManPagePath()
	if path == "" {
		return nil, "", nil
	}

	//nolint:gosec // path is resolved safely within candidate list
	data, err := os.ReadFile(path)
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
		matches: ver == Version,
	}, path, nil
}

func newManCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "man",
		Short: "Manage stamp troff man pages",
		Long:  `Command group to generate, install, and check stamp man pages.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newManInstallCmd())
	cmd.AddCommand(newManCheckCmd())
	return cmd
}

func newManInstallCmd() *cobra.Command {
	var prefix string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the stamp man page to system or user path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			header := &doc.GenManHeader{
				Title:   "STAMP",
				Section: "1",
				Source:  fmt.Sprintf("stamp %s", Version),
				Manual:  "Stamp Manual",
			}

			if prefix == "" {
				prefix = defaultManPrefix()
			}

			manDir := filepath.Join(prefix, "share", "man", "man1")
			if err := os.MkdirAll(manDir, 0750); err != nil {
				return fmt.Errorf("failed to create man directory %s: %w", manDir, err)
			}

			manPath := filepath.Join(manDir, "stamp.1")
			//nolint:gosec // path is controlled by --prefix flag
			f, err := os.Create(manPath)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", manPath, err)
			}
			defer func() { _ = f.Close() }()

			if err := doc.GenMan(cmd.Root(), header, f); err != nil {
				return fmt.Errorf("failed to generate man page: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "man page installed to %s\n", manPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&prefix, "prefix", "", "install prefix (default: ~/.local)")
	return cmd
}

func newManCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Verify installed man page version matches current stamp version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)

			status, installedPath, err := checkInstalledManVersion()
			if app != nil && app.json {
				type jsonReport struct {
					Installed     bool   `json:"installed"`
					ManVersion    string `json:"man_version,omitempty"`
					BinaryVersion string `json:"binary_version"`
					Match         bool   `json:"match"`
					Error         string `json:"error,omitempty"`
				}
				report := jsonReport{
					BinaryVersion: Version,
				}
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
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "❌ Error checking man page: %v\n", err)
				return nil
			}

			if installedPath == "" {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "❌ Man page not installed. Run 'stamp man install' to install.")
				return nil
			}

			if status.matches {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "✅ Man page is up to date (%s)\n", status.version)
			} else {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "⚠️ Man page is outdated (installed %s, current %s). Run 'stamp man install' to update.\n", status.version, Version)
			}

			return nil
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
