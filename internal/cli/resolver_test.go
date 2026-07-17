package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

// testAdapter is a simple adapter implementation for testing.
type testAdapter struct {
	name string
}

func (t *testAdapter) Name() string                                                  { return t.name }
func (t *testAdapter) ListInstalled(_ context.Context) ([]string, error)             { return nil, nil }
func (t *testAdapter) ListRepos(_ context.Context) ([]manager.RepositoryInfo, error) { return nil, nil }
func (t *testAdapter) Install(_ context.Context, _ string) error                     { return nil }
func (t *testAdapter) Reinstall(_ context.Context, _ string) error                   { return nil }
func (t *testAdapter) Remove(_ context.Context, _ string) error                      { return nil }
func (t *testAdapter) Search(_ context.Context, _ string) ([]string, error)          { return nil, nil }
func (t *testAdapter) AddRepo(_ context.Context, _, _ string) error                  { return nil }
func (t *testAdapter) RemoveRepo(_ context.Context, _ string) error                  { return nil }
func (t *testAdapter) Info(_ context.Context, _ string) (string, error)              { return "", nil }
func (t *testAdapter) Doctor(_ context.Context) (string, error)                      { return "mock doctor: all good", nil }
func (t *testAdapter) Update(_ context.Context) error                                { return nil }

func TestResolver_Tier1ExplicitOverride(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&testAdapter{"dnf"}, &testAdapter{"brew"}}
	r := NewResolver(adapters, &Config{Precedence: []string{"dnf", "brew"}})

	// Tier 1: override matches adapter
	a, err := r.Resolve("htop", "brew")
	require.NoError(t, err)
	assert.Equal(t, "brew", a.Name())

	// Tier 1: unknown override
	_, err = r.Resolve("htop", "unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown manager")
}

func TestResolver_Tier2PatternRule(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&testAdapter{"dnf"}, &testAdapter{"flatpak"}}
	cfg := &Config{
		Precedence: []string{"dnf", "flatpak"},
		Rules: []Rule{
			{Pattern: "^com\\.", Prefer: "flatpak"},
		},
	}
	r := NewResolver(adapters, cfg)

	// Pattern matches → picks flatpak (not dnf which has higher precedence)
	a, err := r.Resolve("com.spotify.Client", "")
	require.NoError(t, err)
	assert.Equal(t, "flatpak", a.Name())

	// No pattern match → falls back to precedence: dnf first
	a, err = r.Resolve("htop", "")
	require.NoError(t, err)
	assert.Equal(t, "dnf", a.Name())
}

func TestResolver_Tier3Fallback(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&testAdapter{"brew"}}
	r := NewResolver(adapters, &Config{})

	// No precedence or rules set → ambiguous, requires --manager
	_, err := r.Resolve("htop", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify --manager")
}

func TestResolver_NoAdapters(t *testing.T) {
	t.Parallel()
	r := NewResolver(nil, &Config{})

	_, err := r.Resolve("htop", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no package managers available")
}

func TestResolver_SkipsInvalidRegex(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&testAdapter{"brew"}}
	cfg := &Config{
		Precedence: []string{"brew"},
		Rules: []Rule{
			{Pattern: "[", Prefer: "flatpak"}, // invalid regex, should be skipped
		},
	}
	r := NewResolver(adapters, cfg)

	// Invalid regex pattern is skipped, falls through to global precedence
	a, err := r.Resolve("anything", "")
	require.NoError(t, err)
	assert.Equal(t, "brew", a.Name())
}

func TestResolver_SkipsMissingPrecedenceEntry(t *testing.T) {
	t.Parallel()
	adapters := []manager.Adapter{&testAdapter{"brew"}, &testAdapter{"dnf"}}
	cfg := &Config{
		Precedence: []string{"apt", "brew"}, // apt not installed, should skip to brew
	}
	r := NewResolver(adapters, cfg)

	a, err := r.Resolve("htop", "")
	require.NoError(t, err)
	assert.Equal(t, "brew", a.Name())
}
