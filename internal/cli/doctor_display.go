package cli

import (
	"fmt"
	"io"
	"strings"
)

func renderDoctorTTY(w io.Writer, d *doctorReport, noColor bool) {
	_, _ = fmt.Fprint(w, "▪ System Diagnosis (Stamp Doctor)\n\n")

	_, _ = fmt.Fprintln(w, "Package Managers:")
	_, _ = fmt.Fprintf(w, "  %-10s %-10s %-22s %s\n", "Name", "Status", "Path", "Details")
	for _, m := range d.PackageManagers {
		statusSymbol := "❌ Not Found"
		path := "-"
		if m.Active {
			statusSymbol = "✅ Active"
			if m.Path != "" {
				path = m.Path
			}
		}
		_, _ = fmt.Fprintf(w, "  %-10s %-10s %-22s %s\n", m.Name, statusSymbol, path, m.Details)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Manifest Integrity:")
	_, _ = fmt.Fprintf(w, "  Path:   %s\n", d.Manifest.Path)
	if d.Manifest.Valid {
		_, _ = fmt.Fprintf(w, "  Status: ✅ Healthy (%d package(s))\n", d.Manifest.PackagesCount)
	} else {
		_, _ = fmt.Fprintf(w, "  Status: ❌ %s\n", d.Manifest.Error)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "UNIX Compliance:")
	if noColor {
		_, _ = fmt.Fprintln(w, "  NO_COLOR: ✅ Set")
	} else {
		_, _ = fmt.Fprintln(w, "  NO_COLOR: ❌ Not set")
	}
	_, _ = fmt.Fprintf(w, "  Version:  stamp %s\n", d.Version)

	if d.ManPage.Installed {
		if d.ManPage.Matches {
			_, _ = fmt.Fprintf(w, "  Man Page: ✅ Up to date (%s)\n", d.ManPage.Version)
		} else {
			_, _ = fmt.Fprintf(w, "  Man Page: ⚠️ Outdated (installed %s, current %s) — run 'stamp man install'\n", d.ManPage.Version, d.Version)
		}
	} else {
		_, _ = fmt.Fprintln(w, "  Man Page: ❌ Not found — run 'stamp man install'")
	}

	if d.Completions.Installed {
		_, _ = fmt.Fprintf(w, "  Completions: ✅ Installed (%s)\n", strings.Join(d.Completions.Shells, ", "))
	} else {
		_, _ = fmt.Fprintln(w, "  Completions: ❌ Not installed — run 'stamp completion'")
	}
}
