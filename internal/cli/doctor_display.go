package cli

import (
	"fmt"
	"io"
	"strings"
)

func renderDoctorTTY(w io.Writer, d *doctorReport, noColor bool) {
	tty := isOutputTerminal(w)
	_, _ = fmt.Fprintln(w, iconLine(tty, "▪", "System Diagnosis (Stamp Doctor)"))
	_, _ = fmt.Fprintln(w)

	_, _ = fmt.Fprintln(w, "Package Managers:")
	_, _ = fmt.Fprintf(w, "  %-10s %-10s %-22s %s\n", "Name", "Status", "Path", "Details")
	for _, m := range d.PackageManagers {
		statusSymbol := iconLine(tty, "✗", "Not Found")
		path := "-"
		if m.Active {
			statusSymbol = iconLine(tty, "✓", "Active")
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
		_, _ = fmt.Fprintf(w, "  Status: %s\n", iconLine(tty, "✓", fmt.Sprintf("Healthy (%d package(s))", d.Manifest.PackagesCount)))
		if len(d.Manifest.Missing) > 0 {
			_, _ = fmt.Fprintln(w, "  Missing:")
			for _, p := range d.Manifest.Missing {
				_, _ = fmt.Fprintf(w, "    - %s (%s)\n", p.Name, p.Manager)
			}
			_, _ = fmt.Fprintln(w, "    run 'stamp restore' to reinstall, or 'stamp ls --type missing' for details")
		}
	} else {
		_, _ = fmt.Fprintf(w, "  Status: %s\n", iconLine(tty, "✗", d.Manifest.Error))
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Configuration:")
	_, _ = fmt.Fprintf(w, "  Path:   %s\n", d.Config.Path)
	if d.Config.Valid {
		_, _ = fmt.Fprintf(w, "  Status: %s\n", iconLine(tty, "✓", "Valid"))
	} else {
		_, _ = fmt.Fprintf(w, "  Status: %s\n", iconLine(tty, "✗", "invalid [backup] config:"))
		for _, line := range strings.Split(d.Config.Error, "\n") {
			_, _ = fmt.Fprintf(w, "    %s\n", line)
		}
		_, _ = fmt.Fprintln(w, "    run 'stamp doctor' after fixing config.toml")
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "UNIX Compliance:")
	if noColor {
		_, _ = fmt.Fprintf(w, "  NO_COLOR: %s\n", iconLine(tty, "✓", "Set"))
	} else {
		_, _ = fmt.Fprintf(w, "  NO_COLOR: %s\n", iconLine(tty, "✗", "Not set"))
	}
	_, _ = fmt.Fprintf(w, "  Version:  stamp v%s\n", d.Version)

	if d.ManPage.Installed {
		if d.ManPage.Matches {
			_, _ = fmt.Fprintf(w, "  Man Page: %s\n", iconLine(tty, "✓", fmt.Sprintf("Up to date (%s)", d.ManPage.Version)))
		} else {
			_, _ = fmt.Fprintf(w, "  Man Page: ⚠ Outdated (installed %s, current v%s) — run 'stamp man install'\n", d.ManPage.Version, d.Version)
		}
	} else {
		_, _ = fmt.Fprintf(w, "  Man Page: %s\n", iconLine(tty, "✗", "Not found — run 'stamp man install'"))
	}

	if d.Completions.Installed {
		_, _ = fmt.Fprintf(w, "  Completions: %s\n", iconLine(tty, "✓", fmt.Sprintf("Installed (%s)", strings.Join(d.Completions.Shells, ", "))))
	} else {
		_, _ = fmt.Fprintf(w, "  Completions: %s\n", iconLine(tty, "✗", "Not installed — run 'stamp completion'"))
	}
}
