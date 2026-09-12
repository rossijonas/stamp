package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_VersionIncludesBuildInfo(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = oldVersion, oldCommit, oldDate })
	Version, Commit, Date = "9.9.9", "abc1234", "2026-09-10"

	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--version"})
	require.NoError(t, root.Execute())

	out := buf.String()
	assert.Contains(t, out, "stamp version")
	assert.Contains(t, out, "9.9.9")
	assert.Contains(t, out, "commit abc1234")
	assert.Contains(t, out, "built 2026-09-10")
}
