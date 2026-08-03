package manager

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goExecTest returns an Executor that handles GOBIN-empty and GOPATH=tmpDir.
// Version calls are delegated to onErrorVers (to allow recovery simulation).
// When onErrorVers is nil, version calls return an error (fallback to binary name).
func goExecTest(calls *int, tmpDir string, onErrorVers func(string, ...string) ([]byte, error)) Executor {
	return func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if calls != nil {
			*calls++
		}
		if len(args) >= 2 && args[1] == "GOBIN" {
			return []byte(""), nil
		}
		if len(args) >= 2 && args[1] == "GOPATH" {
			return []byte(tmpDir), nil
		}
		if len(args) >= 1 && args[0] == "version" {
			if onErrorVers != nil {
				return onErrorVers(args[0], args[1:]...)
			}
			return nil, assert.AnError
		}
		return nil, nil
	}
}

func TestGo_Install(t *testing.T) {
	t.Parallel()
	var calls int
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls++
		assert.Equal(t, []string{"install", "github.com/example/tool@latest"}, args)
		return nil, nil
	}
	err := mgr.Install(WithYes(context.Background()), "github.com/example/tool")
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestGo_Install_ShortName(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	err := mgr.Install(WithYes(context.Background()), "tool")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "full module path")
}

func TestGo_Install_InvalidChars(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	err := mgr.Install(WithYes(context.Background()), "github.com/too;l/evil")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid module path")
}

func TestGo_Reinstall(t *testing.T) {
	t.Parallel()
	var calls int
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls++
		assert.Equal(t, []string{"install", "github.com/example/tool@latest"}, args)
		return nil, nil
	}
	err := mgr.Reinstall(WithYes(context.Background()), "github.com/example/tool")
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestGo_ListInstalled(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.MkdirAll(binDir, 0755))
	for _, name := range []string{"htop", "jq", "lazygit"} {
		//nolint:gosec // test fixture needs execute bit
		require.NoError(t, os.WriteFile(filepath.Join(binDir, name), []byte("x"), 0755))
	}
	//nolint:gosec // test fixture
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "readme.md"), []byte("x"), 0644))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, ".hidden"), []byte("x"), 0755))

	mgr := NewGo()
	mgr.exec = goExecTest(nil, tmpDir, nil)

	pkgs, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	// go version -m fails on test fixtures → fallback to binary names
	assert.ElementsMatch(t, []string{"htop", "jq", "lazygit"}, pkgs)
}

func TestGo_ListInstalled_DirNotExist(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	mgr.exec = goExecTest(nil, "/nonexistent/gopath", nil)
	pkgs, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestGo_Remove(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	binPath := filepath.Join(binDir, "tool")
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	mgr := NewGo()
	mgr.exec = goExecTest(nil, tmpDir, nil)

	err := mgr.Remove(WithYes(context.Background()), "github.com/example/tool")
	require.NoError(t, err)
	// Verify the file was actually removed via os.Remove
	_, err = os.Stat(binPath)
	assert.True(t, os.IsNotExist(err))
}

func TestGo_Remove_Missing(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	mgr.exec = goExecTest(nil, "/nonexistent/gopath", nil)
	err := mgr.Remove(WithYes(context.Background()), "github.com/example/tool")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in go bin directory")
}

func TestGo_Remove_InvalidBinName(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	err := mgr.Remove(WithYes(context.Background()), "github.com/example/..")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binary name must be a simple filename")
}

func TestGo_Remove_BaseNormalized(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	binPath := filepath.Join(binDir, "tool")
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	mgr := NewGo()
	mgr.exec = goExecTest(nil, tmpDir, nil)

	err := mgr.Remove(WithYes(context.Background()), "github.com/example/a/b/tool")
	require.NoError(t, err)
	// Verify the correct binary was removed
	_, err = os.Stat(binPath)
	assert.True(t, os.IsNotExist(err))
}

func TestGo_Info(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	binPath := filepath.Join(binDir, "tool")
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))

	mgr := NewGo()
	mgr.exec = goExecTest(nil, tmpDir, func(_ string, args ...string) ([]byte, error) {
		assert.Equal(t, []string{"-m", binPath}, args)
		return []byte("module info"), nil
	})

	res, err := mgr.Info(WithYes(context.Background()), "github.com/example/tool")
	require.NoError(t, err)
	assert.Equal(t, "module info", res)
}

func TestGo_Info_TraversalGuard(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	_, err := mgr.Info(WithYes(context.Background()), "github.com/example/..")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binary name must be a simple filename")
}

func TestGo_Update_Single(t *testing.T) {
	t.Parallel()
	var calls int
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls++
		assert.Equal(t, []string{"install", "github.com/example/tool@latest"}, args)
		return nil, nil
	}
	err := mgr.Update(WithYes(context.Background()), "github.com/example/tool")
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestGo_Update_Batch(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool"), []byte("x"), 0755))

	var calls []string
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) >= 2 && args[1] == "GOBIN" {
			return []byte(""), nil
		}
		if len(args) >= 2 && args[1] == "GOPATH" {
			return []byte(tmpDir), nil
		}
		if len(args) >= 1 && args[0] == "version" {
			return nil, assert.AnError
		}
		return nil, nil
	}

	err := mgr.Update(WithYes(context.Background()), "")
	require.NoError(t, err)
	// "tool" has no "/" → skipped; no install calls
	assert.Len(t, calls, 3) // GOBIN + GOPATH + version -m
}

func TestGo_Update_Batch_AllFail(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool1"), []byte("x"), 0755))
	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool2"), []byte("x"), 0755))

	var installMu sync.Mutex
	var installCalls int
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[1] == "GOBIN" {
			return []byte(""), nil
		}
		if len(args) >= 2 && args[1] == "GOPATH" {
			return []byte(tmpDir), nil
		}
		if len(args) >= 2 && args[0] == "version" && args[1] == "-m" {
			return []byte("tool1: go1.26\npath\tgithub.com/example/tool1\n"), nil
		}
		if len(args) >= 1 && args[0] == "install" {
			installMu.Lock()
			installCalls++
			installMu.Unlock()
			return nil, assert.AnError
		}
		return nil, nil
	}

	err := mgr.Update(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Positive(t, installCalls)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestGo_Update_Batch_PartialFail(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool1"), []byte("x"), 0755))
	//nolint:gosec
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool2"), []byte("x"), 0755))

	var partialMu sync.Mutex
	var callCount int
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[1] == "GOBIN" {
			return []byte(""), nil
		}
		if len(args) >= 2 && args[1] == "GOPATH" {
			return []byte(tmpDir), nil
		}
		if len(args) >= 2 && args[0] == "version" && args[1] == "-m" {
			return []byte("tool1: go1.26\npath\tgithub.com/example/tool1\n"), nil
		}
		if len(args) >= 1 && args[0] == "install" {
			partialMu.Lock()
			callCount++
			first := callCount == 1
			partialMu.Unlock()
			if first {
				return nil, nil
			}
			return nil, assert.AnError
		}
		return nil, nil
	}

	err := mgr.Update(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.Equal(t, 2, callCount)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestGo_Search_Error(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	results, err := mgr.Search(WithYes(context.Background()), "anything")
	require.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "search not supported")
}

func TestGo_UnsupportedOperations(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	ctx := WithYes(context.Background())

	_, err := mgr.Doctor(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	err = mgr.AddRepo(ctx, "repo", "url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	err = mgr.RemoveRepo(ctx, "repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")

	repos, err := mgr.ListRepos(ctx)
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestGo_GoBinDir_Cache(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool"), []byte("x"), 0755))

	mgr := NewGo()
	var execCalls int
	mgr.exec = goExecTest(&execCalls, tmpDir, nil)

	pkgs, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Len(t, pkgs, 1)
	// First call: GOBIN(1) + GOPATH(2) + version-m(3)
	assert.Equal(t, 3, execCalls)

	// Second call: binDir cached, only version -m per binary
	pkgs, err = mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Len(t, pkgs, 1)
	assert.Equal(t, 4, execCalls)
}

func TestGo_GoBinDir_GOBIN(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "mybin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "gtop"), []byte("x"), 0755))

	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[1] == "GOBIN" {
			return []byte(binDir), nil // GOBIN returns the bin dir directly
		}
		return nil, assert.AnError
	}

	pkgs, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"gtop"}, pkgs)
}

func TestGo_GoBinDir_MultiGOPATH(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "gtop"), []byte("x"), 0755))
	otherDir := t.TempDir()
	otherBinDir := filepath.Join(otherDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(otherBinDir, 0755))

	gopath := tmpDir + ":" + otherDir

	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[1] == "GOBIN" {
			return []byte(""), nil
		}
		if len(args) >= 2 && args[1] == "GOPATH" {
			return []byte(gopath), nil
		}
		return nil, assert.AnError
	}

	// Should use the first GOPATH entry
	dir, err := mgr.goBinDir(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "bin"), dir)
}

func TestGo_ListInstalled_ModulePaths(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "tool"), []byte("x"), 0755))

	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[1] == "GOBIN" {
			return []byte(""), nil
		}
		if len(args) >= 2 && args[1] == "GOPATH" {
			return []byte(tmpDir), nil
		}
		if len(args) >= 1 && args[0] == "version" {
			return []byte("path\tgithub.com/example/tool\nmod\tgithub.com/example/tool\tv1.0.0\n"), nil
		}
		return nil, nil
	}

	pkgs, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"github.com/example/tool"}, pkgs)
}

func TestGo_ListInstalled_ModulePathFallback(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(binDir, 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "legacy"), []byte("x"), 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "modern"), []byte("x"), 0755))

	mgr := NewGo()
	mgr.exec = goExecTest(nil, tmpDir, func(_ string, args ...string) ([]byte, error) {
		binPath := args[len(args)-1]
		if strings.HasSuffix(binPath, "/modern") {
			return []byte("path\tgithub.com/example/modern\n"), nil
		}
		// legacy (unrecoverable)
		return nil, assert.AnError
	})

	pkgs, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"github.com/example/modern", "legacy"}, pkgs)
}

func TestGo_ListInstalled_GoBinDirFallback(t *testing.T) {
	tmpDir := t.TempDir()
	relaxDir := filepath.Join(tmpDir, "go", "bin")
	//nolint:gosec // test fixture
	require.NoError(t, os.MkdirAll(relaxDir, 0755))
	//nolint:gosec // test fixture needs execute bit
	require.NoError(t, os.WriteFile(filepath.Join(relaxDir, "x"), []byte("x"), 0755))

	mgr := NewGo()
	var callCount int
	mgr.exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		callCount++
		if len(args) >= 2 && args[1] == "GOBIN" {
			return nil, assert.AnError
		}
		if len(args) >= 2 && args[1] == "GOPATH" {
			return nil, assert.AnError
		}
		// version -m → unrecoverable
		return nil, assert.AnError
	}

	t.Setenv("HOME", tmpDir)
	pkgs, err := mgr.ListInstalled(WithYes(context.Background()))
	require.NoError(t, err)
	assert.NotEmpty(t, pkgs)
	// 2 exec calls (GOBIN + GOPATH), HOME fallback used, then version -m
	assert.Equal(t, 3, callCount)
}

func TestValidateModulePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		pkg   string
		valid bool
	}{
		{"full module path", "github.com/example/tool", true},
		{"with hyphen", "github.com/golangci/golangci-lint", true},
		{"with dot", "golang.org/x/tools/gopls", true},
		{"short name", "tool", false},
		{"empty", "", false},
		{"semicolon", "github.com/too;l/evil", false},
		{"pipe", "github.com/too|l/evil", false},
		{"ampersand", "github.com/too&l/evil", false},
		{"backtick", "github.com/too`l/evil", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateModulePath(tt.pkg)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGo_CheckUpdate_Unsupported(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	_, err := mgr.CheckUpdate(WithYes(context.Background()), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCheckUnsupported)
}

func TestGo_GoBinDir_AllErrors(t *testing.T) {
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, assert.AnError
	}
	t.Setenv("HOME", "/nonexistent_home_for_test")
	// GOBIN + GOPATH fail → falls back to $HOME/go/bin
	dir, err := mgr.goBinDir(WithYes(context.Background()))
	require.NoError(t, err)
	assert.Equal(t, "/nonexistent_home_for_test/go/bin", dir)
}

func TestGo_Name(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	assert.Equal(t, "go", mgr.Name())
}

func TestGo_ProvidesNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	_, err := mgr.Provides(WithYes(context.Background()), "htop")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestGo_AutoRemoveNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	_, err := mgr.AutoRemove(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestGo_CleanNotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	_, err := mgr.Clean(WithYes(context.Background()), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestGo_Hold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	err := mgr.Hold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestGo_Unhold_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	err := mgr.Unhold(WithYes(context.Background()), "nginx")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestGo_ListHeld_NotSupported(t *testing.T) {
	t.Parallel()
	mgr := NewGo()
	_, err := mgr.ListHeld(WithYes(context.Background()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestGo_Remove_ExecError(t *testing.T) {
	t.Parallel()
	call := 0
	mgr := NewGo()
	mgr.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("/home/user/go/bin/tool\n"), nil // go env GOPATH
		}
		return nil, assert.AnError // go version or removal fails
	}
	err := mgr.Remove(WithYes(context.Background()), "github.com/example/tool")
	require.Error(t, err)
}
