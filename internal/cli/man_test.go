package cli

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// stubManExec forces `man -w` resolution to fail so candidate paths drive the
// result deterministically (a dev machine with a real installed man page would
// otherwise leak into the resolver).
func stubManExec(t *testing.T) {
	t.Helper()
	old := manExec
	manExec = func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("man unavailable")
	}
	t.Cleanup(func() { manExec = old })
}

func TestMan_Help(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"man"}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Command group to generate, install, and check")
	assert.Contains(t, output, "install")
	assert.Contains(t, output, "check")
}

func TestMan_Install_Success(t *testing.T) {
	t.Parallel()
	prefix := t.TempDir()
	buf, err := execCmd(t, []string{"man", "install", "--prefix", prefix}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "installed")

	// Verify file was written
	path := filepath.Join(prefix, "share", "man", "man1", "stamp.1")
	_, err = os.Stat(path)
	require.NoError(t, err)
}

func TestMan_Install_Error(t *testing.T) {
	t.Parallel()
	tempFile, err := os.CreateTemp("", "stamp-man-error-*")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()

	_, err = execCmd(t, []string{"man", "install", "--prefix", tempFile.Name()}, []manager.Adapter{})
	require.Error(t, err)
}

func TestMan_Install_CreateFileError(t *testing.T) {
	tmpDir := t.TempDir()
	manDir := filepath.Join(tmpDir, "share", "man", "man1")
	require.NoError(t, os.MkdirAll(manDir, 0750))
	//nolint:gosec // restricting permissions for test isolation
	require.NoError(t, os.Chmod(manDir, 0500))
	//nolint:gosec // restoring permissions for test cleanup
	defer func() { _ = os.Chmod(manDir, 0700) }()

	_, err := execCmd(t, []string{"man", "install", "--prefix", tmpDir}, []manager.Adapter{})
	require.Error(t, err)
}

func TestMan_Check_NotExist(t *testing.T) {
	// Override candidates to point to isolated nonexistent files
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Man page not installed. Run 'stamp man install'")
}

func TestMan_Check_Success(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	// Override candidates to point to our isolated path
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	// Pre-create the man page with matching current version
	// Version is typically defined as Version in cli package
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp ` + Version + `" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manFile, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Man page is up to date")
}

func TestMan_Check_Outdated(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	// Pre-create the man page with outdated version
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp v0.1.0" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manFile, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "⚠ Man page is outdated")
	assert.Contains(t, buf.String(), "v0.1.0")
	assert.Contains(t, buf.String(), Version)
}

func TestMan_Check_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	// Pre-create the man page with matching current version
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp v0.3.0" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manFile, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"man", "check", "--json"}, []manager.Adapter{})
	require.NoError(t, err)

	type jsonReport struct {
		Installed     bool   `json:"installed"`
		ManVersion    string `json:"man_version,omitempty"`
		BinaryVersion string `json:"binary_version"`
		Match         bool   `json:"match"`
		Error         string `json:"error,omitempty"`
	}

	var r jsonReport
	err = json.Unmarshal(buf.Bytes(), &r)
	require.NoError(t, err)

	assert.True(t, r.Installed)
	assert.Equal(t, "v0.3.0", r.ManVersion)
}

func TestDefaultManPrefix(t *testing.T) {
	t.Parallel()
	prefix := defaultManPrefix()
	assert.NotEmpty(t, prefix)
}

func TestDefaultManPageCandidatesPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	expected := filepath.Join(home, ".local", "share", "man", "man1", "stamp.1")
	assert.Equal(t, expected, defaultManPageCandidates()[0])
}

func TestInstalledManPagePath_UserLocal(t *testing.T) {
	home := t.TempDir()
	manPath := filepath.Join(home, ".local", "share", "man", "man1", "stamp.1")

	oldCandidates := manPageCandidates
	manPageCandidates = []string{manPath}
	defer func() { manPageCandidates = oldCandidates }()

	require.NoError(t, os.MkdirAll(filepath.Dir(manPath), 0750))
	require.NoError(t, os.WriteFile(manPath, []byte("garbage"), 0600))

	assert.Equal(t, manPath, installedManPagePath())
}

func TestMan_Check_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	require.NoError(t, os.WriteFile(manFile, []byte("garbage"), 0000))

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Error checking man page")
}

func TestMan_Check_JSON_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	require.NoError(t, os.WriteFile(manFile, []byte("garbage"), 0000))

	buf, err := execCmd(t, []string{"man", "check", "--json"}, []manager.Adapter{})
	require.NoError(t, err)

	var r struct {
		Error string `json:"error"`
	}
	err = json.Unmarshal(buf.Bytes(), &r)
	require.NoError(t, err)
	assert.Contains(t, r.Error, "permission denied")
}

func TestMan_Check_JSON_NotFound(t *testing.T) {
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	buf, err := execCmd(t, []string{"man", "check", "--json"}, []manager.Adapter{})
	require.NoError(t, err)

	var r struct {
		Error string `json:"error"`
	}
	err = json.Unmarshal(buf.Bytes(), &r)
	require.NoError(t, err)
	assert.Equal(t, "not found", r.Error)
}

func TestResolveInstalledManPage_ManW(t *testing.T) {
	old := manExec
	manExec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("/opt/stamp/share/man/man1/stamp.1.gz\n"), nil
	}
	defer func() { manExec = old }()

	assert.Equal(t, "/opt/stamp/share/man/man1/stamp.1.gz", resolveInstalledManPage())
}

func TestResolveInstalledManPage_FallbackToCandidates(t *testing.T) {
	stubManExec(t)

	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")
	require.NoError(t, os.WriteFile(manFile, []byte("garbage"), 0600))

	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	assert.Equal(t, manFile, resolveInstalledManPage())
}

func TestFirstManPagePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		out  string
		want string
	}{
		{name: "single gz path", out: "/usr/share/man/man1/stamp.1.gz", want: "/usr/share/man/man1/stamp.1.gz"},
		{name: "single plain path", out: "/usr/share/man/man1/stamp.1\n", want: "/usr/share/man/man1/stamp.1"},
		{name: "multiple whitespace separated", out: "/usr/share/man/man1/stamp.1 /usr/local/man/man1/stamp.1", want: "/usr/share/man/man1/stamp.1"},
		{name: "no matching suffix", out: "/usr/share/man/stamp.txt\n", want: ""},
		{name: "empty output", out: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, firstManPagePath([]byte(tt.out)))
		})
	}
}

func TestReadManPage_Plain(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")
	content := []byte(`.TH "STAMP" "1" "Jul 2026" "stamp v1.2.3" "Stamp Manual"`)
	require.NoError(t, os.WriteFile(manFile, content, 0600))

	data, err := readManPage(manFile)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestReadManPage_Gzip(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1.gz")
	content := []byte(`.TH "STAMP" "1" "Jul 2026" "stamp v1.2.3" "Stamp Manual"`)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(content)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, os.WriteFile(manFile, buf.Bytes(), 0600))

	data, err := readManPage(manFile)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestCheckInstalledManVersion_Gzip(t *testing.T) {
	stubManExec(t)

	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1.gz")
	content := []byte(`.TH "STAMP" "1" "Jul 2026" "stamp ` + Version + `" "Stamp Manual"`)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(content)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, os.WriteFile(manFile, buf.Bytes(), 0600))

	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	oldIsTerminal := isTerminal
	isTerminal = func(io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	result, path, err := checkInstalledManVersion()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, manFile, path)
	assert.True(t, result.matches, "gzip-compressed page version must match")
	assert.Equal(t, Version, result.version)
}

func TestManCheck_TTYGlyphs(t *testing.T) {
	forceOutputTTY(t)
	stubManExec(t)
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "✗ Man page not installed")
}
