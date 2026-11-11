package okf

import (
	"strings"
)

// bodyLink is one Markdown link found outside a fenced code block.
type bodyLink struct {
	// Text is the link label, kept for presentation only.
	Text string `json:"text"`
	// Target is the destination exactly as written, without any title.
	Target string `json:"target"`
	// Line is the 1-based line inside the document.
	Line int `json:"line"`
}

// bodyFootnote is one Markdown footnote reference and its matching definition.
type bodyFootnote struct {
	Label string `json:"label"`
	// Uses counts how many times the label is referenced in the body.
	Uses int `json:"uses"`
	// Defined reports whether a matching `[^label]: ...` definition exists.
	Defined bool `json:"defined"`
	Line    int  `json:"line,omitempty"`
}

// bodySection is one top-level heading occurrence.
type bodySection struct {
	Heading string `json:"heading"`
	Line    int
}

// bodyScan is the reader's structural summary of a document body. It exists so a
// consumer does not have to re-parse the Markdown, and so the reader can state
// what it did and did not enumerate.
type bodyScan struct {
	Links     []bodyLink     `json:"-"`
	Footnotes []bodyFootnote `json:"-"`
	Headings  []bodySection  `json:"-"`
	// FencedLanguages records the code fence info strings, so a caller can see
	// which sections carry runnable-looking blocks without the reader executing
	// anything.
	FencedLanguages []string `json:"-"`
	// HasCitationsSection reports a legacy v0.1 `# Citations` body section.
	HasCitationsSection bool `json:"-"`
	// HasComputationSection reports a `# Computation` heading.
	HasComputationSection bool `json:"-"`
}

// markdownLinkPattern and friends are deliberately small: this reader needs the
// link destination and footnote labels of a structural Markdown document, not a
// full CommonMark parser. Anything the small scanner does not recognize is
// reported as not enumerated rather than guessed at.
const (
	markdownLinkOpen  = "]("
	markdownLinkClose = ")"
	footnoteMarker    = "[^"
)

// scanBody extracts links, footnotes, headings and fence languages from the body.
// Text inside fenced code blocks is skipped, matching the conventions in OKF
// v0.2 section 4.2 (structured Markdown, fenced code blocks) and the repository's own
// Markdown checks.
func scanBody(body string, firstLine int) bodyScan {
	var scan bodyScan
	footnoteUses := map[string]int{}
	footnoteDefined := map[string]bool{}
	footnoteLine := map[string]int{}

	lines := splitLines(body)
	inFence := false
	fenceIndent := ""
	for i, raw := range lines {
		lineNumber := firstLine + i
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if !inFence {
				inFence = true
				fenceIndent = trimmed[:3]
				info := strings.TrimSpace(strings.TrimLeft(trimmed, "`~"))
				scan.FencedLanguages = append(scan.FencedLanguages, info)
			} else if strings.HasPrefix(trimmed, fenceIndent) {
				inFence = false
			}
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			if level == 1 {
				heading := strings.TrimSpace(trimmed[level:])
				scan.Headings = append(scan.Headings, bodySection{Heading: heading, Line: lineNumber})
				switch heading {
				case "Citations":
					scan.HasCitationsSection = true
				case "Computation":
					scan.HasComputationSection = true
				}
			}
		}
		for _, link := range linksInLine(line, lineNumber) {
			scan.Links = append(scan.Links, link)
		}
		for _, label := range footnoteReferencesInLine(line) {
			footnoteUses[label]++
			if _, ok := footnoteLine[label]; !ok {
				footnoteLine[label] = lineNumber
			}
		}
		if label, ok := footnoteDefinition(line); ok {
			footnoteDefined[label] = true
		}
	}

	labels := make([]string, 0, len(footnoteUses)+len(footnoteDefined))
	seen := map[string]bool{}
	for label := range footnoteUses {
		seen[label] = true
		labels = append(labels, label)
	}
	for label := range footnoteDefined {
		if !seen[label] {
			labels = append(labels, label)
		}
	}
	scan.Footnotes = make([]bodyFootnote, 0, len(labels))
	for _, label := range labels {
		scan.Footnotes = append(scan.Footnotes, bodyFootnote{
			Label:   label,
			Uses:    footnoteUses[label],
			Defined: footnoteDefined[label],
			Line:    footnoteLine[label],
		})
	}
	return scan
}

// linksInLine finds `[text](target)` pairs on one line. Angle-bracket targets
// are unwrapped; a title in quotes is stripped, and both are recorded as
// normalization the reader performed rather than as a rewrite of the source.
func linksInLine(line string, lineNumber int) []bodyLink {
	var out []bodyLink
	for offset := 0; offset < len(line); {
		open := strings.Index(line[offset:], "]( ")
		if open < 0 {
			open = strings.Index(line[offset:], markdownLinkOpen)
		}
		if open < 0 {
			break
		}
		openAt := offset + open
		labelStart := strings.LastIndex(line[:openAt], "[")
		closeAt := strings.Index(line[openAt+2:], markdownLinkClose)
		if closeAt < 0 {
			break
		}
		target := line[openAt+2 : openAt+2+closeAt]
		var text string
		if labelStart >= 0 {
			text = line[labelStart+1 : openAt]
		}
		out = append(out, bodyLink{
			Text:   unescapeMarkdownText(text),
			Target: normalizeLinkTarget(target),
			Line:   lineNumber,
		})
		offset = openAt + 2 + closeAt + 1
	}
	return out
}

// normalizeLinkTarget strips an optional title and angle brackets. It does not
// resolve or decode anything: the target stays the author's path or URL.
func normalizeLinkTarget(raw string) string {
	target := strings.TrimSpace(raw)
	if strings.HasPrefix(target, "<") {
		if end := strings.Index(target, ">"); end > 0 {
			return target[1:end]
		}
	}
	if index := strings.IndexAny(target, " \t"); index >= 0 {
		target = target[:index]
	}
	return target
}

func unescapeMarkdownText(text string) string {
	replacer := strings.NewReplacer(`\[`, "[", `\]`, "]")
	return replacer.Replace(text)
}

// footnoteReferencesInLine returns the labels of `[^label]` references. A
// definition is excluded, because it also matches the same prefix.
func footnoteReferencesInLine(line string) []string {
	var labels []string
	for offset := 0; offset < len(line); {
		start := strings.Index(line[offset:], footnoteMarker)
		if start < 0 {
			break
		}
		startAt := offset + start
		end := strings.Index(line[startAt+len(footnoteMarker):], "]")
		if end < 0 {
			break
		}
		label := line[startAt+len(footnoteMarker) : startAt+len(footnoteMarker)+end]
		rest := line[startAt+len(footnoteMarker)+end+1:]
		if strings.HasPrefix(rest, ":") {
			offset = startAt + len(footnoteMarker) + end + 1
			continue
		}
		if label != "" {
			labels = append(labels, label)
		}
		offset = startAt + len(footnoteMarker) + end + 1
	}
	return labels
}

// footnoteDefinition returns the label of a `[^label]: ...` definition.
func footnoteDefinition(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, footnoteMarker) {
		return "", false
	}
	end := strings.Index(trimmed[len(footnoteMarker):], "]")
	if end <= 0 {
		return "", false
	}
	label := trimmed[len(footnoteMarker) : len(footnoteMarker)+end]
	rest := trimmed[len(footnoteMarker)+end+1:]
	if !strings.HasPrefix(rest, ":") {
		return "", false
	}
	return label, true
}
