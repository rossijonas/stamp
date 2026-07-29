package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCargo_Operations(t *testing.T) {
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
			mockOutput:  "bat v0.25.0:\n    bat\nripgrep v14.1.0:\n    rg\n",
			expectedRes: []string{"bat", "ripgrep"},
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
			pkgName:   "ripgrep",
		},
		{
			name:        "install error",
			operation:   "install",
			pkgName:     "ripgrep",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "ripgrep",
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "ripgrep",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "ripgrep",
		},
		{
			name:        "remove error",
			operation:   "remove",
			pkgName:     "ripgrep",
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
			name:       "search success",
			operation:  "search",
			pkgName:    "serde",
			mockOutput: "serde = \"1.0.215\"\n",
		},
		{
			name:        "search error",
			operation:   "search",
			pkgName:     "serde",
			mockErr:     assert.AnError,
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
			pkgName:    "serde",
			mockOutput: "serde — https://serde.rs\nA serialization framework\n",
		},
		{
			name:        "info error",
			operation:   "info",
			pkgName:     "serde",
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
			pkgName:   "ripgrep",
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
			manager := NewCargo()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "cargo", manager.Name())

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
					assert.NotEmpty(t, res)
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

func TestCargo_ListInstalled_Empty(t *testing.T) {
	t.Parallel()
	manager := NewCargo()
	manager.exec = mockExecutorHelper("", nil)

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestCargo_Update_Single(t *testing.T) {
	t.Parallel()
	manager := NewCargo()
	manager.exec = mockExecutorHelper("", nil)

	err := manager.Update(context.Background(), "ripgrep")
	require.NoError(t, err)
}

func TestCargo_Search_Supported(t *testing.T) {
	t.Parallel()
	manager := NewCargo()
	manager.exec = mockExecutorHelper("serde = \"1.0.215\"\n", nil)

	results, err := manager.Search(context.Background(), "serde")
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestCargo_Info_Supported(t *testing.T) {
	t.Parallel()
	manager := NewCargo()
	expected := "serde — https://serde.rs\nA serialization framework\n"
	manager.exec = mockExecutorHelper(expected, nil)

	info, err := manager.Info(context.Background(), "serde")
	require.NoError(t, err)
	assert.Equal(t, expected, info)
}

func TestCargo_CheckUpdate_NotSupported(t *testing.T) {
	t.Parallel()
	manager := NewCargo()
	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCheckUnsupported)
}

func TestCargo_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	_, err := mgr.Provides(context.Background(), "htop")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestCargo_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	_, err := mgr.AutoRemove(context.Background(), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestCargo_CleanNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	_, err := mgr.Clean(context.Background(), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestCargo_Hold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	err := mgr.Hold(context.Background(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestCargo_Unhold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	err := mgr.Unhold(context.Background(), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestCargo_ListHeld_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	_, err := mgr.ListHeld(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestCargo_Update_Batch_ListError(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	err := mgr.Update(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list installed packages for update")
}

func TestCargo_Update_Single_Error(t *testing.T) {
	t.Parallel()
	mgr := NewCargo()
	mgr.exec = mockExecutorHelper("", assert.AnError)
	err := mgr.Update(context.Background(), "ripgrep")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update")
}
