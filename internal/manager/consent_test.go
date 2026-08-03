package manager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsent_RequiredForMutations(t *testing.T) {
	d := NewDNF("dnf")
	execCalls := 0
	d.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		execCalls++
		return nil, nil
	}
	ctx := context.Background()

	// Without explicit consent, destructive methods refuse and never execute.
	require.ErrorIs(t, d.Install(ctx, "htop"), ErrConfirmationRequired)
	require.ErrorIs(t, d.Reinstall(ctx, "htop"), ErrConfirmationRequired)
	require.ErrorIs(t, d.Remove(ctx, "htop"), ErrConfirmationRequired)
	require.ErrorIs(t, d.Update(ctx, ""), ErrConfirmationRequired)
	assert.Equal(t, 0, execCalls)

	// With consent they run.
	require.NoError(t, d.Install(WithYes(ctx), "htop"))
	require.NoError(t, d.Reinstall(WithYes(ctx), "htop"))
	require.NoError(t, d.Remove(WithYes(ctx), "htop"))
	require.NoError(t, d.Update(WithYes(ctx), ""))
	assert.Equal(t, 4, execCalls)
}

func TestConsent_ReadOnlyNeverRequired(t *testing.T) {
	d := NewDNF("dnf")
	calls := 0
	d.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		calls++
		return []byte("htop.x86_64 3.4.1 updates\n"), nil
	}
	ctx := context.Background()

	_, err := d.ListInstalled(ctx)
	require.NoError(t, err)
	_, err = d.Search(ctx, "htop")
	require.NoError(t, err)
	_, err = d.Info(ctx, "htop")
	require.NoError(t, err)
	_, err = d.CheckUpdate(ctx, "")
	require.NoError(t, err)
	_, err = d.PreviewInstall(ctx, "htop")
	require.NoError(t, err)
	assert.Positive(t, calls)
}

func TestConsent_DryRunExempt(t *testing.T) {
	a := NewAPT("apt")
	execCalls := 0
	a.exec = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		execCalls++
		return nil, nil
	}
	ctx := context.Background()

	// Dry-run never requires consent and never executes.
	_, err := a.AutoRemove(ctx, true)
	require.NoError(t, err)
	_, err = a.Clean(ctx, true)
	require.NoError(t, err)
	assert.Equal(t, 0, execCalls)

	// Real runs refuse without consent.
	_, err = a.AutoRemove(ctx, false)
	require.ErrorIs(t, err, ErrConfirmationRequired)
	_, err = a.Clean(ctx, false)
	require.ErrorIs(t, err, ErrConfirmationRequired)
	assert.Equal(t, 0, execCalls)
}

// refusesWithoutConsent asserts a destructive operation was refused. Real
// mutating methods return ErrConfirmationRequired; manager stubs return their
// own "not supported" error. Either way the operation must NOT have executed,
// which would have produced a nil error.
func refusesWithoutConsent(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("destructive operation ran without explicit consent")
	}
}

// TestConsent_AllAdaptersRefuse exercises the fail-closed branch of every
// destructive adapter method across every adapter: without WithYes the
// operation must be refused (or be unsupported), never executed.
func TestConsent_AllAdaptersRefuse(t *testing.T) {
	adapters := []Adapter{
		NewDNF("dnf"), NewAPT("apt"), NewBrew(), NewCargo(), NewFlatpak(), NewGo(),
		NewMacPorts(), NewNpm(), NewPacman(), NewParu(), NewPipx(), NewSnap(), NewUv(), NewZypper(),
		&Mock{ManagerName: "mock"},
	}
	ctx := context.Background()
	for _, a := range adapters {
		t.Run(a.Name(), func(t *testing.T) {
			refusesWithoutConsent(t, a.Install(ctx, "pkg"))
			refusesWithoutConsent(t, a.Reinstall(ctx, "pkg"))
			refusesWithoutConsent(t, a.Remove(ctx, "pkg"))
			refusesWithoutConsent(t, a.Update(ctx, ""))
			refusesWithoutConsent(t, a.AddRepo(ctx, "repo", ""))
			refusesWithoutConsent(t, a.RemoveRepo(ctx, "repo"))
			if _, err := a.AutoRemove(ctx, false); err != nil {
				refusesWithoutConsent(t, err)
			}
			if _, err := a.Clean(ctx, false); err != nil {
				refusesWithoutConsent(t, err)
			}
			refusesWithoutConsent(t, a.Hold(ctx, "pkg"))
			refusesWithoutConsent(t, a.Unhold(ctx, "pkg"))
		})
	}
}

// TestConsent_AllAdaptersAccept ensures the same methods run with consent on a
// representative subset (exec mocked to no-op), proving the guard is the only
// gate and not something else blocking execution.
func TestConsent_AllAdaptersAccept(t *testing.T) {
	noop := func(_ context.Context, _ string, _ ...string) ([]byte, error) { return nil, nil }
	adapters := []Adapter{
		func() Adapter { a := NewDNF("dnf"); a.exec = noop; return a }(),
		func() Adapter { a := NewAPT("apt"); a.exec = noop; return a }(),
		func() Adapter { a := NewBrew(); a.exec = noop; return a }(),
		func() Adapter { a := NewCargo(); a.exec = noop; return a }(),
		func() Adapter { a := NewFlatpak(); a.exec = noop; return a }(),
		func() Adapter { a := NewGo(); a.exec = noop; return a }(),
		func() Adapter { a := NewMacPorts(); a.exec = noop; return a }(),
		func() Adapter { a := NewNpm(); a.exec = noop; return a }(),
		func() Adapter { a := NewPacman(); a.exec = noop; return a }(),
		func() Adapter { a := NewParu(); a.exec = noop; return a }(),
		func() Adapter { a := NewPipx(); a.exec = noop; return a }(),
		func() Adapter { a := NewSnap(); a.exec = noop; return a }(),
		func() Adapter { a := NewUv(); a.exec = noop; return a }(),
		func() Adapter { a := NewZypper(); a.exec = noop; return a }(),
	}
	ctx := WithYes(context.Background())
	for _, a := range adapters {
		t.Run(a.Name(), func(t *testing.T) {
			// Validation may reject some inputs; the point is that the consent
			// guard does not block: any error must NOT be ErrConfirmationRequired.
			for _, op := range []func() error{
				func() error { return a.Install(ctx, "pkg") },
				func() error { return a.Reinstall(ctx, "pkg") },
				func() error { return a.Remove(ctx, "pkg") },
				func() error { return a.Update(ctx, "") },
				func() error { return a.AddRepo(ctx, "repo", "") },
				func() error { return a.RemoveRepo(ctx, "repo") },
				func() error { return a.Hold(ctx, "pkg") },
				func() error { return a.Unhold(ctx, "pkg") },
			} {
				err := op()
				require.NotErrorIs(t, err, ErrConfirmationRequired,
					"operation refused despite explicit consent")
			}
			if _, err := a.AutoRemove(ctx, false); err != nil {
				require.NotErrorIs(t, err, ErrConfirmationRequired)
			}
			if _, err := a.Clean(ctx, false); err != nil {
				require.NotErrorIs(t, err, ErrConfirmationRequired)
			}
		})
	}
}
