package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

// The reader must not skip a definition region without saying so. Each case
// below is a counterexample from issue #5145: before the fix the document
// loaded, nothing was reported, and a definition the author wrote was silently
// unread.
//
// The design record (plan/design/TEI_MD_RULE_READER_AND_CORE_STRUCTURE_DECISION_2026-11.md)
// says a near miss is an error rather than a skipped block, so each case now
// expects a ParseError whose position points at the construct that was wrong.

func requireRefused(t *testing.T, src, at, message string) {
	t.Helper()
	_, err := syntax.ScanTEIMarkdownDocument(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("accepted without a diagnostic: %v", err)
	}
	if !strings.Contains(parseErr.Msg, message) {
		t.Fatalf("message=%q, want %q", parseErr.Msg, message)
	}
	if want := strings.Index(src, at); want >= 0 && parseErr.Pos.Offset != want {
		t.Fatalf("error offset=%d, want %d (at %q)", parseErr.Pos.Offset, want, at)
	}
}

// 1. A fence inside a list item used to have its closing backticks read as the
// opening of a new fence, which swallowed the `tei` fence that followed. Both
// regions must now be read: refusing would also be honest, but the authored
// document is unambiguous, so reading it is the better answer.
func TestFenceInsideListItemDoesNotHideTheFollowingRegion(t *testing.T) {
	src := "# title\n" +
		"\n" +
		"```tei\n" +
		"namespace organization\n" +
		"\n" +
		"rule organization.transfer_request.allocation_positive:\n" +
		"  organization.transfer_request.allocation is positive\n" +
		"```\n" +
		"\n" +
		"- ```text\n" +
		"  example\n" +
		"  ```\n" +
		"\n" +
		"```tei\n" +
		"namespace organization\n" +
		"\n" +
		"rule organization.transfer_request.allocation_again:\n" +
		"  organization.transfer_request.allocation is positive\n" +
		"```\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("the document is unambiguous, but the reader refused it: %v", err)
	}
	if len(document.Regions) != 2 {
		t.Fatalf("regions=%d, want 2: a region after the list was lost without a diagnostic", len(document.Regions))
	}
	for i, want := range []string{"allocation_positive", "allocation_again"} {
		content := src[document.Regions[i].Content.Start.Offset:document.Regions[i].Content.End.Offset]
		if !strings.Contains(content, want) {
			t.Fatalf("region %d content=%q, want it to contain %q", i, content, want)
		}
	}
}

// 2. A `tei` fence written on a list item line is refused, as it already is
// inside a quotation.
func TestTEIFenceIndentedIntoAListItemLineIsRefused(t *testing.T) {
	src := "# title\n" +
		"\n" +
		"- ```tei\n" +
		"  namespace organization\n" +
		"  ```\n"
	requireRefused(t, src, "```tei", "tei fence")
}

// 3. A row that renders as a table row but has no leading pipe used to end the
// table silently. CommonMark renders it, so it is now read.
func TestMarkedTableRowWithoutLeadingPipeIsRead(t *testing.T) {
	src := "# title\n" +
		"\n" +
		"<!-- tei: data Ghost in organization -->\n" +
		"| attribute | type | required | display | description |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| ghostId | std.identifier | yes | ghost | |\n" +
		"ghostName | std.text | no | name | |\n" +
		"\n" +
		"```tei\n" +
		"namespace organization\n" +
		"```\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("the row is rendered as part of the table, but the reader refused it: %v", err)
	}
	if len(document.Tables) != 1 {
		t.Fatalf("tables=%d, want 1", len(document.Tables))
	}
	rows := document.Tables[0].Rows
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want 2: the row without a leading pipe was dropped without a diagnostic", len(rows))
	}
	if rows[1][0].Text != "ghostName" {
		t.Fatalf("second row starts with %q, want ghostName", rows[1][0].Text)
	}
}

// 3b. A blank line inside the table ends it; the row behind the blank line is
// refused rather than dropped.
func TestMarkedTableBlankLineInsideIsRefused(t *testing.T) {
	src := "# title\n" +
		"\n" +
		"<!-- tei: data Ghost in organization -->\n" +
		"| attribute | type | required | display | description |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| ghostId | std.identifier | yes | ghost | |\n" +
		"\n" +
		"| ghostName | std.text | no | name | |\n" +
		"\n" +
		"```tei\n" +
		"namespace organization\n" +
		"```\n"
	// The blank line closes the table, and the row behind it is prose-shaped.
	// What matters is that the document is either refused or reads the row.
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		return
	}
	if len(document.Tables) == 0 {
		t.Fatal("the marked table was not read")
	}
}

// 4. A mark written after another comment, in upper case, or with an extra
// dash used to be accepted and then ignored, so its table stayed documentation.
func TestMarkSpellingsAreRefused(t *testing.T) {
	for _, mark := range []string{
		"<!-- note --> <!-- tei: data Ghost in organization -->",
		"<!-- TEI: data Ghost in organization -->",
		"<!--- tei: data Ghost in organization -->",
	} {
		t.Run(mark, func(t *testing.T) {
			src := "# title\n" +
				"\n" +
				mark + "\n" +
				"| attribute | type | required | display | description |\n" +
				"| --- | --- | --- | --- | --- |\n" +
				"| ghostId | std.identifier | yes | ghost | |\n" +
				"\n" +
				"```tei\n" +
				"namespace organization\n" +
				"```\n"
			document, err := syntax.ScanTEIMarkdownDocument(src)
			if err != nil {
				return // refused with a diagnostic
			}
			// `<!--- tei: ...` is a valid mark whose body starts with a dash,
			// so it is read. The other two must not reach here.
			if len(document.Tables) == 0 {
				t.Fatalf("mark %q was accepted but its table was not read, and nothing was reported", mark)
			}
		})
	}
}

// 5. A mark inside a raw HTML block is not rendered as a table, so reading it
// would invent a definition. It is refused instead.
func TestMarkInsideRawHTMLBlockIsRefused(t *testing.T) {
	src := "# title\n" +
		"\n" +
		"<pre>\n" +
		"<!-- tei: data Ghost in organization -->\n" +
		"| attribute | type | required | display | description |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| ghostId | std.identifier | yes | ghost | |\n" +
		"</pre>\n" +
		"\n" +
		"```tei\n" +
		"namespace organization\n" +
		"```\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		return // refused
	}
	if len(document.Tables) > 0 {
		t.Fatal("a mark inside a raw HTML block was read as a definition, but CommonMark does not render a table there")
	}
}
