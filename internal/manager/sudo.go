package manager

import (
	"context"
	"os"
)

// stdIn is overridable in tests to simulate pipe vs TTY for sudo decisions.
var stdIn = os.Stdin

// sudoCmd builds a sudo command that is TTY-aware. In non-interactive
// environments (CI/pipes) it adds -n so a required password fails fast instead
// of hanging; on an interactive terminal it omits flags so sudo prompts
// itself. Stamp never handles the password.
func sudoCmd(args ...string) []string {
	cmd := []string{"sudo"}
	stat, err := stdIn.Stat()
	if err == nil && stat.Mode()&os.ModeCharDevice == 0 {
		cmd = append(cmd, "-n")
	}
	return append(cmd, args...)
}

// sudoProbe reports whether sudo can run without a password: NOPASSWD or a
// valid credential cache. -N (no-update) + -n (non-interactive) + -v (validate)
// is sudo's documented credential check; it has no credential or timestamp
// side effects.
// Overridable in tests.
var sudoProbe = func(ctx context.Context) bool {
	_, err := defaultExecutor(ctx, "sudo", "-Nnv")
	return err == nil
}

// SudoReady reports whether the next sudo command will run without prompting.
// It never authenticates and never updates the credential cache.
func SudoReady(ctx context.Context) bool {
	return sudoProbe(ctx)
}

// sudoValidate authenticates and caches sudo credentials once, letting sudo
// prompt natively (supports password, OTP, askpass). Streams to the terminal so
// sudo's own prompt is visible. Intended to run once before a privileged phase.
// Overridable in tests.
var sudoValidate = func(ctx context.Context) error {
	_, err := defaultExecutor(WithStreamIO(ctx), "sudo", "-v")
	return err
}

// EnsureSudo validates and caches sudo credentials. Best-effort: callers treat
// a failure as a signal to degrade (serialize) and let the real command surface
// the error.
func EnsureSudo(ctx context.Context) error {
	return sudoValidate(ctx)
}
