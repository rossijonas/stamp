// Command docgen generates CLI reference documentation (Markdown + man pages)
// from the cobra command tree. Invoked via `task docs`.
package main

import (
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/rossijonas/stamp/internal/cli"
)

func main() {
	root := cli.NewRootCmd()
	root.DisableAutoGenTag = true

	header := &doc.GenManHeader{
		Title:   "STAMP",
		Section: "1",
		Source:  "stamp https://github.com/rossijonas/stamp",
	}

	if err := os.MkdirAll("docs/usage", 0750); err != nil {
		log.Fatalf("failed to create docs/usage dir: %v", err)
	}
	if err := doc.GenMarkdownTree(root, "docs/usage"); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("docs/man", 0750); err != nil {
		log.Fatalf("failed to create docs/man dir: %v", err)
	}
	if err := doc.GenManTree(root, header, "docs/man"); err != nil {
		log.Fatal(err)
	}
}
