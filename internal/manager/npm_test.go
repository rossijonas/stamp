package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpm_Operations(t *testing.T) {
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
			mockOutput:  "/usr/lib\n├── corepack@0.29.4\n├── typescript@5.6.3\n",
			expectedRes: []string{"corepack", "typescript"},
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
			pkgName:   "typescript",
		},
		{
			name:        "install error",
			operation:   "install",
			pkgName:     "typescript",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "reinstall success",
			operation: "reinstall",
			pkgName:   "typescript",
		},
		{
			name:        "reinstall error",
			operation:   "reinstall",
			pkgName:     "typescript",
			mockErr:     assert.AnError,
			expectedErr: true,
		},
		{
			name:      "remove success",
			operation: "remove",
			pkgName:   "typescript",
		},
		{
			name:        "remove error",
			operation:   "remove",
			pkgName:     "typescript",
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
			name:       "info success",
			operation:  "info",
			pkgName:    "typescript",
			mockOutput: "Name: typescript\nVersion: 5.6.3\n",
		},
		{
			name:        "info error",
			operation:   "info",
			pkgName:     "typescript",
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
			pkgName:   "typescript",
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
			manager := NewNpm()
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "npm", manager.Name())

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

func TestNpm_ListInstalled_Empty(t *testing.T) {
	t.Parallel()
	manager := NewNpm()
	manager.exec = mockExecutorHelper("/usr/lib\n", nil)

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestNpm_ListInstalled_NoGlobals(t *testing.T) {
	t.Parallel()
	manager := NewNpm()
	manager.exec = mockExecutorHelper("/usr/lib\n(empty)\n", nil)

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestNpm_Search_NotSupported(t *testing.T) {
	t.Parallel()
	manager := NewNpm()
	_, err := manager.Search(context.Background(), "typescript")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestNpm_CheckUpdate_NotSupported(t *testing.T) {
	t.Parallel()
	manager := NewNpm()
	_, err := manager.CheckUpdate(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCheckUnsupported)
}

func TestParseNpmLs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "standard output",
			input:    "/usr/lib/node_modules\n├── corepack@0.29.4\n├── npm@10.8.2\n└── typescript@5.6.3\n",
			expected: []string{"corepack", "typescript"},
		},
		{
			name:     "single package",
			input:    "/usr/lib\n└── cowsay@1.6.0\n",
			expected: []string{"cowsay"},
		},
		{
			name:     "no globals",
			input:    "/usr/lib\n",
			expected: []string{},
		},
		{
			name:     "empty output",
			input:    "",
			expected: []string{},
		},
		{
			name:     "with deduped prefix",
			input:    "/usr/lib\n├── typescript@5.6.3 deduped\n",
			expected: []string{"typescript"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseNpmLs([]byte(tt.input))
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestNpm_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.Provides(context.Background(), "htop")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestNpm_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.AutoRemove(context.Background(), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestNpm_CleanNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewNpm()
	_, err := mgr.Clean(context.Background(), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}
