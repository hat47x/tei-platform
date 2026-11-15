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
//
// ScanTEIMarkdownDocument additionally reads a Markdown table that directly
// follows a one-line mark `<!-- tei: ... -->`. The table stays an ordinary
// Markdown table, so the document renders the same; the mark is what makes it
// a definition. A table without a mark is documentation.

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

// MarkdownCell is one table cell: its text without surrounding spaces and the
// byte range of that text. An empty cell has an empty range at its position.
type MarkdownCell struct {
	Text   string
	Anchor Anchor
}

// MarkdownTable is a marked table. Words are the words of the mark after
// `tei:`, for the language to interpret. Every row has as many cells as the
// header.
type MarkdownTable struct {
	Mark   Anchor
	Words  []MarkdownCell
	Header []MarkdownCell
	Rows   [][]MarkdownCell
}

// MarkdownDocument is what a TEI Markdown source defines, in source order
// within each kind.
type MarkdownDocument struct {
	Regions []MarkdownRegion
	Tables  []MarkdownTable
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
//
// A `<!-- tei: ... -->` mark is an error here: this entry point reads no
// table, and skipping the mark would leave the table it announces unread.
func ScanTEIMarkdown(src string) ([]MarkdownRegion, error) {
	document, err := scanTEIMarkdown(src, false)
	if err != nil {
		return nil, err
	}
	return document.Regions, nil
}

// ScanTEIMarkdownDocument returns the fenced regions and the marked tables of
// src. A mark must be directly followed by a table with a header row, a
// delimiter row, and at least one body row.
func ScanTEIMarkdownDocument(src string) (MarkdownDocument, error) {
	return scanTEIMarkdown(src, true)
}

type markdownTableState struct {
	table MarkdownTable
	// stage is 0 before the header row, 1 before the delimiter row, and 2
	// while reading body rows.
	stage int
}

func scanTEIMarkdown(src string, readTables bool) (MarkdownDocument, error) {
	var document MarkdownDocument
	var fence *markdownFence
	var table *markdownTableState
	inComment := false

	finishTable := func() error {
		if table.stage < 2 || len(table.table.Rows) == 0 {
			return &ParseError{Pos: table.table.Mark.Start, End: table.table.Mark.End,
				Msg: "a tei mark must be directly followed by a table with a header row, a delimiter row, and at least one row"}
		}
		document.Tables = append(document.Tables, table.table)
		table = nil
		return nil
	}

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

		consumed := false
		if table != nil {
			cells, isRow, err := markdownTableRow(text, at)
			if err != nil {
				return MarkdownDocument{}, err
			}
			switch {
			case !isRow:
				// The table ends at the first line that is not a row. That
				// line is then read as an ordinary line of the document.
				if err := finishTable(); err != nil {
					return MarkdownDocument{}, err
				}
			case table.stage == 0:
				table.table.Header = cells
				table.stage = 1
				consumed = true
			case table.stage == 1:
				if len(cells) != len(table.table.Header) || !markdownDelimiterRow(cells) {
					return MarkdownDocument{}, &ParseError{Pos: lineSpan.Start, End: lineSpan.End,
						Msg: "the second row of a marked table must be a delimiter row with one --- cell per header cell"}
				}
				table.stage = 2
				consumed = true
			default:
				if len(cells) != len(table.table.Header) {
					return MarkdownDocument{}, &ParseError{Pos: lineSpan.Start, End: lineSpan.End,
						Msg: fmt.Sprintf("this row has %d cells, but the header has %d; a cell cannot be left out", len(cells), len(table.table.Header))}
				}
				table.table.Rows = append(table.table.Rows, cells)
				consumed = true
			}
		}

		switch {
		case consumed:
		case fence != nil:
			if closesMarkdownFence(text, fence.char, fence.length) {
				if fence.tei {
					document.Regions = append(document.Regions, MarkdownRegion{
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
				closing := strings.Index(rest, "-->")
				if closing < 0 {
					inComment = true
					break
				}
				mark := strings.TrimSpace(rest[:closing])
				isMark := strings.HasPrefix(mark, "tei:")
				switch {
				case strings.HasPrefix(mark, "/tei") || isMark && !readTables:
					// A one-line comment that addresses TEI is a mark this
					// reader has no meaning for. Skipping it would leave the
					// region it announces unread.
					return MarkdownDocument{}, &ParseError{Pos: at(indent), End: lineSpan.End,
						Msg: "TEI comment marks are not read by this language; only a ```tei fence is a definition region"}
				case isMark:
					if strings.TrimSpace(rest[closing+len("-->"):]) != "" {
						return MarkdownDocument{}, &ParseError{Pos: at(indent), End: lineSpan.End,
							Msg: "a tei mark must be the only thing on its line"}
					}
					words := markdownMarkWords(text, at)
					if len(words) == 0 {
						return MarkdownDocument{}, &ParseError{Pos: at(indent), End: lineSpan.End,
							Msg: "a tei mark must say what the following table defines"}
					}
					table = &markdownTableState{table: MarkdownTable{Mark: Anchor{Start: at(indent), End: lineSpan.End}, Words: words}}
				}
			default:
				quoted := false
				for strings.HasPrefix(body, ">") {
					quoted = true
					_, body = markdownIndent(body[1:])
				}
				if readTables && (quoted || indent > 3) && strings.HasPrefix(body, "<!--") &&
					strings.HasPrefix(strings.TrimSpace(body[len("<!--"):]), "tei:") {
					return MarkdownDocument{}, &ParseError{Pos: at(len(text) - len(body)), End: lineSpan.End,
						Msg: "a tei mark inside a quotation or an indented block is not read; move it out, or remove it if it is an example"}
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
					return MarkdownDocument{}, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
						Msg: "a tei fence inside a quotation is not read; move the definition out of the quotation, or rename the fence if it is an example"}
				case quoted:
					// A quoted fence prefixes every line with '>', so it cannot
					// hide a later region. It stays documentation.
				case indent > 3 && looksTEI:
					return MarkdownDocument{}, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
						Msg: "a tei fence indented by four or more columns is a code block, not a definition region; remove the indentation, or rename the fence if it is an example"}
				case indent > 3:
					// Indented code. Not a fence.
				case looksTEI && !isTEI:
					return MarkdownDocument{}, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
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

	if table != nil {
		if err := finishTable(); err != nil {
			return MarkdownDocument{}, err
		}
	}
	if fence != nil && fence.tei {
		return MarkdownDocument{}, &ParseError{Pos: fence.open.Start, End: fence.open.End,
			Msg: "tei fence is not closed; the rest of the document would be read as a definition"}
	}
	if len(document.Regions) == 0 && len(document.Tables) == 0 {
		start := Position{Offset: 0, Line: 1, Column: 1}
		return MarkdownDocument{}, &ParseError{Pos: start, End: start,
			Msg: "TEI Markdown source declares no ```tei region, so it defines nothing"}
	}
	return document, nil
}

// markdownMarkWords returns the words of a mark line after `tei:`, each with
// its byte range in the document.
func markdownMarkWords(text string, at func(int) Position) []MarkdownCell {
	start := strings.Index(text, "tei:") + len("tei:")
	end := strings.Index(text, "-->")
	var words []MarkdownCell
	i := start
	for i < end {
		for i < end && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
		begin := i
		for i < end && text[i] != ' ' && text[i] != '\t' {
			i++
		}
		if i > begin {
			words = append(words, MarkdownCell{Text: text[begin:i], Anchor: Anchor{Start: at(begin), End: at(i)}})
		}
	}
	return words
}

// markdownTableRow splits one table row into cells. A row starts and ends
// with `|` and is indented by at most three spaces. A cell cannot contain `|`:
// an escaped pipe is refused instead of being read as part of a value, so a
// value with a pipe has to be written outside a table.
func markdownTableRow(text string, at func(int) Position) ([]MarkdownCell, bool, error) {
	indent, body := markdownIndent(text)
	trimmed := strings.TrimRight(body, " \t")
	if indent > 3 || len(trimmed) < 2 || trimmed[0] != '|' || trimmed[len(trimmed)-1] != '|' {
		return nil, false, nil
	}
	base := len(text) - len(body)
	if escaped := strings.Index(trimmed, `\|`); escaped >= 0 {
		return nil, true, &ParseError{Pos: at(base + escaped), End: at(base + escaped + 2),
			Msg: "a cell of a marked table cannot contain |; write the value outside the table"}
	}
	var cells []MarkdownCell
	cellStart := 1
	for i := 1; i < len(trimmed); i++ {
		if trimmed[i] != '|' {
			continue
		}
		raw := trimmed[cellStart:i]
		left := len(raw) - len(strings.TrimLeft(raw, " \t"))
		value := strings.TrimSpace(raw)
		begin := base + cellStart + left
		if value == "" {
			begin = base + cellStart
		}
		cells = append(cells, MarkdownCell{Text: value, Anchor: Anchor{Start: at(begin), End: at(begin + len(value))}})
		cellStart = i + 1
	}
	return cells, true, nil
}

func markdownDelimiterRow(cells []MarkdownCell) bool {
	for _, cell := range cells {
		dashes := strings.TrimSuffix(strings.TrimPrefix(cell.Text, ":"), ":")
		if dashes == "" || strings.Trim(dashes, "-") != "" {
			return false
		}
	}
	return true
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
