package okf

import (
	"strings"
)

// buildIndex reads one reserved `index.md`. OKF v0.2 section 8 allows frontmatter only in
// the bundle root index (and only for `okf_version`); a non-root index carrying
// frontmatter is a format error.
func buildIndex(doc rawDoc) Index {
	parsed := doc.document
	index := Index{
		Path:        doc.path,
		Digest:      doc.digest,
		IsRoot:      doc.path == "index.md",
		Entries:     []IndexEntry{},
		Diagnostics: append([]Diagnostic{}, parsed.Diagnostics...),
	}

	hasFrontmatter := parsed.Fields != nil
	if hasFrontmatter {
		if index.IsRoot {
			if field := parsed.field("okf_version"); field != nil {
				if text, ok := fieldText(field); ok {
					index.OkfVersion = text
				} else {
					index.Diagnostics = append(index.Diagnostics, Diagnostic{
						Code:     CodeRootIndexOkfVersionInvalid,
						Severity: SeverityWarning,
						Path:     doc.path,
						Line:     field.Line,
						Message:  "`okf_version` is not a scalar value; the declaration is not readable",
					})
				}
			} else {
				index.Diagnostics = append(index.Diagnostics, Diagnostic{
					Code:     CodeRootIndexOkfVersionMissing,
					Severity: SeverityInfo,
					Path:     doc.path,
					Message:  "the root index.md declares no `okf_version`; OKF v0.2 section 12 makes the declaration optional",
				})
			}
			// OKF v0.2 section 8 permits `okf_version` and nothing else in an index.
			for _, key := range parsed.Order {
				if key == "okf_version" {
					continue
				}
				index.Diagnostics = append(index.Diagnostics, Diagnostic{
					Code:     CodeIndexFrontmatterRoot,
					Severity: SeverityWarning,
					Path:     doc.path,
					Line:     parsed.Fields[key].Line,
					Message:  "OKF v0.2 section 8 reserves index frontmatter for `okf_version` in the bundle root; the key " + quoteKey(key) + " is not part of that exception",
				})
			}
		} else {
			index.Diagnostics = append(index.Diagnostics, Diagnostic{
				Code:     CodeIndexFrontmatterRoot,
				Severity: SeverityError,
				Path:     doc.path,
				Line:     1,
				Message:  "OKF v0.2 section 8 permits frontmatter only in the bundle root index.md; this index is at " + doc.path,
			})
		}
	}

	index.Entries = indexEntries(doc.path, parsed.Body, parsed.BodyStartLine)
	return index
}

// indexEntries reads the `* [Title](target) - description` items of an index body.
// Entries outside that shape are ignored rather than reported: section 8 says entries
// SHOULD, not MUST, carry a description, and producers format their own prose.
func indexEntries(pathName, body string, firstLine int) []IndexEntry {
	entries := []IndexEntry{}
	lines := splitLines(body)
	for i, raw := range lines {
		lineNumber := firstLine + i
		trimmed := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if !strings.HasPrefix(trimmed, "* ") && !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		item := strings.TrimSpace(trimmed[2:])
		links := linksInLine(item, lineNumber)
		if len(links) == 0 {
			continue
		}
		link := links[0]
		entry := IndexEntry{
			Title:  link.Text,
			Target: link.Target,
			Line:   lineNumber,
		}
		if index := strings.Index(item, ") - "); index >= 0 {
			entry.Description = strings.TrimSpace(item[index+len(") - "):])
		}
		entries = append(entries, entry)
	}
	return entries
}

// resolveIndexEntries fills each entry's Resolution against the bundle's files.
// A missing target is an observation: OKF v0.2 section 8 does not require an index to be
// complete, and the file may be added later.
func resolveIndexEntries(index *Index, existed func(string) bool) {
	for i := range index.Entries {
		entry := &index.Entries[i]
		ref := resolveTarget(index.Path, entry.Target, "index-entry", entry.Line, existed)
		entry.Resolution = ref.Resolution
		entry.Resolved = ref.Resolved
		switch {
		case escapesBundle(index.Path, entry.Target):
			index.Diagnostics = append(index.Diagnostics, Diagnostic{
				Code:     CodeIndexEntryTargetEscapes,
				Severity: SeverityWarning,
				Path:     index.Path,
				Line:     entry.Line,
				Message:  "index entry " + quoteKey(entry.Target) + " attempts to leave the bundle root; the reader does not follow it",
			})
		case ref.Resolution == ResolutionUnresolved:
			index.Diagnostics = append(index.Diagnostics, Diagnostic{
				Code:     CodeIndexEntryTargetMissing,
				Severity: SeverityWarning,
				Path:     index.Path,
				Line:     entry.Line,
				Message:  "index entry " + quoteKey(entry.Target) + " names no file in the bundle; the entry is kept as written",
			})
		}
	}
}

// buildLog reads one reserved `log.md`. OKF v0.2 section 9 describes a flat list of
// date-grouped entries, newest first, with ISO 8601 `YYYY-MM-DD` headings. A log
// that carries frontmatter is common in practice (the upstream sample bundle does),
// so it is an observation rather than an error.
func buildLog(doc rawDoc) Log {
	parsed := doc.document
	log := Log{
		Path:           doc.path,
		Digest:         doc.digest,
		HasFrontmatter: parsed.Fields != nil,
		Entries:        []LogEntry{},
		Diagnostics:    append([]Diagnostic{}, parsed.Diagnostics...),
	}
	if log.HasFrontmatter {
		log.Diagnostics = append(log.Diagnostics, Diagnostic{
			Code:     CodeLogFrontmatterPresent,
			Severity: SeverityInfo,
			Path:     doc.path,
			Message:  "OKF v0.2 section 9 defines log.md as a flat date list and does not define frontmatter; the block is preserved and reported",
		})
	}

	log.Entries = logEntries(doc.path, parsed.Body, parsed.BodyStartLine, &log.Diagnostics)
	return log
}

var logDateLine = okfLogDateHeading()

// logEntries groups `## YYYY-MM-DD` headings with the list lines beneath them.
func logEntries(pathName, body string, firstLine int, diags *[]Diagnostic) []LogEntry {
	entries := []LogEntry{}
	lines := splitLines(body)
	current := -1
	for i, raw := range lines {
		lineNumber := firstLine + i
		trimmed := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if strings.HasPrefix(trimmed, "## ") {
			heading := strings.TrimSpace(trimmed[3:])
			if !logDateLine.MatchString(heading) {
				*diags = append(*diags, Diagnostic{
					Code:     CodeLogDateNotISO,
					Severity: SeverityWarning,
					Path:     pathName,
					Line:     lineNumber,
					Message:  "OKF v0.2 section 9 requires ISO 8601 YYYY-MM-DD date headings; " + quoteKey(heading) + " is not one",
				})
				current = -1
				continue
			}
			entries = append(entries, LogEntry{Date: heading, Entries: []string{}, Line: lineNumber})
			current = len(entries) - 1
			continue
		}
		if current < 0 {
			continue
		}
		if !strings.HasPrefix(trimmed, "* ") && !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		entries[current].Entries = append(entries[current].Entries, strings.TrimSpace(trimmed[2:]))
	}
	return entries
}
