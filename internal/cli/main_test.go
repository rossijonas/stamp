package cli

import (
	"os"
	"testing"
)

// TestMain forces all tests in this package to write stamp state (manifest,
// config, snapshots) to a private temp dir instead of the real user home.
// Without this, snapshot-writing tests (restore/init/reconcile) corrupt the
// developer's real ~/.local/share/stamp/snapshots when running `task check`.
func TestMain(m *testing.M) {
	tmp := os.Getenv("TMPDIR")
	if tmp == "" {
		tmp = os.TempDir()
	}
	dataHome, err := os.MkdirTemp(tmp, "stamp-test-xdg-*")
	if err != nil {
		panic("failed to create temp XDG_DATA_HOME: " + err.Error())
	}
	oldData := os.Getenv("XDG_DATA_HOME")
	oldConfig := os.Getenv("XDG_CONFIG_HOME")
	if err := os.Setenv("XDG_DATA_HOME", dataHome); err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CONFIG_HOME", dataHome); err != nil {
		panic(err)
	}

	code := m.Run()

	if err := os.Unsetenv("XDG_DATA_HOME"); err != nil {
		panic(err)
	}
	if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
		panic(err)
	}
	if oldData != "" {
		_ = os.Setenv("XDG_DATA_HOME", oldData)
	}
	if oldConfig != "" {
		_ = os.Setenv("XDG_CONFIG_HOME", oldConfig)
	}
	//nolint:gosec // dataHome is a MkdirTemp-created path under os.TempDir
	_ = os.RemoveAll(dataHome)
	os.Exit(code)
}
