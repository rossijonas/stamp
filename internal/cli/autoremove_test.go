package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestAutoremoveCmd_Runs(t *testing.T) {
	buf, err := execCmd(t, []string{"autoremove", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestAutoremoveCmd_DryRun(t *testing.T) {
	buf, err := execCmd(t, []string{"autoremove", "--dry-run"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestAutoremoveCmd_ManagerFlag(t *testing.T) {
	buf, err := execCmd(t, []string{"autoremove", "-m", "dnf", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestAutoremoveCmd_WithOrphans(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:      "brew",
			AutoRemoveResult: []string{"libfoo", "libbar"},
		},
	}
	buf, err := execCmd(t, []string{"autoremove", "-y"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed 2 package(s)")
}

func TestAutoremoveCmd_DryRunWithOrphans(t *testing.T) {
	adapters := []manager.Adapter{
		&manager.Mock{
			ManagerName:      "brew",
			AutoRemoveResult: []string{"libfoo"},
		},
	}
	buf, err := execCmd(t, []string{"autoremove", "--dry-run"}, adapters)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "would remove")
	assert.Contains(t, buf.String(), "libfoo")
}

func TestAutoremoveCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"autoremove", "-m", "nonexistent"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}
