package manager

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNF_ResolveInstalledName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cmd         string
		spec        string
		exactOut    string
		providesOut string
		globOut     string
		execErr     error
		want        string
		wantErr     string
		wantCalls   int
	}{
		{
			name:      "exact installed name",
			cmd:       "dnf",
			spec:      "htop",
			exactOut:  "htop\n",
			want:      "htop",
			wantCalls: 1,
		},
		{
			name:      "duplicate exact output deduplicated",
			cmd:       "dnf",
			spec:      "htop",
			exactOut:  "htop\nhtop\n",
			want:      "htop",
			wantCalls: 1,
		},
		{
			name:        "alias resolved via provides",
			cmd:         "dnf",
			spec:        "nodejs",
			providesOut: "nodejs22\n",
			want:        "nodejs22",
			wantCalls:   2,
		},
		{
			name:      "alias resolved via file provides",
			cmd:       "dnf",
			spec:      "yarn",
			globOut:   "nodejs-corepack\n",
			want:      "nodejs-corepack",
			wantCalls: 3,
		},
		{
			name:      "unresolved when nothing matches",
			cmd:       "dnf",
			spec:      "ghost",
			wantErr:   "installed package name could not be resolved",
			wantCalls: 3,
		},
		{
			name:        "multiple provides are ambiguous",
			cmd:         "dnf",
			spec:        "tool",
			providesOut: "tool-a\ntool-b\n",
			wantErr:     "matches multiple packages",
			wantCalls:   2,
		},
		{
			name:      "exec failure propagates",
			cmd:       "dnf",
			spec:      "htop",
			execErr:   assert.AnError,
			wantErr:   "failed to resolve installed name",
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls [][]string
			m := NewDNF(tt.cmd)
			m.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
				call := append([]string{name}, args...)
				calls = append(calls, call)
				if tt.execErr != nil {
					return nil, tt.execErr
				}
				joined := strings.Join(call, " ")
				switch {
				case strings.Contains(joined, "*/"):
					return []byte(tt.globOut), nil
				case strings.Contains(joined, "--whatprovides"):
					return []byte(tt.providesOut), nil
				default:
					return []byte(tt.exactOut), nil
				}
			}

			got, err := m.ResolveInstalledName(context.Background(), tt.spec)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.Len(t, calls, tt.wantCalls)

			for _, call := range calls {
				assert.NotEqual(t, "sudo", call[0], "resolver is read-only")
				assert.Contains(t, call, "--installed")
				assert.Contains(t, call, "--qf")
			}
		})
	}
}

func TestDNF_ResolveInstalledName_YumUsesStandaloneRepoquery(t *testing.T) {
	t.Parallel()
	var calls [][]string
	m := NewDNF("yum")
	m.exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte("htop\n"), nil
	}

	got, err := m.ResolveInstalledName(context.Background(), "htop")
	require.NoError(t, err)
	assert.Equal(t, "htop", got)
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"repoquery", "--installed", "htop", "--qf", dnfQFNameFmt}, calls[0])
}
