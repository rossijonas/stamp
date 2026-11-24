package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePkcheck(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "pkcheck")
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o600)) //nolint:gosec // test fixture, widened below
	require.NoError(t, os.Chmod(executable, 0o755))                            //nolint:gosec // executable fixture
	plain := filepath.Join(dir, "pkcheck-plain")
	require.NoError(t, os.WriteFile(plain, []byte("x"), 0o600))
	missing := filepath.Join(dir, "missing")

	tests := []struct {
		name       string
		candidates []string
		want       string
		wantErr    bool
	}{
		{name: "first executable wins", candidates: []string{executable, plain}, want: executable},
		{name: "skips non-executable", candidates: []string{plain, executable}, want: executable},
		{name: "skips directory", candidates: []string{dir, executable}, want: executable},
		{name: "missing", candidates: []string{missing}, wantErr: true},
		{name: "no candidates", candidates: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := pkcheckCandidates
			pkcheckCandidates = tt.candidates
			t.Cleanup(func() { pkcheckCandidates = old })

			got, err := resolvePkcheck()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNeedsPolkitAuth(t *testing.T) {
	old := polkitChallenge
	t.Cleanup(func() { polkitChallenge = old })

	tests := []struct {
		name      string
		responses map[string]bool
		err       error
		want      bool
	}{
		{name: "no challenge", responses: map[string]bool{}, want: false},
		{name: "metadata challenge", responses: map[string]bool{flatpakPolkitActions[0]: true}, want: true},
		{name: "appstream challenge", responses: map[string]bool{flatpakPolkitActions[1]: true}, want: true},
		{name: "probe error is ignored", err: assert.AnError, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polkitChallenge = func(_ context.Context, action string) (bool, error) {
				if tt.err != nil {
					return false, tt.err
				}
				return tt.responses[action], nil
			}
			assert.Equal(t, tt.want, needsPolkitAuth(context.Background()))
		})
	}
}
