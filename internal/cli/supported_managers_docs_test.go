package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adapterConstructorRe matches adapter registrations in package source files,
// e.g. manager.NewMacPorts. The constructor suffix lowercased is the manager
// name. The trailing "(" is optional so method values are still counted.
var adapterConstructorRe = regexp.MustCompile(`manager\.New([A-Za-z]+)`)

// managerTableHeaders lists the first header cells whose tables enumerate
// managers. Other tables in the same files describe commands, not the set.
var managerTableHeaders = map[string]bool{
	"manager":   true,
	"adapter":   true,
	"tool":      true,
	"toolchain": true,
}

var managerNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// nonManagerLabels are table cells that summarize the remaining managers
// instead of naming one, e.g. "All" or "Others". They are not manager rows.
var nonManagerLabels = map[string]bool{
	"all":    true,
	"other":  true,
	"others": true,
	"none":   true,
}

var markdownLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// adapterNamesFromCode returns the manager names registered by the cli package.
// It scans every non-test .go file so moving a registration out of root.go does
// not silently shrink the set.
func adapterNamesFromCode(t *testing.T) map[string]bool {
	t.Helper()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	names := make(map[string]bool)
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		//nolint:gosec // fixed package source files, not user input
		src, err := os.ReadFile(name)
		require.NoError(t, err)
		scanned++

		for _, m := range adapterConstructorRe.FindAllStringSubmatch(string(src), -1) {
			names[strings.ToLower(m[1])] = true
		}
	}

	require.Positive(t, scanned, "no non-test Go files found")
	require.NotEmpty(t, names, "no adapter constructors found in package sources")
	return names
}

// managerNamesFromDoc extracts manager names from the Markdown tables in path.
func managerNamesFromDoc(t *testing.T, path string) map[string]bool {
	t.Helper()

	//nolint:gosec // path is a fixed repo docs file supplied by the test, not user input
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return extractManagerNames(string(data))
}

// extractManagerNames returns manager names from every Markdown table whose
// first header cell is Manager, Adapter, Tool, or Toolchain. The docs contract
// is that these tables enumerate implemented managers only: planned work lives
// in prose and issues. Equality with the registered adapter set is intentional.
func extractManagerNames(content string) map[string]bool {
	names := make(map[string]bool)
	lines := strings.Split(content, "\n")

	for i := 0; i+1 < len(lines); i++ {
		header := splitTableRow(lines[i])
		if len(header) == 0 || !managerTableHeaders[strings.ToLower(header[0])] {
			continue
		}
		if !isTableSeparator(lines[i+1]) {
			continue
		}
		for j := i + 2; j < len(lines); j++ {
			row := splitTableRow(lines[j])
			if len(row) == 0 {
				break
			}
			if name := normalizeManagerCell(row[0]); name != "" {
				names[name] = true
			}
		}
	}

	return names
}

// splitTableRow splits a Markdown table row, or returns nil when line is not a
// row. Escaped pipes (\|) stay inside their cell.
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = strings.TrimSuffix(line, "|")
	}

	var (
		cells   []string
		current strings.Builder
		escaped bool
	)
	for _, r := range line {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '|':
			cells = append(cells, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	cells = append(cells, strings.TrimSpace(current.String()))

	return cells
}

// isTableSeparator reports whether line is a Markdown table separator row.
func isTableSeparator(line string) bool {
	cells := splitTableRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if strings.Trim(c, ":- ") != "" {
			return false
		}
	}
	return true
}

// normalizeManagerCell turns a table cell into a manager name. It handles bold,
// backticks, Markdown links, and compound labels such as "DNF / YUM" or
// "`go install`".
func normalizeManagerCell(cell string) string {
	if link := markdownLinkRe.FindStringSubmatch(cell); link != nil {
		cell = link[1]
	}
	cell = strings.NewReplacer("**", "", "`", "").Replace(cell)
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return ""
	}

	fields := strings.FieldsFunc(cell, func(r rune) bool {
		return r == '/' || r == ' ' || r == '\t'
	})
	if len(fields) == 0 {
		return ""
	}

	name := strings.ToLower(fields[0])
	if nonManagerLabels[name] || !managerNameRe.MatchString(name) {
		return ""
	}
	return name
}

// TestManagerTableParsing covers the Markdown edge cases the drift test relies on.
func TestManagerTableParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     []string
	}{
		{
			name:     "plain manager table",
			markdown: "| Manager | OS |\n| :--- | :--- |\n| apt | Debian |\n| brew | macOS |\n",
			want:     []string{"apt", "brew"},
		},
		{
			name:     "escaped pipe inside a cell",
			markdown: "| Manager | Notes |\n| --- | --- |\n| apt | uses a \\| b |\n",
			want:     []string{"apt"},
		},
		{
			name:     "markdown link cell",
			markdown: "| Manager | Notes |\n| --- | --- |\n| [`go`](https://go.dev) | toolchain |\n",
			want:     []string{"go"},
		},
		{
			name:     "bold and compound labels",
			markdown: "| Adapter | Notes |\n| --- | --- |\n| **DNF / YUM** | alias |\n| `go install` | toolchain |\n",
			want:     []string{"dnf", "go"},
		},
		{
			name:     "empty cell is skipped",
			markdown: "| Manager | OS |\n| --- | --- |\n|  | Debian |\n| brew | macOS |\n",
			want:     []string{"brew"},
		},
		{
			name:     "table ending at EOF without newline",
			markdown: "| Toolchain | OS |\n| --- | --- |\n| cargo | Linux |",
			want:     []string{"cargo"},
		},
		{
			name:     "pipe-only line does not end the table",
			markdown: "| Manager | OS |\n| --- | --- |\n|\n| pacman | Arch |\n",
			want:     []string{"pacman"},
		},
		{
			name:     "non-manager tables are ignored",
			markdown: "| Feature | DNF |\n| --- | --- |\n| Install | yes |\n",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractManagerNames(tt.markdown)
			if tt.want == nil {
				assert.Empty(t, got)
				return
			}
			names := make([]string, 0, len(got))
			for name := range got {
				names = append(names, name)
			}
			sort.Strings(names)
			assert.Equal(t, tt.want, names)
		})
	}
}

// authoritativeDocs are the pages that must enumerate every registered manager.
// Other docs may list a subset (command-specific tables); they are checked for
// unknown names only.
var authoritativeDocs = map[string]bool{
	"getting-started/supported-managers.md": true,
	"project/features.md":                   true,
	"project/spec.md":                       true,
	"history/os-manager-matrix.md":          true,
}

// TestSupportedManagersPageMatchesAdapters fails when a manager enumeration
// drifts from the adapters registered in the cli package.
func TestSupportedManagersPageMatchesAdapters(t *testing.T) {
	t.Parallel()

	want := adapterNamesFromCode(t)
	checked := make(map[string]bool)

	err := filepath.WalkDir("../../docs", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "_site" || entry.Name() == ".jekyll-cache" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		got := managerNamesFromDoc(t, path)
		if len(got) == 0 {
			return nil
		}

		rel, err := filepath.Rel("../../docs", path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		missing, extra := diffManagerSets(want, got)
		if authoritativeDocs[rel] {
			checked[rel] = true
			assert.Empty(t, missing, "%s: missing managers", rel)
		}
		assert.Empty(t, extra, "%s: managers not registered in the package (aliases are not rows)", rel)
		return nil
	})
	require.NoError(t, err)

	for rel := range authoritativeDocs {
		assert.True(t, checked[rel], "authoritative docs file not checked: %s", rel)
	}
}

// diffManagerSets returns the names in want but not got, and in got but not want.
func diffManagerSets(want, got map[string]bool) (missing, extra []string) {
	for name := range want {
		if !got[name] {
			missing = append(missing, name)
		}
	}
	for name := range got {
		if !want[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}
