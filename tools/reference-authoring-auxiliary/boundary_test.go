package auxiliary

import "testing"

func TestPresentationClaimThatAffectsExecutionIsReported(t *testing.T) {
	r := Analyze(Input{Candidates: []Candidate{
		{ID: "Diagram.Order.position", AffectsExecution: true, PresentationOnly: true},
	}})
	if r.Placements[0].Placement != MeaningCanonical {
		t.Fatalf("placement = %q, want meaning-canonical", r.Placements[0].Placement)
	}
	if !hasFinding(r.Findings, "claim-contradicts-execution") {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestDerivedAndLocalClaimsThatAffectExecutionAreReported(t *testing.T) {
	r := Analyze(Input{Candidates: []Candidate{
		{ID: "a", AffectsExecution: true, RecomputableFromCanonical: true},
		{ID: "b", AffectsExecution: true, PerViewer: true},
	}})
	if !hasFinding(r.Findings, "claim-contradicts-execution") {
		t.Fatalf("findings = %+v", r.Findings)
	}
	seen := map[string]bool{}
	for _, f := range r.Findings {
		if f.Code == "claim-contradicts-execution" {
			seen[f.Subject] = true
		}
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("both contradictions must be reported: %+v", r.Findings)
	}
}

func TestNonContradictoryDisplayClaimHasNoFinding(t *testing.T) {
	r := Analyze(Input{Candidates: []Candidate{{ID: "x", PresentationOnly: true}}})
	if hasFinding(r.Findings, "claim-contradicts-execution") {
		t.Fatalf("findings = %+v", r.Findings)
	}
	if r.Placements[0].Placement != DisplayInfo {
		t.Fatalf("placement = %q, want display-info", r.Placements[0].Placement)
	}
}

func TestMeaningVersionChangesWithCanonicalContent(t *testing.T) {
	base := Analyze(Input{Candidates: []Candidate{
		{ID: "Data.Order.total", Content: "sum(lines)", AffectsExecution: true},
	}}).MeaningVersion
	changed := Analyze(Input{Candidates: []Candidate{
		{ID: "Data.Order.total", Content: "sum(lines) - discount", AffectsExecution: true},
	}}).MeaningVersion
	if base == changed {
		t.Fatal("changing canonical content must move the meaning version")
	}
}

func TestMeaningVersionChangesWithAcceptedDefinition(t *testing.T) {
	base := Analyze(Input{Terms: []Term{
		{ID: "t1", Symbol: "Order", Definition: "a purchase request", Accepted: true, Basis: "req#1"},
	}}).MeaningVersion
	changed := Analyze(Input{Terms: []Term{
		{ID: "t1", Symbol: "Order", Definition: "a confirmed purchase request", Accepted: true, Basis: "req#1"},
	}}).MeaningVersion
	if base == changed {
		t.Fatal("changing an accepted definition must move the meaning version")
	}
}

func TestMeaningVersionIgnoresBasisAndUnacceptedDefinition(t *testing.T) {
	base := Analyze(Input{Terms: []Term{
		{ID: "accepted", Symbol: "Order", Definition: "a purchase request", Accepted: true, Basis: "req#1"},
		{ID: "guessed", Symbol: "Zone", Definition: "guess A", Accepted: false},
	}}).MeaningVersion

	basisChanged := Analyze(Input{Terms: []Term{
		{ID: "accepted", Symbol: "Order", Definition: "a purchase request", Accepted: true, Basis: "req#2"},
		{ID: "guessed", Symbol: "Zone", Definition: "guess A", Accepted: false},
	}}).MeaningVersion
	if basisChanged != base {
		t.Fatal("changing the evidence basis must not move the meaning version")
	}

	guessChanged := Analyze(Input{Terms: []Term{
		{ID: "accepted", Symbol: "Order", Definition: "a purchase request", Accepted: true, Basis: "req#1"},
		{ID: "guessed", Symbol: "Zone", Definition: "guess B", Accepted: false},
	}}).MeaningVersion
	if guessChanged != base {
		t.Fatal("an unaccepted definition must not move the meaning version")
	}
}
