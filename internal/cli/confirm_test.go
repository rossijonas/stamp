package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rossijonas/stamp/internal/manager"
)

func TestConfirmDestructive_YesFlagSkipsEverything(t *testing.T) {
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", RefreshErr: errors.New("offline")}
	got := confirmDestructive(context.Background(), &buf, strings.NewReader("y\n"), true, a, previewInstall, "Install", "htop")
	require.True(t, got)
	assert.Empty(t, buf.String())
}

func TestConfirmDestructive_NonTerminalAborts(t *testing.T) {
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf"}
	got := confirmDestructive(context.Background(), &buf, strings.NewReader("y\n"), false, a, previewInstall, "Install", "htop")
	require.False(t, got)
	assert.Contains(t, buf.String(), "Install: htop@1.0.0")
	assert.Contains(t, buf.String(), "aborted")
}

func TestConfirmDestructive_TerminalAccept(t *testing.T) {
	saveRestoreTerminal(t)
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf"}
	got := confirmDestructive(context.Background(), &buf, newLineReader("y"), false, a, previewInstall, "Install", "htop")
	require.True(t, got)
	assert.Contains(t, buf.String(), "Install: htop@1.0.0")
	assert.Contains(t, buf.String(), "Install htop via dnf? [y/N]: ")
	assert.NotContains(t, buf.String(), "aborted")
}

func TestConfirmDestructive_TerminalDecline(t *testing.T) {
	saveRestoreTerminal(t)
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf"}
	got := confirmDestructive(context.Background(), &buf, newLineReader("n"), false, a, previewInstall, "Install", "htop")
	require.False(t, got)
	assert.Contains(t, buf.String(), "aborted")
}

func TestConfirmDestructive_RefreshErrorWarns(t *testing.T) {
	saveRestoreTerminal(t)
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", RefreshErr: errors.New("offline")}
	got := confirmDestructive(context.Background(), &buf, newLineReader("y"), false, a, previewInstall, "Install", "htop")
	require.True(t, got)
	assert.Contains(t, buf.String(), "refresh failed: offline")
}

func TestConfirmDestructive_PreviewErrorWarnsAndPrompts(t *testing.T) {
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", PreviewInstallErr: errors.New("no dry run")}
	got := confirmDestructive(context.Background(), &buf, strings.NewReader("y\n"), false, a, previewInstall, "Install", "htop")
	require.False(t, got)
	assert.Contains(t, buf.String(), "could not render preview: no dry run")
	assert.NotContains(t, buf.String(), "Name: htop")
	assert.Contains(t, buf.String(), "aborted")
}

func TestConfirmDestructive_PreviewErrorPrompts(t *testing.T) {
	saveRestoreTerminal(t)
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", PreviewInstallErr: errors.New("no dry run")}
	got := confirmDestructive(context.Background(), &buf, newLineReader("y"), false, a, previewInstall, "Install", "htop")
	require.True(t, got) // warn-and-prompt: consent is the gate
	assert.Contains(t, buf.String(), "could not render preview: no dry run")
	assert.Contains(t, buf.String(), "Install htop via dnf? [y/N]: ")
}

func TestConfirmDestructive_NonPreviewerWarnsAndPrompts(t *testing.T) {
	var buf bytes.Buffer
	a := &mockAdapter{name: "dnf"}
	got := confirmDestructive(context.Background(), &buf, strings.NewReader("y\n"), false, a, previewInstall, "Install", "htop")
	require.False(t, got)
	assert.Contains(t, buf.String(), "could not render preview")
	assert.NotContains(t, buf.String(), "Name: htop")
}

func TestConfirmDestructive_RemoveUsesPreviewRemove(t *testing.T) {
	saveRestoreTerminal(t)
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", InstalledPkgs: []string{"htop"}}
	got := confirmDestructive(context.Background(), &buf, newLineReader("y"), false, a, previewRemove, "Remove", "htop")
	require.True(t, got)
	assert.Contains(t, buf.String(), "Remove: htop")
	assert.Contains(t, buf.String(), "Remove htop via dnf? [y/N]: ")
}

func TestConfirmDestructive_ReinstallUsesPreviewReinstall(t *testing.T) {
	saveRestoreTerminal(t)
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf"}
	got := confirmDestructive(context.Background(), &buf, newLineReader("y"), false, a, previewReinstall, "Reinstall", "htop")
	require.True(t, got)
	assert.Contains(t, buf.String(), "Reinstall: htop@1.0.0")
	assert.Contains(t, buf.String(), "Reinstall htop via dnf? [y/N]: ")
}

func TestConfirmDestructive_NoopSkipsPrompt(t *testing.T) {
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", PreviewNoop: true}
	got := confirmDestructive(context.Background(), &buf, strings.NewReader("y\n"), false, a, previewInstall, "Install", "htop")
	require.False(t, got)
	assert.Contains(t, buf.String(), "nothing to do: htop via dnf")
	assert.NotContains(t, buf.String(), "[y/N]: ")
	assert.NotContains(t, buf.String(), "aborted")
}

func TestConfirmDestructive_RemoveNoopFailsFast(t *testing.T) {
	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", PreviewNoop: true}
	got := confirmDestructive(context.Background(), &buf, strings.NewReader("y\n"), false, a, previewRemove, "Remove", "htop")
	require.False(t, got)
	assert.Contains(t, buf.String(), "nothing to do: htop via dnf")
	assert.NotContains(t, buf.String(), "[y/N]: ")
	assert.NotContains(t, buf.String(), "aborted")
}

func TestConfirmDestructive_ContextCanceledAborts(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf"}
	got := confirmDestructive(canceled, &buf, newLineReader("y"), false, a, previewInstall, "Install", "htop")
	require.False(t, got)
	assert.Contains(t, buf.String(), "aborted")
	assert.NotContains(t, buf.String(), "[y/N]: ")
}

func TestConfirmDestructive_ContextCanceledDuringPreview(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	a := &manager.Mock{ManagerName: "dnf", PreviewInstallErr: context.Canceled}
	got := confirmDestructive(canceled, &buf, newLineReader("y"), false, a, previewInstall, "Install", "htop")
	require.False(t, got)
	// The "cannot preview: context canceled" warning is suppressed; we just abort.
	assert.NotContains(t, buf.String(), "cannot preview")
	assert.Contains(t, buf.String(), "aborted")
}

func TestRequireConsent_ContextCanceledAborts(t *testing.T) {
	saveRestoreTerminal(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := &cobra.Command{}
	cmd.SetContext(context.WithValue(canceled, ctxKey{}, &AppContext{yes: false}))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	got := requireConsent(cmd, "Proceed")
	require.False(t, got)
	assert.Contains(t, buf.String(), "aborted")
	assert.NotContains(t, buf.String(), "Proceed?")
}
