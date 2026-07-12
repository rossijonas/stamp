package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

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
	assert.Contains(t, buf.String(), "man page installed to")

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

func TestMan_Check_NotExist(t *testing.T) {
	// Override candidates to point to isolated nonexistent files
	oldCandidates := manPageCandidates
	manPageCandidates = []string{filepath.Join(t.TempDir(), "nonexistent.1")}
	defer func() { manPageCandidates = oldCandidates }()

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "❌ Man page not installed. Run 'stamp man install'")
}

func TestMan_Check_Success(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	// Override candidates to point to our isolated path
	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	// Pre-create the man page with matching current version
	// Version is typically defined as Version in cli package
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp ` + Version + `" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manFile, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "✅ Man page is up to date")
}

func TestMan_Check_Outdated(t *testing.T) {
	oldIsTerminal := isTerminal
	isTerminal = func(_ io.Reader) bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

	oldCandidates := manPageCandidates
	manPageCandidates = []string{manFile}
	defer func() { manPageCandidates = oldCandidates }()

	// Pre-create the man page with outdated version
	manContent := `.TH "STAMP" "1" "Jul 2026" "stamp v0.1.0" "Stamp Manual"`
	require.NoError(t, os.WriteFile(manFile, []byte(manContent), 0600))

	buf, err := execCmd(t, []string{"man", "check"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "⚠️ Man page is outdated")
	assert.Contains(t, buf.String(), "v0.1.0")
	assert.Contains(t, buf.String(), Version)
}

func TestMan_Check_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	manFile := filepath.Join(tmpDir, "stamp.1")

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
