package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/rossijonas/stamp/internal/manager"
	"github.com/rossijonas/stamp/internal/manifest"
)

type managerStatus struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Path    string `json:"path"`
	Details string `json:"details"`
}

type manifestStatus struct {
	Path          string             `json:"path"`
	Valid         bool               `json:"valid"`
	PackagesCount int                `json:"packages_count"`
	Error         string             `json:"error,omitempty"`
	Missing       []manifest.Package `json:"missing,omitempty"`
}

type configStatus struct {
	Path  string `json:"path"`
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type completionStatus struct {
	Installed bool     `json:"installed"`
	Shells    []string `json:"shells,omitempty"`
}

type doctorReport struct {
	System          string           `json:"system"`
	Version         string           `json:"version"`
	PackageManagers []managerStatus  `json:"package_managers"`
	Manifest        manifestStatus   `json:"manifest"`
	Config          configStatus     `json:"config"`
	NoColor         bool             `json:"no_color"`
	ManPage         manPageStatus    `json:"man_page"`
	Completions     completionStatus `json:"completions"`
}

func checkCompletionStatus() completionStatus {
	shells := map[string]string{
		"bash": completionPath("bash"),
		"zsh":  completionPath("zsh"),
		"fish": completionPath("fish"),
	}
	var installed []string
	for name, path := range shells {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			installed = append(installed, name)
		}
	}
	return completionStatus{
		Installed: len(installed) > 0,
		Shells:    installed,
	}
}

func buildManagersReport(adapters []manager.Adapter) []managerStatus {
	adapterNames := make(map[string]bool)
	for _, a := range adapters {
		adapterNames[a.Name()] = true
	}

	knownManagers := []struct {
		name    string
		details string
	}{
		{"apt", "Default system manager (Debian/Ubuntu)"},
		{"dnf", "Default system manager (Fedora/RHEL, alias yum)"},
		{"pacman", "Package manager for Arch Linux"},
		{"paru", "AUR helper for Arch Linux (replaces pacman)"},
		{"zypper", "Package manager for openSUSE/SLES"},
		{"snap", "Universal Linux package manager"},
		{"flatpak", "Sandboxed application distribution"},
		{"brew", "User-space manager"},
		{"macports", "Package manager for macOS"},
		{"go", "Language toolchain — go install"},
		{"npm", "Node.js package manager"},
		{"cargo", "Rust package manager — cargo install"},
		{"pipx", "Python tool installer"},
		{"uv", "Python package manager"},
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
	return managers
}

func buildManifestReport(path string, manifestErr error, manifest *manifest.Manifest) manifestStatus {
	ms := manifestStatus{Path: path}
	if manifestErr != nil {
		ms.Error = manifestErr.Error()
	} else {
		_, statErr := os.Stat(path)
		if os.IsNotExist(statErr) {
			ms.Error = "manifest not found — run 'stamp init'"
		} else {
			ms.Valid = true
			ms.PackagesCount = len(manifest.Packages)
		}
	}
	return ms
}

func newDoctorCmd() *cobra.Command {
	var managerFlag string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose system configuration and manifest health",
		Example: `  # check the whole system
  stamp doctor

  # machine-readable output for scripting
  stamp doctor --json

  # check a single manager's native diagnostics
  stamp doctor -m dnf`,
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
					return catErr(ErrUnavailable, "manager %q not available on this system", managerFlag)
				}
				result, err := adapter.Doctor(cmd.Context())
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s doctor:\n\n%s\n", managerFlag, result)
				return nil
			}

			managers := buildManagersReport(app.adapters)
			ms := buildManifestReport(app.manifestPath, app.manifestErr, app.manifest)
			if ms.Valid {
				ms.Missing = missingFromSystem(cmd.Context(), app.adapters, app.manifest)
			}
			cs := configStatus{Path: app.configPath}
			if err := app.config.Backup.Validate(); err != nil {
				cs.Error = err.Error()
			} else {
				cs.Valid = true
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

			comps := checkCompletionStatus()

			report := doctorReport{
				System:          runtime.GOOS,
				Version:         Version,
				PackageManagers: managers,
				Manifest:        ms,
				Config:          cs,
				NoColor:         app.noColor,
				ManPage: manPageStatus{
					Installed: mpInstalled,
					Path:      mpPath,
					Version:   mpVersion,
					Matches:   mpMatches,
				},
				Completions: comps,
			}

			if app.json {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal doctor report: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			renderDoctorTTY(cmd.OutOrStdout(), &report, app.noColor)
			return nil
		},
	}

	cmd.Flags().StringVarP(&managerFlag, "manager", "m", "", "package manager to check")
	return cmd
}
