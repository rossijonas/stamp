package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
	"github.com/rossijonas/stamp/internal/state"
)

func renderNoBaselineDryRun(w io.Writer) {
	_, _ = fmt.Fprintln(w, "No baseline snapshot exists. Run without --dry-run to take baseline.")
}

// renderReconcileBanner frames reconcile as a fallback and steers users toward
// `stamp install` as the primary intent-tracking path.
func renderReconcileBanner(w io.Writer) {
	_, _ = fmt.Fprintln(w, "note: reconcile is a fallback for packages installed outside stamp.")
	_, _ = fmt.Fprintln(w, "      prefer 'stamp install <pkg> -m <mgr>' so intent is tracked from day one.")
}

// renderReliabilityNotes prints reliability warnings for adapters whose
// ListInstalled output is over-inclusive. Always shown so unexpected drift is
// not mistaken for real package changes.
func renderReliabilityNotes(w io.Writer, adapters []manager.Adapter) {
	for _, a := range adapters {
		if r, ok := a.(manager.ReliabilityReporter); ok && r.ReconcileReliability() == manager.ReliabilityOverInclusive {
			_, _ = fmt.Fprintf(w, "note: %s lists all installed packages; reconcile may detect system packages\n", a.Name())
		}
	}
}

func renderBaselineTaken(w io.Writer) {
	_, _ = fmt.Fprintln(w, "initial baseline snapshot taken")
}

func renderNoDrift(w io.Writer) {
	_, _ = fmt.Fprintln(w, "No drift detected")
}

func renderDiscovered(w io.Writer, pkgs []discoveredPkg, repos []discoveredRepo) {
	if len(pkgs) > 0 {
		_, _ = fmt.Fprintf(w, "Discovered %d new package(s):\n", len(pkgs))
		for _, p := range pkgs {
			_, _ = fmt.Fprintf(w, "  - %s (%s)\n", p.Name, p.Manager)
		}
	}
	if len(repos) > 0 {
		_, _ = fmt.Fprintf(w, "Discovered %d new repository(ies):\n", len(repos))
		for _, r := range repos {
			_, _ = fmt.Fprintf(w, "  - %s (%s)\n", r.Name, r.Manager)
		}
	}
}

func renderDryRunHint(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Use `stamp reconcile` without --dry-run to track")
}

func renderTrackedSummary(w io.Writer, pkgCount, repoCount int) {
	_, _ = fmt.Fprintf(w, "Tracked %d package(s), %d repository(ies)\n", pkgCount, repoCount)
}

// printSnapshotWarnings prints non-fatal warnings from the snapshot pipeline to stderr/writer.
func printSnapshotWarnings(w io.Writer, snaps []state.Snapshot) {
	for _, s := range snaps {
		for _, warn := range s.Warnings {
			_, _ = fmt.Fprintf(w, "warning: %s: %s\n", s.Manager, warn)
		}
	}
}

// renderMissing warns that manifest-tracked packages are absent from the
// system. Names are listed inline up to a small count, then the message points
// at 'stamp ls --type missing' and 'stamp restore' to keep it terse.
func renderMissing(w io.Writer, pkgs []manifest.Package) {
	if len(pkgs) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "warning: %d manifest package(s) not installed", len(pkgs))
	if len(pkgs) <= 5 {
		names := make([]string, 0, len(pkgs))
		for _, p := range pkgs {
			names = append(names, fmt.Sprintf("%s (%s)", p.Name, p.Manager))
		}
		_, _ = fmt.Fprintf(w, ": %s\n", strings.Join(names, ", "))
	} else {
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, "         run 'stamp ls --type missing' for the full list, or 'stamp restore' to reinstall")
}
