package scenariodiff

import "testing"

func baseModel() Model {
	return Model{
		Data: []DataDef{
			{ID: "Order", Fields: []Field{
				{Name: "id", Type: "string", Key: true},
				{Name: "customer_id", Type: "string"},
				{Name: "status", Type: "string"},
				{Name: "total", Type: "decimal"},
			}},
			{ID: "Customer", Fields: []Field{
				{Name: "id", Type: "string", Key: true},
				{Name: "name", Type: "string"},
			}},
		},
		Relations: []Relation{{ID: "order-customer", From: "Order.customer_id", To: "Customer.id"}},
	}
}

func beforeSnapshot() Snapshot {
	return Snapshot{ID: "before", Dataset: []DataSet{
		{Data: "Order", Rows: []map[string]any{
			{"id": "o1", "customer_id": "c1", "status": "pending", "total": float64(100)},
		}},
		{Data: "Customer", Rows: []map[string]any{{"id": "c1", "name": "Acme"}}},
	}}
}

func afterSnapshot() Snapshot {
	return Snapshot{ID: "after", Dataset: []DataSet{
		{Data: "Order", Rows: []map[string]any{
			{"id": "o1", "customer_id": "c1", "status": "shipped", "total": float64(100)},
			{"id": "o2", "customer_id": "c1", "status": "pending", "total": float64(50)},
		}},
		{Data: "Customer", Rows: []map[string]any{{"id": "c1", "name": "Acme"}}},
	}}
}

func hasFinding(fs []Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

func findCell(cells []Cell, data, field string) (Cell, bool) {
	for _, c := range cells {
		if c.Data == data && c.Field == field {
			return c, true
		}
	}
	return Cell{}, false
}

func stepByName(r Report, scenario, step string) (StepResult, bool) {
	for _, s := range r.Scenarios {
		if s.Scenario != scenario {
			continue
		}
		for _, st := range s.Steps {
			if st.Step == step {
				return st, true
			}
		}
	}
	return StepResult{}, false
}

func TestModelValidation(t *testing.T) {
	m := Model{
		Data: []DataDef{
			{ID: "Bad", Fields: []Field{{Name: "x", Type: "number"}}},
			{ID: "Ok", Fields: []Field{{Name: "k", Type: "string", Key: true}}},
		},
		Relations: []Relation{{ID: "r", From: "Ok.missing", To: "Nope.k"}},
	}
	r := Plan(Input{Model: m})
	if !hasFinding(r.ModelFindings, "data-key-missing") {
		t.Errorf("missing key not reported: %+v", r.ModelFindings)
	}
	if !hasFinding(r.ModelFindings, "unknown-field-type") {
		t.Errorf("unknown type not reported: %+v", r.ModelFindings)
	}
	if !hasFinding(r.ModelFindings, "relation-source-missing") || !hasFinding(r.ModelFindings, "relation-target-missing") {
		t.Errorf("relation refs not reported: %+v", r.ModelFindings)
	}
}

func TestSnapshotDiff(t *testing.T) {
	r := Plan(Input{Model: baseModel(), Snapshots: []Snapshot{beforeSnapshot(), afterSnapshot()}})
	if len(r.SnapshotDiffs) != 1 {
		t.Fatalf("diffs = %d", len(r.SnapshotDiffs))
	}
	d := r.SnapshotDiffs[0]
	changed, ok := findCell(d.Changed, "Order", "status")
	if !ok {
		t.Fatalf("changed status not found: %+v", d.Changed)
	}
	if changed.Before != "pending" || changed.After != "shipped" {
		t.Errorf("changed cell = %+v", changed)
	}
	if _, ok := findCell(d.Added, "Order", "id"); !ok {
		t.Errorf("added o2 not found: %+v", d.Added)
	}
	if len(d.Removed) != 0 {
		t.Errorf("unexpected removals: %+v", d.Removed)
	}
}

func TestUnchangedRowIsNotReportedAsChanged(t *testing.T) {
	r := Plan(Input{Model: baseModel(), Snapshots: []Snapshot{beforeSnapshot(), beforeSnapshot()}})
	d := r.SnapshotDiffs[0]
	if len(d.Changed) != 0 || len(d.Added) != 0 || len(d.Removed) != 0 {
		t.Errorf("identical snapshots produced a diff: %+v", d)
	}
}

func TestDanglingReference(t *testing.T) {
	s := Snapshot{ID: "s", Dataset: []DataSet{
		{Data: "Order", Rows: []map[string]any{{"id": "o1", "customer_id": "missing", "status": "pending", "total": float64(1)}}},
		{Data: "Customer", Rows: []map[string]any{{"id": "c1", "name": "Acme"}}},
	}}
	r := Plan(Input{Model: baseModel(), Snapshots: []Snapshot{s}})
	if !hasFinding(r.DataFindings, "dangling-reference") {
		t.Errorf("dangling reference not reported: %+v", r.DataFindings)
	}
}

func TestTypeMismatchAndUnknownField(t *testing.T) {
	s := Snapshot{ID: "s", Dataset: []DataSet{
		{Data: "Order", Rows: []map[string]any{{"id": "o1", "status": "ok", "total": "not-a-number", "bogus": true}}},
	}}
	r := Plan(Input{Model: baseModel(), Snapshots: []Snapshot{s}})
	if !hasFinding(r.DataFindings, "field-type-mismatch") {
		t.Errorf("type mismatch not reported: %+v", r.DataFindings)
	}
	if !hasFinding(r.DataFindings, "unknown-field") {
		t.Errorf("unknown field not reported: %+v", r.DataFindings)
	}
}

func scenarioWith(s1, s2 Step) Scenario {
	return Scenario{ID: "sc", Steps: []Step{s1, s2}}
}

func passStep(id string, ord int) Step {
	return Step{
		ID: id, Action: "transition", PresentationOrder: ord,
		Expected: []Invariant{{Data: "Order", Field: "status", Op: "eq", Value: "shipped"}},
		Observed: []Invariant{{Data: "Order", Field: "status", Op: "eq", Value: "shipped"}},
	}
}

func TestExpectedMatchesObserved(t *testing.T) {
	s := passStep("s1", 1)
	r := Plan(Input{Model: baseModel(), Scenarios: []Scenario{{ID: "sc", Steps: []Step{s}}}})
	st, ok := stepByName(r, "sc", "s1")
	if !ok || st.Status != "passed" {
		t.Fatalf("status = %q", st.Status)
	}
}

func TestExpectedMismatchFailed(t *testing.T) {
	s := passStep("s1", 1)
	s.Observed = []Invariant{{Data: "Order", Field: "status", Op: "eq", Value: "pending"}}
	r := Plan(Input{Model: baseModel(), Scenarios: []Scenario{{ID: "sc", Steps: []Step{s}}}})
	st, _ := stepByName(r, "sc", "s1")
	if st.Status != "failed" {
		t.Fatalf("status = %q", st.Status)
	}
	if len(st.Results) != 1 || st.Results[0].Actual != "pending" {
		t.Errorf("actual not recorded: %+v", st.Results)
	}
}

func TestUnobservedManualExternalIsUnverified(t *testing.T) {
	s := Step{
		ID: "s1", Action: "ship", PresentationOrder: 1,
		Expected:     []Invariant{{Data: "Order", Field: "status", Op: "eq", Value: "shipped"}},
		Observations: []Observation{{Source: "external", Data: "Order", Field: "status", Observed: false}},
	}
	r := Plan(Input{Model: baseModel(), Scenarios: []Scenario{{ID: "sc", Steps: []Step{s}}}})
	st, _ := stepByName(r, "sc", "s1")
	if st.Status != "unverified" {
		t.Fatalf("status = %q, want unverified", st.Status)
	}
	if r.Summary.Unverified != 1 || r.Summary.Failed != 0 {
		t.Errorf("summary = %+v", r.Summary)
	}
}

func TestInternalObservationIsNotUnverified(t *testing.T) {
	s := Step{
		ID: "s1", Action: "check", PresentationOrder: 1,
		Expected:     []Invariant{{Data: "Order", Field: "status", Op: "exists"}},
		Observations: []Observation{{Source: "internal", Data: "Order", Field: "status", Observed: true}},
	}
	r := Plan(Input{Model: baseModel(), Scenarios: []Scenario{{ID: "sc", Steps: []Step{s}}}})
	st, _ := stepByName(r, "sc", "s1")
	if st.Status != "failed" {
		t.Fatalf("status = %q, want failed (no observed invariant)", st.Status)
	}
	if !hasFinding(st.Findings, "expected-not-observed") {
		t.Errorf("findings = %+v", st.Findings)
	}
}

func TestPresentationOrderDoesNotAffectAnalysis(t *testing.T) {
	a := scenarioWith(passStep("s1", 1), passStep("s2", 2))
	b := scenarioWith(passStep("s1", 2), passStep("s2", 1))

	ra := Plan(Input{Model: baseModel(), Scenarios: []Scenario{a}})
	rb := Plan(Input{Model: baseModel(), Scenarios: []Scenario{b}})

	if ra.Fingerprint != rb.Fingerprint {
		t.Errorf("fingerprint changed with presentation order: %s vs %s", ra.Fingerprint, rb.Fingerprint)
	}
	// Analysis order keeps the declared step order, never the presentation order.
	if ra.Scenarios[0].Steps[0].Step != "s1" || ra.Scenarios[0].Steps[1].Step != "s2" {
		t.Errorf("analysis order changed: %+v", ra.Scenarios[0].Steps)
	}
	// Presentation order is exposed separately.
	if ra.Scenarios[0].Presentation[0] != "s1" || rb.Scenarios[0].Presentation[0] != "s2" {
		t.Errorf("presentation order not kept separate: %+v / %+v", ra.Scenarios[0].Presentation, rb.Scenarios[0].Presentation)
	}
}

func TestGoldenIsDeterministic(t *testing.T) {
	in := Input{Model: baseModel(), Snapshots: []Snapshot{beforeSnapshot(), afterSnapshot()}, Scenarios: []Scenario{scenarioWith(passStep("s1", 1), passStep("s2", 2))}}
	first := Plan(in)
	second := Plan(in)
	if first.Fingerprint != second.Fingerprint {
		t.Errorf("fingerprint not deterministic: %s vs %s", first.Fingerprint, second.Fingerprint)
	}
}
