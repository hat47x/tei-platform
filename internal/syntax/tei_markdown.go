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
//
// A row of a marked table may omit its leading pipe, but it must keep the
// trailing one: a line that ends with `|` and has as many cells as the header is
// read as a row, and a line that does not is not. Without the trailing pipe the
// rule could not tell a row from prose that merely contains a pipe, so that case
// is documentation rather than a silent loss. An isolated row immediately after
// a marked table and blank lines is ambiguous; report it unless it starts a
// separate documentation table. Nothing that looks like a `tei`
// fence or a `tei:` mark is ever skipped without a diagnostic: a region or table
// the scanner did not read is reported with its position.

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

// saysOverNoRows reports whether the mark states the empty relation explicitly.
func saysOverNoRows(words []MarkdownCell) bool {
	return len(words) == 8 && words[5].Text == "over" && words[6].Text == "no" && words[7].Text == "rows"
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
// delimiter row, and at least one body row, except that a row-set example's
// mark carrying `over no rows` states the empty relation and needs no body row
// (decision section 8.2).
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
	var endedTableColumns int // only across blank lines immediately after a marked table
	var documentation []Anchor
	rawHTMLBlockOpen := ""
	inComment := false
	commentStart := Position{}

	finishTable := func() error {
		// Section 8.2: the empty relation is written as the header and the
		// delimiter of a row-set table with no body row, and only when the mark
		// says `over no rows`. A table whose body was left unwritten is then
		// still reported: the clause says the empty collection was meant, so a
		// missing row is not silently promoted to an example. An ordinary
		// table (`data`, `example`, `cases of <operator>`) never takes that
		// clause and keeps needing a body row.
		emptyRowSetExample := table.stage == 2 && len(table.table.Rows) == 0 &&
			saysOverNoRows(table.table.Words)
		if table.stage < 2 || len(table.table.Rows) == 0 && !emptyRowSetExample {
			return &ParseError{Pos: table.table.Mark.Start, End: table.table.Mark.End,
				Msg: "a tei mark must be directly followed by a table with a header row, a delimiter row, and at least one row; an empty relation is written in a row-set example's mark with `over no rows`"}
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
		if endedTableColumns != 0 && strings.TrimSpace(text) != "" {
			// A blank line ends a Markdown table. A following row of the same
			// shape is easily mistaken for another definition or test case.
			// Do not turn it into a definition implicitly or drop it silently.
			cells, isRow, err := markdownTableRow(text, at)
			if !isRow {
				cells, isRow, err = markdownTableRowDetached(text, at, endedTableColumns)
			}
			if isRow && err == nil && len(cells) == endedTableColumns &&
				!startsDocumentationTable(src, next, len(cells)) {
				return MarkdownDocument{}, &ParseError{Pos: lineSpan.Start, End: lineSpan.End,
					Msg: "a row is separated from its marked table by a blank line; remove the blank line or start a separate table"}
			}
			endedTableColumns = 0
		}
		if table != nil {
			cells, isRow, err := markdownTableRow(text, at)
			if err != nil {
				return MarkdownDocument{}, err
			}
			switch {
			case !isRow:
				// A line that is still a row of this table is read, not
				// dropped: CommonMark renders a row with or without the
				// leading pipe (#5145).
				if table.stage == 2 {
					if detached, ok, err := markdownTableRowDetached(text, at, len(table.table.Header)); err != nil {
						return MarkdownDocument{}, err
					} else if ok {
						table.table.Rows = append(table.table.Rows, detached)
						consumed = true
						break
					}
					// A line that renders as a row but cannot be read is
					// reported, so it is not lost the way the defect describes.
					if markdownRendersAsRow(text, len(table.table.Header)) {
						return MarkdownDocument{}, &ParseError{Pos: lineSpan.Start, End: lineSpan.End,
							Msg: fmt.Sprintf("this line renders as a row of the marked table above, but it has %d cells where the header has %d; write the missing cells, or end the table with a blank line", strings.Count(strings.TrimSpace(text), "|")+1, len(table.table.Header))}
					}
				}
				if table.stage == 2 && strings.TrimSpace(text) == "" {
					endedTableColumns = len(table.table.Header)
				}
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
				} else {
					// A fence that is not `tei` is documentation by design.
					// Remember its whole span so a `tei` fence quoted inside it
					// stays documentation too.
					documentation = append(documentation, Anchor{Start: fence.open.Start, End: lineSpan.End})
				}
				fence = nil
			}
		case inComment:
			if strings.Contains(text, "-->") {
				documentation = append(documentation, Anchor{Start: commentStart, End: lineSpan.End})
				inComment = false
			}
		default:
			indent, body := markdownIndent(text)
			switch {
			case rawHTMLBlockOpen != "" && indent <= 3:
				// A mark or table inside a raw HTML block is not rendered as a
				// table; reading it would invent a definition (#5145).
				if strings.TrimSpace(body) == "" {
					// A block opened by <div>-style markup ends at a blank line
					// (rawHTMLBlockBlankLine). A block opened by <pre>-style
					// markup continues through blank lines.
					if rawHTMLBlockOpen == rawHTMLBlockBlankLine {
						rawHTMLBlockOpen = ""
					}
					break
				}
				if readTables && strings.HasPrefix(body, "<!--") &&
					strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(body, "<!--"))), "tei:") {
					return MarkdownDocument{}, &ParseError{Pos: at(indent), End: lineSpan.End,
						Msg: "a tei mark inside a raw HTML block is not read; move it out of the HTML block, or remove it if it is an example"}
				}
				if rawHTMLBlockOpen != rawHTMLBlockBlankLine && strings.Contains(strings.ToLower(body), rawHTMLBlockOpen) {
					rawHTMLBlockOpen = ""
				}
			case indent <= 3 && rawHTMLBlockTag(body) != "":
				// An HTML block is not Markdown: a table written inside it is
				// not rendered as one, so a mark there announces nothing.
				rawHTMLBlockOpen = rawHTMLBlockTag(body)
			case indent <= 3 && strings.HasPrefix(body, "<!--"):
				rest := body[len("<!--"):]
				closing := strings.Index(rest, "-->")
				if closing < 0 {
					inComment = true
					commentStart = at(indent)
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
				// A list marker puts the fence inside a list item. The closing
				// fence of that item then appears indented, and without this
				// the closing backticks were read as the opening of a new fence
				// that swallowed the `tei` fence after the list (#5145).
				bodyOffset := len(text) - len(body)
				inListItem := false
				if indent <= 3 {
					if rest, consumed, ok := stripMarkdownListMarker(body); ok {
						_, inner := markdownIndent(rest)
						body = inner
						bodyOffset += consumed
						inListItem = true
					}
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
				fenceSpan := Anchor{Start: at(bodyOffset), End: lineSpan.End}
				switch {
				case quoted && looksTEI:
					return MarkdownDocument{}, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
						Msg: "a tei fence inside a quotation is not read; move the definition out of the quotation, or rename the fence if it is an example"}
				case inListItem && looksTEI:
					// A definition cannot start inside a list item: the item's
					// continuation lines belong to the item, so the region
					// boundary would be ambiguous. A quotation is refused the
					// same way (#5145).
					return MarkdownDocument{}, &ParseError{Pos: fenceSpan.Start, End: fenceSpan.End,
						Msg: "a tei fence on a list item line is not read; move the definition out of the list, or rename the fence if it is an example"}
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
	if fence != nil {
		// A non-tei fence that never closes is still documentation to the end
		// of the document, so a `tei` fence quoted inside it stays an example.
		documentation = append(documentation, wantSpan(fence.open.Start.Offset, len(src)))
	}
	if inComment && commentStart.Offset < len(src) {
		documentation = append(documentation, wantSpan(commentStart.Offset, len(src)))
	}
	if err := rejectUnreadTEITokens(src, document, documentation); err != nil {
		return MarkdownDocument{}, err
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

// rawHTMLBlockBlankLine marks an HTML block that ends at a blank line rather
// than at a closing tag.
const rawHTMLBlockBlankLine = "\x00blank-line"

// rawHTMLBlockTag returns the marker that ends the raw HTML block body opens, or
// "" when body does not open one.
//
// Inside these blocks CommonMark does not interpret Markdown, so a table written
// there is not a table and a mark announces nothing.
func rawHTMLBlockTag(body string) string {
	lowered := strings.ToLower(body)
	for _, tag := range []string{"pre", "script", "style", "textarea"} {
		if !opensRawHTMLTag(lowered, tag) {
			continue
		}
		// `<pre>` and `<pre class="x">` open a block; `<pre>example</pre>`
		// closes on its own line and does not.
		rest := lowered[len(tag)+1:]
		if end := strings.IndexByte(rest, '>'); end >= 0 && strings.Contains(rest[end:], "</"+tag+">") {
			return ""
		}
		return "</" + tag + ">"
	}
	// Block-level elements are not raw text, so they end at a blank line. A
	// mark inside them is still not rendered as a table.
	for _, tag := range []string{
		"address", "article", "aside", "blockquote", "body", "caption", "center", "col", "colgroup",
		"dd", "details", "dialog", "dir", "div", "dl", "dt", "fieldset", "figcaption", "figure",
		"footer", "form", "frame", "frameset", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header",
		"hr", "html", "iframe", "legend", "li", "link", "main", "menu", "menuitem", "nav", "noframes",
		"ol", "optgroup", "option", "p", "param", "section", "source", "summary", "table", "tbody",
		"td", "tfoot", "th", "thead", "title", "tr", "track", "ul",
	} {
		if opensRawHTMLTag(lowered, tag) {
			return rawHTMLBlockBlankLine
		}
	}
	return ""
}

// opensRawHTMLTag reports whether body starts with `<tag` and the tag name ends
// there, as CommonMark requires: a space, a tab, `>`, or `/>` must follow.
//
// Without the delimiter check `<table-of-contents>` would be read as `<table>`,
// opening a block that never ends and refusing a later legitimate mark.
func opensRawHTMLTag(lowered, tag string) bool {
	open := "<" + tag
	if !strings.HasPrefix(lowered, open) {
		return false
	}
	rest := lowered[len(open):]
	if rest == "" {
		return true
	}
	switch rest[0] {
	case ' ', '\t', '>', '\r', '\n':
		return true
	case '/':
		return strings.HasPrefix(rest, "/>")
	}
	return false
}

// stripMarkdownListMarker removes a leading list marker and the spaces after
// it, returning the rest, the number of bytes removed, and whether a marker was
// present. Only "-", "*", "+", or an ordered marker is accepted, so ordinary
// prose is left alone.
func stripMarkdownListMarker(body string) (rest string, consumed int, ok bool) {
	if body == "" {
		return body, 0, false
	}
	switch body[0] {
	case '-', '*', '+':
		if len(body) < 2 || (body[1] != ' ' && body[1] != '\t') {
			return body, 0, false
		}
		return body[2:], 2, true
	}
	digits := 0
	for digits < len(body) && body[digits] >= '0' && body[digits] <= '9' {
		digits++
	}
	if digits == 0 || digits > 9 || digits+1 >= len(body) {
		return body, 0, false
	}
	if body[digits] != '.' && body[digits] != ')' {
		return body, 0, false
	}
	if body[digits+1] != ' ' && body[digits+1] != '\t' {
		return body, 0, false
	}
	return body[digits+2:], digits + 2, true
}

// markdownRendersAsRow reports whether CommonMark would render text as a row of
// a table with the given number of cells: it carries at least one pipe and is
// not a block-level start.
//
// Such a line is what a reader loses when it stops at the first line that
// `markdownTableRow` does not accept, so it is reported instead of becoming an
// ordinary paragraph.
func markdownRendersAsRow(text string, cells int) bool {
	indent, body := markdownIndent(text)
	if indent > 3 {
		return false
	}
	trimmed := strings.TrimSpace(body)
	if trimmed == "" || !strings.Contains(trimmed, "|") {
		return false
	}
	switch trimmed[0] {
	case '#', '<', '`', '~', '>':
		return false
	}
	return true
}

// markdownTableRowDetached reads a row that does not start with `|`.
//
// CommonMark renders a table row with or without the leading pipe, so a line
// that ends with `|` and has exactly as many cells as the header is a row of the
// marked table. Reading it keeps the reader faithful to what the document
// displays; dropping it silently was the defect in #5145.
//
// The trailing pipe is required because it is what tells a row from prose that
// merely mentions a pipe: "Use `a | b` to filter rows." has the cell count of a
// two-cell table but does not end with one. A row written with no surrounding
// pipe at all stays documentation.
func markdownTableRowDetached(text string, at func(int) Position, cells int) ([]MarkdownCell, bool, error) {
	indent, body := markdownIndent(text)
	trimmed := strings.TrimRight(body, " \t")
	if indent > 3 || trimmed == "" || trimmed[0] == '|' || !strings.HasSuffix(trimmed, "|") {
		return nil, false, nil
	}
	// A block-level start is never a row.
	switch trimmed[0] {
	case '#', '<', '-', '*', '+', '`', '~', '>':
		return nil, false, nil
	}
	base := len(text) - len(body)
	if escaped := strings.Index(trimmed, `\|`); escaped >= 0 {
		return nil, true, &ParseError{Pos: at(base + escaped), End: at(base + escaped + 2),
			Msg: "a cell of a marked table cannot contain |; write the value outside the table"}
	}
	parts := strings.Split(strings.TrimSuffix(trimmed, "|"), "|")
	if len(parts) != cells {
		return nil, false, nil
	}
	var row []MarkdownCell
	start := base
	for _, raw := range parts {
		left := len(raw) - len(strings.TrimLeft(raw, " \t"))
		value := strings.TrimSpace(raw)
		begin := start + left
		if value == "" {
			begin = start
		}
		row = append(row, MarkdownCell{Text: value, Anchor: Anchor{Start: at(begin), End: at(begin + len(value))}})
		start += len(raw) + 1
	}
	return row, true, nil
}

// startsDocumentationTable recognises the delimiter immediately following a
// candidate header, including optional outside pipes. It does not read the
// documentation table as a definition.
func startsDocumentationTable(src string, offset, columns int) bool {
	if offset >= len(src) {
		return false
	}
	line, _, _ := strings.Cut(src[offset:], "\n")
	indent, body := markdownIndent(strings.TrimSuffix(line, "\r"))
	if indent > 3 {
		return false
	}
	body = strings.TrimSpace(body)
	body = strings.TrimPrefix(body, "|")
	body = strings.TrimSuffix(body, "|")
	parts := strings.Split(body, "|")
	if len(parts) != columns {
		return false
	}
	cells := make([]MarkdownCell, len(parts))
	for i, part := range parts {
		cells[i].Text = strings.TrimSpace(part)
	}
	return markdownDelimiterRow(cells)
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

// rejectUnreadTEITokens reports a `tei` token that stayed in the source without
// becoming a region or a marked table.
//
// The scanner decides region boundaries line by line, so a construct it does not
// model can hide a definition: the closing backticks of a fence inside a list
// item are read as the beginning of a new fence, and the `tei` fence that
// follows is taken as its content. That case, a table row without a leading
// pipe, and a mark written as `TEI:` or after another comment all loaded without
// a diagnostic before this check existed (#5145).
//
// Masking what was read and looking for `tei` tokens in the rest keeps the
// decision in one place: a new near miss does not need its own condition, and a
// region that was read cannot be reported twice.
func rejectUnreadTEITokens(src string, document MarkdownDocument, documentation []Anchor) error {
	residual := maskReadTEIContent(src, document, documentation)

	// Walk the residual once, line by line. Masking replaces bytes with spaces
	// and keeps line feeds, so a column in the residual is the same column in
	// the source: the position needs no second pass over the document.
	line := 1
	for offset := 0; offset <= len(residual); {
		end := strings.IndexByte(residual[offset:], '\n')
		next := len(residual) + 1
		if end < 0 {
			end = len(residual)
		} else {
			end += offset
			next = end + 1
		}
		text := strings.TrimSuffix(residual[offset:end], "\r")
		at := func(column int) Position {
			return Position{Offset: offset + column, Line: line, Column: column + 1}
		}
		if indent, body := markdownIndent(text); indent <= 3 {
			lowered := strings.ToLower(body)
			if strings.HasPrefix(lowered, "```tei") || strings.HasPrefix(lowered, "~~~tei") {
				return &ParseError{Pos: at(indent), End: at(len(text)),
					Msg: "a tei fence was left unread; a fence inside a list item or a raw HTML block can hide it. Move the definition out, or rename the fence if it is an example"}
			}
			// A mark must be the only thing on its line, so a `tei:` comment
			// that shares the line with other content is not one: it used to be
			// accepted and then ignored (#5145).
			if strings.Contains(lowered, "<!--") {
				for start := 0; ; {
					open := strings.Index(lowered[start:], "<!--")
					if open < 0 {
						break
					}
					column := indent + start + open
					rest := body[column+len("<!--"):]
					closing := strings.Index(rest, "-->")
					if closing >= 0 {
						mark := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(rest[:closing]), "-"))
						if strings.HasPrefix(strings.ToLower(mark), "tei:") {
							return &ParseError{Pos: at(column), End: at(len(text)),
								Msg: "a tei mark was left unread; a mark must be the only thing on its line, written in lower case, directly before its table"}
						}
						start += open + len("<!--") + closing
						continue
					}
					break
				}
			}
		}
		if next > len(residual) {
			break
		}
		offset = next
		line++
	}
	return nil
}

// maskReadTEIContent blanks every byte this reader has accounted for and returns
// the rest. Line feeds are kept so a position in the result still maps to the
// same line of the source.
func maskReadTEIContent(src string, document MarkdownDocument, documentation []Anchor) string {
	masked := []byte(src)
	blank := func(start, end int) {
		if start < 0 {
			start = 0
		}
		if end > len(masked) {
			end = len(masked)
		}
		for i := start; i < end; i++ {
			if masked[i] != '\n' {
				masked[i] = ' '
			}
		}
	}
	for _, region := range document.Regions {
		blank(region.Open.Start.Offset, region.Close.End.Offset)
	}
	for _, span := range documentation {
		blank(span.Start.Offset, span.End.Offset)
	}
	for _, table := range document.Tables {
		blank(table.Mark.Start.Offset, table.Mark.End.Offset)
		end := table.Header[len(table.Header)-1].Anchor.End.Offset
		if rows := len(table.Rows); rows > 0 {
			end = table.Rows[rows-1][len(table.Rows[rows-1])-1].Anchor.End.Offset
		}
		blank(table.Mark.Start.Offset, end)
	}
	return string(masked)
}

// wantSpan builds an anchor from byte offsets alone. Masking only reads the
// offsets, so the line and column are left unset.
func wantSpan(start, end int) Anchor {
	return Anchor{Start: Position{Offset: start}, End: Position{Offset: end}}
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
