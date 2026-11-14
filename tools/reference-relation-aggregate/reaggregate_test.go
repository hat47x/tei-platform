package relationaggregate

import (
	"encoding/json"
	"strings"
	"testing"
)

// The JSON requests follow #2593 MC-02 and MC-03. Expected values are fixed by
// hand from the base facts: 180/10 = 18 and 9/12 = 3/4.

const mc02Request = `{
  "measureRef": "synthetic.score",
  "law": "mean",
  "baseGrain": "synthetic.observation",
  "scope": "all departments",
  "axis": {"name": "department", "kind": "dimension"},
  "partition": "disjoint",
  "time": {"kind": "period-flow"},
  "nullPolicy": "propagate",
  "components": {
    "sum":   {"anchor": "org", "relation": "dept.score_sum",   "key": ["dept.id"], "unit": "point",       "complete": true,
              "groups": [{"id": "A", "state": "present", "amount": "20"}, {"id": "B", "state": "present", "amount": "160"}]},
    "count": {"anchor": "org", "relation": "dept.score_count", "key": ["dept.id"], "unit": "observation", "complete": true,
              "groups": [{"id": "A", "state": "present", "amount": "2"},  {"id": "B", "state": "present", "amount": "8"}]}
  }
}`

func yes() *bool { v := true; return &v }
func no() *bool  { v := false; return &v }

func decodeReaggregation(t *testing.T, raw string) ReaggregationInput {
	t.Helper()
	var in ReaggregationInput
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		t.Fatal(err)
	}
	return in
}

func TestReaggregationJSONMeanFromSumAndCount(t *testing.T) {
	report := EvaluateReaggregation(decodeReaggregation(t, mc02Request))
	if report.Verdict != "eligible" || report.Code != "OK" || report.Exact != "18" || report.Result != "18" || len(report.Findings) != 0 {
		t.Fatalf("report=%+v want eligible 18", report)
	}
}

func TestReaggregationJSONLostStateAndMissingDeclarations(t *testing.T) {
	onlyMeans := decodeReaggregation(t, mc02Request)
	onlyMeans.Components = map[string]ReaggregationComponent{"value": {
		Anchor: "org", Relation: "dept.score_mean", Key: []string{"dept.id"}, Unit: "point", Complete: yes(),
		Groups: []Element{line("A", "10"), line("B", "20")},
	}}
	report := EvaluateReaggregation(onlyMeans)
	if report.Verdict != "ineligible" || report.Code != "INTERMEDIATE_STATE_LOST" || report.Gap != "missing-input" || report.Result != "" || report.Exact != "" {
		t.Fatalf("report=%+v want ineligible without a value", report)
	}
	if strings.Join(report.Requires, ",") != "sum,count" {
		t.Fatalf("requires=%v", report.Requires)
	}

	// Omitted JSON fields stay undeclared.
	undeclared := decodeReaggregation(t, mc02Request)
	undeclared.Scope, undeclared.Partition = "", ""
	report = EvaluateReaggregation(undeclared)
	if report.Verdict != "unknown" || report.Gap != "missing-semantics" || len(report.Findings) != 2 || report.Exact != "" {
		t.Fatalf("report=%+v want unknown with two findings", report)
	}

	// An unavailable group is not NULL and not zero; a page is not the whole.
	unavailable := decodeReaggregation(t, mc02Request)
	unavailable.Components["count"].Groups[1] = Element{ID: "B", State: ElementInputUnavailable}
	if report = EvaluateReaggregation(unavailable); report.Verdict != "unknown" || report.Code != "INPUT_UNAVAILABLE" {
		t.Fatalf("report=%+v want INPUT_UNAVAILABLE", report)
	}
	page := decodeReaggregation(t, mc02Request)
	count := page.Components["count"]
	count.Complete = no()
	page.Components["count"] = count
	if report = EvaluateReaggregation(page); report.Verdict != "ineligible" || report.Code != "INCOMPLETE_SET" {
		t.Fatalf("report=%+v want INCOMPLETE_SET", report)
	}
}

func TestReaggregationJSONRatioAndRepeatedOccurrence(t *testing.T) {
	ratio := ReaggregationInput{
		Law: "ratio", BaseGrain: "synthetic.attempt", Scope: "all departments",
		Axis: ReaggregationAxis{Name: "department", Kind: "dimension"}, Partition: "disjoint",
		Time: ReaggregationTime{Kind: "period-flow"},
		Components: map[string]ReaggregationComponent{
			"numerator":   {Anchor: "org", Relation: "dept.passed", Key: []string{"dept.id"}, Unit: "attempt", Complete: yes(), Groups: []Element{line("A", "1"), line("B", "8")}},
			"denominator": {Anchor: "org", Relation: "dept.attempted", Key: []string{"dept.id"}, Unit: "attempt", Complete: yes(), Groups: []Element{line("A", "2"), line("B", "10")}},
		},
	}
	report := EvaluateReaggregation(ratio)
	if report.Verdict != "eligible" || report.Exact != "3/4" || report.Result != "0.75" {
		t.Fatalf("report=%+v want 3/4", report)
	}

	// MC-01: the same line occurring three times after a JOIN.
	fanout := ReaggregationInput{
		Law: "sum", BaseGrain: "sales.order_line", Scope: "order H",
		Axis: ReaggregationAxis{Name: "line", Kind: "dimension"}, Partition: "disjoint",
		Time: ReaggregationTime{Kind: "period-flow"},
		Components: map[string]ReaggregationComponent{
			"sum": {Anchor: "order.H", Relation: "sales.order_lines", Key: []string{"order.id"}, Unit: "JPY", Complete: yes(),
				Groups: []Element{line("L1", "100"), line("L2", "100"), line("L1", "100"), line("L2", "100"), line("L1", "100"), line("L2", "100")}},
		},
	}
	report = EvaluateReaggregation(fanout)
	if report.Verdict != "ineligible" || report.Code != "FACT_OCCURRENCE_REPEATED" || report.Result != "" {
		t.Fatalf("report=%+v want FACT_OCCURRENCE_REPEATED", report)
	}
}

// An omitted "complete" is unknown (never read), not a definite partial read.
func TestReaggregationJSONOmittedCompleteStaysUnknown(t *testing.T) {
	in := decodeReaggregation(t, mc02Request)
	count := in.Components["count"]
	count.Complete = nil
	in.Components["count"] = count
	report := EvaluateReaggregation(in)
	if report.Verdict != "unknown" || report.Code != "INPUT_UNAVAILABLE" || report.Exact != "" {
		t.Fatalf("report=%+v want unknown INPUT_UNAVAILABLE", report)
	}
}

// Exponent and fraction literals would be rounded silently by the sum; they
// are refused as invalid instead.
func TestReaggregationJSONNonPlainDecimalIsRefused(t *testing.T) {
	for _, amount := range []string{"1e-5", "1/3", "0x10"} {
		in := decodeReaggregation(t, mc02Request)
		in.Components["sum"].Groups[0].Amount = amount
		report := EvaluateReaggregation(in)
		if report.Verdict != "unknown" || report.Code != "BINDING_INVALID" || report.Exact != "" {
			t.Fatalf("amount %q: report=%+v want unknown BINDING_INVALID", amount, report)
		}
	}
}
