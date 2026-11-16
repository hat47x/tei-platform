package syntax

import (
	"strings"
	"testing"
)

// The five Data item table column names are a closed vocabulary: a header cell is
// accepted only when it names one of them. A constant no test mentions cannot be
// shown to be accepted, and the column it names would then mean less than its
// name suggests (#4032).
//
// `attribute` and `type` are required, so each name is exercised alongside them;
// the column under test is what varies.
func TestDataTableColumnNamesAreAccepted(t *testing.T) {
	dataTable := func(header ...string) MarkdownTable {
		cells := make([]MarkdownCell, 0, len(header))
		for _, text := range header {
			cells = append(cells, MarkdownCell{Text: text})
		}
		// Each column has its own value format: type is a qualified canonical
		// id, required is yes/no, the rest are free text.
		value := func(column string) string {
			switch column {
			case string(DataTableColumnType):
				return "std.customer_id"
			case string(DataTableColumnRequired):
				return "yes"
			default:
				return "customer_id"
			}
		}
		row := make([]MarkdownCell, 0, len(header))
		for _, text := range header {
			row = append(row, MarkdownCell{Text: value(text)})
		}
		return MarkdownTable{
			Words: []MarkdownCell{
				{Text: "data"}, {Text: "Order"}, {Text: "in"}, {Text: "sales"},
			},
			Header: cells,
			Rows:   [][]MarkdownCell{row},
		}
	}

	// The two required columns on their own are accepted.
	if _, err := ParseDataTable(dataTable(string(DataTableColumnAttribute), string(DataTableColumnType))); err != nil {
		t.Fatalf("the required columns were rejected: %v", err)
	}

	// Each of the five names is accepted as a header cell.
	for _, column := range []DataTableColumn{
		DataTableColumnAttribute,
		DataTableColumnType,
		DataTableColumnRequired,
		DataTableColumnDisplay,
		DataTableColumnDescription,
	} {
		header := []string{string(DataTableColumnAttribute), string(DataTableColumnType)}
		if column != DataTableColumnAttribute && column != DataTableColumnType {
			header = append(header, string(column))
		}
		if _, err := ParseDataTable(dataTable(header...)); err != nil {
			t.Errorf("header %q was rejected: %v", column, err)
		}
	}
}

// The list is what the parser iterates, so a column missing from it is a column no
// table can use. Keep the list and the constants from drifting apart.
func TestDataTableColumnsListCoversEveryColumnName(t *testing.T) {
	want := map[DataTableColumn]bool{
		DataTableColumnAttribute:   true,
		DataTableColumnType:        true,
		DataTableColumnRequired:    true,
		DataTableColumnDisplay:     true,
		DataTableColumnDescription: true,
	}
	got := map[DataTableColumn]bool{}
	for _, column := range dataTableColumns {
		if got[column] {
			t.Errorf("%q appears twice in dataTableColumns", column)
		}
		got[column] = true
	}
	if len(got) != len(want) {
		t.Errorf("dataTableColumns has %d entries, want %d", len(got), len(want))
	}
	for column := range want {
		if !got[column] {
			t.Errorf("%q is declared but missing from dataTableColumns", column)
		}
	}
	// An unknown header stays rejected: the list is closed.
	unknown := MarkdownTable{
		Words: []MarkdownCell{
			{Text: "data"}, {Text: "Order"}, {Text: "in"}, {Text: "sales"},
		},
		Header: []MarkdownCell{{Text: "frobnicate"}},
		Rows:   [][]MarkdownCell{{{Text: "x"}}},
	}
	if _, err := ParseDataTable(unknown); err == nil {
		t.Fatal("an unknown column header was accepted")
	}
}

// RulePredicateOperator and RulePredicateNotOverlapping are sentinels for the two
// multi-word forms, not members of the single-word list: an author never writes
// `operator` or `not-overlapping` bare. They are surfaced to the author through
// the accepted-predicate text, which is what a diagnostic quotes.
func TestMultiWordRulePredicatesAreSurfacedToTheAuthor(t *testing.T) {
	text := validRulePredicatesText()
	for _, form := range []string{
		"not after <attribute>",
		"not overlapping per <relation>",
	} {
		if !strings.Contains(text, form) {
			t.Errorf("validRulePredicatesText does not show %q: %s", form, text)
		}
	}

	// The sentinels are distinct from each other and from the single-word forms,
	// so a reader can tell which shape was authored.
	if RulePredicateOperator == RulePredicateNotOverlapping {
		t.Error("RulePredicateOperator and RulePredicateNotOverlapping are the same predicate")
	}
	for _, single := range rulePredicates {
		if RulePredicateOperator == single || RulePredicateNotOverlapping == single {
			t.Errorf("a multi-word sentinel equals the single-word predicate %q", single)
		}
	}

	// Deliberately not valid() as a single word: an author cannot write it bare.
	if RulePredicateOperator.valid() {
		t.Error("RulePredicateOperator.valid() = true, but an author never writes it bare")
	}
	if RulePredicateNotOverlapping.valid() {
		t.Error("RulePredicateNotOverlapping.valid() = true, but an author never writes it bare")
	}
}
