package manager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type contextKey int

const (
	streamIOKey contextKey = iota
	outputPrefixKey
)

// WithStreamIO returns a new context that signals the executor to stream I/O.
func WithStreamIO(ctx context.Context) context.Context {
	return context.WithValue(ctx, streamIOKey, true)
}

func isStreamIO(ctx context.Context) bool {
	b, _ := ctx.Value(streamIOKey).(bool)
	return b
}

// WithOutputPrefix returns a context with a label prefix for streaming output.
// When set, each line of the command's output is prefixed with the given string
// (e.g. "[brew] "), making concurrent output identifiable.
func WithOutputPrefix(ctx context.Context, prefix string) context.Context {
	return context.WithValue(ctx, outputPrefixKey, prefix)
}

func getOutputPrefix(ctx context.Context) string {
	p, _ := ctx.Value(outputPrefixKey).(string)
	return p
}

// prefixWriter prepends a label prefix to each line of output.
type prefixWriter struct {
	prefix string
	w      io.Writer
	buf    []byte
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	origLen := len(p)

	if len(pw.buf) > 0 {
		p = append(pw.buf, p...)
		pw.buf = pw.buf[:0]
	}

	for {
		idx := bytes.IndexByte(p, '\n')
		if idx < 0 {
			pw.buf = append(pw.buf, p...)
			break
		}
		line := p[:idx+1]
		if _, err := io.WriteString(pw.w, pw.prefix); err != nil {
			return 0, err
		}
		if _, err := pw.w.Write(line); err != nil {
			return 0, err
		}
		p = p[idx+1:]
	}

	return origLen, nil
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

		if prefix := getOutputPrefix(ctx); prefix != "" {
			cmd.Stdout = &prefixWriter{prefix: prefix, w: os.Stdout}
			cmd.Stderr = &prefixWriter{prefix: prefix, w: os.Stderr}
		}

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
