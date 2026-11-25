package syntax_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/syntax"
)

const dataTableDocument = "# 異動申請\n" +
	"\n" +
	"申請が持つ項目を定義する。\n" +
	"\n" +
	"<!-- tei: data TransferRequest in organization -->\n" +
	"| attribute | type | required | display | description |\n" +
	"| --- | :---: | --- | --- | --- |\n" +
	"| transferRequestId | std.identifier | yes | 異動申請ID | |\n" +
	"| allocation | std.decimal | yes | 配分 | 兼務のときの割合 |\n" +
	"| effectiveTo | std.datetime | no | 終了日 | |\n" +
	"| reason | std.text | | | |\n" +
	"\n" +
	"参考の表。印がないので定義ではない。\n" +
	"\n" +
	"| attribute | type |\n" +
	"| --- | --- |\n" +
	"| notDefined | std.text |\n"

// The same Data written in the native grammar. It is the independent
// expected value: the table must mean exactly what this declaration means.
const dataTableNativeEquivalent = `namespace organization

data TransferRequest {
    attribute transferRequestId type std.identifier display "異動申請ID" required
    attribute allocation type std.decimal description "兼務のときの割合" display "配分" required
    attribute effectiveTo type std.datetime display "終了日"
    attribute reason type std.text
}
`

func TestDataTableMeansTheSameAsTheNativeDeclaration(t *testing.T) {
	for name, src := range map[string]string{
		"lf":   dataTableDocument,
		"crlf": strings.ReplaceAll(dataTableDocument, "\n", "\r\n"),
		"bom":  "\xef\xbb\xbf" + dataTableDocument,
	} {
		t.Run(name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(document.Tables) != 1 || len(document.Regions) != 0 {
				t.Fatalf("tables=%d regions=%d, want only the marked table", len(document.Tables), len(document.Regions))
			}
			parsed, err := syntax.ParseDataTable(document.Tables[0])
			if err != nil {
				t.Fatal(err)
			}
			native, err := syntax.NewParser(dataTableNativeEquivalent).ParseSemanticFile()
			if err != nil {
				t.Fatal(err)
			}
			if len(native.Data) != 1 || !reflect.DeepEqual(parsed.Value, native.Data[0]) {
				t.Fatalf("table and native declaration differ:\n table=%+v\nnative=%+v", parsed.Value, native.Data)
			}
			if parsed.Namespace != native.ID {
				t.Fatalf("namespace=%s, want %s", parsed.Namespace, native.ID)
			}

			// Spans are located independently in the document.
			checks := []struct {
				what string
				span syntax.Anchor
				text string
			}{
				{"data name", parsed.SymbolSpan, "TransferRequest"},
				{"namespace", parsed.NamespaceSpan, "organization"},
				{"first attribute", parsed.Attributes[0].SymbolSpan, "transferRequestId"},
				{"second attribute type", parsed.Attributes[1].TypeSpan, "std.decimal"},
				{"last attribute", parsed.Attributes[3].SymbolSpan, "reason"},
			}
			for _, check := range checks {
				at := uniqueOffset(t, src, check.text)
				if check.span.Start.Offset != at || src[check.span.Start.Offset:check.span.End.Offset] != check.text {
					t.Errorf("%s spans [%d,%d), want %q at %d", check.what, check.span.Start.Offset, check.span.End.Offset, check.text, at)
				}
				wantLine := 1 + strings.Count(src[:at], "\n")
				if check.span.Start.Line != wantLine {
					t.Errorf("%s line=%d, want %d", check.what, check.span.Start.Line, wantLine)
				}
			}
		})
	}
}

func TestDataTableEmptyCellsMeanNotStated(t *testing.T) {
	document, err := syntax.ScanTEIMarkdownDocument(dataTableDocument)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := syntax.ParseDataTable(document.Tables[0])
	if err != nil {
		t.Fatal(err)
	}
	reason := parsed.Value.Attributes[3]
	if reason.Required || reason.Display != "" || reason.Description != "" || reason.Value != nil {
		t.Fatalf("empty cells were read as values: %+v", reason)
	}
	// "no" and an empty required cell agree; only "yes" requires.
	if parsed.Value.Attributes[2].Required || !parsed.Value.Attributes[0].Required {
		t.Fatalf("required column misread: %+v", parsed.Value.Attributes)
	}
}

func TestMarkedTableScanRefusals(t *testing.T) {
	const mark = "<!-- tei: data Note in sales -->\n"
	const header = "| attribute | type |\n| --- | --- |\n"
	cases := []struct {
		name, src, at, message string
	}{
		{"blank line between mark and table", mark + "\n" + header + "| a | std.text |\n", "<!-- tei:", "directly followed by a table"},
		{"mark followed by prose", mark + "ただの文章。\n", "<!-- tei:", "directly followed by a table"},
		{"mark at the end of the document", "text\n\n" + mark, "<!-- tei:", "directly followed by a table"},
		// A table with no body row is now read by the scanner (a row-scoped
		// example states an empty relation that way), so the scanner is no
		// longer where such a table is refused. ParseDataTable refuses it,
		// and TestDataTableWithoutRowsIsRefusedWhereItIsInterpreted covers that.
		{"row with a missing cell", mark + header + "| a |\n", "| a |", "cannot be left out"},
		{"row with an extra cell", mark + header + "| a | std.text | x |\n", "| a | std.text | x |", "cannot be left out"},
		{"escaped pipe in a cell", mark + header + "| a\\|b | std.text |\n", "\\|", "cannot contain |"},
		{"text after the mark", "<!-- tei: data Note in sales --> 補足\n" + header + "| a | std.text |\n", "<!-- tei:", "only thing on its line"},
		{"mark without words", "<!-- tei: -->\n" + header + "| a | std.text |\n", "<!-- tei:", "must say what"},
		{"mark inside a quotation", "> " + mark + "> " + "| attribute | type |\n", "<!-- tei:", "not read"},
		{"mark in an indented block", "text\n\n    " + mark, "<!-- tei:", "not read"},
		{"closing mark of another notation", "text\n<!-- /tei -->\n", "<!-- /tei", "comment marks are not read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ScanTEIMarkdownDocument(tc.src)
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("accepted or unlocated: %v", err)
			}
			if !strings.Contains(parseErr.Msg, tc.message) {
				t.Fatalf("message=%q, want %q", parseErr.Msg, tc.message)
			}
			if want := uniqueOffset(t, tc.src, tc.at); parseErr.Pos.Offset != want {
				t.Fatalf("error offset=%d, want %d", parseErr.Pos.Offset, want)
			}
		})
	}

	// The fence-only entry point still refuses every mark, so a Workspace that
	// selected the older language is told about a table it does not read.
	if _, err := syntax.ScanTEIMarkdown(mark + header + "| a | std.text |\n"); err == nil ||
		!strings.Contains(err.Error(), "comment marks are not read") {
		t.Fatalf("the fence-only scan read or skipped a marked table: %v", err)
	}
}

func TestDataTableRefusals(t *testing.T) {
	table := func(mark, header, row string) string {
		return mark + "\n" + header + "\n| --- | --- | --- |\n" + row + "\n"
	}
	const mark = "<!-- tei: data Note in sales -->"
	const header = "| attribute | type | required |"
	cases := []struct {
		name, src, at, message string
	}{
		{"unknown mark form", table("<!-- tei: membership in sales -->", header, "| a | std.text | |"), "<!-- tei:", "unknown tei mark"},
		{"data name with a namespace", table("<!-- tei: data sales.Note in sales -->", header, "| a | std.text | |"), "sales.Note", "without a namespace"},
		{"nested namespace", table("<!-- tei: data Note in sales.sub -->", header, "| a | std.text | |"), "sales.sub", "top-level canonical id"},
		{"unknown column", table(mark, "| attribute | type | nullable |", "| a | std.text | |"), "nullable", "unknown column"},
		{"column written twice", table(mark, "| attribute | type | type |", "| a | std.text | std.text |"), "type |\n", "more than once"},
		{"no type column", table(mark, "| attribute | display | required |", "| a | x | |"), "<!-- tei:", "needs the column \"type\""},
		{"empty attribute cell", table(mark, header, "|  | std.text | yes |"), "|  | std.text", "attribute cell is empty"},
		{"empty type cell", table(mark, header, "| a |  | yes |"), "| a |  |", "has no type"},
		{"unqualified type", table(mark, header, "| a | text | |"), "text |", "not a qualified canonical id"},
		{"attribute written twice", table(mark, header, "| a | std.text | |\n| a | std.decimal | |"), "a | std.decimal", "more than once"},
		{"required spelled another way", table(mark, header, "| a | std.text | 必須 |"), "必須", "yes or no"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(tc.src)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			_, err = syntax.ParseDataTable(document.Tables[0])
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("accepted or unlocated: %v", err)
			}
			if !strings.Contains(parseErr.Msg, tc.message) {
				t.Fatalf("message=%q, want %q", parseErr.Msg, tc.message)
			}
			want := uniqueOffset(t, tc.src, tc.at)
			// For an empty cell the error sits inside the row that tc.at locates.
			if parseErr.Pos.Offset < want || parseErr.Pos.Offset > want+len(tc.at) {
				t.Fatalf("error offset=%d, want within %q at %d", parseErr.Pos.Offset, tc.at, want)
			}
		})
	}

	// A description that the native grammar could not hold is refused, so
	// every table can be written back as a native declaration.
	src := "<!-- tei: data Note in sales -->\n| attribute | type | description |\n| --- | --- | --- |\n| a | std.text | 引用符 \" を含む |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := syntax.ParseDataTable(document.Tables[0]); err == nil || !strings.Contains(err.Error(), "double quote") {
		t.Fatalf("a description with a double quote was accepted: %v", err)
	}
}

// An ordinary one-line comment is not a mark. The table after it stays
// documentation, and the comment does not disturb a marked table later on.
func TestOrdinaryCommentsDoNotMarkATable(t *testing.T) {
	src := "<!-- 編集メモ: 後で見直す -->\n" +
		"| attribute | type |\n| --- | --- |\n| notDefined | std.text |\n" +
		"\n<!-- teika: これも印ではない -->\n" +
		"\n<!-- tei: data Note in sales -->\n" +
		"| attribute | type |\n| --- | --- |\n| body | std.text |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Tables) != 1 {
		t.Fatalf("tables=%d, want only the table under the tei mark", len(document.Tables))
	}
	parsed, err := syntax.ParseDataTable(document.Tables[0])
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Value.ID != "sales.note" || len(parsed.Value.Attributes) != 1 || parsed.Value.Attributes[0].ID != "sales.note.body" {
		t.Fatalf("read the wrong table: %+v", parsed.Value)
	}
}

// A Data item table with no body row states no attribute. The scanner reads
// such a table (a row-scoped example states an empty relation that way), so the
// refusal is here, where the table is interpreted, with the table's own mark.
func TestDataTableWithoutRowsIsRefusedWhereItIsInterpreted(t *testing.T) {
	const src = "<!-- tei: data Note in sales -->\n| attribute | type |\n| --- | --- |\n\nafter\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("the scanner refused a header-only table: %v", err)
	}
	_, err = syntax.ParseDataTable(document.Tables[0])
	if err == nil {
		t.Fatal("a Data item table with no attribute row was accepted")
	}
	if !strings.Contains(err.Error(), "needs at least one attribute row") {
		t.Fatalf("err=%v, want it to say a row is needed", err)
	}
}
