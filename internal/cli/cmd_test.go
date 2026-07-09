// Package cli provides the Cobra command tree for stamp.
package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args prints help", []string{}, false},
		{"install without pkg fails", []string{"install"}, true},
		{"remove without pkg fails", []string{"remove"}, true},
		{"search without pkg fails", []string{"search"}, true},
		{"repo no subcommand prints help", []string{"repo"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			root := NewRootCmd()
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, buf.String())
			}
		})
	}
}

func TestInstallAlias(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"add", "htop"})

	err := root.Execute()
	require.NoError(t, err)
}

func TestRemoveAliases(t *testing.T) {
	t.Parallel()
	for _, alias := range []string{"uninstall", "rm", "delete", "del"} {
		t.Run(alias, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			root := NewRootCmd()
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{alias, "htop"})

			err := root.Execute()
			require.NoError(t, err)
		})
	}
}

func TestRepoAddRequiresManager(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "add", "foo"})

	err := root.Execute()
	require.Error(t, err)
}

func TestRepoAddWithManager(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "add", "foo", "-m", "brew"})

	err := root.Execute()
	require.NoError(t, err)
}

func TestRepoRemoveRequiresManager(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "remove", "foo"})

	err := root.Execute()
	require.Error(t, err)
}

func TestRepoRemoveWithManager(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"repo", "remove", "foo", "-m", "brew"})

	err := root.Execute()
	require.NoError(t, err)
}

func TestRepoAliases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{"repo install alias", []string{"repo", "install", "foo", "-m", "brew"}},
		{"repo uninstall alias", []string{"repo", "uninstall", "foo", "-m", "brew"}},
		{"repo delete alias", []string{"repo", "delete", "foo", "-m", "brew"}},
		{"repo del alias", []string{"repo", "del", "foo", "-m", "brew"}},
		{"repo ls alias", []string{"repo", "ls"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			root := NewRootCmd()
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			require.NoError(t, err)
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{"verbose flag", []string{"--verbose"}},
		{"verbose short", []string{"-v"}},
		{"yes flag", []string{"--yes"}},
		{"yes short", []string{"-y"}},
		{"json flag", []string{"--json"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			root := NewRootCmd()
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			require.NoError(t, err)
			assert.NotEmpty(t, buf.String())
		})
	}
}
