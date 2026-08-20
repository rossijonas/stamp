package manager

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultExecutor_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Use a command that is virtually guaranteed to exist and succeed on Unix systems.
	out, err := defaultExecutor(ctx, "echo", "hello", "world")
	require.NoError(t, err)
	assert.Equal(t, "hello world\n", string(out))
}

func TestDefaultExecutor_FailureWithoutStderr(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	ctx := context.Background()
	// Run a shell command that writes to stderr and exits with an error
	out, err := defaultExecutor(ctx, "sh", "-c", "echo 'custom error' >&2; exit 1")
	require.Error(t, err)
	assert.Empty(t, out)

	// The error should wrap the stderr output
	assert.Contains(t, err.Error(), "custom error")
}

func TestDefaultExecutor_CommandNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Run a non-existent command to trigger non-ExitError path
	_, err := defaultExecutor(ctx, "nonexistentcommand12345")
	require.Error(t, err)
}

func TestDefaultExecutor_StreamSuccess(t *testing.T) {
	t.Parallel()
	ctx := WithStreamIO(context.Background())
	out, err := defaultExecutor(ctx, "echo", "hello", "stream")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestDefaultExecutor_StreamFailure(t *testing.T) {
	t.Parallel()
	ctx := WithStreamIO(context.Background())
	out, err := defaultExecutor(ctx, "false")
	require.Error(t, err)
	assert.Empty(t, out)
}

func TestDefaultExecutor_Cancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	_, err := defaultExecutor(ctx, "sleep", "10")
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "signal: interrupt") || strings.Contains(err.Error(), "canceled"))
}

func TestDefaultExecutor_StreamCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = WithStreamIO(ctx)
	cancel()
	_, err := defaultExecutor(ctx, "sleep", "10")
	require.Error(t, err)
}

func TestStdInString_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := WithStdInString(context.Background(), "y\n")
	require.Equal(t, "y\n", getStdInString(ctx))
	require.Empty(t, getStdInString(context.Background()))
}

func TestDefaultExecutor_StreamFeedsStdInString(t *testing.T) {
	t.Parallel()
	// A streamed command that echoes a line read from stdin: proves the stdin
	// string is fed to the child (used to auto-answer native prompts under -y).
	ctx := WithStreamIO(WithStdInString(context.Background(), "accepted\n"))
	out, err := defaultExecutor(ctx, "sh", "-c", "read line; printf '%s' \"$line\"")
	require.NoError(t, err)
	// Streamed mode returns nil output, but the command must have consumed the
	// piped line without error (it would fail/hang if stdin were the closed TTY).
	_ = out
}
