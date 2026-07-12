package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocgenMain(t *testing.T) {
	// Create temporary directories to avoid polluting or overwriting official files during tests
	tmpDir, err := os.MkdirTemp("", "docgen-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Since main() writes to docs/usage and docs/man, let's temporarily change the working directory to the temp dir,
	// or we can just let it run. But wait, main() writes to relative paths: "docs/usage" and "docs/man".
	// If we change working directory to tmpDir, it will create "docs/usage" and "docs/man" inside tmpDir!
	// This is clean and doesn't modify our repository files during go test.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	main()

	// Verify files were generated
	if _, err := os.Stat(filepath.Join("docs/usage", "stamp.md")); os.IsNotExist(err) {
		t.Errorf("stamp.md was not generated in docs/usage")
	}
	if _, err := os.Stat(filepath.Join("docs/man", "stamp.1")); os.IsNotExist(err) {
		t.Errorf("stamp.1 was not generated in docs/man")
	}
}
