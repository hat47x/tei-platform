package procedureexpand

import "testing"

func base() Input {
	return Input{
		Procedure: "order-acceptance",
		Steps: []Step{
			{ID: "s1", Action: "accept-order", Nature: "machine", Authority: "perm:accept", Conditions: []string{"amount>=0"}},
			{ID: "s2", Action: "review", Nature: "manual"},
			{ID: "s3", Action: "notify-carrier", Nature: "external", ExternalEffect: true},
			{ID: "s4", Action: "approve", Nature: "machine", Authority: "perm:approve", Conditions: []string{"<unresolved>"}},
			{ID: "s5", Action: "none"},
		},
		Bindings: []Binding{
			{Step: "s1", Target: "function", Ref: "Fn.AcceptOrder", Authority: "perm:accept"},
			{Step: "s1", Target: "explain", Ref: "doc:accept"},
			{Step: "s2", Target: "screen", Ref: "Screen.Review"},
			{Step: "s3", Target: "external", Ref: "Ext.Carrier"},
			{Step: "s4", Target: "function", Ref: "Fn.Approve", Authority: "perm:approve"},
			{Step: "s4", Target: "explain", Ref: "doc:approve"},
			{Step: "s5", Target: "ai", Ref: "AI.None"},
		},
	}
}

func optionReady(r Report, step, target string) (bool, []string, bool) {
	for _, o := range r.Options {
		if o.Step == step && o.Target == target {
			return o.Ready, o.Missing, true
		}
	}
	return false, nil, false
}

func stepStatus(r Report, id string) string {
	for _, s := range r.Steps {
		if s.Step == id {
			return s.Status
		}
	}
	return ""
}

func stepPlan(r Report, id string) (StepPlan, bool) {
	for _, s := range r.Steps {
		if s.Step == id {
			return s, true
		}
	}
	return StepPlan{}, false
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestPlanEnumeratesRealization(t *testing.T) {
	r := Plan(base())
	cases := map[string]string{
		"s1": "executable",
		"s2": "manual",
		"s3": "external",
		"s4": "unresolved",
		"s5": "unresolved",
	}
	for id, want := range cases {
		if got := stepStatus(r, id); got != want {
			t.Errorf("%s status = %q, want %q", id, got, want)
		}
	}
	if r.Summary.Executable != 1 || r.Summary.Manual != 1 || r.Summary.External != 1 || r.Summary.Unresolved != 2 {
		t.Errorf("summary = %+v", r.Summary)
	}
}

func TestScreenBindingKeepsMeaning(t *testing.T) {
	in := Input{
		Steps: []Step{{ID: "s", Action: "review", Nature: "manual"}},
		Bindings: []Binding{
			{Step: "s", Target: "screen", Ref: "Screen.Review"},
			{Step: "s", Target: "explain", Ref: "doc:review"},
		},
	}
	r := Plan(in)
	p, ok := stepPlan(r, "s")
	if !ok {
		t.Fatal("step missing")
	}
	if !p.MeaningUnchanged {
		t.Fatal("a screen/explain-only expansion changed the business meaning")
	}
	if ready, _, _ := optionReady(r, "s", "screen"); !ready {
		t.Fatal("the human work list was not available")
	}
	if p.Status != "manual" {
		t.Errorf("status = %q, want manual", p.Status)
	}
}

func TestUndeterminedConditionBlocksAutoNotExplanation(t *testing.T) {
	r := Plan(base())
	ready, missing, ok := optionReady(r, "s4", "function")
	if !ok {
		t.Fatal("function option missing")
	}
	if ready {
		t.Fatal("an undetermined condition did not block the function")
	}
	if !contains(missing, "undetermined-condition") {
		t.Errorf("missing = %+v", missing)
	}
	if eReady, _, _ := optionReady(r, "s4", "explain"); !eReady {
		t.Fatal("the explanation was blocked by an undetermined execution condition")
	}
	if !hasFinding(r.Findings, "undetermined-condition") {
		t.Errorf("missing undetermined-condition in %+v", r.Findings)
	}
	if stepStatus(r, "s4") != "unresolved" {
		t.Errorf("s4 status = %q, want unresolved", stepStatus(r, "s4"))
	}
}

func TestAuthorityNotInProcedureRejected(t *testing.T) {
	in := Input{
		Steps:    []Step{{ID: "s", Action: "approve", Nature: "machine", Authority: "perm:approve"}},
		Bindings: []Binding{{Step: "s", Target: "function", Ref: "Fn.Approve", Authority: "perm:admin"}},
	}
	r := Plan(in)
	ready, missing, _ := optionReady(r, "s", "function")
	if ready {
		t.Fatal("an authority absent from the procedure was accepted")
	}
	if !contains(missing, "authority-not-in-procedure") {
		t.Errorf("missing = %+v", missing)
	}
	if !hasFinding(r.Findings, "authority-not-in-procedure") {
		t.Errorf("missing authority-not-in-procedure in %+v", r.Findings)
	}
}

func TestResultUnknownBlocksAuto(t *testing.T) {
	in := Input{
		Steps:    []Step{{ID: "s", Action: "charge", Nature: "machine", ResultUnknown: true}},
		Bindings: []Binding{{Step: "s", Target: "function", Ref: "Fn.Charge"}},
	}
	r := Plan(in)
	ready, missing, _ := optionReady(r, "s", "function")
	if ready {
		t.Fatal("a result-unknown action was allowed to run automatically")
	}
	if !contains(missing, "result-unknown") {
		t.Errorf("missing = %+v", missing)
	}
	if stepStatus(r, "s") != "unresolved" {
		t.Errorf("status = %q, want unresolved", stepStatus(r, "s"))
	}
}

func TestExternalNeedsDeclaredEffect(t *testing.T) {
	in := Input{
		Steps:    []Step{{ID: "s", Action: "notify", Nature: "external"}},
		Bindings: []Binding{{Step: "s", Target: "external", Ref: "Ext.Carrier"}},
	}
	r := Plan(in)
	ready, missing, _ := optionReady(r, "s", "external")
	if ready {
		t.Fatal("an external call without a declared effect was enabled")
	}
	if !contains(missing, "external-effect-not-declared") {
		t.Errorf("missing = %+v", missing)
	}
}

func TestAiIsNotExecutableWithoutAcceptance(t *testing.T) {
	r := Plan(base())
	ready, missing, _ := optionReady(r, "s5", "ai")
	if ready {
		t.Fatal("an AI target was executable without acceptance")
	}
	if !contains(missing, "requires-acceptance") {
		t.Errorf("missing = %+v", missing)
	}
}

func TestCompareShowsAffectedAssets(t *testing.T) {
	before := base()
	after := base()
	// The approve step loses its machine nature, so its function binding is no longer ready.
	for i := range after.Steps {
		if after.Steps[i].ID == "s1" {
			after.Steps[i].Nature = "manual"
		}
	}
	cmp := Compare(before, after)
	if len(cmp.ChangedSteps) != 1 || cmp.ChangedSteps[0] != "s1" {
		t.Errorf("changedSteps = %+v", cmp.ChangedSteps)
	}
	found := false
	for _, ref := range cmp.AffectedAssets {
		if ref == "Fn.AcceptOrder" {
			found = true
		}
	}
	if !found {
		t.Errorf("affected assets = %+v", cmp.AffectedAssets)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
