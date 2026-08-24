package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

// notifySigint returns a channel that receives SIGINT. Overridable in tests to
// avoid delivering real signals to the test process.
var notifySigint = func() chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT)
	return ch
}

// killJobGroup sends SIGKILL to the entire job process group so no child
// (including privileged sudo/dnf) survives a forced abort. Overridable in tests.
var killJobGroup = func() {
	_ = syscall.Kill(-syscall.Getpgrp(), syscall.SIGKILL)
}

// forceExit terminates the process immediately. Overridable in tests.
var forceExit = func(code int) { os.Exit(code) }

// restoreTerminalEcho re-enables terminal echo via stty, guarding against a
// child (e.g. sudo) leaving the terminal echo disabled when interrupted.
func restoreTerminalEcho() {
	if stty, err := lookPath("stty"); err == nil {
		//nolint:gosec // stty path comes from LookPath, not user input
		_ = exec.Command(stty, "echo").Run()
	}
}

// cancelOnInterrupt installs a two-phase SIGINT handler on the given context:
//   - First SIGINT: print a newline, restore terminal echo, and cancel ctx so
//     exec.CommandContext kills the running child.
//   - Second SIGINT: SIGKILL the whole job process group and exit 130 so no
//     orphaned (possibly privileged) child survives a forced abort.
//
// The returned cleanup stops the handler and blocks until its goroutine exits.
func cancelOnInterrupt(ctx context.Context, errOut io.Writer) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	return ctx, watchSigint(errOut, notifySigint(), cancel)
}

// watchSigint runs the two-phase SIGINT handler described by cancelOnInterrupt.
// It takes ownership of cancel: the first SIGINT invokes it, and the returned
// cleanup invokes it again after stopping the handler.
func watchSigint(errOut io.Writer, sigCh chan os.Signal, cancel context.CancelFunc) func() {
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		select {
		case <-sigCh:
			_, _ = fmt.Fprintln(errOut)
			restoreTerminalEcho()
			cancel()
			select {
			case <-sigCh:
				killJobGroup()
				forceExit(130)
			case <-done:
			}
		case <-done:
		}
	}()

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			close(done)
			signal.Stop(sigCh)
			cancel()
			<-exited
		})
	}
	return cleanup
}
