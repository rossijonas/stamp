package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain_Runs(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"docgen"}
	defer func() { os.Args = oldArgs }()

	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	main()

	usageDir := filepath.Join(tmpDir, "docs", "usage")
	if _, err := os.Stat(usageDir); err != nil {
		t.Errorf("docs/usage directory not created: %v", err)
	}

	manDir := filepath.Join(tmpDir, "docs", "man")
	if _, err := os.Stat(manDir); err != nil {
		t.Errorf("docs/man directory not created: %v", err)
	}
}

func TestGenerate_MkdirUsageError(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	// Create "docs" as a file — MkdirAll("docs/usage") will fail
	require.NoError(t, os.WriteFile("docs", []byte{}, 0600))

	root := &cobra.Command{Use: "test"}
	header := &doc.GenManHeader{Title: "Test"}
	err := generate(root, header)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create docs/usage dir")
}

func TestGenerate_MkdirManError(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	require.NoError(t, os.MkdirAll("docs/usage", 0750))

	// Create "docs/man" as a file — MkdirAll("docs/man") will fail after usage succeeds
	require.NoError(t, os.WriteFile("docs/man", []byte{}, 0600))

	root := &cobra.Command{Use: "test"}
	header := &doc.GenManHeader{Title: "Test"}
	err := generate(root, header)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create docs/man dir")
}

func TestGenerate_GenManTreeError(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	require.NoError(t, os.MkdirAll("docs/usage", 0750))
	require.NoError(t, os.MkdirAll("docs/man", 0750))
	//nolint:gosec // test fixture: read-only dir to trigger GenManTree error
	require.NoError(t, os.Chmod("docs/man", 0500))
	t.Cleanup(func() { _ = os.Chmod("docs/man", 0700) }) //nolint:gosec // restore permissions

	root := &cobra.Command{Use: "test"}
	header := &doc.GenManHeader{Title: "Test"}
	err := generate(root, header)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate man pages")
}

func TestGenerate_MarkdownError(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	// Create docs/usage as writable, then remove write perms
	require.NoError(t, os.MkdirAll("docs/usage", 0750))
	//nolint:gosec // test fixture: read-only dir to trigger GenMarkdownTree error
	require.NoError(t, os.Chmod("docs/usage", 0500))
	t.Cleanup(func() { _ = os.Chmod("docs/usage", 0700) }) //nolint:gosec // restore permissions

	root := &cobra.Command{Use: "test"}
	header := &doc.GenManHeader{Title: "Test"}
	err := generate(root, header)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate markdown")
}
