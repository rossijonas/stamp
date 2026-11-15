package manager

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSpecForManager(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		manager string
		spec    string
		want    string
		wantErr string
	}{
		{name: "apt name", manager: "apt", spec: "htop", want: "htop"},
		{name: "apt version", manager: "apt", spec: "htop=3.2.1", want: "htop"},
		{name: "apt release", manager: "apt", spec: "htop/bookworm", want: "htop"},
		{name: "apt arch", manager: "apt", spec: "htop:amd64", want: "htop"},
		{name: "apt plus name", manager: "apt", spec: "libstdc++", want: "libstdc++"},
		{name: "apt modifier plus", manager: "apt", spec: "htop+", wantErr: "apt modifier"},
		{name: "apt modifier minus", manager: "apt", spec: "htop-", wantErr: "apt modifier"},

		{name: "dnf name", manager: "dnf", spec: "htop", want: "htop"},
		{name: "dnf nevra", manager: "dnf", spec: "htop-3.2.2-1.fc44.x86_64", want: "htop-3.2.2-1.fc44.x86_64"},
		{name: "dnf file provides", manager: "dnf", spec: "/usr/bin/htop", want: "/usr/bin/htop"},
		{name: "dnf rich dep", manager: "dnf", spec: "(htop or atop)", want: "(htop or atop)"},
		{name: "dnf provides comparison", manager: "dnf", spec: "htop >= 3.0", want: "htop >= 3.0"},
		{name: "dnf plain space", manager: "dnf", spec: "foo bar", wantErr: "invalid"},
		{name: "dnf group", manager: "dnf", spec: "@development-tools", wantErr: "use --group"},
		{name: "dnf non-path slash", manager: "dnf", spec: "owner/tap/formula", wantErr: "file-provides"},
		{name: "dnf url", manager: "dnf", spec: "https://example.com/foo.rpm", wantErr: "paths and URLs"},

		{name: "pacman name", manager: "pacman", spec: "htop", want: "htop"},
		{name: "pacman repo", manager: "pacman", spec: "extra/htop", want: "htop"},
		{name: "pacman constraint", manager: "pacman", spec: "htop>=3.2", want: "htop"},

		{name: "zypper name", manager: "zypper", spec: "htop", want: "htop"},
		{name: "zypper repo", manager: "zypper", spec: "oss:htop", want: "htop"},
		{name: "zypper capability", manager: "zypper", spec: "htop>=3.2", want: "htop"},
		{name: "zypper arch", manager: "zypper", spec: "htop.x86_64", want: "htop"},
		{name: "zypper dotted name", manager: "zypper", spec: "python3.11", want: "python3.11"},
		{name: "zypper pattern", manager: "zypper", spec: "pattern:lamp_server", wantErr: "selectors"},

		{name: "snap name", manager: "snap", spec: "hello", want: "hello"},
		{name: "snap instance", manager: "snap", spec: "hello_foo", want: "hello_foo"},
		{name: "snap channel inline", manager: "snap", spec: "hello=stable", wantErr: "invalid"},

		{name: "flatpak id", manager: "flatpak", spec: "org.gnome.Calculator", want: "org.gnome.Calculator"},
		{name: "flatpak ref", manager: "flatpak", spec: "app/org.gnome.Calculator/x86_64/stable", wantErr: "app ID"},

		{name: "brew formula", manager: "brew", spec: "htop", want: "htop"},
		{name: "brew tap", manager: "brew", spec: "owner/tap/formula", want: "owner/tap/formula"},
		{name: "brew versioned", manager: "brew", spec: "python@3.14", want: "python@3.14"},

		{name: "macports name", manager: "macports", spec: "htop", want: "htop"},
		{name: "macports version", manager: "macports", spec: "htop@1.2_0", want: "htop"},
		{name: "macports variant", manager: "macports", spec: "htop+universal", wantErr: "variants"},

		{name: "go path", manager: "go", spec: "github.com/example/tool", want: "github.com/example/tool"},
		{name: "go version", manager: "go", spec: "github.com/example/tool@v1.2.3", wantErr: "version pins"},

		{name: "cargo name", manager: "cargo", spec: "ripgrep", want: "ripgrep"},
		{name: "cargo version", manager: "cargo", spec: "ripgrep@14.1.0", want: "ripgrep"},

		{name: "npm scoped", manager: "npm", spec: "@scope/name@latest", want: "@scope/name"},

		{name: "pipx name", manager: "pipx", spec: "black", want: "black"},
		{name: "pipx version", manager: "pipx", spec: "black==24.1.0", want: "black"},
		{name: "pipx extras", manager: "pipx", spec: "black[colorama]", want: "black"},
		{name: "pipx vcs", manager: "pipx", spec: "git+https://example.com/x", wantErr: "VCS"},

		{name: "uv name", manager: "uv", spec: "ruff", want: "ruff"},
		{name: "uv equals", manager: "uv", spec: "ruff==0.6.0", want: "ruff"},
		{name: "uv at", manager: "uv", spec: "ruff@0.6.0", want: "ruff"},
		{name: "uv extras", manager: "uv", spec: "ruff[dev]", want: "ruff"},

		{name: "empty", manager: "apt", spec: "", wantErr: "cannot be empty"},
		{name: "leading dash", manager: "apt", spec: "-x", wantErr: "start with '-'"},
		{name: "shell metachar", manager: "apt", spec: "htop;rm", wantErr: "shell metacharacters"},
		{name: "control char", manager: "apt", spec: "htop\n", wantErr: "control characters"},
		{name: "too long", manager: "apt", spec: strings.Repeat("a", 513), wantErr: "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec, err := ParseSpecForManager(tt.manager, tt.spec)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.spec, spec.Raw)
			assert.Equal(t, tt.want, spec.Name)
		})
	}
}
