package designoutput

import (
	"fmt"
	"sort"
	"strings"
)

// DocumentFormat is an output-format plugin for derived documents. A format may
// change the presentation, but it must not add, drop, reorder, or reword the
// semantic lines: CheckSemanticParity verifies that the flattened facts are the
// same. DDL is owned by the database provider and is not reformatted here.
type DocumentFormat struct {
	Name      string
	Extension string
	render    func(markdown string) string
}

var documentFormats = map[string]DocumentFormat{
	"markdown": {Name: "markdown", Extension: "md", render: func(markdown string) string { return markdown }},
	"text":     {Name: "text", Extension: "txt", render: renderText},
}

// DocumentFormats lists the available formats in a stable order.
func DocumentFormats() []string {
	names := make([]string, 0, len(documentFormats))
	for name := range documentFormats {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func documentFormat(name string) (DocumentFormat, error) {
	if strings.TrimSpace(name) == "" {
		name = "markdown"
	}
	format, ok := documentFormats[name]
	if !ok {
		return DocumentFormat{}, fmt.Errorf("unknown document format %q (want one of %v)", name, DocumentFormats())
	}
	return format, nil
}

// renderText flattens Markdown into plain text: headings lose their markers,
// table rows become tab-separated cells, separators and quotes are dropped, and
// inline code markers are removed. The order of lines is preserved.
func renderText(markdown string) string {
	return strings.Join(SemanticLines(markdown), "\n")
}

// SemanticLines is the representation compared by CheckSemanticParity. It keeps
// every fact line in order and drops only presentation: empty lines, table
// separators, heading markers, quote markers, table pipes, and inline code
// markers. It is idempotent, so a plain-text rendering compares equal to the
// Markdown it came from.
func SemanticLines(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isTableSeparator(trimmed) {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			trimmed = strings.ReplaceAll(strings.TrimSpace(strings.TrimLeft(trimmed, "#")), "`", "")
			if trimmed != "" {
				out = append(out, trimmed)
			}
			continue
		}
		for strings.HasPrefix(trimmed, ">") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
		}
		if strings.HasPrefix(trimmed, "|") {
			cells := strings.Split(strings.Trim(trimmed, "|"), "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(strings.ReplaceAll(cells[i], "`", ""))
			}
			out = append(out, strings.Join(cells, "\t"))
			continue
		}
		out = append(out, strings.ReplaceAll(trimmed, "`", ""))
	}
	return out
}

func isTableSeparator(line string) bool {
	if !strings.HasPrefix(line, "|") {
		return false
	}
	for _, cell := range strings.Split(strings.Trim(line, "|"), "|") {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		for _, r := range cell {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

// CheckSemanticParity verifies that two bundles of different document formats
// carry the same semantic lines for every document artifact. DDL scripts and
// the navigation page are excluded because they are not document-format output.
func CheckSemanticParity(left, right Bundle) error {
	leftDocs := documentArtifacts(left)
	rightDocs := documentArtifacts(right)
	if len(leftDocs) != len(rightDocs) {
		return fmt.Errorf("document artifact count differs: %d vs %d", len(leftDocs), len(rightDocs))
	}
	for name, leftContent := range leftDocs {
		rightContent, ok := rightDocs[name]
		if !ok {
			return fmt.Errorf("document artifact %q is missing from the second bundle", name)
		}
		a := SemanticLines(string(leftContent))
		b := SemanticLines(string(rightContent))
		if len(a) != len(b) {
			return fmt.Errorf("document %q has %d semantic lines vs %d", name, len(a), len(b))
		}
		for i := range a {
			if a[i] != b[i] {
				return fmt.Errorf("document %q differs at semantic line %d: %q vs %q", name, i, a[i], b[i])
			}
		}
	}
	return nil
}

func documentArtifacts(bundle Bundle) map[string][]byte {
	out := map[string][]byte{}
	for _, file := range bundle.Files {
		if file.Path == "index.html" || strings.HasSuffix(file.Path, ".sql") || file.Path == "manifest.json" {
			continue
		}
		if strings.HasPrefix(file.Path, "generated/") {
			continue
		}
		out[artifactNameForPath(bundle.Manifest, file.Path)] = file.Content
	}
	return out
}

func artifactNameForPath(manifest Manifest, path string) string {
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == path {
			return artifact.Name
		}
	}
	return path
}
