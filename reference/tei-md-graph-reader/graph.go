// Package teimdreader provides a strictly read-only reference probe for one
// Markdown table profile. It is not the selected TEI Workspace/Language Reader.
package teimdreader

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type Span struct {
	Start int // Inclusive byte offset in original source.
	End   int // Exclusive byte offset in original source.
	Line  int // One-based source line.
}

type Diagnostic struct {
	Code   string
	At     Span
	Detail string
}

func (e *Diagnostic) Error() string {
	return fmt.Sprintf("%s at line %d, byte %d: %s", e.Code, e.At.Line, e.At.Start, e.Detail)
}

type Node struct {
	ID, Label string
	IDSource  Span
}
type Edge struct {
	ID, From, To                   string
	IDSource, FromSource, ToSource Span
}
type Candidate struct {
	Nodes []Node
	Edges []Edge
}

type textLine struct {
	text          string
	start, number int
}

func sourceLines(src []byte) []textLine {
	out := make([]textLine, 0, 32)
	start, number := 0, 1
	for i, b := range src {
		if b == '\n' {
			out = append(out, textLine{text: strings.TrimSuffix(string(src[start:i]), "\r"), start: start, number: number})
			start = i + 1
			number++
		}
	}
	if start < len(src) {
		out = append(out, textLine{text: strings.TrimSuffix(string(src[start:]), "\r"), start: start, number: number})
	}
	return out
}

type cell struct {
	value string
	span  Span
}

func tableCells(ln textLine) ([]cell, error) {
	raw := ln.text
	leading := len(raw) - len(strings.TrimLeft(raw, " \t"))
	end := len(strings.TrimRight(raw, " \t"))
	at := Span{Start: ln.start + leading, End: ln.start + end, Line: ln.number}
	if leading >= end || raw[leading] != '|' || raw[end-1] != '|' {
		return nil, &Diagnostic{Code: "UNSUPPORTED_TABLE", At: at, Detail: "table must start and end with an unescaped pipe"}
	}
	cells := []cell{}
	prev := leading
	for p := leading + 1; p < end; p++ {
		if raw[p] != '|' {
			continue
		}
		if p > 0 && raw[p-1] == '\\' {
			return nil, &Diagnostic{Code: "UNSUPPORTED_TABLE", At: at, Detail: "escaped table pipes require a full Markdown reader"}
		}
		left, right := prev+1, p
		for left < right && (raw[left] == ' ' || raw[left] == '\t') {
			left++
		}
		for right > left && (raw[right-1] == ' ' || raw[right-1] == '\t') {
			right--
		}
		if strings.ContainsAny(raw[left:right], "`\\") {
			return nil, &Diagnostic{Code: "UNSUPPORTED_TABLE", At: at, Detail: "inline code/escape in semantic cells requires a full Markdown reader"}
		}
		cells = append(cells, cell{value: raw[left:right], span: Span{Start: ln.start + left, End: ln.start + right, Line: ln.number}})
		prev = p
	}
	if prev != end-1 {
		return nil, &Diagnostic{Code: "UNSUPPORTED_TABLE", At: at, Detail: "malformed pipe table"}
	}
	return cells, nil
}
func headerMatches(cells []cell, expected ...string) bool {
	if len(cells) != len(expected) {
		return false
	}
	for i, v := range expected {
		if cells[i].value != v {
			return false
		}
	}
	return true
}
func isSeparator(cells []cell) bool {
	for _, c := range cells {
		v := strings.Trim(c.value, ":")
		if len(v) < 3 || strings.Trim(v, "-") != "" {
			return false
		}
	}
	return len(cells) > 0
}

// markdownCodeFence tracks only the fenced-code exclusion boundary; it does
// not interpret Markdown or select a TEI Language/Role. Markdown fence markers
// may be backticks or tildes, and a closing run must match the marker and be at
// least as long as the opening run.
type markdownCodeFence struct {
	marker byte
	width  int
	opened Span
}

func markdownFenceRun(raw string) (byte, int, string) {
	indent := 0
	for indent < len(raw) && raw[indent] == ' ' {
		indent++
	}
	if indent > 3 || indent >= len(raw) {
		return 0, 0, ""
	}
	marker := raw[indent]
	if marker != '`' && marker != '~' {
		return 0, 0, ""
	}
	width := 0
	for indent+width < len(raw) && raw[indent+width] == marker {
		width++
	}
	if width < 3 {
		return 0, 0, ""
	}
	return marker, width, raw[indent+width:]
}

func (f *markdownCodeFence) skip(ln textLine) bool {
	raw := ln.text
	if ln.number == 1 {
		raw = strings.TrimPrefix(raw, "\ufeff")
	}
	marker, width, rest := markdownFenceRun(raw)
	if f.width > 0 {
		if marker == f.marker && width >= f.width && strings.TrimSpace(rest) == "" {
			f.width = 0
		}
		return true
	}
	if width < 3 {
		return false
	}
	f.marker, f.width = marker, width
	f.opened = Span{Start: ln.start, End: ln.start + len(ln.text), Line: ln.number}
	return true
}

// ParseCandidate only interprets the two explicitly named tables in the
// fixed fixture profile. It never admits meaning, selects Providers, or
// changes the Workspace's TEI_MD_PARSER_UNBOUND rejection.
func ParseCandidate(source []byte) (Candidate, error) {
	if !utf8.Valid(source) {
		return Candidate{}, &Diagnostic{Code: "INVALID_UTF8", Detail: "source is not UTF-8"}
	}
	var out Candidate
	const (
		none  = 0
		nodes = 1
		edges = 2
	)
	mode := none
	rows := [3]int{}
	seen := [3]bool{}
	nodeIDs := map[string]struct{}{}
	edgeIDs := map[string]struct{}{}
	var fence markdownCodeFence
	for _, ln := range sourceLines(source) {
		trim := strings.TrimSpace(ln.text)
		if ln.number == 1 {
			trim = strings.TrimPrefix(trim, "\ufeff")
		}
		if strings.HasPrefix(trim, "> ") || trim == ">" {
			continue
		}
		if fence.skip(ln) {
			// Fenced examples are never candidate declarations. A closing
			// fence cannot silently resume the previous table section.
			mode = none
			continue
		}
		if strings.HasPrefix(ln.text, "    ") || strings.HasPrefix(ln.text, "\t") {
			// Markdown indented code is presentation, not a heading/table.
			mode = none
			continue
		}
		if strings.HasPrefix(trim, "#") {
			mode = none
			switch trim {
			case "## ノードの定義":
				mode = nodes
			case "## 関係の定義":
				mode = edges
			}
			if mode != none {
				if seen[mode] {
					return Candidate{}, &Diagnostic{Code: "DUPLICATE_SECTION", At: Span{Start: ln.start, End: ln.start + len(ln.text), Line: ln.number}, Detail: trim}
				}
				seen[mode] = true
			}
			continue
		}
		if mode == none || !strings.HasPrefix(trim, "|") {
			continue
		}
		cells, err := tableCells(ln)
		if err != nil {
			return Candidate{}, err
		}
		phase := rows[mode]
		rows[mode]++
		if phase == 0 {
			good := false
			if mode == nodes {
				good = headerMatches(cells, "ID", "表示名")
			} else {
				good = headerMatches(cells, "Edge ID", "起点", "終点")
			}
			if !good {
				return Candidate{}, &Diagnostic{Code: "UNSUPPORTED_TABLE_HEADER", At: cells[0].span, Detail: "table columns do not match the selected fixture profile"}
			}
			continue
		}
		if phase == 1 {
			if !isSeparator(cells) {
				return Candidate{}, &Diagnostic{Code: "INVALID_TABLE_SEPARATOR", At: cells[0].span, Detail: "expected Markdown table separator"}
			}
			continue
		}
		if mode == nodes {
			if len(cells) != 2 || cells[0].value == "" || cells[1].value == "" {
				return Candidate{}, &Diagnostic{Code: "INVALID_NODE_ROW", At: cells[0].span, Detail: "node requires ID and label"}
			}
			id := cells[0].value
			if _, exists := nodeIDs[id]; exists {
				return Candidate{}, &Diagnostic{Code: "DUPLICATE_NODE_ID", At: cells[0].span, Detail: id}
			}
			nodeIDs[id] = struct{}{}
			out.Nodes = append(out.Nodes, Node{ID: id, Label: cells[1].value, IDSource: cells[0].span})
		} else {
			if len(cells) != 3 || cells[0].value == "" || cells[1].value == "" || cells[2].value == "" {
				return Candidate{}, &Diagnostic{Code: "INVALID_EDGE_ROW", At: cells[0].span, Detail: "edge requires ID, source and target"}
			}
			id := cells[0].value
			if _, exists := edgeIDs[id]; exists {
				return Candidate{}, &Diagnostic{Code: "DUPLICATE_EDGE_ID", At: cells[0].span, Detail: id}
			}
			edgeIDs[id] = struct{}{}
			out.Edges = append(out.Edges, Edge{ID: id, From: cells[1].value, To: cells[2].value, IDSource: cells[0].span, FromSource: cells[1].span, ToSource: cells[2].span})
		}
	}
	if fence.width > 0 {
		return Candidate{}, &Diagnostic{Code: "UNTERMINATED_FENCE", At: fence.opened, Detail: "Markdown example fence is not closed"}
	}
	for _, mode := range []int{nodes, edges} {
		if !seen[mode] || rows[mode] < 3 {
			return Candidate{}, &Diagnostic{Code: "MISSING_REQUIRED_TABLE", Detail: "selected heading or at least one row is missing"}
		}
	}
	for _, e := range out.Edges {
		if _, ok := nodeIDs[e.From]; !ok {
			return Candidate{}, &Diagnostic{Code: "UNKNOWN_EDGE_ENDPOINT", At: e.FromSource, Detail: e.From}
		}
		if _, ok := nodeIDs[e.To]; !ok {
			return Candidate{}, &Diagnostic{Code: "UNKNOWN_EDGE_ENDPOINT", At: e.ToSource, Detail: e.To}
		}
	}
	return out, nil
}

func (c Candidate) Outgoing() map[string][]string {
	out := map[string][]string{}
	for _, n := range c.Nodes {
		out[n.ID] = []string{}
	}
	for _, e := range c.Edges {
		out[e.From] = append(out[e.From], e.ID)
	}
	for _, ids := range out {
		sort.Strings(ids)
	}
	return out
}

// The explicit aliases are supplied by the consumer of this *reference view*;
// they must never be inferred from duplicate display labels. This compares a
// deliberately tiny Mermaid subset and does not interpret Mermaid as meaning.
var tinyArrow = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9]*)(?:\[[^\]]*\])?\s+-->\|([^|]+)\|\s+([A-Za-z][A-Za-z0-9]*)(?:\[[^\]]*\])?\s*$`)

func CheckDerivedView(source []byte, c Candidate, aliases map[string]string) (bool, error) {
	inMermaid, found := false, false
	diagram := map[string][2]string{}
	for _, ln := range sourceLines(source) {
		v := strings.TrimSpace(ln.text)
		if !inMermaid {
			if v == "```mermaid" {
				inMermaid = true
				found = true
			}
			continue
		}
		if v == "```" {
			inMermaid = false
			continue
		}
		if strings.HasPrefix(v, "flowchart ") || v == "" {
			continue
		}
		m := tinyArrow.FindStringSubmatch(v)
		if m == nil {
			return false, &Diagnostic{Code: "UNSUPPORTED_MERMAID_VIEW", At: Span{Start: ln.start, End: ln.start + len(ln.text), Line: ln.number}, Detail: v}
		}
		from, ok1 := aliases[m[1]]
		to, ok2 := aliases[m[3]]
		if !ok1 || !ok2 {
			return false, &Diagnostic{Code: "UNBOUND_DISPLAY_ALIAS", At: Span{Start: ln.start, End: ln.start + len(ln.text), Line: ln.number}, Detail: v}
		}
		if _, ok := diagram[m[2]]; ok {
			return false, &Diagnostic{Code: "DUPLICATE_VIEW_EDGE", At: Span{Start: ln.start, End: ln.start + len(ln.text), Line: ln.number}, Detail: m[2]}
		}
		diagram[m[2]] = [2]string{from, to}
	}
	if !found || inMermaid {
		return false, &Diagnostic{Code: "UNSUPPORTED_MERMAID_VIEW", Detail: "missing or unclosed Mermaid fence"}
	}
	if len(diagram) != len(c.Edges) {
		return false, nil
	}
	for _, edge := range c.Edges {
		endpoints, ok := diagram[edge.ID]
		if !ok || endpoints != [2]string{edge.From, edge.To} {
			return false, nil
		}
	}
	return true, nil
}
