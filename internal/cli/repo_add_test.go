package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// repoAddRecorder captures AddRepo invocations for auto-name tests. It embeds
// mockAdapter (same package) to satisfy manager.Adapter, overriding only AddRepo.
type repoAddRecorder struct {
	mockAdapter
	name string
	url  string
}

func newRepoAddRecorder(managerName string) *repoAddRecorder {
	return &repoAddRecorder{mockAdapter: mockAdapter{name: managerName}}
}

func (r *repoAddRecorder) AddRepo(_ context.Context, name, url string) error {
	r.name = name
	r.url = url
	return nil
}

func TestDeriveRepoName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"enpass repofile", "https://yum.enpass.io/enpass-yum.repo", "enpass-yum"},
		{"brave repofile", "https://brave-browser-rpm-release.s3.brave.com/brave-browser.repo", "brave-browser"},
		{"uppercase extension", "https://example.com/repo.REPO", "repo"},
		{"query string", "https://example.com/foo.repo?token=abc", "foo"},
		{"plain baseurl", "https://example.com/repo", "repo"},
		{"trailing slash", "https://example.com/repo/", "repo"},
		{"root path", "https://example.com", "example.com"},
		{"root path with slash", "https://example.com/", "example.com"},
		{"nested path", "https://example.com/vendor/stable.repo", "stable"},
		{"invalid url", "://bad", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveRepoName(tt.url))
		})
	}
}

func TestRepoAddCmd_SingleURL_AutoName(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mPath := filepath.Join(tmpDir, "manifest.toml")
	cPath := filepath.Join(tmpDir, "config.toml")

	rec := newRepoAddRecorder("dnf")
	buf, err := runRepoAdd(rec, mPath, cPath, []string{"repo", "add", "https://yum.enpass.io/enpass-yum.repo", "-m", "dnf", "-y"})
	require.NoError(t, err)
	assert.Equal(t, "enpass-yum", rec.name)
	assert.Equal(t, "https://yum.enpass.io/enpass-yum.repo", rec.url)
	assert.Contains(t, buf.String(), "added repo enpass-yum via dnf")

	// manifest entry persists the URL for restore
	//nolint:gosec // mPath comes from t.TempDir in the test
	data, err := os.ReadFile(mPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "https://yum.enpass.io/enpass-yum.repo")
}

func TestRepoAddCmd_SingleBareURL_HostFallback(t *testing.T) {
	t.Parallel()

	rec := newRepoAddRecorder("dnf")
	buf, err := execCmd(t, []string{"repo", "add", "https://example.com", "-m", "dnf", "-y"},
		[]manager.Adapter{rec})
	require.NoError(t, err)
	assert.Equal(t, "example.com", rec.name)
	assert.Equal(t, "https://example.com", rec.url)
	assert.Contains(t, buf.String(), "added repo example.com via dnf")
}

func TestRepoAddCmd_SingleURL_Undervivable(t *testing.T) {
	t.Parallel()

	_, err := execCmd(t, []string{"repo", "add", "https://example.com/..", "-m", "dnf", "-y"},
		[]manager.Adapter{newRepoAddRecorder("dnf")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot derive repository name")
	assert.Equal(t, ExitUsage, exitCodeFor(err))
}

func TestRepoAddCmd_NameOnly_StillWorks(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, args string }{
		{"tap", "mytap"},
		{"copr", "petersen/cava"},
		{"ppa", "ppa:git-core/ppa"},
		{"brew tap", "homebrew/cask"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := newRepoAddRecorder("dnf")
			buf, err := execCmd(t, []string{"repo", "add", tc.args, "-m", "dnf", "-y"},
				[]manager.Adapter{rec})
			require.NoError(t, err)
			assert.Equal(t, tc.args, rec.name, "name should stay the single arg for %q", tc.args)
			assert.Empty(t, rec.url)
			assert.Contains(t, buf.String(), "added repo "+tc.args)
		})
	}
}

// runRepoAdd executes the repo add command with explicit config/manifest paths.
func runRepoAdd(adapter manager.Adapter, mPath, cPath string, args []string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	root := NewRootCmd(WithAdapters([]manager.Adapter{adapter}), WithConfigPath(cPath), WithManifestPath(mPath))
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	root.SetIn(r)
	_ = w.Close() // stdin reads EOF immediately (non-interactive)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err = root.Execute()
	return buf, err
}
