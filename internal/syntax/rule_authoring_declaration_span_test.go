package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A declaration's byte span must include every modifier the author wrote.
// The earlier implementation stopped at "number" even when "of each row",
// "absent as", or "refutes on some rows" followed it.
func TestOperatorDeclarationSpanIncludesTrailingModifiers(t *testing.T) {
	cases := []string{
		"takes number",
		"takes value: number of each row",
		"takes value: number absent as open",
		"takes value: number refutes on some rows",
		"takes value: number of each row absent as open refutes on some rows",
	}
	for _, tail := range cases {
		t.Run(tail, func(t *testing.T) {
			declaration := "operator sales.op: " + tail
			src := "namespace sales\r\n\r\n" + declaration + "\r\n"
			file, err := syntax.ParseRuleAuthoringFile(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(file.Operators) != 1 {
				t.Fatalf("operators=%d, want 1", len(file.Operators))
			}
			span := file.Operators[0].DeclarationSpan
			start := strings.Index(src, declaration)
			if span.Start.Offset != start || span.End.Offset != start+len(declaration) {
				t.Fatalf("declaration span [%d,%d), want [%d,%d)", span.Start.Offset, span.End.Offset, start, start+len(declaration))
			}
			if got := src[span.Start.Offset:span.End.Offset]; got != declaration {
				t.Fatalf("span text=%q, want %q", got, declaration)
			}
		})
	}
}

// The final named value determines the end of a rule, including when an
// earlier assignment had an absent-value mapping. The parser keeps this
// source-level information even though semantic normalization currently
// refuses assigned values.
func TestRuleDeclarationSpanIncludesAllNamedAssignments(t *testing.T) {
	cases := []string{
		"sales.a.b is sales.op",
		"sales.a.b is sales.op low: 0",
		"sales.a.b is sales.op low: 0 high: 100",
		"sales.a.b is sales.op end: 0 or open high: 100",
		"sales.a.b is sales.op low: 0 high: 100 or open",
	}
	for _, tail := range cases {
		t.Run(tail, func(t *testing.T) {
			declaration := "rule sales.r: " + tail
			src := "namespace sales\r\n\r\n" + declaration + "\r\n"
			file, err := syntax.ParseRuleAuthoringFile(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(file.Rules) != 1 {
				t.Fatalf("rules=%d, want 1", len(file.Rules))
			}
			span := file.Rules[0].DeclarationSpan
			start := strings.Index(src, declaration)
			if span.Start.Offset != start || span.End.Offset != start+len(declaration) {
				t.Fatalf("declaration span [%d,%d), want [%d,%d)", span.Start.Offset, span.End.Offset, start, start+len(declaration))
			}
			if got := src[span.Start.Offset:span.End.Offset]; got != declaration {
				t.Fatalf("span text=%q, want %q", got, declaration)
			}
		})
	}
}
