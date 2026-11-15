package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// The expected offsets in this file are found with strings.Index on text that
// occurs once in the document. They do not come from the scanner or the
// parser, so a wrong span cannot agree with itself.

func uniqueOffset(t *testing.T, src, text string) int {
	t.Helper()
	if strings.Count(src, text) != 1 {
		t.Fatalf("test text %q must occur exactly once, found %d", text, strings.Count(src, text))
	}
	return strings.Index(src, text)
}

const teiMarkdownTwoRegions = "# 受注明細の規則\n" +
	"\n" +
	"数量は正の数にする。\n" +
	"\n" +
	"```tei\n" +
	"namespace sales\n" +
	"rule sales.order_line.quantity_positive:\n" +
	"  sales.order_line.quantity is positive\n" +
	"```\n" +
	"\n" +
	"引用の例。定義ではない。\n" +
	"\n" +
	"```text\n" +
	"rule sales.quoted_only: sales.order_line.unit_price is positive\n" +
	"```\n" +
	"\n" +
	"~~~tei\n" +
	"namespace sales\n" +
	"rule sales.order.lines.line_number_unique:\n" +
	"  sales.order_line.line_number is unique per sales.order.lines\n" +
	"~~~\n"

func TestScanTEIMarkdownReturnsOnlyTEIFences(t *testing.T) {
	regions, err := syntax.ScanTEIMarkdown(teiMarkdownTwoRegions)
	if err != nil {
		t.Fatal(err)
	}
	if len(regions) != 2 {
		t.Fatalf("regions=%d, want the two tei fences and not the text fence", len(regions))
	}
	first := teiMarkdownTwoRegions[regions[0].Content.Start.Offset:regions[0].Content.End.Offset]
	const wantFirst = "namespace sales\n" +
		"rule sales.order_line.quantity_positive:\n" +
		"  sales.order_line.quantity is positive\n"
	if first != wantFirst {
		t.Fatalf("first region content=%q", first)
	}
	second := teiMarkdownTwoRegions[regions[1].Content.Start.Offset:regions[1].Content.End.Offset]
	if !strings.HasPrefix(second, "namespace sales\nrule sales.order.lines.line_number_unique:") ||
		strings.Contains(second, "~~~") {
		t.Fatalf("second region content=%q", second)
	}
	// Line numbers are those of the document, counted independently here.
	wantLine := 1 + strings.Count(teiMarkdownTwoRegions[:uniqueOffset(t, teiMarkdownTwoRegions, "```tei\n")], "\n")
	if regions[0].Open.Start.Line != wantLine || regions[0].Content.Start.Line != wantLine+1 {
		t.Fatalf("open line=%d content line=%d, want %d and %d",
			regions[0].Open.Start.Line, regions[0].Content.Start.Line, wantLine, wantLine+1)
	}
}

func TestMaskMarkdownRegionKeepsOnlyTheRegionAtItsOffsets(t *testing.T) {
	for _, src := range []string{
		teiMarkdownTwoRegions,
		"\xef\xbb\xbf" + teiMarkdownTwoRegions,
		strings.ReplaceAll(teiMarkdownTwoRegions, "\n", "\r\n"),
	} {
		regions, err := syntax.ScanTEIMarkdown(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, region := range regions {
			masked := syntax.MaskMarkdownRegion(src, region)
			start, end := region.Content.Start.Offset, region.Content.End.Offset
			if len(masked) != end {
				t.Fatalf("masked length=%d, want the content end %d", len(masked), end)
			}
			if masked[start:end] != src[start:end] {
				t.Fatal("region bytes changed")
			}
			for i := 0; i < start; i++ {
				if src[i] == '\n' && masked[i] != '\n' {
					t.Fatalf("line feed at %d was not kept", i)
				}
				if src[i] != '\n' && masked[i] != ' ' {
					t.Fatalf("byte %d outside the region survived as %q", i, masked[i])
				}
			}
		}
	}
}

func TestScanTEIMarkdownDoesNotReadNestedOrCommentedFences(t *testing.T) {
	cases := map[string]string{
		"example inside a longer fence": "````markdown\n```tei\nnamespace quoted\nrule quoted.a: quoted.b.c is positive\n```\n````\n\n" +
			"```tei\nnamespace sales\nrule sales.r: sales.a.b is positive\n```\n",
		"example inside a tilde fence": "~~~\n```tei\nnamespace quoted\n```\n~~~\n\n" +
			"```tei\nnamespace sales\nrule sales.r: sales.a.b is positive\n```\n",
		"fence inside a multi-line comment": "<!--\n```tei\nnamespace quoted\n```\n-->\n\n" +
			"```tei\nnamespace sales\nrule sales.r: sales.a.b is positive\n```\n",
		"fence indented by three spaces":    "   ```tei\n   namespace sales\n   rule sales.r: sales.a.b is positive\n   ```\n",
		"closing fence longer than opening": "```tei\nnamespace sales\nrule sales.r: sales.a.b is positive\n`````\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			regions, err := syntax.ScanTEIMarkdown(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(regions) != 1 {
				t.Fatalf("regions=%d, want exactly the one real region", len(regions))
			}
			content := src[regions[0].Content.Start.Offset:regions[0].Content.End.Offset]
			if !strings.Contains(content, "namespace sales") || strings.Contains(content, "quoted") {
				t.Fatalf("read the wrong region: %q", content)
			}
		})
	}
}

func TestScanTEIMarkdownRefusesNearMisses(t *testing.T) {
	const body = "namespace sales\nrule sales.r: sales.a.b is positive\n"
	cases := []struct {
		name, src, at, message string
	}{
		{"upper case name", "intro\n\n```TEI\n" + body + "```\n", "```TEI", "unknown TEI fence"},
		{"old profile name", "```tei-definition id=q\n" + body + "```\n", "```tei-definition", "unknown TEI fence"},
		{"extra info word", "```tei rule\n" + body + "```\n", "```tei rule", "unknown TEI fence"},
		{"indented code", "text\n\n    ```tei\n    " + body + "    ```\n", "```tei", "four or more columns"},
		{"tab indented", "text\n\n\t```tei\n" + body + "\t```\n", "```tei", "four or more columns"},
		{"inside a quotation", "> ```tei\n> namespace sales\n> ```\n", "```tei", "inside a quotation"},
		{"never closed", "```tei\n" + body + "\nmore prose\n", "```tei", "not closed"},
		{"closed by the other fence character", "```tei\n" + body + "~~~\n", "```tei", "not closed"},
		{"comment mark of a later notation", "<!-- tei: membership in example.company -->\n| id |\n| --- |\n", "<!-- tei:", "comment marks are not read"},
		{"closing comment mark", "text\n<!-- /tei -->\n", "<!-- /tei", "comment marks are not read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ScanTEIMarkdown(tc.src)
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("a near miss must be a located error, got %v", err)
			}
			if !strings.Contains(parseErr.Msg, tc.message) {
				t.Fatalf("message=%q, want it to mention %q", parseErr.Msg, tc.message)
			}
			if want := uniqueOffset(t, tc.src, tc.at); parseErr.Pos.Offset != want {
				t.Fatalf("error offset=%d, want %d (%q)", parseErr.Pos.Offset, want, tc.at)
			}
		})
	}
}

func TestScanTEIMarkdownRefusesADocumentWithoutRegions(t *testing.T) {
	for _, src := range []string{"", "# notes only\n", "```text\nrule a: b.c is positive\n```\n"} {
		if _, err := syntax.ScanTEIMarkdown(src); err == nil ||
			!strings.Contains(err.Error(), "declares no ```tei region") {
			t.Fatalf("a document that defines nothing was accepted: %q -> %v", src, err)
		}
	}
}
