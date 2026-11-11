package okf

import (
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// pathBase returns the final path segment of a bundle-relative path.
func pathBase(rel string) string {
	return path.Base(rel)
}

// frontmatterDelimiter separates the YAML frontmatter block from the body.
const frontmatterDelimiter = "---"

// maxAliasDepth bounds alias expansion so a crafted document cannot make the
// reader walk an alias chain deeper than the parser already allows.
const maxAliasDepth = 64

// frontmatterField is one top-level frontmatter key with its parsed node. When
// a key repeats, the last occurrence wins (matching the upstream reference
// implementation) and the repeat is reported as a diagnostic.
type frontmatterField struct {
	Key  string
	Node *yaml.Node
	Line int
}

// parsedDocument is the read-only result of splitting and parsing one Markdown
// file. It holds no interpretation of the content: values stay as the text the
// author wrote.
type parsedDocument struct {
	// Fields holds the top-level mapping entries. It is nil when the file has
	// no frontmatter block.
	Fields map[string]*frontmatterField
	// Order lists the top-level keys in document order, so a caller can report
	// unknown keys in a stable, author-visible order.
	Order []string
	// Body is everything after the closing delimiter. It is the raw text: no
	// trimming, re-wrapping, or line-ending normalization is applied.
	Body string
	// BodyStartLine is the 1-based line number of the first body line.
	BodyStartLine int
	// HadBOM records one leading UTF-8 BOM, which is stripped before parsing.
	HadBOM bool
	// HadCRLF records at least one CRLF line ending in the raw bytes.
	HadCRLF bool
	// Diagnostics carries format and observation findings for this file.
	Diagnostics []Diagnostic
}

// formatInvalid reports whether this document failed a format (F0) check.
func (d *parsedDocument) formatInvalid() bool {
	for _, diag := range d.Diagnostics {
		if diag.IsFormatError() {
			return true
		}
	}
	return false
}

// field returns one top-level frontmatter entry, or nil when absent.
func (d *parsedDocument) field(key string) *frontmatterField {
	if d == nil || d.Fields == nil {
		return nil
	}
	return d.Fields[key]
}

// parseDocument splits a Markdown file into frontmatter and body. It does not
// reject unknown keys, unknown types, or content outside the format.
func parseDocument(path, conceptID string, raw []byte) *parsedDocument {
	doc := &parsedDocument{BodyStartLine: 1}
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		doc.HadBOM = true
		raw = raw[3:]
		doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
			Code:      CodeBOMPresent,
			Severity:  SeverityInfo,
			Path:      path,
			ConceptID: conceptID,
			Line:      1,
			Message:   "the file starts with a UTF-8 BOM; the BOM is not part of the frontmatter",
		})
	}
	text := string(raw)
	doc.HadCRLF = strings.Contains(text, "\r\n")
	if doc.HadCRLF {
		doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
			Code:      CodeCRLFLineEndings,
			Severity:  SeverityInfo,
			Path:      path,
			ConceptID: conceptID,
			Message:   "the file uses CRLF line endings; the body bytes are reported unchanged",
		})
	}

	lines := splitLines(text)
	doc.Body = text
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != frontmatterDelimiter {
		// OKF v0.2 section 8 and 9 describe index.md and log.md as lists, not as
		// concepts; an index without frontmatter is the normal case (only the
		// bundle root index may carry `okf_version`). A concept without
		// frontmatter is a format violation under section 4.
		if !isReservedName(pathBase(path)) {
			doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
				Code:      CodeFrontmatterMissing,
				Severity:  SeverityError,
				Path:      path,
				ConceptID: conceptID,
				Line:      1,
				Message:   "OKF v0.2 section 4 requires a YAML frontmatter block starting with --- on its own line",
			})
		}
		return doc
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == frontmatterDelimiter {
			end = i
			break
		}
	}
	if end < 0 {
		doc.Diagnostics = append(doc.Diagnostics, Diagnostic{
			Code:      CodeFrontmatterUnterminat,
			Severity:  SeverityError,
			Path:      path,
			ConceptID: conceptID,
			Line:      1,
			Message:   "the YAML frontmatter block has no closing --- line",
		})
		return doc
	}

	block := strings.Join(lines[1:end], "\n")
	doc.Body = strings.Join(lines[end+1:], "\n")
	doc.BodyStartLine = end + 2

	fields, order, diags := parseFrontmatterMapping(path, conceptID, block)
	doc.Fields = fields
	doc.Order = order
	doc.Diagnostics = append(doc.Diagnostics, diags...)
	return doc
}

// parseFrontmatterMapping decodes the block as one YAML mapping. Duplicate keys,
// multiple documents, and non-mapping roots are reported; a duplicate key keeps
// the last value so the rest of the document is still readable.
func parseFrontmatterMapping(path, conceptID, block string) (map[string]*frontmatterField, []string, []Diagnostic) {
	var diags []Diagnostic
	if strings.TrimSpace(block) == "" {
		// An empty block is still a mapping with no keys. OKF requires `type`,
		// which the caller reports as TYPE_MISSING.
		return map[string]*frontmatterField{}, nil, diags
	}

	decoder := yaml.NewDecoder(strings.NewReader(block))
	var root yaml.Node
	if err := decoder.Decode(&root); err != nil {
		diags = append(diags, Diagnostic{
			Code:      CodeYAMLInvalid,
			Severity:  SeverityError,
			Path:      path,
			ConceptID: conceptID,
			Line:      1,
			Message:   "the YAML frontmatter block is not parseable: " + err.Error(),
		})
		return nil, nil, diags
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err == nil {
		diags = append(diags, Diagnostic{
			Code:      CodeYAMLMultiDocument,
			Severity:  SeverityError,
			Path:      path,
			ConceptID: conceptID,
			Line:      extra.Line,
			Message:   "the frontmatter block contains more than one YAML document",
		})
		return nil, nil, diags
	}

	mapping := resolveAlias(&root)
	// Decode yields a document node whose single child is the mapping. The
	// document wrapper is not part of the frontmatter's shape, so unwrap it
	// before asking whether the block is a mapping.
	if mapping != nil && mapping.Kind == yaml.DocumentNode && len(mapping.Content) == 1 {
		mapping = resolveAlias(mapping.Content[0])
	}
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		diags = append(diags, Diagnostic{
			Code:      CodeYAMLNotMapping,
			Severity:  SeverityError,
			Path:      path,
			ConceptID: conceptID,
			Line:      1,
			Message:   "the frontmatter block must be a YAML mapping",
		})
		return nil, nil, diags
	}

	diags = append(diags, duplicateKeyDiagnostics(path, conceptID, mapping)...)

	fields := make(map[string]*frontmatterField, len(mapping.Content)/2)
	var order []string
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := resolveAlias(mapping.Content[i])
		if keyNode == nil || keyNode.Kind != yaml.ScalarNode {
			continue
		}
		key := keyNode.Value
		if _, seen := fields[key]; !seen {
			order = append(order, key)
		}
		fields[key] = &frontmatterField{Key: key, Node: resolveAlias(mapping.Content[i+1]), Line: keyNode.Line}
	}
	return fields, order, diags
}

// duplicateKeyDiagnostics reports every repeated key at any depth. YAML requires
// mapping keys to be unique, so a repeat is a format violation rather than a
// stylistic issue; the value that wins is still the last one.
func duplicateKeyDiagnostics(path, conceptID string, n *yaml.Node) []Diagnostic {
	var diags []Diagnostic
	var walk func(node *yaml.Node, depth int)
	walk = func(node *yaml.Node, depth int) {
		node = resolveAlias(node)
		if node == nil || depth > maxAliasDepth {
			return
		}
		switch node.Kind {
		case yaml.MappingNode:
			seen := map[string]int{}
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := resolveAlias(node.Content[i])
				if key != nil && key.Kind == yaml.ScalarNode {
					if firstLine, ok := seen[key.Value]; ok {
						diags = append(diags, Diagnostic{
							Code:      CodeYAMLDuplicateKey,
							Severity:  SeverityError,
							Path:      path,
							ConceptID: conceptID,
							Line:      key.Line,
							Message:   "mapping key " + quoteKey(key.Value) + " repeats (first seen at line " + itoa(firstLine) + "); the last value wins",
						})
					} else {
						seen[key.Value] = key.Line
					}
				}
				walk(node.Content[i+1], depth+1)
			}
		case yaml.SequenceNode, yaml.DocumentNode:
			for _, child := range node.Content {
				walk(child, depth+1)
			}
		}
	}
	walk(n, 0)
	return diags
}

// resolveAlias follows YAML aliases so callers never need to special-case a
// node of kind AliasNode. A nil result means the chain could not be resolved
// within maxAliasDepth.
func resolveAlias(n *yaml.Node) *yaml.Node {
	for depth := 0; n != nil && n.Kind == yaml.AliasNode && depth < maxAliasDepth; depth++ {
		n = n.Alias
	}
	if n != nil && n.Kind == yaml.AliasNode {
		return nil
	}
	return n
}

// fieldText returns the text of a top-level frontmatter field exactly as written.
func fieldText(f *frontmatterField) (string, bool) {
	if f == nil {
		return "", false
	}
	return scalarText(f.Node)
}

// scalarText returns the text of a scalar node exactly as written. The bool
// result is false for null, non-scalar, or unresolvable nodes.
func scalarText(n *yaml.Node) (string, bool) {
	n = resolveAlias(n)
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag == "!!null" {
		return "", false
	}
	return n.Value, true
}

// stringText returns the text only when the node is a YAML string. It is used
// where the specification requires a string rather than any scalar.
func stringText(n *yaml.Node) (string, bool) {
	n = resolveAlias(n)
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
		return "", false
	}
	return n.Value, true
}

// sequenceItems returns the items of a sequence node, resolving aliases. A bare
// mapping is returned as a one-element sequence, which is the tolerant reading
// OKF v0.2 §5.2 requires for `verified` and that producers commonly rely on for
// `sources`.
func sequenceItems(n *yaml.Node) ([]*yaml.Node, bool) {
	n = resolveAlias(n)
	if n == nil {
		return nil, false
	}
	switch n.Kind {
	case yaml.SequenceNode:
		items := make([]*yaml.Node, 0, len(n.Content))
		for _, child := range n.Content {
			items = append(items, resolveAlias(child))
		}
		return items, true
	case yaml.MappingNode:
		return []*yaml.Node{n}, true
	}
	return nil, false
}

// mappingEntries returns the key/value pairs of a mapping node. When a key
// repeats, only the last pair is kept for lookup while the duplicate itself was
// already reported.
func mappingEntries(n *yaml.Node) (map[string]*yaml.Node, bool) {
	n = resolveAlias(n)
	if n == nil || n.Kind != yaml.MappingNode {
		return nil, false
	}
	entries := make(map[string]*yaml.Node, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := resolveAlias(n.Content[i])
		if key == nil || key.Kind != yaml.ScalarNode {
			continue
		}
		entries[key.Value] = resolveAlias(n.Content[i+1])
	}
	return entries, true
}

// splitLines splits on LF only and keeps any CR at the end of a line, so line
// numbers match the file the author wrote. Python's str.splitlines() splits on
// additional separators (U+2028, U+000C, ...) and so silently renumbers lines;
// this reader does not.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

func quoteKey(key string) string {
	return "\"" + strings.ReplaceAll(key, "\"", "\\\"") + "\""
}
