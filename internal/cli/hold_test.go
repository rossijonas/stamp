package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestHoldCmd_Success(t *testing.T) {
	buf, err := execCmd(t, []string{"hold", "nginx", "-m", "apt", "-y"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "held nginx via apt")
}

func TestHoldCmd_UnsupportedManager(t *testing.T) {
	// mockAdapter.Hold returns ErrNotSupported → CLI skips it and falls through
	_, err := execCmd(t, []string{"hold", "nginx", "-m", "dnf", "-y"}, []manager.Adapter{
		&mockAdapter{name: "dnf"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no manager supports hold")
}

func TestHoldCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"hold", "nginx", "-m", "nonexistent"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestUnholdCmd_Success(t *testing.T) {
	buf, err := execCmd(t, []string{"unhold", "nginx", "-m", "apt", "-y"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "unheld nginx via apt")
}

func TestUnholdCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"unhold", "nginx", "-m", "nonexistent"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestHeldCmd_WithResults(t *testing.T) {
	buf, err := execCmd(t, []string{"held"}, []manager.Adapter{
		&manager.Mock{
			ManagerName: "apt",
			HeldPkgs:    []string{"nginx", "redis"},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nginx (apt)")
	assert.Contains(t, buf.String(), "redis (apt)")
}

func TestHeldCmd_NoResults(t *testing.T) {
	buf, err := execCmd(t, []string{"held"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no packages held")
}

func TestHeldCmd_WithManagerFlag(t *testing.T) {
	buf, err := execCmd(t, []string{"held", "-m", "apt"}, []manager.Adapter{
		&manager.Mock{
			ManagerName: "apt",
			HeldPkgs:    []string{"nginx"},
		},
		&manager.Mock{ManagerName: "brew"},
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nginx (apt)")
	assert.NotContains(t, buf.String(), "brew")
}

func TestHeldCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"held", "-m", "nonexistent"}, []manager.Adapter{
		&manager.Mock{ManagerName: "apt"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}
