package cli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
)

type managerStatus struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Path    string `json:"path"`
	Details string `json:"details"`
}

type manifestStatus struct {
	Path          string `json:"path"`
	Valid         bool   `json:"valid"`
	PackagesCount int    `json:"packages_count"`
	Error         string `json:"error,omitempty"`
}

type doctorReport struct {
	System          string          `json:"system"`
	Version         string          `json:"version"`
	PackageManagers []managerStatus `json:"package_managers"`
	Manifest        manifestStatus  `json:"manifest"`
	NoColor         bool            `json:"no_color"`
	ManPage         manPageStatus   `json:"man_page"`
}

func newDoctorCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose system configuration and manifest health",
		Long: `Check package manager availability and manifest integrity.
Reports which managers are installed and whether the manifest is valid.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCtx(cmd)
			if app == nil {
				return fmt.Errorf("app context not initialized")
			}

			if managerFlag != "" {
				var adapter manager.Adapter
				for _, a := range app.adapters {
					if a.Name() == manager.ResolveManager(managerFlag) {
						adapter = a
						break
					}
				}
				if adapter == nil {
					return fmt.Errorf("manager %q not available on this system", managerFlag)
				}
				result, err := adapter.Doctor(cmd.Context())
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s doctor:\n\n%s\n", managerFlag, result)
				return nil
			}

			adapterNames := make(map[string]bool)
			for _, a := range app.adapters {
				adapterNames[a.Name()] = true
			}

			knownManagers := []struct {
				name    string
				details string
			}{
				{"dnf", "Default system manager"},
				{"brew", "User-space manager"},
				{"flatpak", "Sandboxed application distribution"},
			}

			managers := make([]managerStatus, 0, len(knownManagers))
			for _, km := range knownManagers {
				status := managerStatus{
					Name:    km.name,
					Details: "Executable not found in $PATH",
				}
				if adapterNames[km.name] {
					status.Active = true
					status.Details = km.details
					path, err := exec.LookPath(km.name)
					if err == nil {
						status.Path = path
					}
				}
				managers = append(managers, status)
			}

			ms := manifestStatus{Path: app.manifestPath}
			if app.manifestErr != nil {
				ms.Error = app.manifestErr.Error()
			} else {
				ms.Valid = true
				ms.PackagesCount = len(app.manifest.Packages)
			}

			var mpInstalled bool
			var mpPath string
			var mpVersion string
			var mpMatches bool
			status, mp, _ := checkInstalledManVersion()
			if mp != "" {
				mpInstalled = true
				mpPath = mp
				mpVersion = status.version
				mpMatches = status.matches
			}

			if app.json {
				report := doctorReport{
					System:          runtime.GOOS,
					Version:         Version,
					PackageManagers: managers,
					Manifest:        ms,
					NoColor:         app.noColor,
					ManPage: manPageStatus{
						Installed: mpInstalled,
						Path:      mpPath,
						Version:   mpVersion,
						Matches:   mpMatches,
					},
				}
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal doctor report: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprint(out, "▪ System Diagnosis (Stamp Doctor)\n\n")

			_, _ = fmt.Fprintln(out, "Package Managers:")
			_, _ = fmt.Fprintf(out, "  %-10s %-10s %-22s %s\n", "Name", "Status", "Path", "Details")
			for _, m := range managers {
				statusSymbol := "❌ Not Found"
				path := "-"
				if m.Active {
					statusSymbol = "✅ Active"
					if m.Path != "" {
						path = m.Path
					}
				}
				_, _ = fmt.Fprintf(out, "  %-10s %-10s %-22s %s\n", m.Name, statusSymbol, path, m.Details)
			}

			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out, "Manifest Integrity:")
			_, _ = fmt.Fprintf(out, "  Path:   %s\n", ms.Path)
			if ms.Valid {
				_, _ = fmt.Fprintf(out, "  Status: ✅ Healthy (%d package(s))\n", ms.PackagesCount)
			} else {
				_, _ = fmt.Fprintf(out, "  Status: ❌ %s\n", ms.Error)
			}

			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out, "UNIX Compliance:")
			if app.noColor {
				_, _ = fmt.Fprintln(out, "  NO_COLOR: ✅ Set")
			} else {
				_, _ = fmt.Fprintln(out, "  NO_COLOR: ❌ Not set")
			}
			_, _ = fmt.Fprintf(out, "  Version:  stamp %s\n", Version)

			if mpInstalled {
				if mpMatches {
					_, _ = fmt.Fprintf(out, "  Man Page: ✅ Up to date (%s)\n", mpVersion)
				} else {
					_, _ = fmt.Fprintf(out, "  Man Page: ⚠️ Outdated (installed %s, current %s) — run 'stamp man install'\n", mpVersion, Version)
				}
			} else {
				_, _ = fmt.Fprintln(out, "  Man Page: ❌ Not found — run 'stamp man install'")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to check")
	return cmd
}
