package main

import (
	"os"
	"path/filepath"
	"testing"
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
