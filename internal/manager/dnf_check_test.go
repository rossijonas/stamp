package manager

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNF_CheckInstalled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		cmd          string
		pkgs         []string
		installedOut []byte
		installedErr error
		provides     map[string]string
		providesErr  error
		want         []string
		wantErr      string
		wantCalls    int
	}{
		{
			name:         "all present via system set, no provides query",
			cmd:          "dnf",
			pkgs:         []string{"htop", "vim"},
			installedOut: []byte("htop\nvim\nnodejs22\n"),
			want:         nil,
			wantCalls:    1,
		},
		{
			name:         "dependency-installed package counts as present",
			cmd:          "dnf",
			pkgs:         []string{"ffmpeg-free", "flatpak"},
			installedOut: []byte("ffmpeg-free\nflatpak\nglibc\n"),
			want:         nil,
			wantCalls:    1,
		},
		{
			name:         "alias resolved via whatprovides",
			cmd:          "dnf",
			pkgs:         []string{"nodejs"},
			installedOut: []byte("nodejs22\n"),
			provides:     map[string]string{"nodejs": "nodejs22"},
			want:         nil,
			wantCalls:    2,
		},
		{
			name:         "mixed present, alias and absent",
			cmd:          "dnf",
			pkgs:         []string{"htop", "nodejs", "ghost"},
			installedOut: []byte("htop\nnodejs22\n"),
			provides:     map[string]string{"nodejs": "nodejs22"},
			want:         []string{"ghost"},
			wantCalls:    3,
		},
		{
			name:      "empty input skips queries",
			cmd:       "dnf",
			pkgs:      nil,
			want:      nil,
			wantCalls: 0,
		},
		{
			name:         "invalid name reported absent without exec",
			cmd:          "dnf",
			pkgs:         []string{"-bad", "ok"},
			installedOut: []byte("ok\n"),
			want:         []string{"-bad"},
			wantCalls:    1,
		},
		{
			name:      "invalid-only input skips queries",
			cmd:       "dnf",
			pkgs:      []string{"-bad"},
			want:      []string{"-bad"},
			wantCalls: 0,
		},
		{
			name:         "full set query failure fails whole check",
			cmd:          "dnf",
			pkgs:         []string{"htop"},
			installedErr: fmt.Errorf("boom"),
			wantErr:      "failed to list installed packages",
		},
		{
			name:         "whatprovides failure fails whole check",
			cmd:          "dnf",
			pkgs:         []string{"ghost"},
			installedOut: []byte("htop\n"),
			providesErr:  fmt.Errorf("boom"),
			wantErr:      "failed to resolve provides for ghost",
		},
		{
			name:         "available-only provider still absent",
			cmd:          "dnf",
			pkgs:         []string{"foo"},
			installedOut: []byte("unrelated\n"),
			provides:     map[string]string{},
			want:         []string{"foo"},
			wantCalls:    2,
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
				joined := strings.Join(call, " ")
				switch {
				case strings.Contains(joined, "--whatprovides"):
					if tt.providesErr != nil {
						return nil, tt.providesErr
					}
					for i, a := range args {
						if a == "--whatprovides" && i+1 < len(args) {
							if out, ok := tt.provides[args[i+1]]; ok {
								return []byte(out), nil
							}
						}
					}
					return []byte(""), nil
				case strings.Contains(joined, "--installed"):
					return tt.installedOut, tt.installedErr
				}
				return nil, fmt.Errorf("unexpected exec: %s", joined)
			}

			got, err := m.CheckInstalled(context.Background(), tt.pkgs)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Len(t, calls, tt.wantCalls)

			for _, call := range calls {
				assert.NotEqual(t, "sudo", call[0], "CheckInstalled is read-only and never needs root")
				assert.Contains(t, call, "--qf")
				// Alias resolution must be scoped to installed packages: a
				// bare --whatprovides also matches available repo providers,
				// hiding genuinely-missing packages.
				if strings.Contains(strings.Join(call, " "), "--whatprovides") {
					assert.Contains(t, call, "--installed", "whatprovides must be scoped to installed packages")
				}
			}
		})
	}
}

func TestDNF_CheckInstalled_YumUsesStandaloneRepoquery(t *testing.T) {
	t.Parallel()
	var binaries []string
	m := NewDNF("yum")
	m.exec = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		binaries = append(binaries, name)
		return []byte("vim-enhanced\nvim\n"), nil
	}

	got, err := m.CheckInstalled(context.Background(), []string{"vim"})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, []string{"repoquery"}, binaries)
}
