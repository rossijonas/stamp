package manager

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrew_CheckInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		pkgs       []string
		formulae   string
		formulaErr error
		casks      string
		caskErr    error
		want       []string
		wantErr    string
		wantCalls  int
	}{
		{
			name:      "all present",
			pkgs:      []string{"htop", "jq"},
			formulae:  "htop\n\njq\n",
			want:      nil,
			wantCalls: 2,
		},
		{
			name:      "dependency-installed formula counts as present",
			pkgs:      []string{"lean-ctx", "tokensave"},
			formulae:  "lean-ctx\nopenssl@3\ntokensave\n",
			want:      nil,
			wantCalls: 2,
		},
		{
			name:      "missing formula returned",
			pkgs:      []string{"htop", "ghost"},
			formulae:  "htop\n",
			want:      []string{"ghost"},
			wantCalls: 2,
		},
		{
			name:      "tap-qualified entry matches short formula name",
			pkgs:      []string{"nklmilojevic/sofka/sofka"},
			formulae:  "sofka\n",
			want:      nil,
			wantCalls: 2,
		},
		{
			name:      "tap-qualified entry absent",
			pkgs:      []string{"owner/tap/ghost"},
			formulae:  "sofka\n",
			want:      []string{"owner/tap/ghost"},
			wantCalls: 2,
		},
		{
			name:      "cask present",
			pkgs:      []string{"firefox"},
			formulae:  "",
			casks:     "firefox\n",
			want:      nil,
			wantCalls: 2,
		},
		{
			name:      "cask list error tolerated",
			pkgs:      []string{"htop"},
			formulae:  "htop\n",
			caskErr:   assert.AnError,
			want:      nil,
			wantCalls: 2,
		},
		{
			name:       "formula list error fails the check",
			pkgs:       []string{"htop"},
			formulaErr: assert.AnError,
			wantErr:    "failed to list installed formulae",
			wantCalls:  1,
		},
		{
			name:      "empty input skips exec",
			pkgs:      nil,
			want:      nil,
			wantCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls [][]string
			m := NewBrew()
			m.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
				call := append([]string{name}, args...)
				calls = append(calls, call)
				joined := strings.Join(call, " ")
				switch {
				case strings.Contains(joined, brewFlagCask):
					return []byte(tt.casks), tt.caskErr
				case strings.Contains(joined, "--formula"):
					return []byte(tt.formulae), tt.formulaErr
				}
				return nil, fmt.Errorf("unexpected exec: %s", joined)
			}

			got, err := m.CheckInstalled(context.Background(), tt.pkgs)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.Len(t, calls, tt.wantCalls)

			for _, call := range calls {
				assert.Equal(t, "brew", call[0], "CheckInstalled is read-only and never needs root")
				assert.Contains(t, call, "list")
			}
		})
	}
}
