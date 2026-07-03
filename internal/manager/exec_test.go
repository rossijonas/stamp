package manager

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultExecutor_Success(t *testing.T) {
	ctx := context.Background()
	// Use a command that is virtually guaranteed to exist and succeed on Unix systems.
	out, err := defaultExecutor(ctx, "echo", "hello", "world")
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(out))
}

func TestDefaultExecutor_FailureWithoutStderr(t *testing.T) {
	ctx := context.Background()
	// Use a command that will fail (exit status 1) but produces no stderr output.
	// `false` command exits with 1.
	out, err := defaultExecutor(ctx, "false")
	require.Error(t, err)
	assert.Empty(t, out)

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
}

func TestDefaultExecutor_FailureWithStderr(t *testing.T) {
	ctx := context.Background()
	// Run a shell command that writes to stderr and exits with an error
	out, err := defaultExecutor(ctx, "sh", "-c", "echo 'custom error' >&2; exit 1")
	require.Error(t, err)
	assert.Empty(t, out)

	// The error should wrap the stderr output
	assert.Contains(t, err.Error(), "custom error")
}
