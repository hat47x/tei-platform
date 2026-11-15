package syntax

import (
	"fmt"
	"strings"
)

// This file reads the regions of a TEI Markdown source (*.tei.md).
//
// A TEI Markdown source is an ordinary Markdown document. Only a fenced code
// block whose info string is exactly `tei` is a definition region. Prose,
// tables, other fences, and HTML comments stay documentation: they are kept in
// the source and never become a definition.
//
// The scanner decides *where* to read. It does not decide which language
// reads the region; the caller passes the region to the language its Workspace
// selected. Spans are UTF-8 byte offsets into the whole document, so a
// diagnostic and a Source Index range point at the authored bytes.
//
// A near miss is an error, not a skipped block. A fence named `TEI`, a `tei`
// fence that is indented into a code block or sits inside a quotation, and a
// `tei` fence that never closes would otherwise leave a definition unread
// while the document still loads.

// TEIMarkdownFenceName is the info string that marks a definition region.
const TEIMarkdownFenceName = "tei"

// MarkdownRegion is one definition region of a TEI Markdown source.
type MarkdownRegion struct {
	// Open is the opening fence line without its line ending.
	Open Anchor
	// Content is every byte between the opening line and the closing line.
	// It may be empty. It always starts at the beginning of a line.
	Content Anchor
	// Close is the closing fence line without its line ending.
	Close Anchor
}

type markdownFence struct {
	char    byte
	length  int
	tei     bool
	open    Anchor
	content Position
}

// ScanTEIMarkdown returns the definition regions of src in source order.
//
// Fences follow the CommonMark rules that matter for a region boundary: three
// or more backticks or tildes, at most three spaces of indentation, and a
// closing fence of the same character that is at least as long. A fence inside
// another fence or inside a multi-line HTML comment is not a region.
func ScanTEIMarkdown(src string) ([]MarkdownRegion, error) {
	var regions []MarkdownRegion
	var fence *markdownFence
	inComment := false

	offset, line := 0, 1
	for offset <= len(src) {
		lineStart := offset
		lineEnd := strings.IndexByte(src[offset:], '\n')
		next := len(src) + 1
		if lineEnd < 0 {
			lineEnd = len(src)
		} else {
			lineEnd += offset
			next = lineEnd + 1
		}
		text := strings.TrimSuffix(src[lineStart:lineEnd], "\r")
		at := func(column int) Position {
			return Position{Offset: lineStart + column, Line: line, Column: column + 1}
		}
		lineSpan := Anchor{Start: at(0), End: at(len(text))}

		switch {
		case fence != nil:
			if closesMarkdownFence(text, fence.char, fence.length) {
				if fence.tei {
					regions = append(regions, MarkdownRegion{
						Open:    fence.open,
						Content: Anchor{Start: fence.content, End: at(0)},
						Close:   lineSpan,
					})
				}
				fence = nil
			}
		case inComment:
			if strings.Contains(text, "-->") {
				inComment = false
			}
		default:
			indent, body := markdownIndent(text)
			switch {
			case indent <= 3 && strings.HasPrefix(body, "<!--"):
				rest := body[len("<!--"):]
				if !strings.Contains(rest, "-->") {
					inComment = true
					break
				}
				// A one-line comment that addresses TEI is a mark of a later
				// notation. This reader has no meaning for it, and skipping it
				// would leave the region it announces unread.
				mark := strings.TrimSpace(rest)
				if strings.HasPrefix(mark, "tei:") || strings.HasPrefix(mark, "/tei") {
					return nil, &ParseError{Pos: at(indent), End: lineSpan.End,
						Msg: "TEI comment marks are not read by this language; only a ```tei fence is a definition region"}
				}
			default:
				quoted := false
				for strings.HasPrefix(body, ">") {
					quoted = true
					_, body = markdownIndent(body[1:])
				}
				char, length, info, ok := opensMarkdownFence(body)
				if !ok {
					break
				}
				word, extra := markdownInfoWord(info)
				looksTEI := strings.HasPrefix(strings.ToLower(word), TEIMarkdownFenceName)
				isTEI := word == TEIMarkdownFenceName && !extra
				fenceSpan := Anchor{Start: at(len(text) - len(body)), End: lineSpan.End}
				switch {
				case quoted && looksTEI:
					return nil, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
						Msg: "a tei fence inside a quotation is not read; move the definition out of the quotation, or rename the fence if it is an example"}
				case quoted:
					// A quoted fence prefixes every line with '>', so it cannot
					// hide a later region. It stays documentation.
				case indent > 3 && looksTEI:
					return nil, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
						Msg: "a tei fence indented by four or more columns is a code block, not a definition region; remove the indentation, or rename the fence if it is an example"}
				case indent > 3:
					// Indented code. Not a fence.
				case looksTEI && !isTEI:
					return nil, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
						Msg: fmt.Sprintf("unknown TEI fence %q; a definition region is a fence named exactly %q", strings.TrimSpace(info), TEIMarkdownFenceName)}
				default:
					fence = &markdownFence{char: char, length: length, tei: isTEI, open: fenceSpan}
					fence.content = Position{Offset: min(next, len(src)), Line: line + 1, Column: 1}
				}
			}
		}

		if next > len(src) {
			break
		}
		offset = next
		line++
	}

	if fence != nil && fence.tei {
		return nil, &ParseError{Pos: fence.open.Start, End: fence.open.End,
			Msg: "tei fence is not closed; the rest of the document would be read as a definition"}
	}
	if len(regions) == 0 {
		start := Position{Offset: 0, Line: 1, Column: 1}
		return nil, &ParseError{Pos: start, End: start,
			Msg: "TEI Markdown source declares no ```tei region, so it defines nothing"}
	}
	return regions, nil
}

// MaskMarkdownRegion returns a source in which only the region content is
// present, at its original byte offsets. Every byte before the content becomes
// a space, except that line feeds are kept, and the source ends where the
// content ends.
//
// Parsing the result with an existing parser yields offsets, lines, and
// columns of the whole document without a second position computation: the
// value and its spans come from one parse of the authored bytes.
func MaskMarkdownRegion(src string, region MarkdownRegion) string {
	start, end := region.Content.Start.Offset, region.Content.End.Offset
	masked := make([]byte, end)
	for i := 0; i < start; i++ {
		if src[i] == '\n' {
			masked[i] = '\n'
		} else {
			masked[i] = ' '
		}
	}
	copy(masked[start:], src[start:end])
	return string(masked)
}

func markdownIndent(text string) (int, string) {
	columns := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case ' ':
			columns++
		case '\t':
			columns += 4 - columns%4
		default:
			return columns, text[i:]
		}
	}
	return columns, ""
}

func opensMarkdownFence(body string) (char byte, length int, info string, ok bool) {
	if body == "" || (body[0] != '`' && body[0] != '~') {
		return 0, 0, "", false
	}
	char = body[0]
	for length < len(body) && body[length] == char {
		length++
	}
	if length < 3 {
		return 0, 0, "", false
	}
	info = body[length:]
	if char == '`' && strings.ContainsRune(info, '`') {
		return 0, 0, "", false
	}
	return char, length, info, true
}

func closesMarkdownFence(text string, char byte, length int) bool {
	indent, body := markdownIndent(text)
	if indent > 3 {
		return false
	}
	run := 0
	for run < len(body) && body[run] == char {
		run++
	}
	return run >= length && strings.TrimSpace(body[run:]) == ""
}

// markdownInfoWord returns the first word of a fence info string and whether
// anything follows it.
func markdownInfoWord(info string) (word string, extra bool) {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return "", false
	}
	return fields[0], len(fields) > 1
}
