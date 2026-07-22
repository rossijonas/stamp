package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMacPorts_Operations(t *testing.T) {
	t.Parallel()
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
			mockOutput:  "The following ports are installed:\n  htop @3.2.2 (active)\n  ripgrep @13.0.0 (active)\n",
			expectedRes: []string{"htop", "ripgrep"},
		},
		{
			name:        "list installed error",
			operation:   "list",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "list installed no ports",
			operation:   "list",
			mockOutput:  "The following ports are installed:\n",
			expectedRes: []string{},
		},
		{
			name:      "install success",
			operation: "install",
			pkgName:   "htop",
		},
		{
			name:        "install error",
			operation:   "install",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "htop",
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "htop",
		},
		{
			name:        "remove error",
			operation:   "remove",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "htop",
			mockOutput:  "htop @3.2.2 (sysutils, interactive process viewer)\n",
			expectedRes: []string{"htop"},
		},
		{
			name:        "search error",
			operation:   "search",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "install validation error",
			operation:   "install",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:        "remove validation error",
			operation:   "remove",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:        "search validation error",
			operation:   "search",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:       "info success",
			operation:  "info",
			pkgName:    "htop",
			mockOutput: "htop @3.2.2 (sysutils)\n",
		},
		{
			name:        "info error",
			operation:   "info",
			pkgName:     "htop",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "info validation error",
			operation:   "info",
			pkgName:     "-invalid",
			expectedErr: true,
		},
		{
			name:      "update success",
			operation: "update",
		},
		{
			name:        "update error",
			operation:   "update",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "doctor not supported",
			operation:   "doctor",
			expectedErr: true,
		},
		{
			name:        "add repo not supported",
			operation:   "addrepo",
			expectedErr: true,
		},
		{
			name:        "remove repo not supported",
			operation:   "removerepo",
			expectedErr: true,
		},
		{
			name:        "list repos not supported",
			operation:   "listrepos",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewMacPorts()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "macports", manager.Name())

			var err error
			ctx := context.Background()

			switch tt.operation {
			case "list":
				res, err := manager.ListInstalled(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedRes, res)
				}
			case "install":
				err = manager.Install(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "reinstall":
				err = manager.Reinstall(ctx, tt.pkgName)
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
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedRes, res)
				}
			case "info":
				res, err := manager.Info(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.mockOutput, res)
				}
			case "doctor":
				_, err = manager.Doctor(ctx)
				require.Error(t, err)
			case "update":
				err = manager.Update(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "addrepo":
				err = manager.AddRepo(ctx, "repo", "")
				require.Error(t, err)
			case "removerepo":
				err = manager.RemoveRepo(ctx, "repo")
				require.Error(t, err)
			case "listrepos":
				_, err = manager.ListRepos(ctx)
				require.Error(t, err)
			}
		})
	}
}

func TestParsePortInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "standard output with header",
			input: "The following ports are installed:\n" +
				"  htop @3.2.2 (active)\n" +
				"  curl @8.0.1  (active)\n",
			expected: []string{"htop", "curl"},
		},
		{
			name:     "no ports installed",
			input:    "The following ports are installed:\n",
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parsePortInstalled([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestParsePortSearch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "standard search output",
			input:    "htop @3.2.2 (sysutils, interactive process viewer)\n",
			expected: []string{"htop"},
		},
		{
			name:     "no results",
			input:    "",
			expected: []string{},
		},
		{
			name:     "multiple results",
			input:    "htop @3.2.2 (sysutils)\nhtop-debug @1.0.0 (devel)\n",
			expected: []string{"htop", "htop-debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parsePortSearch([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
