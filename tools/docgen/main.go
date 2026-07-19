// Command docgen generates CLI reference documentation (Markdown + man pages)
// from the cobra command tree. Invoked via `task docs`.
package main

import (
	"fmt"
	"log"
	"os"

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

	if err := os.MkdirAll("docs/man", 0750); err != nil {
		return fmt.Errorf("failed to create docs/man dir: %w", err)
	}
	if err := doc.GenManTree(root, header, "docs/man"); err != nil {
		return fmt.Errorf("failed to generate man pages: %w", err)
	}

	return nil
}
