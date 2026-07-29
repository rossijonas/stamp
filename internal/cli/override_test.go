package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestOverrideCmd_Filesystem(t *testing.T) {
	adapters := []manager.Adapter{&manager.Mock{
		ManagerName: "flatpak",
		OverrideFunc: func(_ context.Context, _ string, _ manager.OverrideFlags) error {
			return nil
		},
	}}
	_, err := execCmd(t, []string{"override", "firefox", "-m", "flatpak", "--filesystem=host"}, adapters)
	require.NoError(t, err)
}

func TestOverrideCmd_NonFlatpak(t *testing.T) {
	_, err := execCmd(t, []string{"override", "firefox", "-m", "brew"}, []manager.Adapter{
		&manager.Mock{ManagerName: "brew"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supported for flatpak")
}

func TestOverrideCmd_UnknownManager(t *testing.T) {
	_, err := execCmd(t, []string{"override", "firefox", "-m", "nonexistent"}, []manager.Adapter{
		&manager.Mock{ManagerName: "flatpak"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestOverrideCmd_InvalidName(t *testing.T) {
	_, err := execCmd(t, []string{"override", "bad!", "-m", "flatpak"}, []manager.Adapter{
		&manager.Mock{ManagerName: "flatpak"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid app-id")
}
