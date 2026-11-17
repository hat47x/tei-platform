package syntax_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"tei/internal/syntax"
)

// These guard the defects an independent review found in the first version of
// the #5145 fix. Each one was a regression the fix itself introduced, so they
// matter as much as the counterexamples.

// Prose with a pipe after a table must not be swallowed silently. GFM does
// continue a table across such a line, so refusing it is the honest answer: the
// author is told the sentence is being read as a table row. What must not happen
// is the table quietly changing shape.
func TestProseWithAPipeIsNotSilentlySwallowed(t *testing.T) {
	cases := map[string]string{
		"two-cell table": "<!-- tei: m -->\n| id | name |\n| --- | --- |\n| 1 | one |\nUse `a | b` to filter rows.\n\n```tei\nnamespace o\n```\n",
		"one-cell table": "<!-- tei: m -->\n| id |\n| --- |\n| 1 |\nUse a | b here.\n\n```tei\nnamespace o\n```\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(src)
			if err != nil {
				// Refused with a diagnostic that points at the line.
				var parseErr *syntax.ParseError
				if !errors.As(err, &parseErr) {
					t.Fatalf("refused without a located diagnostic: %v", err)
				}
				return
			}
			// Or read as a row, which is what GFM renders. Either is honest;
			// silently keeping the old shape is not.
			if len(document.Tables) != 1 {
				t.Fatalf("tables=%d, want 1", len(document.Tables))
			}
			if rows := len(document.Tables[0].Rows); rows != 1 && rows != 2 {
				t.Fatalf("rows=%d, want 1 (table ended) or 2 (row read)", rows)
			}
		})
	}
}

// An HTML element that closes on its own line opens no block, so a mark after
// it is still a mark.
func TestOneLineHTMLElementDoesNotSwallowTheNextMark(t *testing.T) {
	src := "# title\n" +
		"\n" +
		"<pre>example</pre>\n" +
		"\n" +
		"<!-- tei: data Ghost in organization -->\n" +
		"| attribute | type |\n" +
		"| --- | --- |\n" +
		"| ghostId | std.identifier |\n" +
		"\n" +
		"```tei\n" +
		"namespace organization\n" +
		"```\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("a mark after a self-closing element was refused: %v", err)
	}
	if len(document.Tables) != 1 {
		t.Fatalf("tables=%d, want 1", len(document.Tables))
	}
}

// A non-tei fence that never closes is documentation to the end of the document.
// A `tei` fence quoted inside it is an example, not an unread definition.
func TestUnclosedDocumentationFenceStaysDocumentation(t *testing.T) {
	src := "```tei\n" +
		"namespace o\n" +
		"```\n" +
		"\n" +
		"```markdown\n" +
		"```tei\n" +
		"namespace example_only\n" +
		"```\n"
	if _, err := syntax.ScanTEIMarkdownDocument(src); err != nil {
		t.Fatalf("an example inside an unclosed documentation fence was reported: %v", err)
	}
}

// A mark inside a <div> is not rendered as a table either, so it is refused
// rather than inventing a definition.
func TestMarkInsideBlockLevelHTMLIsRefused(t *testing.T) {
	src := "<div>\n" +
		"<!-- tei: data Ghost in organization -->\n" +
		"| attribute | type |\n" +
		"| --- | --- |\n" +
		"| ghostId | std.identifier |\n" +
		"</div>\n" +
		"\n" +
		"```tei\n" +
		"namespace organization\n" +
		"```\n"
	requireRefused(t, src, "<!-- tei:", "raw HTML block")
}

// Scanning a large document must stay linear in its size. The first version
// recomputed a line and column for every byte, which took minutes on a few
// megabytes.
func TestScanIsLinearInDocumentSize(t *testing.T) {
	if testing.Short() {
		t.Skip("timing check")
	}
	body := strings.Repeat("plain text line here\n", 200000)
	src := body + "\n```tei\nnamespace o\n```\n"
	start := time.Now()
	if _, err := syntax.ScanTEIMarkdownDocument(src); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("%d bytes took %v; the scan looks super-linear", len(src), elapsed)
	}
}
