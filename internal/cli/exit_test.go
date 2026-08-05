package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitCodeFor_Categories(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"usage", catErr(ErrUsage, "bad argument"), ExitUsage},
		{"data", catErr(ErrData, "bad data"), ExitDataErr},
		{"no input", catErr(ErrNoInput, "missing"), ExitNoInput},
		{"unavailable", catErr(ErrUnavailable, "nope"), ExitUnavailable},
		{"cannot create", catErr(ErrCanTCreate, "cannot write"), ExitCanTCreate},
		{"config", catErr(ErrConfig, "misconfigured"), ExitConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, exitCodeFor(tt.err))
		})
	}
}

func TestExitCodeFor_Unwrap(t *testing.T) {
	t.Parallel()
	inner := catErr(ErrUsage, "unknown type %q", "bogus")
	outer := fmt.Errorf("context: %w", inner)
	assert.Equal(t, ExitUsage, exitCodeFor(outer))
}

func TestExitCodeFor_Join(t *testing.T) {
	t.Parallel()
	err := errors.Join(
		fmt.Errorf("plain failure"),
		catErr(ErrConfig, "manifest missing"),
	)
	assert.Equal(t, ExitConfig, exitCodeFor(err))
}

func TestExitCodeFor_Default(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1, exitCodeFor(fmt.Errorf("generic error")))
	assert.Zero(t, exitCodeFor(nil))
}

func TestCategorizedError_MessagePreserved(t *testing.T) {
	t.Parallel()
	err := catErr(ErrUsage, "unknown type %q; valid types: %s", "xyz", "packages")
	assert.Equal(t, `unknown type "xyz"; valid types: packages`, err.Error())
	assert.ErrorIs(t, err, ErrUsage)
}

func TestCategorizedError_IsByIdentity(t *testing.T) {
	t.Parallel()
	err := catErr(ErrData, "boom")
	require.ErrorIs(t, err, ErrData)
	assert.NotErrorIs(t, err, ErrUsage, "must not match a different category")
}

func TestCategorizedError_Unwrap(t *testing.T) {
	t.Parallel()
	base := fmt.Errorf("root cause")
	err := catErr(ErrConfig, "wrapped: %w", base)
	require.ErrorIs(t, err, base)
	// Chain is categorizedError -> fmt.wrapError -> base.
	inner := errors.Unwrap(err)
	require.Error(t, inner)
	assert.ErrorIs(t, inner, base)
}

func TestFlagParseErrorIsUsage(t *testing.T) {
	tmpDir := t.TempDir()
	root := NewRootCmd(WithManifestPath(fmt.Sprintf("%s/manifest.toml", tmpDir)), WithConfigPath(fmt.Sprintf("%s/config.toml", tmpDir)))
	root.SetArgs([]string{"list", "--bogus"})
	err := root.Execute()
	require.Error(t, err)
	assert.Equal(t, ExitUsage, exitCodeFor(err))
}
