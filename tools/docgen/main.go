// Command docgen generates CLI reference documentation (Markdown + man pages)
// from the cobra command tree. Invoked via `task docs`.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/rossijonas/stamp/internal/cli"
)

func main() {
	root := cli.NewRootCmd()
	root.DisableAutoGenTag = true

	header := &doc.GenManHeader{
		Title:   "STAMP",
		Section: "1",
		Source:  fmt.Sprintf("stamp %s", cli.Version),
		Manual:  "Stamp Manual",
	}

	if err := generate(root, header); err != nil {
		log.Fatal(err)
	}
}

func generate(root *cobra.Command, header *doc.GenManHeader) error {
	if err := os.MkdirAll("docs/usage", 0750); err != nil {
		return fmt.Errorf("failed to create docs/usage dir: %w", err)
	}
	if err := doc.GenMarkdownTree(root, "docs/usage"); err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Post-process: replace .md links with .html for Jekyll compatibility
	entries, err := os.ReadDir("docs/usage")
	if err != nil {
		return fmt.Errorf("failed to read docs/usage: %w", err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join("docs/usage", entry.Name())
		//nolint:gosec // path is from cobra docgen output (trusted source)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		data = bytes.ReplaceAll(data, []byte(".md)"), []byte(".html)"))
		// Add Jekyll front matter if missing
		if !bytes.HasPrefix(data, []byte("---\n")) {
			data = append([]byte("---\n---\n\n"), data...)
		}
		//nolint:gosec // 0644 for Jekyll-readible text files
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	if err := os.MkdirAll("docs/man", 0750); err != nil {
		return fmt.Errorf("failed to create docs/man dir: %w", err)
	}
	if err := doc.GenManTree(root, header, "docs/man"); err != nil {
		return fmt.Errorf("failed to generate man pages: %w", err)
	}

	return nil
}
