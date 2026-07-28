package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestProvidesCmd_NoMatches(t *testing.T) {
	buf, err := execCmd(t, []string{"provides", "/usr/bin/htop"}, []manager.Adapter{&mockAdapter{name: "dnf", err: nil}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no packages provide")
}

func TestProvidesCmd_ManagerFlag(t *testing.T) {
	buf, err := execCmd(t, []string{"provides", "/usr/bin/htop", "-m", "dnf"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestProvidesCmd_MatchFound(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:    "dnf",
			ProvidesResult: []string{"/usr/bin/htop"},
		},
	}
	buf, err := execCmd(t, []string{"provides", "/usr/bin/htop"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "/usr/bin/htop (dnf)")
}

func TestProvidesCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"provides", "/usr/bin/htop", "-m", "nonexistent"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}
