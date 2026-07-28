package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipx_Operations(t *testing.T) {
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
			name:        "list installed via json output",
			operation:   "list",
			mockOutput:  `{"venvs":{"black":{"metadata":{}},"httpie":{"metadata":{}}}}`,
			expectedRes: []string{"black", "httpie"},
		},
		{
			name:        "list installed via text fallback",
			operation:   "list_text",
			mockOutput:  "   package black 23.1.0, installed using Python 3.11\n   package httpie 3.2.2, installed using Python 3.11\n",
			expectedRes: []string{"black", "httpie"},
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
			mockOutput: `{"venvs":{"black":{"metadata":{"version":"23.1.0"}}}}`,
		},
		{
			name:        "info not found",
			operation:   "info",
			pkgName:     "missing",
			mockOutput:  `{"venvs":{"black":{}}}`,
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
			mgr := NewPipx()
			mgr.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "pipx", mgr.Name())

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
			case "list_text":
				// Force text fallback by making --json fail first
				first := true
				mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
					if first {
						first = false
						return nil, assert.AnError
					}
					return []byte(tt.mockOutput), nil
				}
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
				_, err = mgr.Info(ctx, tt.pkgName)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
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

func TestPipx_CheckUpdate_Unsupported(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
	_, err := mgr.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCheckUnsupported)
}

func TestPipx_UnsupportedOperations(t *testing.T) {
	t.Parallel()
	mgr := NewPipx()
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
