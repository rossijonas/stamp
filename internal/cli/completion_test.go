package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestCompletion_AllShells_Stdout(t *testing.T) {
	t.Parallel()
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			t.Parallel()
			buf, err := execCmd(t, []string{"completion", shell, "--stdout"}, []manager.Adapter{})
			require.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestCompletion_InvalidShell(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"completion", "invalid"}, []manager.Adapter{})
	require.Error(t, err)
}

func TestCompletion_ExtraArgs(t *testing.T) {
	t.Parallel()
	_, err := execCmd(t, []string{"completion", "bash", "extra"}, []manager.Adapter{})
	require.Error(t, err)
}

func TestCompletion_Help(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"completion", "--help"}, []manager.Adapter{})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Generate")
	assert.Contains(t, output, "bash")
	assert.Contains(t, output, "zsh")
	assert.Contains(t, output, "fish")
	assert.Contains(t, output, "powershell")
}

func TestCompletion_StdoutFlag(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"completion", "bash", "--stdout"}, []manager.Adapter{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "bash")
	assert.Contains(t, buf.String(), "stamp")
}

func TestCompletion_AutoDetect(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	adapters := []manager.Adapter{&manager.Mock{ManagerName: "brew"}}

	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	zfDir := filepath.Join(home, ".zfunc")
	require.NoError(t, os.MkdirAll(zfDir, 0750))
	t.Setenv("HOME", home)

	buf, err := execCmd(t, []string{"completion"}, adapters)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "completion installed to")
	assert.Contains(t, output, "_stamp")
	assert.Contains(t, output, "add BEFORE compinit")
	assert.Contains(t, output, "fpath=("+zfDir)
	assert.Contains(t, output, "autoload -U compinit; compinit")
}

func TestCompletion_StripCompdef(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "_stamp")
	content := "#compdef stamp\ncompdef _stamp stamp\n\n_stamp() { : }\n"
	//nolint:gosec // test fixture
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))
	require.NoError(t, stripZshCompdef(tmpFile))
	//nolint:gosec // test readback
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "#compdef stamp")
	assert.NotContains(t, string(data), "compdef _stamp")
}

func TestCompletion_ZshInstructions(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	zfDir := filepath.Join(home, ".zfunc")
	require.NoError(t, os.MkdirAll(zfDir, 0750))
	t.Setenv("HOME", home)

	buf, err := execCmd(t, []string{"completion"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "completion installed to")
	assert.Contains(t, output, "add BEFORE compinit")
	assert.Contains(t, output, "fpath=("+zfDir)
	assert.Contains(t, output, "autoload -U compinit; compinit")
}

func TestCompletion_BashNoInstructions(t *testing.T) {
	t.Parallel()
	buf, err := execCmd(t, []string{"completion", "bash"}, []manager.Adapter{&manager.Mock{ManagerName: "brew"}})
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "completion installed to")
	assert.NotContains(t, output, "compinit")
}

func TestCompletion_DetectShell(t *testing.T) {
	tests := []struct {
		env  string
		want string
	}{
		{"/bin/bash", "bash"},
		{"/bin/zsh", "zsh"},
		{"/usr/bin/fish", "fish"},
		{"/bin/sh", "sh"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Setenv("SHELL", tt.env)
			got := detectShell()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompletion_NoShellDetected(t *testing.T) {
	t.Setenv("SHELL", "")
	shell := detectShell()
	assert.Empty(t, shell)
}

func TestCompletion_RunCompletion_NoShell(t *testing.T) {
	t.Setenv("SHELL", "")
	root := NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetErr(buf)
	runCompletion(root)
	assert.Contains(t, buf.String(), "cannot detect shell")
}

func TestCompletion_InstallPath(t *testing.T) {
	t.Parallel()
	path := completionPath("bash")
	assert.Contains(t, path, "stamp")
	path = completionPath("zsh")
	assert.Contains(t, path, "_stamp")
	path = completionPath("fish")
	assert.Contains(t, path, "stamp.fish")
	path = completionPath("powershell")
	assert.Empty(t, path)
}
