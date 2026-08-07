package cli

import (
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Overridable in tests for deterministic platform behavior.
var currentOS = runtime.GOOS

// Overridable in tests to capture systemctl/launchctl calls.
var execCommand = osexec.Command

// osExecutable is declared in selfupdate.go
const systemdServiceTemplate = `[Unit]
Description=Stamp auto-reconcile
Documentation=https://gostamp.dev

[Service]
Type=oneshot
ExecStart=%s reconcile
`

const systemdTimerTemplate = `[Unit]
Description=Stamp auto-reconcile timer
Documentation=https://gostamp.dev

[Timer]
OnCalendar=%s
Persistent=true

[Install]
WantedBy=timers.target
`

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.gostamp.stamp-reconcile</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>reconcile</string>
    </array>
    <key>StartInterval</key>
    <integer>%d</integer>
    <key>StandardOutPath</key>
    <string>/tmp/stamp-reconcile.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/stamp-reconcile.log</string>
</dict>
</plist>`

func newAutoReconcileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-reconcile",
		Short: "Manage automated reconcile timer",
		Example: `  # install the automated reconcile timer (hourly by default)
  stamp auto-reconcile on

  # run on a weekly schedule
  stamp auto-reconcile on --period weekly

  # remove the timer
  stamp auto-reconcile off`,
		Long: `Install or remove a system timer that runs stamp reconcile automatically.
On Linux: uses systemd user timer.
On macOS: uses launchd agent.`,
	}
	cmd.AddCommand(newAutoReconcileOnCmd())
	cmd.AddCommand(newAutoReconcileOffCmd())
	return cmd
}

func newAutoReconcileOnCmd() *cobra.Command {
	var period string

	cmd := &cobra.Command{
		Use:   "on",
		Short: "Install automated reconcile timer",
		Example: `  # install the automated reconcile timer (daily by default)
  stamp auto-reconcile on

  # run on a different schedule
  stamp auto-reconcile on --period hourly
  stamp auto-reconcile on --period weekly`,
		Long: `Install a system timer to run stamp reconcile automatically.
Use --period to set the interval (daily, hourly, weekly).
On Linux: creates systemd user service + timer.
On macOS: creates a launchd agent.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			period = strings.ToLower(period)
			switch period {
			case "hourly", "daily", "weekly":
			default:
				return fmt.Errorf("invalid period %q (valid: hourly, daily, weekly)", period)
			}

			bin, err := osExecutable()
			if err != nil {
				return fmt.Errorf("failed to resolve stamp binary path: %w", err)
			}
			bin, err = filepath.EvalSymlinks(bin)
			if err != nil {
				return fmt.Errorf("failed to resolve symlinks: %w", err)
			}

			switch currentOS {
			case "linux":
				return installSystemdTimer(cmd.ErrOrStderr(), bin, period)
			case "darwin":
				return installLaunchdPlist(cmd.ErrOrStderr(), bin, period)
			default:
				return fmt.Errorf("auto-reconcile not supported on %s", currentOS)
			}
		},
	}

	cmd.Flags().StringVarP(&period, "period", "p", "daily", "timer period: hourly, daily, weekly")
	return cmd
}

func newAutoReconcileOffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "off",
		Short: "Remove automated reconcile timer",
		Example: `  # remove the automated reconcile timer
  stamp auto-reconcile off`,
		Long: `Remove the timer installed by stamp auto-reconcile on.
Stops the active timer and deletes the timer files.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch currentOS {
			case "linux":
				return removeSystemdTimer(cmd.ErrOrStderr())
			case "darwin":
				return removeLaunchdPlist(cmd.ErrOrStderr())
			default:
				return fmt.Errorf("auto-reconcile not supported on %s", currentOS)
			}
		},
	}
	return cmd
}

func timerDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch currentOS {
	case "linux":
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "systemd", "user"), nil
		}
		return filepath.Join(home, ".config", "systemd", "user"), nil
	case "darwin":
		return filepath.Join(home, "Library", "LaunchAgents"), nil
	default:
		return "", fmt.Errorf("auto-reconcile not supported on %s", currentOS)
	}
}

func installSystemdTimer(w io.Writer, bin, period string) error {
	dir, err := timerDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create timer directory: %w", err)
	}

	serviceContent := fmt.Sprintf(systemdServiceTemplate, bin)
	servicePath := filepath.Join(dir, "stamp-reconcile.service")
	if err := atomicWriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}

	cal := systemdCalendar(period)
	timerContent := fmt.Sprintf(systemdTimerTemplate, cal)
	timerPath := filepath.Join(dir, "stamp-reconcile.timer")
	if err := atomicWriteFile(timerPath, []byte(timerContent), 0644); err != nil {
		return err
	}

	if err := systemctl("daemon-reload"); err != nil {
		return fmt.Errorf("systemd user session unavailable: %w\n  Timer files installed to %s — activate manually with: systemctl --user enable --now stamp-reconcile.timer", err, dir)
	}
	if err := systemctl("enable", "--now", "stamp-reconcile.timer"); err != nil {
		_, _ = fmt.Fprintf(w, "warning: timer installed but activation failed: %v\n", err)
	}

	_, _ = fmt.Fprintf(w, "✓ auto-reconcile enabled (period: %s)\n", period)
	return nil
}

func installLaunchdPlist(w io.Writer, bin, period string) error {
	dir, err := timerDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create launchd directory: %w", err)
	}

	interval := launchdInterval(period)
	plistContent := fmt.Sprintf(launchdPlistTemplate, bin, interval)
	plistPath := filepath.Join(dir, "dev.gostamp.stamp-reconcile.plist")
	if err := atomicWriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return err
	}

	if err := launchctl("load", plistPath); err != nil {
		return fmt.Errorf("failed to load launchd agent: %w", err)
	}

	_, _ = fmt.Fprintf(w, "✓ auto-reconcile enabled (period: %s)\n", period)
	return nil
}

func removeSystemdTimer(w io.Writer) error {
	dir, err := timerDir()
	if err != nil {
		return err
	}
	timerPath := filepath.Join(dir, "stamp-reconcile.timer")
	servicePath := filepath.Join(dir, "stamp-reconcile.service")

	if _, err := os.Stat(timerPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintln(w, "no timer found")
		return nil
	}

	_ = systemctl("disable", "--now", "stamp-reconcile.timer")
	_ = os.Remove(timerPath)
	_ = os.Remove(servicePath)
	_ = systemctl("daemon-reload")

	_, _ = fmt.Fprintln(w, "✓ auto-reconcile disabled")
	return nil
}

func removeLaunchdPlist(w io.Writer) error {
	dir, err := timerDir()
	if err != nil {
		return err
	}
	plistPath := filepath.Join(dir, "dev.gostamp.stamp-reconcile.plist")

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintln(w, "no timer found")
		return nil
	}

	_ = launchctl("unload", plistPath)
	_ = os.Remove(plistPath)

	_, _ = fmt.Fprintln(w, "✓ auto-reconcile disabled")
	return nil
}

func systemdCalendar(period string) string {
	switch period {
	case "hourly":
		return "hourly"
	case "daily":
		return "daily"
	case "weekly":
		return "weekly"
	}
	return "daily"
}

func launchdInterval(period string) int {
	switch period {
	case "hourly":
		return 3600
	case "daily":
		return 86400
	case "weekly":
		return 604800
	}
	return 86400
}

func systemctl(args ...string) error {
	cmd := execCommand("systemctl", append([]string{"--user"}, args...)...)
	return cmd.Run()
}

func launchctl(args ...string) error {
	cmd := execCommand("launchctl", args...)
	return cmd.Run()
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".stamp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := f.Name()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write: %w", err)
	}
	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename %s: %w", path, err)
	}
	return nil
}
