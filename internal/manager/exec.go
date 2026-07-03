package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type contextKey int

const streamIOKey contextKey = iota

// WithStreamIO returns a new context that signals the executor to stream I/O.
func WithStreamIO(ctx context.Context) context.Context {
	return context.WithValue(ctx, streamIOKey, true)
}

func isStreamIO(ctx context.Context) bool {
	b, _ := ctx.Value(streamIOKey).(bool)
	return b
}

// Executor defines a function signature for running shell commands.
// This allows us to inject a mock executor during tests.
type Executor func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultExecutor is the standard implementation that uses os/exec.
func defaultExecutor(ctx context.Context, name string, args ...string) ([]byte, error) {
	//nolint:gosec // execution is restricted to hardcoded manager names
	cmd := exec.CommandContext(ctx, name, args...)

	// Graceful cancellation handling (SIGINT instead of SIGKILL)
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return cmd.Process.Signal(os.Interrupt)
		}
		return nil
	}

	if isStreamIO(ctx) {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return out, fmt.Errorf("%w: %s", err, string(exitErr.Stderr))
		}
		return out, err
	}
	return out, nil
}
