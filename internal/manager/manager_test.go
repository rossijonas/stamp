package manager

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutorHelper returns an Executor that injects a predefined string output.
func mockExecutorHelper(output string, err error) Executor {
	return func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(output), err
	}
}

func TestMockManager(t *testing.T) {
	t.Parallel()
	mock := &Mock{
		ManagerName:   "mock",
		InstalledPkgs: []string{"git", "curl"},
		AvailablePkgs: []string{"git", "curl", "htop", "jq", "docker"},
	}

	ctx := context.Background()

	assert.Equal(t, "mock", mock.Name())

	installed, err := mock.ListInstalled(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"git", "curl"}, installed)

	err = mock.Install(ctx, "jq")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.Contains(t, installed, "jq")

	err = mock.Install(ctx, "jq")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.Len(t, installed, 3)

	err = mock.Remove(ctx, "curl")
	require.NoError(t, err)
	installed, _ = mock.ListInstalled(ctx)
	assert.NotContains(t, installed, "curl")
	assert.Contains(t, installed, "jq")
	assert.Contains(t, installed, "git")

	results, err := mock.Search(ctx, "to")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"htop"}, results)

	err = mock.AddRepo(ctx, "test-repo", "url")
	require.NoError(t, err)
	assert.Contains(t, mock.TrackedRepos, "test-repo")

	err = mock.RemoveRepo(ctx, "test-repo")
	require.NoError(t, err)
	assert.NotContains(t, mock.TrackedRepos, "test-repo")
}

func TestMockManagerErrors(t *testing.T) {
	t.Parallel()
	expectedErr := assert.AnError
	mock := &Mock{
		ListErr:       expectedErr,
		InstallErr:    expectedErr,
		RemoveErr:     expectedErr,
		SearchErr:     expectedErr,
		AddRepoErr:    expectedErr,
		RemoveRepoErr: expectedErr,
		ListReposErr:  expectedErr,
		UpdateErr:     expectedErr,
	}

	ctx := context.Background()

	_, err := mock.ListInstalled(ctx)
	require.ErrorIs(t, err, expectedErr)

	err = mock.Install(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	err = mock.Remove(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	_, err = mock.Search(ctx, "htop")
	require.ErrorIs(t, err, expectedErr)

	err = mock.AddRepo(ctx, "repo", "url")
	require.ErrorIs(t, err, expectedErr)

	err = mock.RemoveRepo(ctx, "repo")
	require.ErrorIs(t, err, expectedErr)

	_, err = mock.ListRepos(ctx)
	require.ErrorIs(t, err, expectedErr)

	err = mock.Update(ctx)
	require.ErrorIs(t, err, expectedErr)
}

func TestParseLines(t *testing.T) {
	t.Parallel()
	input := []byte(" line1 \nline2\n\n  line3  \n")
	expected := []string{"line1", "line2", "line3"}
	actual := parseLines(input)
	assert.ElementsMatch(t, expected, actual)
}

func TestValidatePackageName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pkg   string
		valid bool
	}{
		{"htop", true},
		{"google-chrome", true},
		{"foo_bar.baz+qux", true},
		{"--remove-all", false},
		{"-y", false},
		{"foo;rm -rf /", false},
		{"curl|bash", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			t.Parallel()
			err := ValidatePackageName(tt.pkg)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPrefixWriter_SingleLine(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	pw := &prefixWriter{prefix: "[test] ", w: &buf}

	n, err := pw.Write([]byte("hello\n"))
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "[test] hello\n", buf.String())
}

func TestPrefixWriter_MultipleLines(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	pw := &prefixWriter{prefix: "[brew] ", w: &buf}

	_, err := pw.Write([]byte("line1\nline2\n"))
	require.NoError(t, err)
	assert.Equal(t, "[brew] line1\n[brew] line2\n", buf.String())
}

func TestPrefixWriter_PartialLine(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	pw := &prefixWriter{prefix: "[x] ", w: &buf}

	_, err := pw.Write([]byte("partial"))
	require.NoError(t, err)
	assert.Empty(t, buf.String())

	_, err = pw.Write([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, "[x] partial\n", buf.String())
}

func TestPrefixWriter_EmptyInput(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	pw := &prefixWriter{prefix: "[x] ", w: &buf}

	n, err := pw.Write(nil)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, buf.String())
}

func TestWithOutputPrefix(t *testing.T) {
	ctx := context.Background()
	prefixed := WithOutputPrefix(ctx, "[brew] ")

	prefix := getOutputPrefix(prefixed)
	assert.Equal(t, "[brew] ", prefix)

	empty := getOutputPrefix(ctx)
	assert.Empty(t, empty)
}
