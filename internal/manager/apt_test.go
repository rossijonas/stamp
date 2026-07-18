package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPT_Operations(t *testing.T) {
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
			mockOutput:  "Listing...\nhtop/stable,now 3.2.1 amd64 [installed]\njq/stable,now 1.6 amd64 [installed]\n",
			expectedRes: []string{"htop", "jq"},
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
			name:      "remove success",
			operation: "remove",
			pkgName:   "htop",
		},
		{
			name:        "search success",
			operation:   "search",
			pkgName:     "htop",
			mockOutput:  "htop - interactive process viewer\nhtop-dev - development headers\n",
			expectedRes: []string{"htop", "htop-dev"},
		},
		{
			name:       "info success",
			operation:  "info",
			pkgName:    "htop",
			mockOutput: "Package: htop\nVersion: 3.2.1\n",
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
			name:        "info validation error",
			operation:   "info",
			pkgName:     "-invalid",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manager := NewAPT("apt")
			manager.exec = mockExecutorHelper(tt.mockOutput, tt.mockErr)

			assert.Equal(t, "apt", manager.Name())

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
			case "update":
				err = manager.Update(ctx)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestAPT_ListInstalled_Fallback(t *testing.T) {
	t.Parallel()
	calls := 0
	manager := NewAPT("apt")
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		if calls == 1 {
			// apt list --installed fails
			return nil, assert.AnError
		}
		// dpkg-query succeeds
		return []byte("installed htop\ninstalled jq\nnot-installed curl\n"), nil
	}

	pkgs, err := manager.ListInstalled(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop", "jq"}, pkgs)
	assert.Equal(t, 2, calls)
}

func TestAPT_ListInstalled_BothFail(t *testing.T) {
	t.Parallel()
	calls := 0
	manager := NewAPT("apt")
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		return nil, assert.AnError
	}

	_, err := manager.ListInstalled(context.Background())
	require.Error(t, err)
	assert.Equal(t, 2, calls)
}

func TestAPT_Info_Fallback(t *testing.T) {
	t.Parallel()
	manager := NewAPT("apt")
	manager.exec = mockExecutorHelper("Package: htop\nVersion: 3.2.1\n", nil)

	res, err := manager.Info(context.Background(), "htop")
	require.NoError(t, err)
	assert.Equal(t, "Package: htop\nVersion: 3.2.1\n", res)
}

func TestAPT_Info_AptGet(t *testing.T) {
	t.Parallel()
	manager := NewAPT("apt-get")
	manager.exec = mockExecutorHelper("Package: htop\nVersion: 3.2.1\n", nil)

	res, err := manager.Info(context.Background(), "htop")
	require.NoError(t, err)
	assert.Equal(t, "Package: htop\nVersion: 3.2.1\n", res)
}

func TestAPT_Doctor_NotSupported(t *testing.T) {
	t.Parallel()
	manager := NewAPT("apt")
	_, err := manager.Doctor(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doctor not supported")
}

func TestAPT_ListRepos(t *testing.T) {
	oldDir := aptSourcesDir
	oldFile := aptSourcesFile
	aptSourcesDir = t.TempDir()
	aptSourcesFile = filepath.Join(t.TempDir(), "sources.list")
	defer func() {
		aptSourcesDir = oldDir
		aptSourcesFile = oldFile
	}()

	// Create a custom .list file in sources.list.d
	//nolint:gosec // test fixtures in temp dir
	require.NoError(t, os.WriteFile(filepath.Join(aptSourcesDir, "google-chrome.list"),
		[]byte("deb [arch=amd64] https://dl.google.com/linux/chrome/deb/ stable main\n"), 0644))
	//nolint:gosec // test fixtures in temp dir
	require.NoError(t, os.WriteFile(filepath.Join(aptSourcesDir, "vscode.list"),
		[]byte("deb [arch=amd64] https://packages.microsoft.com/repos/code stable main\n"), 0644))

	// Create a system repo in sources.list — should be filtered
	//nolint:gosec // test fixtures in temp dir
	require.NoError(t, os.WriteFile(aptSourcesFile,
		[]byte("deb http://archive.ubuntu.com/ubuntu/ jammy main restricted\n"), 0644))

	repos, err := parseAPTSources()
	require.NoError(t, err)
	require.Len(t, repos, 2)

	names := make(map[string]string)
	for _, r := range repos {
		names[r.Name] = r.URL
	}

	assert.Contains(t, names, "dl.google.com/linux/chrome/deb")
	assert.Equal(t, "https://dl.google.com/linux/chrome/deb/", names["dl.google.com/linux/chrome/deb"])
	assert.Contains(t, names, "packages.microsoft.com/repos/code")
	assert.NotContains(t, names, "archive.ubuntu.com/ubuntu/")
}

func TestAPT_ListRepos_EmptyDir(t *testing.T) {
	oldDir := aptSourcesDir
	oldFile := aptSourcesFile
	aptSourcesDir = t.TempDir()
	aptSourcesFile = filepath.Join(t.TempDir(), "sources.list")
	defer func() {
		aptSourcesDir = oldDir
		aptSourcesFile = oldFile
	}()

	repos, err := parseAPTSources()
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestAPT_ListRepos_MissingDir(t *testing.T) {
	oldDir := aptSourcesDir
	aptSourcesDir = "/nonexistent/sources"
	defer func() { aptSourcesDir = oldDir }()

	repos, err := parseAPTSources()
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestParseAPTSources_WithOptions(t *testing.T) {
	oldDir := aptSourcesDir
	oldFile := aptSourcesFile
	aptSourcesDir = t.TempDir()
	aptSourcesFile = filepath.Join(t.TempDir(), "sources.list")
	defer func() {
		aptSourcesDir = oldDir
		aptSourcesFile = oldFile
	}()

	// File with options, mixed deb/deb-src, and comments
	//nolint:gosec // test fixtures in temp dir
	require.NoError(t, os.WriteFile(filepath.Join(aptSourcesDir, "custom.list"),
		[]byte("# comment\n"+
			"deb [signed-by=/usr/share/keyrings/custom.gpg] https://custom-repo.example.com/ubuntu focal main\n"+
			"deb-src https://custom-repo.example.com/ubuntu focal main\n"), 0644))

	repos, err := parseAPTSources()
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "custom-repo.example.com/ubuntu", repos[0].Name)
}

func TestAPT_AddRepo_CustomURL(t *testing.T) {
	t.Parallel()
	manager := NewAPT("apt")
	manager.exec = mockExecutorHelper("", assert.AnError)

	err := manager.AddRepo(context.Background(), "myrepo", "https://example.com/apt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add repository")
}

func TestAPT_Update_Phase2Fails(t *testing.T) {
	t.Parallel()
	calls := 0
	manager := NewAPT("apt")
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte{}, nil // update succeeds
		}
		return nil, assert.AnError // upgrade fails
	}

	err := manager.Update(context.Background())
	require.Error(t, err)
	assert.Equal(t, 2, calls)
}

func TestAPT_Update_Phase1Fails(t *testing.T) {
	t.Parallel()
	calls := 0
	manager := NewAPT("apt")
	manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		return nil, assert.AnError
	}

	err := manager.Update(context.Background())
	require.Error(t, err)
	assert.Equal(t, 1, calls) // update should fail before upgrade is called
}

func TestAPT_RemoveRepo_NoFile(t *testing.T) {
	t.Parallel()
	oldLookPath := lookPath
	lookPath = func(string) (string, error) { return "", assert.AnError }
	defer func() { lookPath = oldLookPath }()

	manager := NewAPT("apt")
	manager.exec = mockExecutorHelper("", nil)

	err := manager.RemoveRepo(context.Background(), "nonexistent-repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add-apt-repository not found")
}

func TestAPT_RemoveRepo_WithFile(t *testing.T) {
	oldDir := aptSourcesDir
	aptSourcesDir = t.TempDir()
	defer func() { aptSourcesDir = oldDir }()

	// Create a .list file
	//nolint:gosec // test fixtures in temp dir
	require.NoError(t, os.WriteFile(filepath.Join(aptSourcesDir, "myrepo.list"),
		[]byte("deb https://example.com/apt ./\n"), 0644))

	manager := NewAPT("apt")
	manager.exec = mockExecutorHelper("", nil)

	// Now the file exists in our overridden aptSourcesDir, and sudo rm is mocked
	err := manager.RemoveRepo(context.Background(), "myrepo")
	require.NoError(t, err)
}

func TestAPT_ListRepos_ThroughAdapter(t *testing.T) {
	oldDir := aptSourcesDir
	oldFile := aptSourcesFile
	aptSourcesDir = t.TempDir()
	aptSourcesFile = filepath.Join(t.TempDir(), "sources.list")
	defer func() {
		aptSourcesDir = oldDir
		aptSourcesFile = oldFile
	}()

	//nolint:gosec // test fixtures in temp dir
	require.NoError(t, os.WriteFile(filepath.Join(aptSourcesDir, "custom.list"),
		[]byte("deb https://myrepo.example.com/apt ./\n"), 0644))

	manager := NewAPT("apt")
	repos, err := manager.ListRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "myrepo.example.com/apt", repos[0].Name)
}

func TestAPT_Reinstall_Error(t *testing.T) {
	t.Parallel()
	manager := NewAPT("apt")
	manager.exec = mockExecutorHelper("", assert.AnError)

	err := manager.Reinstall(context.Background(), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reinstall")
}

func TestAPT_Remove_Error(t *testing.T) {
	t.Parallel()
	manager := NewAPT("apt")
	manager.exec = mockExecutorHelper("", assert.AnError)

	err := manager.Remove(context.Background(), "htop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove")
}

func TestAPT_AddRepo_PPA(t *testing.T) {
	t.Parallel()
	oldLookPath := lookPath
	lookPath = func(string) (string, error) { return "", assert.AnError }
	defer func() { lookPath = oldLookPath }()

	manager := NewAPT("apt")
	manager.exec = mockExecutorHelper("", nil)

	err := manager.AddRepo(context.Background(), "ppa:git-core/ppa", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add-apt-repository not found")
}

func TestParseAPTListInstalled(t *testing.T) {
	t.Parallel()
	input := []byte("Listing...\nhtop/stable,now 3.2.1 amd64 [installed]\njq/stable,now 1.6 amd64 [installed]\nlocal-pkg,now 1.0 amd64 [installed]\n")
	expected := []string{"htop", "jq", "local-pkg"}
	result := parseAPTListInstalled(input)
	assert.ElementsMatch(t, expected, result)
}

func TestParseDPKGQueryInstalled(t *testing.T) {
	t.Parallel()
	input := []byte("installed htop\ninstalled jq\nrc removed-pkg\nnot-installed curl\n")
	expected := []string{"htop", "jq"}
	result := parseDPKGQueryInstalled(input)
	assert.ElementsMatch(t, expected, result)
}
