package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoReconcile_On_Linux(t *testing.T) {
	oldOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // test fixture needs execute bit for EvalSymlinks
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	oldExec := osExecutable
	osExecutable = func() (string, error) { return binPath, nil }
	defer func() { osExecutable = oldExec }()

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	var systemctlCalls []string
	oldCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "systemctl" {
			systemctlCalls = append(systemctlCalls, args...)
			return exec.Command("true")
		}
		return oldCmd(name, args...)
	}
	defer func() { execCommand = oldCmd }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on"})
	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "auto-reconcile enabled")

	systemdDir := filepath.Join(tmpDir, ".config", "systemd", "user")
	servicePath := filepath.Join(systemdDir, "stamp-reconcile.service")
	timerPath := filepath.Join(systemdDir, "stamp-reconcile.timer")

	//nolint:gosec // temp files in isolated test directory
	serviceData, err := os.ReadFile(servicePath)
	require.NoError(t, err)
	assert.Contains(t, string(serviceData), "stamp reconcile")

	//nolint:gosec // temp files in isolated test directory
	timerData, err := os.ReadFile(timerPath)
	require.NoError(t, err)
	assert.Contains(t, string(timerData), "OnCalendar=daily")
	assert.Contains(t, string(timerData), "Persistent=true")

	assert.Contains(t, systemctlCalls, "daemon-reload")
	assert.Contains(t, systemctlCalls, "enable")
	assert.Contains(t, systemctlCalls, "--now")
	assert.Contains(t, systemctlCalls, "stamp-reconcile.timer")
}

func TestAutoReconcile_On_Linux_XDG(t *testing.T) {
	oldOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // test fixture needs execute bit for EvalSymlinks
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	oldExec := osExecutable
	osExecutable = func() (string, error) { return binPath, nil }
	defer func() { osExecutable = oldExec }()

	xdgDir := t.TempDir()
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", xdgDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	oldCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "systemctl" {
			return exec.Command("true")
		}
		return oldCmd(name, args...)
	}
	defer func() { execCommand = oldCmd }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on"})
	err := root.Execute()
	require.NoError(t, err)

	systemdDir := filepath.Join(xdgDir, "systemd", "user")
	servicePath := filepath.Join(systemdDir, "stamp-reconcile.service")
	_, err = os.Stat(servicePath)
	require.NoError(t, err, "service file should be in XDG_CONFIG_HOME dir")
}

func TestAutoReconcile_On_Darwin(t *testing.T) {
	oldOS := currentOS
	currentOS = "darwin"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // test fixture needs execute bit for EvalSymlinks
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	oldExec := osExecutable
	osExecutable = func() (string, error) { return binPath, nil }
	defer func() { osExecutable = oldExec }()

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	oldCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "launchctl" {
			return exec.Command("true")
		}
		return oldCmd(name, args...)
	}
	defer func() { execCommand = oldCmd }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on"})
	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "auto-reconcile enabled")

	launchDir := filepath.Join(tmpDir, "Library", "LaunchAgents")
	plistPath := filepath.Join(launchDir, "dev.gostamp.stamp-reconcile.plist")
	//nolint:gosec // temp file in isolated test directory
	plistData, err := os.ReadFile(plistPath)
	require.NoError(t, err)

	content := string(plistData)
	assert.Contains(t, content, binPath)
	assert.Contains(t, content, "86400")
	assert.Contains(t, content, "StandardOutPath")
	assert.Contains(t, content, "stamp-reconcile.log")
}

func TestAutoReconcile_On_Period_Invalid(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on", "--period", "monthly"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid period")
}

func TestAutoReconcile_On_UnsupportedOS(t *testing.T) {
	oldOS := currentOS
	currentOS = "windows"
	defer func() { currentOS = oldOS }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on windows")
}

func TestAutoReconcile_Off_Linux(t *testing.T) {
	oldOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = oldOS }()

	oldExec := osExecutable
	osExecutable = func() (string, error) { return "/usr/local/bin/stamp", nil }
	defer func() { osExecutable = oldExec }()

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	systemdDir := filepath.Join(tmpDir, ".config", "systemd", "user")
	require.NoError(t, os.MkdirAll(systemdDir, 0750))
	//nolint:gosec // dummy fixture files in temp directory
	require.NoError(t, os.WriteFile(filepath.Join(systemdDir, "stamp-reconcile.service"), []byte("dummy"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(systemdDir, "stamp-reconcile.timer"), []byte("dummy"), 0600))

	var systemctlCalls []string
	oldCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "systemctl" {
			systemctlCalls = append(systemctlCalls, args...)
			return exec.Command("true")
		}
		return oldCmd(name, args...)
	}
	defer func() { execCommand = oldCmd }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "off"})
	err := root.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "auto-reconcile disabled")
	assert.Contains(t, systemctlCalls, "disable")

	_, err = os.Stat(filepath.Join(systemdDir, "stamp-reconcile.service"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(systemdDir, "stamp-reconcile.timer"))
	assert.True(t, os.IsNotExist(err))
}

func TestAutoReconcile_Off_NotInstalled(t *testing.T) {
	oldOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "off"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no timer found")
}

func TestAutoReconcile_Off_UnsupportedOS(t *testing.T) {
	oldOS := currentOS
	currentOS = "windows"
	defer func() { currentOS = oldOS }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "off"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on windows")
}

func TestAutoReconcile_On_Period_Hourly(t *testing.T) {
	oldOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // test fixture needs execute bit for EvalSymlinks
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	oldExec := osExecutable
	osExecutable = func() (string, error) { return binPath, nil }
	defer func() { osExecutable = oldExec }()

	_ = os.Unsetenv("XDG_CONFIG_HOME")
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	oldCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "systemctl" {
			return exec.Command("true")
		}
		return oldCmd(name, args...)
	}
	defer func() { execCommand = oldCmd }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on", "-p", "hourly"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "enabled (period: hourly)")

	systemdDir := filepath.Join(tmpDir, ".config", "systemd", "user")
	//nolint:gosec // temp file in isolated test directory
	timerData, err := os.ReadFile(filepath.Join(systemdDir, "stamp-reconcile.timer"))
	require.NoError(t, err)
	assert.Contains(t, string(timerData), "OnCalendar=hourly")
}

func TestAutoReconcile_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test-output.txt")
	content := []byte("test content")
	err := atomicWriteFile(testPath, content, 0644)
	require.NoError(t, err)

	//nolint:gosec // temp file in isolated test directory
	data, err := os.ReadFile(testPath)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(data))
}

func TestAutoReconcile_Help(t *testing.T) {
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "--help"})
	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "auto-reconcile")
	assert.Contains(t, output, "on")
	assert.Contains(t, output, "off")
}

func TestAutoReconcile_Off_Darwin(t *testing.T) {
	oldOS := currentOS
	currentOS = "darwin"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	launchDir := filepath.Join(tmpDir, "Library", "LaunchAgents")
	plistPath := filepath.Join(launchDir, "dev.gostamp.stamp-reconcile.plist")
	require.NoError(t, os.MkdirAll(launchDir, 0750))
	//nolint:gosec // dummy test fixture
	require.NoError(t, os.WriteFile(plistPath, []byte("dummy"), 0644))

	var launchctlCalls []string
	oldCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "launchctl" {
			launchctlCalls = append(launchctlCalls, args...)
			return exec.Command("true")
		}
		return oldCmd(name, args...)
	}
	defer func() { execCommand = oldCmd }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "off"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "auto-reconcile disabled")
	assert.Contains(t, launchctlCalls, "unload")
	_, err = os.Stat(plistPath)
	assert.True(t, os.IsNotExist(err))
}

func TestAutoReconcile_Off_Darwin_NotInstalled(t *testing.T) {
	oldOS := currentOS
	currentOS = "darwin"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "off"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no timer found")
}

func TestAutoReconcile_On_Period_Weekly_Darwin(t *testing.T) {
	oldOS := currentOS
	currentOS = "darwin"
	defer func() { currentOS = oldOS }()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "stamp")
	//nolint:gosec // test fixture needs execute bit for EvalSymlinks
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	oldExec := osExecutable
	osExecutable = func() (string, error) { return binPath, nil }
	defer func() { osExecutable = oldExec }()

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	oldCmd := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "launchctl" {
			return exec.Command("true")
		}
		return oldCmd(name, args...)
	}
	defer func() { execCommand = oldCmd }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on", "-p", "weekly"})
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "enabled (period: weekly)")

	launchDir := filepath.Join(tmpDir, "Library", "LaunchAgents")
	//nolint:gosec // temp file in isolated test directory
	plistData, err := os.ReadFile(filepath.Join(launchDir, "dev.gostamp.stamp-reconcile.plist"))
	require.NoError(t, err)
	assert.Contains(t, string(plistData), "604800")
}

func TestAutoReconcile_SystemdCalendar(t *testing.T) {
	tests := []struct {
		period   string
		expected string
	}{
		{"hourly", "hourly"},
		{"daily", "daily"},
		{"weekly", "weekly"},
		{"monthly", "daily"}, // default fallback
		{"", "daily"},        // default fallback
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			got := systemdCalendar(tt.period)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAutoReconcile_LaunchdInterval(t *testing.T) {
	tests := []struct {
		period   string
		expected int
	}{
		{"hourly", 3600},
		{"daily", 86400},
		{"weekly", 604800},
		{"monthly", 86400}, // default fallback
		{"", 86400},        // default fallback
	}
	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			got := launchdInterval(tt.period)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAutoReconcile_AtomicWrite_ReadOnlyDir(t *testing.T) {
	readonlyDir := t.TempDir()
	//nolint:gosec // intentionally restrictive permissions for test
	require.NoError(t, os.Chmod(readonlyDir, 0500))

	err := atomicWriteFile(filepath.Join(readonlyDir, "out.txt"), []byte("x"), 0644)
	require.Error(t, err)
}

func TestAutoReconcile_On_ExecutableError(t *testing.T) {
	oldOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = oldOS }()

	// Force osExecutable to return an error so EvalSymlinks never runs
	oldExec := osExecutable
	osExecutable = func() (string, error) { return "", assert.AnError }
	defer func() { osExecutable = oldExec }()

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"auto-reconcile", "on"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve stamp binary path")
}

func TestAutoReconcile_TimerDir_HomeError(t *testing.T) {
	oldOS := currentOS
	currentOS = "linux"
	defer func() { currentOS = oldOS }()

	// Unset HOME so UserHomeDir fails
	oldHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	_, err := timerDir()
	require.Error(t, err)
}

func TestAutoReconcile_TimerDir_UnsupportedOS(t *testing.T) {
	oldOS := currentOS
	currentOS = "windows"
	defer func() { currentOS = oldOS }()

	_, err := timerDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
