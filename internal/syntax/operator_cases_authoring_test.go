package syntax_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

const operatorCasesDocument = "# 値引きの検査\n\n0以上であることを確かめる。\n\n" +
	"<!-- tei: cases of sales.non_negative -->\n" +
	"| value | expect | note |\n" +
	"| --- | --- | --- |\n" +
	"| 300 | satisfied | |\n" +
	"| 0 | satisfied | 値引きなしは正しい入力 |\n" +
	"| -0.01 | not-satisfied | |\n"

func TestOperatorCasesTableIsReadAsWritten(t *testing.T) {
	for name, src := range map[string]string{
		"lf":   operatorCasesDocument,
		"crlf": strings.ReplaceAll(operatorCasesDocument, "\n", "\r\n"),
	} {
		t.Run(name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(src)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
			if err != nil {
				t.Fatal(err)
			}
			// Expected values are the rows of the table, written out by hand.
			want := []semantic.OperatorCase{
				{Value: "300", Satisfied: true},
				{Value: "0", Satisfied: true},
				{Value: "-0.01", Satisfied: false},
			}
			var got []semantic.OperatorCase
			for _, example := range parsed.Cases {
				got = append(got, example.Value)
			}
			if parsed.Operator != "sales.non_negative" || !reflect.DeepEqual(got, want) {
				t.Fatalf("operator=%s cases=%+v, want %+v", parsed.Operator, got, want)
			}
			for what, check := range map[string]struct {
				span syntax.Anchor
				text string
			}{
				"operator":    {parsed.OperatorSpan, "sales.non_negative"},
				"last value":  {parsed.Cases[2].ValueSpan, "-0.01"},
				"last expect": {parsed.Cases[2].ExpectSpan, "not-satisfied"},
			} {
				at := uniqueOffset(t, src, check.text)
				if check.span.Start.Offset != at || src[check.span.Start.Offset:check.span.End.Offset] != check.text {
					t.Errorf("%s spans [%d,%d), want %q at %d", what, check.span.Start.Offset, check.span.End.Offset, check.text, at)
				}
			}
		})
	}
}

func TestOperatorCasesTableRefusals(t *testing.T) {
	table := func(mark, header, rows string) string {
		// The delimiter row must have one cell per header cell, and tables now
		// stand either one value column or one column per declared input.
		cells := strings.Count(header, "|") - 1
		delimiter := "|" + strings.Repeat(" --- |", cells)
		return mark + "\n" + header + "\n" + delimiter + "\n" + rows + "\n"
	}
	const mark = "<!-- tei: cases of sales.op -->"
	const header = "| value | expect |"
	cases := []struct {
		name, src, at, message string
	}{
		{"mark in another form", table("<!-- tei: cases for sales.op -->", header, "| 1 | satisfied |"), "<!-- tei:", "unknown tei mark"},
		{"operator without a namespace", table("<!-- tei: cases of op -->", header, "| 1 | satisfied |"), "op -->", "not a qualified canonical id"},
		{"no expect column", table(mark, "| value | note |", "| 1 | x |"), "<!-- tei:", "needs the column \"expect\""},
		{"column written twice", table(mark, "| value | value | expect |", "| 1 | 1 | satisfied |"), "value | value |", "more than once"},
		{"empty value", table(mark, header, "|  | satisfied |"), "|  | satisfied", "value cell is empty"},
		{"empty expectation", table(mark, header, "| 1 |  |"), "| 1 |  |", "no expected result"},
		{"expectation in other words", table(mark, header, "| 1 | ok |"), "ok |", "satisfied or not-satisfied"},
		{"expectation with a space", table(mark, header, "| 1 | not satisfied |"), "not satisfied", "satisfied or not-satisfied"},
		{"exponent", table(mark, header, "| 1e5 | satisfied |"), "1e5", "not a decimal number"},
		{"digit grouping", table(mark, header, "| 1,000 | satisfied |"), "1,000", "not a decimal number"},
		{"leading plus", table(mark, header, "| +1 | satisfied |"), "+1", "not a decimal number"},
		{"leading zeros", table(mark, header, "| 007 | satisfied |"), "007", "not a decimal number"},
		{"no digit before the point", table(mark, header, "| .5 | satisfied |"), ".5", "not a decimal number"},
		{"no digit after the point", table(mark, header, "| 5. | satisfied |"), "5.", "not a decimal number"},
		{"a word", table(mark, header, "| none | satisfied |"), "none", "not a decimal number"},
		{"same value twice", table(mark, header, "| 5 | satisfied |\n| 5 | not-satisfied |"), "5 | not-satisfied", "more than once"},
		{"same number written differently", table(mark, header, "| 0 | satisfied |\n| 0.0 | not-satisfied |"), "0.0", "more than once"},
		{"minus zero", table(mark, header, "| 0 | satisfied |\n| -0 | satisfied |"), "-0", "more than once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document, err := syntax.ScanTEIMarkdownDocument(tc.src)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
			var parseErr *syntax.ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("accepted or unlocated: %v", err)
			}
			if !strings.Contains(parseErr.Msg, tc.message) {
				t.Fatalf("message=%q, want %q", parseErr.Msg, tc.message)
			}
			want := uniqueOffset(t, tc.src, tc.at)
			if parseErr.Pos.Offset < want || parseErr.Pos.Offset > want+len(tc.at) {
				t.Fatalf("error offset=%d, want within %q at %d", parseErr.Pos.Offset, tc.at, want)
			}
		})
	}
}

// A multi-input operator states its example as one column per declared input.
// The design's within(value, low, high) names its first input "value", so that
// column is an input name here, not the single-value form.
func TestOperatorCasesTableReadsOneColumnPerInput(t *testing.T) {
	src := "<!-- tei: cases of sales.within -->\n" +
		"| value | low | high | expect |\n" +
		"| --- | --- | --- | --- |\n" +
		"| 3 | 0 | 5 | satisfied |\n" +
		"| 6 | 0 | 5 | not-satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	parsed, err := syntax.ParseOperatorCasesTable(document.Tables[0])
	if err != nil {
		t.Fatalf("per-input example table was refused: %v", err)
	}
	if len(parsed.Cases) != 2 {
		t.Fatalf("cases=%d, want 2", len(parsed.Cases))
	}
	first := parsed.Cases[0].Value
	if first.Inputs == nil {
		t.Fatal("the example carries no per-input values")
	}
	if len(first.Inputs) != 3 || first.Inputs["value"] != "3" ||
		first.Inputs["low"] != "0" || first.Inputs["high"] != "5" {
		t.Fatalf("inputs=%v, want value/low/high", first.Inputs)
	}
	if !first.Satisfied {
		t.Fatal("the first example should be satisfied")
	}
	if parsed.Cases[1].Value.Satisfied {
		t.Fatal("the second example should be not-satisfied")
	}
	// The cell spans point at each input's own cell, so a diagnostic can name it.
	for name := range first.Inputs {
		span, ok := parsed.Cases[0].InputSpans[name]
		if !ok || span.End.Offset <= span.Start.Offset {
			t.Fatalf("input %q has no usable span: %+v", name, span)
		}
	}
}

// Two examples that state the same values are the same example, however they are
// written.
func TestOperatorCasesTableRefusesTheSamePerInputExampleTwice(t *testing.T) {
	src := "<!-- tei: cases of sales.within -->\n" +
		"| value | low | high | expect |\n" +
		"| --- | --- | --- | --- |\n" +
		"| 3 | 0 | 5 | satisfied |\n" +
		"| 3.0 | 0 | 5 | not-satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("a repeated per-input example was accepted or unlocated: %v", err)
	}
	if !strings.Contains(parseErr.Msg, "more than once") {
		t.Fatalf("message=%q, want the duplicate refusal", parseErr.Msg)
	}
}

// A blank cell states nothing. It must not be read as zero.
func TestOperatorCasesTableRefusesABlankInputCell(t *testing.T) {
	src := "<!-- tei: cases of sales.within -->\n" +
		"| value | low | high | expect |\n" +
		"| --- | --- | --- | --- |\n" +
		"| 3 |  | 5 | satisfied |\n"
	document, err := syntax.ScanTEIMarkdownDocument(src)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	_, err = syntax.ParseOperatorCasesTable(document.Tables[0])
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("a blank input cell was accepted or unlocated: %v", err)
	}
	if !strings.Contains(parseErr.Msg, "is empty") {
		t.Fatalf("message=%q, want the empty-cell refusal", parseErr.Msg)
	}
}
