package cli

import (
	"fmt"
	"io"
)

func renderNoBaselineDryRun(w io.Writer) {
	_, _ = fmt.Fprintln(w, "No baseline snapshot exists. Run without --dry-run to take baseline.")
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
