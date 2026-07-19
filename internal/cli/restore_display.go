package cli

import (
	"fmt"
	"io"

	"github.com/rossijonas/stamp/internal/manifest"
)

func renderRestoreDryRun(w io.Writer, repos []manifest.Repository, pkgs []manifest.Package) {
	_, _ = fmt.Fprintln(w, "▪ Dry Run (Preview):")
	if len(repos) > 0 {
		_, _ = fmt.Fprintln(w, "Repositories:")
		for _, r := range repos {
			_, _ = fmt.Fprintf(w, "  - %s (%s) %s\n", r.Name, r.Manager, r.URL)
		}
	}
	if len(pkgs) > 0 {
		_, _ = fmt.Fprintln(w, "Packages:")
		for _, p := range pkgs {
			_, _ = fmt.Fprintf(w, "  - %s (%s)\n", p.Name, p.Manager)
		}
	}
}

func renderRestoreErrors(w io.Writer, errs []restoreError) {
	_, _ = fmt.Fprintln(w, "Some packages failed to restore:")
	for _, e := range errs {
		_, _ = fmt.Fprintf(w, "  - %s (%s): %v\n", e.Pkg, e.Manager, e.Err)
	}
}

func renderRestoreComplete(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Restore completed successfully")
}
