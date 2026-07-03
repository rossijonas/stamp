package manager

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// Executor defines a function signature for running shell commands.
// This allows us to inject a mock executor during tests.
type Executor func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultExecutor is the standard implementation that uses os/exec.
func defaultExecutor(ctx context.Context, name string, args ...string) ([]byte, error) {
	//nolint:gosec // execution is restricted to hardcoded manager names
	cmd := exec.CommandContext(ctx, name, args...)
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
