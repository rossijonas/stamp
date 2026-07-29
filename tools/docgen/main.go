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
		Source:  fmt.Sprintf("stamp v%s", cli.Version),
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

	if err := reformatManExamples(); err != nil {
		return fmt.Errorf("failed to reformat man examples: %w", err)
	}

	return nil
}

// reformatManExamples post-processes cobra-generated man pages to clean up
// the .SH EXAMPLE section: strips "  # " comment prefixes and wraps command
// lines in .EX/.EE for proper troff formatting.
func reformatManExamples() error {
	entries, err := os.ReadDir("docs/man")
	if err != nil {
		return fmt.Errorf("failed to read docs/man: %w", err)
	}

	exampleSection := []byte(".SH EXAMPLE")
	seeAlso := []byte(".SH SEE ALSO")

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".1") {
			continue
		}
		manPath := filepath.Join("docs/man", entry.Name())
		//nolint:gosec // manPath is scoped to docs/man/ directory, entry name from ReadDir
		data, err := os.ReadFile(manPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", manPath, err)
		}

		// Find the .SH EXAMPLE section
		startIdx := bytes.Index(data, exampleSection)
		if startIdx < 0 {
			continue
		}

		// Find the end of the EXAMPLE section (next .SH or EOF)
		endIdx := bytes.Index(data[startIdx+1:], seeAlso)
		var sectionEnd int
		if endIdx >= 0 {
			sectionEnd = startIdx + 1 + endIdx
		} else {
			sectionEnd = len(data)
		}

		// Parse the example section content (skip the .SH EXAMPLE header line)
		sectionContent := data[startIdx:sectionEnd]
		lines := bytes.Split(sectionContent, []byte("\n"))

		// Rebuild the section with formatted examples
		var newSection bytes.Buffer
		newSection.WriteString(".SH EXAMPLES\n")
		for i := 0; i < len(lines); i++ {
			line := string(lines[i])
			// Skip the original .SH EXAMPLE header itself
			if strings.HasPrefix(line, ".SH ") {
				continue
			}
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				newSection.WriteString("\n")
				continue
			}
			// Lines starting with "  # " are descriptions — strip the prefix
			if strings.HasPrefix(line, "  # ") {
				desc := strings.TrimPrefix(line, "  # ")
				newSection.WriteString(desc + "\n")
				continue
			}
			// Lines starting with "  stamp" are commands — wrap in .EX/.EE
			if strings.HasPrefix(trimmed, "stamp") {
				newSection.WriteString(".EX\n")
				newSection.WriteString(trimmed + "\n")
				newSection.WriteString(".EE\n")
				continue
			}
		}

		// Replace the old EXAMPLE section with the new one
		before := data[:startIdx]
		after := data[sectionEnd:]
		var newData []byte
		newData = append(newData, before...)
		newData = append(newData, newSection.Bytes()...)
		newData = append(newData, after...)

		//nolint:gosec // generated man pages must be world-readable
		if err := os.WriteFile(manPath, newData, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", manPath, err)
		}
	}

	return nil
}
