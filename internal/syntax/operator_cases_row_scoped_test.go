package syntax_test

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// A row-scoped operator reads every row of a relation, so its example is a
// table whose rows are the rows of that relation and whose columns are the
// operator's inputs; the answer is stated once, in the mark (decision section
// 8, "one table is one example"; section 8.1 fixes the form).
//
// The expectations below are the rows written in the document, by hand.
const rowScopedCasesDocument = "# 明細の上限\n\n" +
	"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
	"| value | limit |\n" +
	"| --- | --- |\n" +
	"| 2 | 10 |\n" +
	"| 5 | 5 |\n\n" +
	"<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n" +
	"| value | limit |\n" +
	"| --- | --- |\n" +
	"| 2 | 10 |\n" +
	"| 5 | 4 |\n"

func parseRowScopedTables(t *testing.T, src string) []syntax.OperatorCasesAuthoring {
	t.Helper()
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]syntax.OperatorCasesAuthoring, 0, len(document.Tables))
	for _, table := range document.Tables {
		parsed, err := syntax.ParseOperatorCasesTable(table)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		out = append(out, parsed)
	}
	return out
}

func TestARowScopedCasesTableIsOneExampleOfRows(t *testing.T) {
	parsed := parseRowScopedTables(t, rowScopedCasesDocument)
	if len(parsed) != 2 {
		t.Fatalf("tables=%d, want two", len(parsed))
	}
	want := []struct {
		satisfied bool
		rows      []map[string]string
	}{
		{
			satisfied: true,
			rows: []map[string]string{
				{"value": "2", "limit": "10"},
				{"value": "5", "limit": "5"},
			},
		},
		{
			satisfied: false,
			rows: []map[string]string{
				{"value": "2", "limit": "10"},
				{"value": "5", "limit": "4"},
			},
		},
	}
	for index, table := range parsed {
		if table.Operator != "sales.every_line_at_most" {
			t.Fatalf("table %d operator=%s", index, table.Operator)
		}
		if table.RowAnswer == nil || *table.RowAnswer != want[index].satisfied {
			t.Fatalf("table %d rowAnswer=%v, want %v", index, table.RowAnswer, want[index].satisfied)
		}
		if len(table.Cases) != 1 {
			t.Fatalf("table %d cases=%d, want one example", index, len(table.Cases))
		}
		example := table.Cases[0].Value
		if example.RowAnswer == nil || *example.RowAnswer != want[index].satisfied {
			t.Fatalf("table %d example answer=%v", index, example.RowAnswer)
		}
		if !reflect.DeepEqual(example.Rows, want[index].rows) {
			t.Fatalf("table %d rows=%v, want %v", index, example.Rows, want[index].rows)
		}
		// The one-record fields are not used by a row-scoped example.
		if example.Value != "" || example.Inputs != nil {
			t.Fatalf("table %d also states a one-record value: %+v", index, example)
		}
	}
}

// A table marked with the row-scoped answer but carrying an `expect` column
// says the answer twice. The column is the one-record form, so it is refused
// rather than read as another row.
func TestARowScopedTableRefusesAnExpectColumn(t *testing.T) {
	src := "# x\n\n" +
		"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
		"| value | expect |\n" +
		"| --- | --- |\n" +
		"| 2 | satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
	if err == nil {
		t.Fatal("a row-scoped table with an expect column was accepted")
	}
	if !strings.Contains(err.Error(), "remove the expect column") {
		t.Fatalf("err=%v, want it to ask for removing the expect column", err)
	}
}

func TestRowScopedCasesTableRefusals(t *testing.T) {
	for _, tc := range []struct {
		name, src, want string
	}{
		{
			name: "the answer is missing",
			src:  "# x\n\n<!-- tei: cases of sales.every_line_at_most expecting -->\n| value |\n| --- |\n| 2 |\n",
			want: "expecting",
		},
		{
			name: "the answer is not satisfied or not-satisfied",
			src:  "# x\n\n<!-- tei: cases of sales.every_line_at_most expecting maybe -->\n| value |\n| --- |\n| 2 |\n",
			want: "satisfied or not-satisfied",
		},
		{
			name: "a column is empty",
			src:  "# x\n\n<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n| value | limit |\n| --- | --- |\n| 2 |  |\n",
			want: "empty",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(tc.src)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
			if err == nil {
				t.Fatal("a malformed row-scoped table was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want it to mention %q", err, tc.want)
			}
		})
	}
}

// The same rows written twice are one example, whatever the answer: one
// example has one expected result.
func TestRowScopedCasesTableRefusesRepeatedRows(t *testing.T) {
	src := "# x\n\n" +
		"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n| 2 | 10 |\n\n" +
		"<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->\n" +
		"| value | limit |\n| --- | --- |\n| 2.0 | 10 |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	first, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := syntax.ParseOperatorCasesTable(document.Tables[1])
	if err != nil {
		t.Fatal(err)
	}
	if semantic.SameOperatorCaseStates(first.Cases[0].Value, second.Cases[0].Value) == false {
		t.Fatal("2 and 2.0 are the same row values and must be recognised as the same example")
	}
}

// Equal rows in one table are two rows of the relation, not a repeated example.
func TestRowScopedCasesTableKeepsEqualRows(t *testing.T) {
	parsed := parseRowScopedTables(t, "# x\n\n"+
		"<!-- tei: cases of sales.every_line_at_most expecting satisfied -->\n"+
		"| value | limit |\n| --- | --- |\n| 2 | 10 |\n| 2 | 10 |\n")
	if len(parsed) != 1 || len(parsed[0].Cases[0].Value.Rows) != 2 {
		t.Fatalf("parsed=%+v, want one example of two equal rows", parsed)
	}
}
