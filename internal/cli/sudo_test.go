package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/rossijonas/stamp/internal/manager"
)

// inflight records the peak number of concurrently running operations, so
// serial vs parallel execution can be asserted deterministically.
type inflight struct {
	mu  sync.Mutex
	cur int
	max int
}

func (f *inflight) enter() {
	f.mu.Lock()
	f.cur++
	if f.cur > f.max {
		f.max = f.cur
	}
	f.mu.Unlock()
}

func (f *inflight) exit() {
	f.mu.Lock()
	f.cur--
	f.mu.Unlock()
}

func (f *inflight) peak() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.max
}

// run records one in-flight operation, holding briefly so concurrent callers overlap.
func (f *inflight) run() error {
	f.enter()
	time.Sleep(30 * time.Millisecond)
	defer f.exit()
	return nil
}

// newPreflightCmd builds a minimal command for preflight tests; the isTerminal
// gate is controlled separately by overriding isTerminal.
func newPreflightCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func overrideSudo(t *testing.T, ready func(context.Context) bool, ensure func(context.Context) error) {
	t.Helper()
	oldReady, oldEnsure := sudoReady, sudoEnsure
	sudoReady, sudoEnsure = ready, ensure
	t.Cleanup(func() { sudoReady, sudoEnsure = oldReady, oldEnsure })
}

func overrideIsTerminal(t *testing.T, terminal bool) {
	t.Helper()
	old := isTerminal
	isTerminal = func(io.Reader) bool { return terminal }
	t.Cleanup(func() { isTerminal = old })
}

func TestSudoPreflight_NoSudoManager(t *testing.T) {
	ensureCalls := 0
	overrideSudo(t, func(context.Context) bool { return false }, func(context.Context) error {
		ensureCalls++
		return nil
	})
	overrideIsTerminal(t, true)

	ok := sudoPreflight(newPreflightCmd(), []manager.Adapter{&mockAdapter{name: "brew"}}, new(bytes.Buffer))
	assert.True(t, ok)
	assert.Zero(t, ensureCalls, "no sudo manager must not authenticate")
}

func TestSudoPreflight_AlreadyReady(t *testing.T) {
	ensureCalls := 0
	overrideSudo(t, func(context.Context) bool { return true }, func(context.Context) error {
		ensureCalls++
		return nil
	})
	overrideIsTerminal(t, true)

	ok := sudoPreflight(newPreflightCmd(), []manager.Adapter{&mockAdapter{name: "dnf"}}, new(bytes.Buffer))
	assert.True(t, ok)
	assert.Zero(t, ensureCalls, "ready sudo must not authenticate")
}

func TestSudoPreflight_NotReadyNonTTY(t *testing.T) {
	ensureCalls := 0
	overrideSudo(t, func(context.Context) bool { return false }, func(context.Context) error {
		ensureCalls++
		return nil
	})
	overrideIsTerminal(t, false)

	ok := sudoPreflight(newPreflightCmd(), []manager.Adapter{&mockAdapter{name: "dnf"}}, new(bytes.Buffer))
	assert.True(t, ok, "non-interactive: no-op, sudo -n fails fast")
	assert.Zero(t, ensureCalls)
}

func TestSudoPreflight_NotReadyTTYCachedAfterValidate(t *testing.T) {
	readyCalls := 0
	// not ready on the first probe, valid cache after `sudo -v`.
	overrideSudo(t, func(context.Context) bool {
		readyCalls++
		return readyCalls > 1
	}, func(context.Context) error { return nil })
	overrideIsTerminal(t, true)

	ok := sudoPreflight(newPreflightCmd(), []manager.Adapter{&mockAdapter{name: "dnf"}}, new(bytes.Buffer))
	assert.True(t, ok, "validated cache permits parallel")
}

func TestSudoPreflight_NotReadyTTYNonCaching(t *testing.T) {
	overrideSudo(t, func(context.Context) bool { return false }, func(context.Context) error { return nil })
	overrideIsTerminal(t, true)

	ok := sudoPreflight(newPreflightCmd(), []manager.Adapter{&mockAdapter{name: "dnf"}}, new(bytes.Buffer))
	assert.False(t, ok, "non-caching sudo policy must serialize")
}

func TestSudoPreflight_ValidateErrorSerializesAndWarnsOnce(t *testing.T) {
	overrideSudo(t, func(context.Context) bool { return false }, func(context.Context) error { return assert.AnError })
	overrideIsTerminal(t, true)

	cmd := newPreflightCmd()
	adapters := []manager.Adapter{&mockAdapter{name: "apt"}}
	var buf bytes.Buffer

	assert.False(t, sudoPreflight(cmd, adapters, &buf))
	assert.False(t, sudoPreflight(cmd, adapters, &buf))
	assert.Equal(t, 1, strings.Count(buf.String(), "sudo validation failed"), "warning emitted once per command")
}

func TestSudoPreflight_CanceledContextStaysQuiet(t *testing.T) {
	readyCalls, ensureCalls := 0, 0
	overrideSudo(t,
		func(context.Context) bool { readyCalls++; return false },
		func(context.Context) error { ensureCalls++; return assert.AnError },
	)
	overrideIsTerminal(t, true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := newPreflightCmd()
	cmd.SetContext(ctx)

	var buf bytes.Buffer
	assert.False(t, sudoPreflight(cmd, []manager.Adapter{&mockAdapter{name: "apt"}}, &buf))
	assert.Empty(t, buf.String(), "canceled context must not emit the sudo warning")
	assert.Zero(t, readyCalls, "canceled context must not probe sudo")
	assert.Zero(t, ensureCalls, "canceled context must not authenticate sudo")
}
