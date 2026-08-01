package cli

import (
	"bytes"
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sendAndWait(t *testing.T, sigCh chan os.Signal) {
	t.Helper()
	sigCh <- os.Interrupt
	require.Eventually(t, func() bool {
		return len(sigCh) == 0
	}, time.Second, time.Millisecond)
}

func TestCancelOnInterrupt_FirstSignalCancels(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	old := notifySigint
	notifySigint = func() chan os.Signal { return sigCh }
	t.Cleanup(func() { notifySigint = old })

	buf := new(bytes.Buffer)
	derived, cleanup := cancelOnInterrupt(context.Background(), buf)
	defer cleanup()

	require.NoError(t, derived.Err())
	sendAndWait(t, sigCh)
	require.Eventually(t, func() bool {
		return derived.Err() == context.Canceled
	}, time.Second, time.Millisecond)
	assert.Equal(t, "\n", buf.String())
}

func TestCancelOnInterrupt_SecondSignalForceExits(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	oldNotify := notifySigint
	notifySigint = func() chan os.Signal { return sigCh }
	t.Cleanup(func() { notifySigint = oldNotify })

	var groupKilled atomic.Bool
	oldKill := killJobGroup
	killJobGroup = func() { groupKilled.Store(true) }
	t.Cleanup(func() { killJobGroup = oldKill })

	var exitCode atomic.Int64
	var exited atomic.Bool
	oldExit := forceExit
	forceExit = func(code int) { exitCode.Store(int64(code)); exited.Store(true) }
	t.Cleanup(func() { forceExit = oldExit })

	derived, cleanup := cancelOnInterrupt(context.Background(), new(bytes.Buffer))
	defer cleanup()

	sendAndWait(t, sigCh)
	require.Eventually(t, func() bool {
		return derived.Err() == context.Canceled
	}, time.Second, time.Millisecond)

	sigCh <- os.Interrupt
	require.Eventually(t, func() bool {
		return exited.Load() && groupKilled.Load()
	}, time.Second, time.Millisecond)
	assert.Equal(t, int64(130), exitCode.Load())
}

func TestCancelOnInterrupt_CleanupStopsHandler(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	oldNotify := notifySigint
	notifySigint = func() chan os.Signal { return sigCh }
	t.Cleanup(func() { notifySigint = oldNotify })

	var groupKilled atomic.Bool
	oldKill := killJobGroup
	killJobGroup = func() { groupKilled.Store(true) }
	t.Cleanup(func() { killJobGroup = oldKill })

	var exited atomic.Bool
	oldExit := forceExit
	forceExit = func(int) { exited.Store(true) }
	t.Cleanup(func() { forceExit = oldExit })

	buf := new(bytes.Buffer)
	_, cleanup := cancelOnInterrupt(context.Background(), buf)
	cleanup()
	cleanup() // idempotent

	sigCh <- os.Interrupt
	time.Sleep(50 * time.Millisecond)
	assert.False(t, exited.Load())
	assert.False(t, groupKilled.Load())
	assert.Empty(t, buf.String())
}

func TestCancelOnInterrupt_FirstSignalThenCleanupStopsHandler(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	oldNotify := notifySigint
	notifySigint = func() chan os.Signal { return sigCh }
	t.Cleanup(func() { notifySigint = oldNotify })

	var exited atomic.Bool
	oldExit := forceExit
	forceExit = func(int) { exited.Store(true) }
	t.Cleanup(func() { forceExit = oldExit })
	oldKill := killJobGroup
	killJobGroup = func() {}
	t.Cleanup(func() { killJobGroup = oldKill })

	derived, cleanup := cancelOnInterrupt(context.Background(), new(bytes.Buffer))
	sendAndWait(t, sigCh)
	require.Eventually(t, func() bool {
		return derived.Err() == context.Canceled
	}, time.Second, time.Millisecond)

	cleanup()
	sigCh <- os.Interrupt
	time.Sleep(50 * time.Millisecond)
	assert.False(t, exited.Load())
}

func TestRestoreTerminalEcho(t *testing.T) {
	old := lookPath
	t.Cleanup(func() { lookPath = old })

	lookPath = func(name string) (string, error) {
		assert.Equal(t, "stty", name)
		return "", assert.AnError
	}
	restoreTerminalEcho() // stty missing: no-op, no panic

	lookPath = func(name string) (string, error) {
		return name, nil
	}
	restoreTerminalEcho() // stty present: executes without panic
}
