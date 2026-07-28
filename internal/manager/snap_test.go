package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnap_Operations(t *testing.T) {
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
			mockOutput:  "Name    Version   Rev   Tracking         Publisher   Notes\nhtop    3.2.2     123   latest/stable    canonical✓  -\ncore    16-2.59   456   latest/stable    canonical✓  core\n",
			expectedRes: []string{"htop", "core"},
		},
		{
			name:        "list installed error",
			operation:   "list",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "list installed no snaps",
			operation:   "list",
			mockOutput:  "Name    Version   Rev   Tracking         Publisher   Notes\n",
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
			mockOutput:  "Name               Version   Publisher   Notes    Summary\nhtop               3.2.2     canonical   -        Interactive process viewer\n",
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
			mockOutput: "name: htop\nversion: 3.2.2\n",
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
			name:      "update single package",
			operation: "update_single",
			pkgName:   "htop",
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
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewSnap()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "snap", manager.Name())

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
				err = manager.Update(ctx, "")
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "update_single":
				err = manager.Update(ctx, tt.pkgName)
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
				res, err := manager.ListRepos(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Empty(t, res)
				}
			}
		})
	}
}

func TestParseSnapTabular(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "standard snap list output",
			input: "Name    Version   Rev   Tracking         Publisher   Notes\n" +
				"htop    3.2.2     123   latest/stable    canonical✓  -\n" +
				"core    16-2.59   456   latest/stable    canonical✓  core\n",
			expected: []string{"htop", "core"},
		},
		{
			name:     "no snaps installed",
			input:    "Name    Version   Rev   Tracking         Publisher   Notes\n",
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name: "single snap",
			input: "Name   Version  Rev   Tracking  Publisher  Notes\n" +
				"hello  2.12     123   stable    canonical  -\n",
			expected: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseSnapTabular([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestSnap_CheckUpdateExecError(t *testing.T) {
	t.Parallel()
	manager := NewSnap()
	manager.exec = mockExecutorHelper("", assert.AnError)
	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check updates")
}

func TestSnap_CheckUpdate(t *testing.T) {
	t.Parallel()
	manager := NewSnap()
	manager.exec = mockExecutorHelper("htop                 3.2.2     123   stable\ngit                  2.43.2    456   stable\n", nil)

	updates, err := manager.CheckUpdate(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
}

func TestParseSnapRefreshList(t *testing.T) {
	t.Parallel()
	input := []byte("Name                 Version   Rev   Latest\nhtop                 3.2.2     123   stable\ngit                  2.43.2    456   stable\n")
	updates := parseSnapRefreshList(input)
	require.Len(t, updates, 2)
	assert.Equal(t, "htop", updates[0].Package)
	assert.Equal(t, "git", updates[1].Package)
}
