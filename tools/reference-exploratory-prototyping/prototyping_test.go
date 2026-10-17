package prototyping

import "testing"

func base() Input {
	return Input{
		Profile:         "exploratory",
		PluginAvailable: true,
		Iterations: []Iteration{
			{ID: "it1", Hypothesis: "h1", InputFixture: "fixture/a.csv", Revision: "abc123", Expected: "sum=10", Observed: "sum=10", Observation: "execution", NextDecision: "adopt"},
			{ID: "it2", Hypothesis: "h2", InputFixture: "fixture/b.csv", Revision: "abc124", Expected: "sum=20", Observed: "sum=21", Observation: "execution", NextDecision: "discard"},
			{ID: "it3", Hypothesis: "h3", InputFixture: "fixture/c.csv", Revision: "abc125", Expected: "sum=30", Observed: "sum=30", Observation: "execution", NextDecision: "unresolved"},
		},
		Selections: []Selection{
			{Iteration: "it1", Kind: "data", Ref: "Data.Sum"},
			{Iteration: "it2", Kind: "discard", Ref: "tmp/report"},
			{Iteration: "it3", Kind: "hold", Ref: "Scenario.SumCandidate"},
		},
	}
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func adoptionReason(r Report, kind, ref string) string {
	for _, a := range r.Adoptions {
		if a.Kind == kind && a.Ref == ref {
			return a.Reason
		}
	}
	return ""
}

func TestPlanRequiresPluginAndProfile(t *testing.T) {
	off := base()
	off.PluginAvailable = false
	r := Plan(off)
	if r.Applicable {
		t.Fatal("a missing plugin was reported as applicable")
	}
	if !hasFinding(r.Findings, "plugin-unavailable") {
		t.Errorf("missing plugin-unavailable in %+v", r.Findings)
	}
	if len(r.Adopted) != 0 {
		t.Errorf("adopted without a plugin: %+v", r.Adopted)
	}

	rig := base()
	rig.Profile = "rigorous"
	r2 := Plan(rig)
	if !hasFinding(r2.Findings, "profile-not-exploratory") {
		t.Errorf("missing profile-not-exploratory in %+v", r2.Findings)
	}
	if len(r2.Adopted) != 0 {
		t.Errorf("adopted under a rigorous profile: %+v", r2.Adopted)
	}
}

func TestIterationObservationMustBeMeasured(t *testing.T) {
	in := base()
	in.Iterations[0].Observation = "ai-candidate"
	r := Plan(in)
	if !hasFinding(r.Findings, "observation-not-measured") {
		t.Errorf("missing observation-not-measured in %+v", r.Findings)
	}
	for _, v := range r.Iterations {
		if v.ID == "it1" && v.Measured {
			t.Error("an AI candidate was reported as measured")
		}
	}
}

func TestInsufficientIterations(t *testing.T) {
	in := base()
	in.Iterations = in.Iterations[:1]
	r := Plan(in)
	if !hasFinding(r.Findings, "insufficient-iterations") {
		t.Errorf("missing insufficient-iterations in %+v", r.Findings)
	}
}

func TestSelfDerivedExpectationBlocks(t *testing.T) {
	in := base()
	in.Iterations[0].SelfDerived = true
	r := Plan(in)
	if !hasFinding(r.Findings, "self-derived-expectation") {
		t.Errorf("missing self-derived-expectation in %+v", r.Findings)
	}
	if contains(r.Adopted, "data:Data.Sum") {
		t.Error("a self-derived expectation was adopted")
	}
	if got := adoptionReason(r, "data", "Data.Sum"); got != "self-derived-expectation" {
		t.Errorf("reason = %q", got)
	}
}

func TestMeaningBearingRequiresAcceptedIteration(t *testing.T) {
	in := base()
	in.Iterations[0].NextDecision = "continue"
	in.Selections = append(in.Selections, Selection{Iteration: "it1", Kind: "scenario", Ref: "Scenario.SumTrial"})
	r := Plan(in)
	if contains(r.Adopted, "data:Data.Sum") {
		t.Error("a meaning-bearing unit was adopted from a non-accepted iteration")
	}
	if got := adoptionReason(r, "data", "Data.Sum"); got != "iteration-not-accepted" {
		t.Errorf("reason = %q", got)
	}
	if !contains(r.Adopted, "scenario:Scenario.SumTrial") {
		t.Error("a scenario candidate from an exploratory iteration was not kept")
	}
}

func TestPartialAdoptionRepresentsAllThree(t *testing.T) {
	r := Plan(base())
	if !contains(r.Adopted, "data:Data.Sum") {
		t.Errorf("adopted = %+v", r.Adopted)
	}
	if !contains(r.Discarded, "discard:tmp/report") {
		t.Errorf("discarded = %+v", r.Discarded)
	}
	if !contains(r.Held, "hold:Scenario.SumCandidate") {
		t.Errorf("held = %+v", r.Held)
	}
	if r.Summary.Adopted != 1 || r.Summary.Discarded != 1 || r.Summary.Held != 1 {
		t.Errorf("summary = %+v", r.Summary)
	}
}

func TestNoDiscardOrHoldIsFlagged(t *testing.T) {
	in := base()
	in.Selections = []Selection{{Iteration: "it1", Kind: "data", Ref: "Data.Sum"}}
	r := Plan(in)
	if !hasFinding(r.Findings, "no-discard-or-hold") {
		t.Errorf("missing no-discard-or-hold in %+v", r.Findings)
	}
}

func TestCompareEffortRecordsUnmeasured(t *testing.T) {
	rigorous := Effort{Revisions: 4, ManualInputs: 6, ReusableAssets: 2, DefectsFound: 1, Measured: true}
	exploratory := Effort{Revisions: 9, ManualInputs: 2, ReusableAssets: 5, DefectsFound: 3, Measured: false}
	c := CompareEffort(rigorous, exploratory)
	if c.Measured {
		t.Error("an unmeasured side was compared as measured")
	}
	if c.Note == "" {
		t.Error("an unmeasured comparison carried no note")
	}
	exploratory.Measured = true
	c2 := CompareEffort(rigorous, exploratory)
	if !c2.Measured || c2.Note != "" {
		t.Errorf("measured comparison = %+v", c2)
	}
}
