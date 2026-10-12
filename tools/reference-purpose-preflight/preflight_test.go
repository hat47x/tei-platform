package main

import (
	"bytes"
	"strings"
	"testing"
)

func i64(v int64) *int64 { return &v }

func accepted() RuleFixture {
	return RuleFixture{Meaning: "accepted", MeaningRef: "order.approval-required", SourceRevision: "R1", Basis: "tax-included-order-total", Currency: "JPY", ThresholdMinor: i64(500000)}
}

func order(v int64) OrderFixture {
	return OrderFixture{AmountMinor: i64(v), Basis: "tax-included-order-total", Currency: "JPY", Revision: "O2", ExpectedRevision: "O2"}
}

func TestBusinessGuardCases(t *testing.T) {
	cases := []struct {
		name   string
		c      Case
		status string
		guard  string
	}{
		{"unknown-meaning", Case{ID: "a", Subject: "order.confirm-shipment", Purpose: "execute-preflight", Rule: RuleFixture{Meaning: "candidate"}}, "semantic-unresolved", ""},
		{"runtime-unknown", Case{ID: "b", Subject: "order.confirm-shipment", Purpose: "execute-preflight", Rule: accepted()}, "deferred-to-runtime", ""},
		{"below-threshold", Case{ID: "c", Subject: "order.confirm-shipment", Purpose: "execute-preflight", Rule: accepted(), Order: order(499999)}, "evaluated", "false"},
		{"at-threshold", Case{ID: "d", Subject: "order.confirm-shipment", Purpose: "execute-preflight", Rule: accepted(), Order: order(500000)}, "evaluated", "true"},
		{"missing-currency", Case{ID: "e", Subject: "order.confirm-shipment", Purpose: "execute-preflight", Rule: accepted(), Order: OrderFixture{AmountMinor: i64(500000), Revision: "O2", ExpectedRevision: "O2"}}, "indeterminate", ""},
		{"stale-order", Case{ID: "f", Subject: "order.confirm-shipment", Purpose: "execute-preflight", Rule: accepted(), Order: OrderFixture{AmountMinor: i64(500000), Basis: "tax-included-order-total", Currency: "JPY", Revision: "O3", ExpectedRevision: "O2"}}, "blocked", ""},
		{"meaning-missing-basis", Case{ID: "g", Subject: "order.confirm-shipment", Purpose: "execute-preflight", Rule: RuleFixture{Meaning: "accepted", MeaningRef: "order.approval-required", SourceRevision: "R1", Currency: "JPY", ThresholdMinor: i64(500000)}}, "semantic-unresolved", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.c)
			if got.Status != tc.status || got.Guard != tc.guard {
				t.Fatalf("got status=%q guard=%q; want status=%q guard=%q", got.Status, got.Guard, tc.status, tc.guard)
			}
			if got.ExecutionAllowed {
				t.Fatal("fixture-only preflight must never grant execution")
			}
		})
	}
}

func TestJSONRejectsUnrecognizedOrTrailingInput(t *testing.T) {
	for _, input := range []string{
		`{"cases":[{"id":"c","subject":"s","purpose":"describe","unexpected":true}]}`,
		`{"cases":[]} `,
		`{"cases":[{"id":"c","subject":"s","purpose":"describe"}]} {}`,
	} {
		var buf bytes.Buffer
		if err := run(strings.NewReader(input), &buf); err == nil {
			t.Fatalf("invalid input accepted: %s", input)
		}
	}
}

func TestReportIsDeterministic(t *testing.T) {
	fixture := `{"cases":[{"id":"a","subject":"order.confirm-shipment","purpose":"describe","rule":{"meaning":"candidate"}}]}`
	var first, second bytes.Buffer
	if err := run(strings.NewReader(fixture), &first); err != nil {
		t.Fatal(err)
	}
	if err := run(strings.NewReader(fixture), &second); err != nil {
		t.Fatal(err)
	}
	if first.String() != second.String() || !strings.Contains(first.String(), `"authority": "fixture-only-no-execution"`) {
		t.Fatal("report is not deterministic or fails to label fixture authority")
	}
}
