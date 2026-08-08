package manager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRepofileURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"brave repofile", "https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo", true},
		{"enpass repofile", "https://yum.enpass.io/enpass-yum.repo", true},
		{"uppercase extension", "https://example.com/repo.REPO", true},
		{"mixed case extension", "https://example.com/repo.Repo", true},
		{"query string", "https://example.com/foo.repo?token=abc", true},
		{"bare baseurl", "https://example.com/repo", false},
		{"trailing slash", "https://example.com/repo/", false},
		{"flatpakrepo not matched", "https://dl.flathub.org/repo/flathub.flatpakrepo", false},
		{"bare host", "https://example.com", false},
		{"invalid url", "://bad", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRepofileURL(tt.url))
		})
	}
}

func TestDNFRepoFetcher_Success(t *testing.T) {
	t.Parallel()

	body := "[brave-browser-rpm-release]\nname=Brave Browser\nbaseurl=https://brave-browser-rpm-release.s3.brave.com/x86_64\ngpgcheck=1\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	got, err := dnfRepoFetcher(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, body, string(got))
}

func TestDNFRepoFetcher_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	_, err := dnfRepoFetcher(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDNFRepoFetcher_EmptyBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	got, err := dnfRepoFetcher(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDNFRepoFetcher_Oversized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", maxRepoFileBytes+1)))
	}))
	defer srv.Close()

	_, err := dnfRepoFetcher(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestDNFRepoFetcher_CancelledContext(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := dnfRepoFetcher(ctx, srv.URL)
	require.Error(t, err)
}

// TestDNF_AddRepo_Repofile_Success verifies a .repo URL is fetched and moved
// verbatim into dnfReposDir with gpg settings preserved.
func TestDNF_AddRepo_Repofile_Success(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	oldFetcher := dnfRepoFetcher
	repoContent := "[brave-browser-rpm-release]\n" +
		"name=Brave Browser\n" +
		"baseurl=https://brave-browser-rpm-release.s3.brave.com/x86_64\n" +
		"gpgcheck=1\n" +
		"gpgkey=https://brave-browser-rpm-release.s3.brave.com/brave-core.asc\n"
	dnfRepoFetcher = func(_ context.Context, _ string) ([]byte, error) {
		return []byte(repoContent), nil
	}
	defer func() { dnfRepoFetcher = oldFetcher }()

	var mvArgs []string
	var movedContent []byte
	manager := NewDNF("dnf")
	manager.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		mvArgs = append([]string{name}, args...)
		// capture temp file content before the deferred cleanup removes it
		if i := indexOf(args, "mv"); i >= 0 && i+1 < len(args) {
			movedContent, _ = os.ReadFile(args[i+1])
		}
		return nil, nil
	}

	err := manager.AddRepo(WithYes(context.Background()), "brave", "https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo")
	require.NoError(t, err)

	// sudo [-n] mv <tmp> <dnfReposDir>/brave.repo — locate "mv" by content
	// since sudoCmd injects -n only in non-TTY environments.
	mvIdx := indexOf(mvArgs, "mv")
	require.NotEqual(t, -1, mvIdx, "expected mv in args: %v", mvArgs)
	assert.Equal(t, "sudo", mvArgs[0])
	assert.Equal(t, filepath.Join(dnfReposDir, "brave.repo"), mvArgs[mvIdx+2])

	// temp file content preserved verbatim (gpgcheck=1 survives)
	assert.Equal(t, repoContent, string(movedContent))
}

// TestDNF_AddRepo_Repofile_FetchError verifies fetch failures wrap the error.
func TestDNF_AddRepo_Repofile_FetchError(t *testing.T) {
	oldFetcher := dnfRepoFetcher
	dnfRepoFetcher = func(_ context.Context, _ string) ([]byte, error) {
		return nil, assert.AnError
	}
	defer func() { dnfRepoFetcher = oldFetcher }()

	manager := NewDNF("dnf")
	err := manager.AddRepo(WithYes(context.Background()), "brave", "https://example.com/brave.repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch repo file")
}

// TestDNF_AddRepo_Repofile_InvalidContent verifies garbage/empty fetched
// content is rejected before any write happens.
func TestDNF_AddRepo_Repofile_InvalidContent(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"empty", "", "no repo sections"},
		{"html 404 page", "<html><body>Not Found</body></html>", "no repo sections"},
		{"section without baseurl", "[repo]\nname=Repo\n", "no baseurl, metalink, or mirrorlist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFetcher := dnfRepoFetcher
			dnfRepoFetcher = func(_ context.Context, _ string) ([]byte, error) {
				return []byte(tt.content), nil
			}
			defer func() { dnfRepoFetcher = oldFetcher }()

			execCalled := false
			manager := NewDNF("dnf")
			manager.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				execCalled = true
				return nil, nil
			}

			err := manager.AddRepo(WithYes(context.Background()), "bad", "https://example.com/bad.repo")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.False(t, execCalled, "no write should occur for invalid content")
		})
	}
}

// TestDNF_AddRepo_Repofile_MvError verifies a failed sudo mv wraps the error.
func TestDNF_AddRepo_Repofile_MvError(t *testing.T) {
	oldFetcher := dnfRepoFetcher
	dnfRepoFetcher = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("[repo]\nname=Repo\nbaseurl=https://example.com/repo\n"), nil
	}
	defer func() { dnfRepoFetcher = oldFetcher }()

	manager := NewDNF("dnf")
	manager.exec = mockExecutorHelper("", assert.AnError)

	err := manager.AddRepo(WithYes(context.Background()), "repo", "https://example.com/repo.repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add repo")
}

// TestDNF_AddRepo_BaseURL_StillWorks guards the plain baseurl path.
func TestDNF_AddRepo_BaseURL_StillWorks(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	var mvArgs []string
	var movedContent []byte
	manager := NewDNF("dnf")
	manager.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		mvArgs = append([]string{name}, args...)
		if i := indexOf(args, "mv"); i >= 0 && i+1 < len(args) {
			movedContent, _ = os.ReadFile(args[i+1])
		}
		return nil, nil
	}

	err := manager.AddRepo(WithYes(context.Background()), "custom", "https://example.com/repo")
	require.NoError(t, err)

	mvIdx := indexOf(mvArgs, "mv")
	require.NotEqual(t, -1, mvIdx, "expected mv in args: %v", mvArgs)
	assert.Equal(t, "sudo", mvArgs[0])
	assert.Equal(t, filepath.Join(dnfReposDir, "custom.repo"), mvArgs[mvIdx+2])

	assert.Contains(t, string(movedContent), "baseurl=https://example.com/repo")
}

// indexOf returns the first index of needle in haystack, or -1.
func indexOf(haystack []string, needle string) int {
	for i, s := range haystack {
		if s == needle {
			return i
		}
	}
	return -1
}

// TestDNF_AddRepo_Copr_StillWorks guards the name-only COPR path.
func TestDNF_AddRepo_Copr_StillWorks(t *testing.T) {
	var args []string
	manager := NewDNF("dnf")
	manager.exec = func(_ context.Context, name string, a ...string) ([]byte, error) {
		args = append([]string{name}, a...)
		return nil, nil
	}

	err := manager.AddRepo(WithYes(context.Background()), "petersen/cava", "")
	require.NoError(t, err)

	assert.Contains(t, args, "copr")
	assert.Contains(t, args, "enable")
	assert.Contains(t, args, "petersen/cava")
}

// TestDNF_RemoveRepo_File verifies a URL-added repo is removed by deleting
// its .repo file.
func TestDNF_RemoveRepo_File(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	//nolint:gosec // test temp dir
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "enpass.repo"), []byte(
		"[enpass]\nbaseurl=https://yum.enpass.io\n"), 0o644))

	var rmArgs []string
	manager := NewDNF("dnf")
	manager.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		rmArgs = append([]string{name}, args...)
		return nil, nil
	}

	err := manager.RemoveRepo(WithYes(context.Background()), "enpass")
	require.NoError(t, err)

	rmIdx := indexOf(rmArgs, "rm")
	require.NotEqual(t, -1, rmIdx, "expected rm in args: %v", rmArgs)
	assert.Equal(t, "sudo", rmArgs[0])
	assert.Equal(t, "-f", rmArgs[rmIdx+1])
	assert.Equal(t, filepath.Join(dnfReposDir, "enpass.repo"), rmArgs[rmIdx+2])
}

// TestDNF_RemoveRepo_FileMissing_CoprFallback verifies the copr path is used
// when no .repo file exists for the name.
func TestDNF_RemoveRepo_FileMissing_CoprFallback(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	var args []string
	manager := NewDNF("dnf")
	manager.exec = func(_ context.Context, name string, a ...string) ([]byte, error) {
		args = append([]string{name}, a...)
		return nil, nil
	}

	err := manager.RemoveRepo(WithYes(context.Background()), "petersen/cava")
	require.NoError(t, err)

	assert.Contains(t, args, "copr")
	assert.Contains(t, args, "disable")
	assert.Contains(t, args, "petersen/cava")
}

// TestDNF_RemoveRepo_File_ExecError verifies a failed sudo rm wraps the error.
func TestDNF_RemoveRepo_File_ExecError(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	//nolint:gosec // test temp dir
	require.NoError(t, os.WriteFile(filepath.Join(dnfReposDir, "enpass.repo"), []byte("x"), 0o644))

	manager := NewDNF("dnf")
	manager.exec = mockExecutorHelper("", assert.AnError)

	err := manager.RemoveRepo(WithYes(context.Background()), "enpass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove repo file")
}

// TestDNF_RemoveRepo_Copr_ExecError verifies a failed copr disable wraps the error.
func TestDNF_RemoveRepo_Copr_ExecError(t *testing.T) {
	oldDir := dnfReposDir
	dnfReposDir = t.TempDir()
	defer func() { dnfReposDir = oldDir }()

	manager := NewDNF("dnf")
	manager.exec = mockExecutorHelper("", assert.AnError)

	err := manager.RemoveRepo(WithYes(context.Background()), "petersen/cava")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to disable copr")
}
