package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestSetupCmd_AutoAccept(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}
	buf, err := execCmd(t, []string{"setup", "--yes"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Stamp Setup Wizard")
	// Auto-accept should not contain prompts
	assert.NotContains(t, output, "[Y/n]:")
}

func TestHelloCmd_StillWorks(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"hello"}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Stamp Setup Wizard")
}

func TestRootCmd_DefaultSuggestsSetup(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Don't know where to start? Try:")
	assert.Contains(t, output, "stamp setup")
}

func TestSetupCmd_Interactive_AllYes(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{ManagerName: "brew", InstalledPkgs: []string{}},
	}

	root := NewRootCmd(WithAdapters(adapters), WithConfigPath(""), WithManifestPath(""))
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader("y\ny\ny\n"))
	root.SetArgs([]string{"setup"})

	err := root.Execute()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Setup complete")
}
