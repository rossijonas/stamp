package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestShowAlias(t *testing.T) {
	buf, err := execCmd(t, []string{"show", "htop", "-m", "dnf"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "htop")
}

func TestViewAlias(t *testing.T) {
	buf, err := execCmd(t, []string{"view", "htop", "-m", "dnf"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "htop")
}

func TestRefreshAlias(t *testing.T) {
	buf, err := execCmd(t, []string{"refresh"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestOutdatedCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"outdated"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestCheckUpdateCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"check-update"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestTapCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"tap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "added tap mytap via brew")
}

func TestTapCmd_BrewNotAvailable(t *testing.T) {
	_, err := execCmd(t, []string{"tap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brew is not available")
}

func TestTapCmd_RefusesWithoutYesNonInteractive(t *testing.T) {
	_, err := execCmd(t, []string{"tap", "mytap"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.ErrorIs(t, err, errNonInteractive)
}

func TestUntapCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"untap", "mytap", "-y"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "removed tap mytap via brew")
}

func TestUntapCmd_RefusesWithoutYesNonInteractive(t *testing.T) {
	_, err := execCmd(t, []string{"untap", "mytap"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.ErrorIs(t, err, errNonInteractive)
}

func TestTapsCmd(t *testing.T) {
	buf, err := execCmd(t, []string{"taps"}, []manager.Adapter{&mockAdapter{name: "brew"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "no taps added")
}

func TestTapsCmd_BrewNotAvailable(t *testing.T) {
	_, err := execCmd(t, []string{"taps"}, []manager.Adapter{&mockAdapter{name: "dnf"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "brew is not available")
}
