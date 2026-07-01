package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutorHelper returns an Executor that injects a predefined string output.
func mockExecutorHelper(output string, err error) Executor {
	return func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(output), err
	}
}

func TestDnfManager_ListInstalled(t *testing.T) {
	mockOutput := "htop\nripgrep\n"
	manager := &DnfManager{exec: mockExecutorHelper(mockOutput, nil)}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop", "ripgrep"}, pkgs)
}

func TestBrewManager_ListInstalled(t *testing.T) {
	mockOutput := "jq\nfzf\ntmux\n"
	manager := &BrewManager{exec: mockExecutorHelper(mockOutput, nil)}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"jq", "fzf", "tmux"}, pkgs)
}

func TestFlatpakManager_ListInstalled(t *testing.T) {
	mockOutput := "com.spotify.Client\norg.mozilla.firefox\n"
	manager := &FlatpakManager{exec: mockExecutorHelper(mockOutput, nil)}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"com.spotify.Client", "org.mozilla.firefox"}, pkgs)
}

func TestParseLines(t *testing.T) {
	input := []byte(" line1 \nline2\n\n  line3  \n")
	expected := []string{"line1", "line2", "line3"}
	actual := parseLines(input)
	assert.ElementsMatch(t, expected, actual)
}
