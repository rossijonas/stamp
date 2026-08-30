package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/backup"
	"github.com/rossijonas/stamp/internal/manifest"
)

const (
	// historyTimeLayout is the human-readable timestamp shown by history and
	// accepted by diff. Backup filenames use backup.BackupTimeLayout instead.
	historyTimeLayout = "2006-01-02T15:04:05Z"
	// hashPrefixLen is how many hex characters of a backup's SHA-256 are shown.
	hashPrefixLen = 12
	// minHashPrefix is the shortest accepted content-hash prefix for diff.
	minHashPrefix = 6
	// errNoBackupFound is the common error message when no backup matches a target.
	errNoBackupFound = "no backup found for %s"
)

// historyEntry is one row of `stamp manifest history`.
type historyEntry struct {
	Timestamp string `json:"timestamp"`
	Hash      string `json:"hash"`
	Current   bool   `json:"current"`
	Unchanged bool   `json:"unchanged,omitempty"`
	Packages  int    `json:"packages"`
	Repos     int    `json:"repos"`
}

// diffItem is one added/removed entry in `stamp manifest diff`.
type diffItem struct {
	Name    string `json:"name"`
	Manager string `json:"manager"`
	Origin  string `json:"origin"`
	Kind    string `json:"kind"`
}

// diffResult is the JSON shape of `stamp manifest diff`.
type diffResult struct {
	Baseline string     `json:"baseline"`
	Added    []diffItem `json:"added"`
	Removed  []diffItem `json:"removed"`
}

// contentHash returns the lowercase hex SHA-256 of the file at path.
func contentHash(path string) (string, error) {
	//nolint:gosec // path is the manifest or a backup resolved via internal config
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func shortHash(h string) string {
	if len(h) <= hashPrefixLen {
		return h
	}
	return h[:hashPrefixLen]
}

// requireManifest fails with a friendly message when the manifest file does
// not exist. Parse errors surface via AppContext.manifestErr instead.
func requireManifest(app *AppContext) error {
	if _, err := os.Stat(app.manifestPath); err != nil {
		if os.IsNotExist(err) {
			return catErr(ErrConfig, "manifest not found; run stamp init first")
		}
		return catErr(ErrConfig, "failed to access manifest: %w", err)
	}
	return nil
}

// currentManifestTimestamp returns the current manifest's UpdatedAt as a
// human-readable timestamp, falling back to the file modification time for
// pre-history manifests that lack a populated updated_at field.
func currentManifestTimestamp(app *AppContext) string {
	if !app.manifest.UpdatedAt.IsZero() {
		return app.manifest.UpdatedAt.UTC().Format(historyTimeLayout)
	}
	if fi, err := os.Stat(app.manifestPath); err == nil {
		return fi.ModTime().UTC().Format(historyTimeLayout)
	}
	return "unknown"
}

// buildHistoryEntries builds the list of history entries (current + backups).
// Warnings for unreadable backups are emitted to warn (typically cmd.ErrOrStderr).
func buildHistoryEntries(app *AppContext, warn io.Writer) ([]historyEntry, error) {
	currentHash, err := contentHash(app.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to hash current manifest: %w", err)
	}

	entries := []historyEntry{{
		Timestamp: currentManifestTimestamp(app),
		Hash:      shortHash(currentHash),
		Current:   true,
		Packages:  len(app.manifest.Packages),
		Repos:     len(app.manifest.Repositories),
	}}

	backups, err := backup.List(app.manifestPath + ".*.bak")
	if err != nil {
		return nil, err
	}
	for _, b := range backups {
		m, mErr := manifest.Load(b.Path)
		if mErr != nil {
			_, _ = fmt.Fprintf(warn, "warning: skipping unreadable backup %s: %v\n", b.Path, mErr)
			continue
		}
		hash, hErr := contentHash(b.Path)
		if hErr != nil {
			_, _ = fmt.Fprintf(warn, "warning: skipping unreadable backup %s: %v\n", b.Path, hErr)
			continue
		}
		entries = append(entries, historyEntry{
			Timestamp: b.Time.Format(historyTimeLayout),
			Hash:      shortHash(hash),
			Unchanged: hash == currentHash,
			Packages:  len(m.Packages),
			Repos:     len(m.Repositories),
		})
	}
	return entries, nil
}

// renderHistoryText writes the human-readable history table to w.
func renderHistoryText(w io.Writer, entries []historyEntry) {
	_, _ = fmt.Fprintln(w, "Available manifest backups:")
	for _, e := range entries {
		marker := "  "
		if e.Current {
			marker = "* "
		}
		line := fmt.Sprintf("%s%s %s", marker, e.Timestamp, e.Hash)
		if e.Current {
			line += fmt.Sprintf("  %d packages, %d repos  (current)", e.Packages, e.Repos)
		} else {
			line += fmt.Sprintf("  %d packages, %d repos", e.Packages, e.Repos)
			if e.Unchanged {
				line += "  (unchanged)"
			}
		}
		_, _ = fmt.Fprintln(w, line)
	}
	if len(entries) == 1 {
		_, _ = fmt.Fprintln(w, "No backups found. Backups are created on re-init and reconcile.")
	}
}

// buildDiffItems converts added/removed packages and repos into diffItems.
func buildDiffItems(addedPkgs, removedPkgs []manifest.Package, addedRepos, removedRepos []manifest.Repository) (added, removed []diffItem) {
	added = make([]diffItem, 0, len(addedPkgs)+len(addedRepos))
	for _, p := range addedPkgs {
		added = append(added, diffItem{Name: p.Name, Manager: p.Manager, Origin: p.OriginEffective(), Kind: "package"})
	}
	for _, r := range addedRepos {
		added = append(added, diffItem{Name: r.Name, Manager: r.Manager, Origin: r.OriginEffective(), Kind: "repo"})
	}
	removed = make([]diffItem, 0, len(removedPkgs)+len(removedRepos))
	for _, p := range removedPkgs {
		removed = append(removed, diffItem{Name: p.Name, Manager: p.Manager, Origin: p.OriginEffective(), Kind: "package"})
	}
	for _, r := range removedRepos {
		removed = append(removed, diffItem{Name: r.Name, Manager: r.Manager, Origin: r.OriginEffective(), Kind: "repo"})
	}
	return
}

// renderDiffText writes the human-readable diff table to w.
func renderDiffText(w io.Writer, baselineLabel string, added, removed []diffItem) {
	_, _ = fmt.Fprintf(w, "Comparing: current vs %s\n\n", baselineLabel)
	if len(added) == 0 && len(removed) == 0 {
		_, _ = fmt.Fprintln(w, "no differences")
		return
	}
	for _, a := range added {
		_, _ = fmt.Fprintf(w, "+ %s (%s)\n", a.Name, a.Manager)
	}
	for _, r := range removed {
		_, _ = fmt.Fprintf(w, "- %s (%s)\n", r.Name, r.Manager)
	}
	_, _ = fmt.Fprintf(w, "\n  %d added, %d removed\n", len(added), len(removed))
}

// matchHashPrefix finds backups whose content hash starts with the given prefix.
func matchHashPrefix(target string, backups []backup.Entry) ([]backup.Entry, error) {
	matched := []backup.Entry{}
	for _, b := range backups {
		h, err := contentHash(b.Path)
		if err != nil {
			continue
		}
		if strings.HasPrefix(h, strings.ToLower(target)) {
			matched = append(matched, b)
		}
	}
	if len(matched) == 0 {
		return nil, catErr(ErrNoInput, errNoBackupFound, target)
	}
	if len(matched) > 1 {
		labels := make([]string, 0, len(matched))
		for _, m := range matched {
			labels = append(labels, m.Time.Format(historyTimeLayout))
		}
		return nil, catErr(ErrData, "ambiguous hash %s; matches: %s", target, strings.Join(labels, ", "))
	}
	return matched, nil
}

// matchTimestamp finds the backup matching a parsed timestamp.
func matchTimestamp(target string, backups []backup.Entry) (string, string, error) {
	ts, ok := parseBackupTimestamp(target)
	if !ok {
		return "", "", catErr(ErrUsage, errNoBackupFound, target)
	}
	for _, b := range backups {
		if b.Time.Equal(ts) {
			return b.Path, b.Time.Format(historyTimeLayout), nil
		}
	}
	return "", "", catErr(ErrNoInput, errNoBackupFound, target)
}

func newManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Inspect manifest backups and changes",
		Example: `  # list manifest backups (newest first, with content hashes)
  stamp manifest history

  # diff the current manifest against a backup
  stamp manifest diff`,
		Long: `Command group for manifest management: list backup history and
diff the current manifest against a backup.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newManifestHistoryCmd())
	cmd.AddCommand(newManifestDiffCmd())
	return cmd
}

func newManifestHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "history",
		Aliases: []string{"backups"},
		Short:   "List manifest backups",
		Example: `  # list manifest backups, newest first
  stamp manifest history

  # machine-readable history
  stamp manifest history -j`,
		Long: `List the current manifest and all timestamped backups, newest first.
Each row shows the backup timestamp, a short content hash, and package/repo
counts. The current manifest is marked with '*'. Backups whose content is
identical to the current manifest are marked as unchanged.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}
			if err := requireManifest(app); err != nil {
				return err
			}

			entries, err := buildHistoryEntries(app, cmd.ErrOrStderr())
			if err != nil {
				return err
			}

			if app.json {
				data, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal manifest history: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			renderHistoryText(cmd.OutOrStdout(), entries)
			return nil
		},
	}
	return cmd
}

func newManifestDiffCmd() *cobra.Command {
	var (
		managerFlag string
		originFlag  string
	)

	cmd := &cobra.Command{
		Use:   "diff [timestamp|hash]",
		Short: "Compare current manifest against a backup",
		Example: `  # diff the current manifest against the most recent backup
  stamp manifest diff

  # diff against a specific backup by timestamp or hash prefix
  stamp manifest diff 2026-08-02T09:15:00Z
  stamp manifest diff a1b2c3d4e5f6

  # filter by manager and origin
  stamp manifest diff -m brew --origin stamped`,
		Long: `Show the difference between the current manifest and a specific backup.
Defaults to the most recent backup. The argument may be a backup timestamp
(2026-08-02T09:15:00Z or 20260802T091500Z) or a content-hash prefix shown by
stamp manifest history. Added entries are prefixed with '+', removed with '-'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCtx(cmd)
			if app.manifestErr != nil {
				return app.manifestErr
			}
			if err := requireManifest(app); err != nil {
				return err
			}
			if err := validateOriginFlag(originFlag); err != nil {
				return err
			}

			target := ""
			if len(args) == 1 {
				target = args[0]
			}

			baselinePath, baselineLabel, err := resolveBaseline(app.manifestPath, target)
			if err != nil {
				return err
			}
			baseline, err := manifest.Load(baselinePath)
			if err != nil {
				return catErr(ErrData, "failed to parse backup at %s: %w", baselinePath, err)
			}

			addedPkgs, removedPkgs, addedRepos, removedRepos := diffManifests(app.manifest, baseline)
			addedPkgs = filterPackages(addedPkgs, managerFlag, originFlag)
			removedPkgs = filterPackages(removedPkgs, managerFlag, originFlag)
			addedRepos = filterRepositories(addedRepos, managerFlag, originFlag)
			removedRepos = filterRepositories(removedRepos, managerFlag, originFlag)

			added, removed := buildDiffItems(addedPkgs, removedPkgs, addedRepos, removedRepos)

			if app.json {
				data, err := json.MarshalIndent(diffResult{Baseline: baselineLabel, Added: added, Removed: removed}, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal manifest diff: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			renderDiffText(cmd.OutOrStdout(), baselineLabel, added, removed)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "filter by package manager")
	cmd.Flags().StringVar(&originFlag, "origin", "", "filter by origin: stamped or reconciled")
	//nolint:errcheck // the --origin flag is registered above, so this cannot fail
	_ = cmd.RegisterFlagCompletionFunc("origin", originCompletion)
	return cmd
}

// originCompletion completes the diff --origin flag.
var originCompletion = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{manifest.OriginStamped, manifest.OriginReconciled}, cobra.ShellCompDirectiveNoFileComp
}

func validateOriginFlag(o string) error {
	if o == "" {
		return nil
	}
	if o != manifest.OriginStamped && o != manifest.OriginReconciled {
		return catErr(ErrUsage, "invalid origin %q; valid origins: %s, %s", o, manifest.OriginStamped, manifest.OriginReconciled)
	}
	return nil
}

// resolveBaseline resolves the backup to diff against. An empty target picks
// the most recent backup. A pure-hex target (>= 6 chars) is treated as a
// content-hash prefix; anything else is parsed as a timestamp in either the
// human (2026-08-02T09:15:00Z) or filename (20260802T091500Z) layout.
func resolveBaseline(manifestPath, target string) (path, label string, err error) {
	backups, err := backup.List(manifestPath + ".*.bak")
	if err != nil {
		return "", "", err
	}
	if len(backups) == 0 {
		return "", "", catErr(ErrNoInput, "no backup to compare against")
	}
	if target == "" {
		return backups[0].Path, backups[0].Time.Format(historyTimeLayout), nil
	}

	if isHexHash(target) {
		matched, mErr := matchHashPrefix(target, backups)
		if mErr != nil {
			return "", "", mErr
		}
		return matched[0].Path, matched[0].Time.Format(historyTimeLayout), nil
	}

	return matchTimestamp(target, backups)
}

// isHexHash reports whether s looks like a content-hash prefix: at least
// minHashPrefix hex characters and nothing else. Compact timestamps contain
// 'T' and 'Z', so they never match.
func isHexHash(s string) bool {
	if len(s) < minHashPrefix {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// parseBackupTimestamp parses a diff target in the human or filename layout.
func parseBackupTimestamp(s string) (time.Time, bool) {
	for _, layout := range []string{historyTimeLayout, backup.BackupTimeLayout} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

// diffManifests returns entries present in current but absent from baseline
// (added) and vice versa (removed), for both packages and repositories.
// Identity is name+manager, matching manifest.HasPackage/HasRepository.
func diffManifests(current, baseline *manifest.Manifest) (addedPkgs, removedPkgs []manifest.Package, addedRepos, removedRepos []manifest.Repository) {
	currentPkgs := pkgKeySet(current.Packages)
	baselinePkgs := pkgKeySet(baseline.Packages)
	for _, p := range current.Packages {
		if !baselinePkgs[keyOf(p.Name, p.Manager)] {
			addedPkgs = append(addedPkgs, p)
		}
	}
	for _, p := range baseline.Packages {
		if !currentPkgs[keyOf(p.Name, p.Manager)] {
			removedPkgs = append(removedPkgs, p)
		}
	}

	currentRepos := repoKeySet(current.Repositories)
	baselineRepos := repoKeySet(baseline.Repositories)
	for _, r := range current.Repositories {
		if !baselineRepos[keyOf(r.Name, r.Manager)] {
			addedRepos = append(addedRepos, r)
		}
	}
	for _, r := range baseline.Repositories {
		if !currentRepos[keyOf(r.Name, r.Manager)] {
			removedRepos = append(removedRepos, r)
		}
	}
	return
}

func keyOf(name, manager string) string {
	return name + "\x00" + manager
}

func pkgKeySet(pkgs []manifest.Package) map[string]bool {
	set := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		set[keyOf(p.Name, p.Manager)] = true
	}
	return set
}

func repoKeySet(repos []manifest.Repository) map[string]bool {
	set := make(map[string]bool, len(repos))
	for _, r := range repos {
		set[keyOf(r.Name, r.Manager)] = true
	}
	return set
}
