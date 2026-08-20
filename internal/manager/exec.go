package manager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type contextKey int

const (
	streamIOKey contextKey = iota
	outputPrefixKey
	combinedOutputKey
	stdInStringKey
)

// WithStreamIO returns a new context that signals the executor to stream I/O.
func WithStreamIO(ctx context.Context) context.Context {
	return context.WithValue(ctx, streamIOKey, true)
}

func isStreamIO(ctx context.Context) bool {
	b, _ := ctx.Value(streamIOKey).(bool)
	return b
}

// WithStdInString returns a context that feeds the given string to the child
// process's stdin instead of the terminal. Used to auto-answer a native
// manager prompt (e.g. Homebrew's "Do you want to proceed? [y/n]") when the
// operator has already consented via stamp's own gate (-y/--yes). It mirrors
// the sudo-password stdin pattern and only applies when consent is set.
func WithStdInString(ctx context.Context, s string) context.Context {
	return context.WithValue(ctx, stdInStringKey, s)
}

func getStdInString(ctx context.Context) string {
	s, _ := ctx.Value(stdInStringKey).(string)
	return s
}

// WithCombinedOutput returns a new context that signals the executor to
// capture stdout and stderr together (cmd.CombinedOutput). Used by transaction
// previews: managers such as dnf5 render their transaction UI to stderr and
// exit non-zero when the dry-run aborts, so stdout-only capture loses it.
func WithCombinedOutput(ctx context.Context) context.Context {
	return context.WithValue(ctx, combinedOutputKey, true)
}

func isCombinedOutput(ctx context.Context) bool {
	b, _ := ctx.Value(combinedOutputKey).(bool)
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

	// Pipe sudo password if cached (applies to both StreamIO and non-StreamIO paths)
	if name == "sudo" && len(sudoPassword) > 0 {
		pw := make([]byte, len(sudoPassword)+1)
		copy(pw, sudoPassword)
		pw[len(pw)-1] = '\n'
		cmd.Stdin = bytes.NewReader(pw)
	}

	if isStreamIO(ctx) {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if s := getStdInString(ctx); s != "" {
			// Auto-answer a native manager prompt when the operator has already
			// consented via stamp's gate. Takes precedence over the terminal.
			cmd.Stdin = strings.NewReader(s)
		} else if cmd.Stdin == nil {
			cmd.Stdin = os.Stdin
		}

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

	if isCombinedOutput(ctx) {
		// CombinedOutput returns the captured stdout+stderr even when the
		// command exits non-zero (e.g. a dnf --assumeno dry-run that displays
		// the transaction then aborts).
		return cmd.CombinedOutput()
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
