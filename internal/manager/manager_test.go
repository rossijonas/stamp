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

func TestDnfManager_Operations(t *testing.T) {
	tests := []struct {
		name        string
		operation   string // "list", "install", "remove", "search"
		pkgName     string
		mockOutput  string
		mockErr     error
		expectedErr bool
		expectedRes []string
	}{
		{
			name:        "list installed success",
			operation:   "list",
			mockOutput:  "htop\nripgrep\n",
			expectedRes: []string{"htop", "ripgrep"},
		},
		{
			name:      "install success",
			operation: "install",
			pkgName:   "htop",
			mockErr:   nil,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "htop",
			mockErr:   nil,
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "htop",
			mockOutput:  "htop\nhtop-debuginfo\n",
			expectedRes: []string{"htop", "htop-debuginfo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewDnfManager()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "dnf", manager.Name()) // hit the name method

			var err error
			ctx := context.Background()

			switch tt.operation {
			case "list":
				res, err := manager.ListInstalled(ctx)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedRes, res)
			case "install":
				err = manager.Install(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "remove":
				err = manager.Remove(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "search":
				res, err := manager.Search(ctx, tt.pkgName)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedRes, res)
			}
		})
	}
}

func TestBrewManager_Operations(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		pkgName     string
		mockOutput  string
		mockErr     error
		expectedErr bool
		expectedRes []string
	}{
		{
			name:        "list installed success",
			operation:   "list",
			mockOutput:  "jq\nfzf\ntmux\n",
			expectedRes: []string{"jq", "fzf", "tmux"},
		},
		{
			name:      "install success",
			operation: "install",
			pkgName:   "jq",
			mockErr:   nil,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "jq",
			mockErr:   nil,
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "jq",
			mockOutput:  "jq\njq-debug\n",
			expectedRes: []string{"jq", "jq-debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewBrewManager()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "brew", manager.Name())

			var err error
			ctx := context.Background()

			switch tt.operation {
			case "list":
				res, err := manager.ListInstalled(ctx)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedRes, res)
			case "install":
				err = manager.Install(ctx, tt.pkgName)
				require.NoError(t, err)
			case "remove":
				err = manager.Remove(ctx, tt.pkgName)
				require.NoError(t, err)
			case "search":
				res, err := manager.Search(ctx, tt.pkgName)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedRes, res)
			}
		})
	}
}

func TestFlatpakManager_Operations(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		pkgName     string
		mockOutput  string
		mockErr     error
		expectedErr bool
		expectedRes []string
	}{
		{
			name:        "list installed success",
			operation:   "list",
			mockOutput:  "com.spotify.Client\norg.mozilla.firefox\n",
			expectedRes: []string{"com.spotify.Client", "org.mozilla.firefox"},
		},
		{
			name:      "install success",
			operation: "install",
			pkgName:   "com.spotify.Client",
			mockErr:   nil,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "com.spotify.Client",
			mockErr:   nil,
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "spotify",
			mockOutput:  "com.spotify.Client\n",
			expectedRes: []string{"com.spotify.Client"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFlatpakManager()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "flatpak", manager.Name())

			var err error
			ctx := context.Background()

			switch tt.operation {
			case "list":
				res, err := manager.ListInstalled(ctx)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedRes, res)
			case "install":
				err = manager.Install(ctx, tt.pkgName)
				require.NoError(t, err)
			case "remove":
				err = manager.Remove(ctx, tt.pkgName)
				require.NoError(t, err)
			case "search":
				res, err := manager.Search(ctx, tt.pkgName)
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedRes, res)
			}
		})
	}
}

func TestParseLines(t *testing.T) {
	input := []byte(" line1 \nline2\n\n  line3  \n")
	expected := []string{"line1", "line2", "line3"}
	actual := parseLines(input)
	assert.ElementsMatch(t, expected, actual)
}
