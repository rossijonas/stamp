package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUv_Operations(t *testing.T) {
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
			mockOutput:  "black v24.8.0\n- black\nruff v0.6.1\n- ruff\n",
			expectedRes: []string{"black", "ruff"},
		},
		{
			name:        "list installed error",
			operation:   "list",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "install success",
			operation: "install",
			pkgName:   "black",
		},
		{
			name:        "install error",
			operation:   "install",
			pkgName:     "black",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "black",
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "black",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "black",
		},
		{
			name:        "remove error",
			operation:   "remove",
			pkgName:     "black",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "search error",
			operation:   "search",
			pkgName:     "anything",
			expectedErr: true,
		},
		{
			name:       "info success",
			operation:  "info",
			pkgName:    "black",
			mockOutput: "black v24.8.0\n- black\n",
		},
		{
			name:        "info not found",
			operation:   "info",
			pkgName:     "missing",
			mockOutput:  "black v24.8.0\n",
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
			name:      "update single success",
			operation: "update_single",
			pkgName:   "black",
		},
		{
			name:        "update single error",
			operation:   "update_single",
			pkgName:     "black",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "update all success",
			operation: "update_all",
		},
		{
			name:        "update all error",
			operation:   "update_all",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:        "doctor error",
			operation:   "doctor",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mgr := NewUv()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "uv", mgr.Name())

			var err error
			ctx := context.Background()

			switch tt.operation {
			case "list":
				res, err := mgr.ListInstalled(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.ElementsMatch(t, tt.expectedRes, res)
				}
			case "install":
				err = mgr.Install(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "reinstall":
				err = mgr.Reinstall(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "remove":
				err = mgr.Remove(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "search":
				_, err = mgr.Search(ctx, tt.pkgName)
				require.Error(t, err)
			case "info":
				res, err := mgr.Info(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Contains(t, res, tt.pkgName)
				}
			case "update_single":
				err = mgr.Update(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "update_all":
				err = mgr.Update(ctx, "")
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case "doctor":
				_, err = mgr.Doctor(ctx)
				require.Error(t, err)
			}
		})
	}
}

func TestUv_UnsupportedOperations(t *testing.T) {
	t.Parallel()
	mgr := NewUv()
	ctx := context.Background()

	err := mgr.AddRepo(ctx, "repo", "url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	err = mgr.RemoveRepo(ctx, "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	repos, err := mgr.ListRepos(ctx)
	require.NoError(t, err)
	assert.Empty(t, repos)
}
