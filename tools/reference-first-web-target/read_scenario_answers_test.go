//go:build !tei_repair_only

package main

import (
	"context"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// TestReadScenarioAnswersEveryDeclaredRulePerLine pins the row answers this
// fixture produces.
//
// The expected answers below are written from the declared rules and the
// fixture values, not copied from the run: quantity_positive asks whether
// quantity is positive and line_number_positive asks whether line_number is
// positive, so both lines of order 1001 satisfy both. line_number_unique is a
// rule about the whole relation, which stays undecided while the read cannot
// certify it saw every row.
func TestReadScenarioAnswersEveryDeclaredRulePerLine(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner := newReferenceReadScenarioRunner(loaded.World)

	result, err := runner.Run(context.Background(), readScenarioInput{
		From:     "2026-09-18T08:30:00Z",
		RowIndex: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.OrderLineRows) != 2 {
		t.Fatalf("order-line rows=%d", len(result.OrderLineRows))
	}
	if len(result.RowAnswers) != len(result.OrderLineRows) {
		t.Fatalf("row answers=%d for %d rows", len(result.RowAnswers), len(result.OrderLineRows))
	}

	want := map[string]string{
		"sales.order_line.quantity_positive":    "satisfied",
		"sales.order_line.line_number_positive": "satisfied",
	}
	for index, row := range result.RowAnswers {
		if row.RowIndex != index {
			t.Fatalf("row %d is labelled %d", index, row.RowIndex)
		}
		if len(row.Answers) != len(want) {
			t.Fatalf("row %d answers=%+v; the Application declares 2 row rules", index, row.Answers)
		}
		for _, answer := range row.Answers {
			if expected, ok := want[answer.Rule]; ok && answer.Result != expected {
				t.Fatalf("row %d rule %s: got %q want %q", index, answer.Rule, answer.Result, expected)
			}
			if answer.Reason != "" {
				t.Fatalf("row %d rule %s is decided but carries a reason %q", index, answer.Rule, answer.Reason)
			}
		}
	}

	// A relation-wide rule is not satisfied by finding no duplicate in a
	// partial read; it is undecided, and says why.
	if len(result.RelationAnswers) != 1 {
		t.Fatalf("relation answers=%+v; the Application declares 1 relation rule", result.RelationAnswers)
	}
	line := result.RelationAnswers[0]
	if !strings.Contains(line, "sales.order.lines.line_number_unique") ||
		!strings.Contains(line, "undecided") {
		t.Fatalf("relation answer=%q", line)
	}
}
