package auxiliary

import "testing"

func TestClassifyChecksExecutionBeforeRecompute(t *testing.T) {
	// A recomputable item that still changes execution is part of the meaning.
	// The ordered questions must not put "derived" first.
	got := Classify(Candidate{ID: "a", AffectsExecution: true, RecomputableFromCanonical: true})
	if got.Placement != MeaningCanonical {
		t.Fatalf("placement = %q, want %q", got.Placement, MeaningCanonical)
	}
	if got.Reason == "" {
		t.Fatal("placement decided without a reason")
	}

	cases := []struct {
		name string
		in   Candidate
		want Placement
	}{
		{"derived", Candidate{ID: "b", RecomputableFromCanonical: true}, Derived},
		{"local", Candidate{ID: "c", PerViewer: true}, UserLocal},
		{"display", Candidate{ID: "d", PresentationOnly: true}, DisplayInfo},
		{"operational", Candidate{ID: "e"}, OperationalRecord},
	}
	for _, tc := range cases {
		if got := Classify(tc.in); got.Placement != tc.want || got.Reason == "" {
			t.Errorf("%s: got (%q, %q), want %q with a reason", tc.name, got.Placement, got.Reason, tc.want)
		}
	}
}

func TestListAnnotationsUsesStructuredDeclarations(t *testing.T) {
	items := []Annotation{
		{ID: "n2", Subject: "Data.Customer", Kind: "note", State: "open", Author: "a", Text: "review the key"},
		{ID: "n1", Subject: "Data.Order", Kind: "question", State: "done", Author: "b", Text: "is this optional?"},
	}
	got, findings := ListAnnotations(items)
	if len(findings) != 0 {
		t.Fatalf("unexpected findings: %+v", findings)
	}
	if got[0].ID != "n1" || got[1].ID != "n2" {
		t.Fatalf("annotations are not in a stable order: %+v", got)
	}
}

func TestListAnnotationsRejectsEmbeddedMarker(t *testing.T) {
	items := []Annotation{
		{ID: "n1", Subject: "Data.Order", Kind: "note", State: "open", Author: "a", Text: "note: @NOTE@ check the total"},
	}
	if _, findings := ListAnnotations(items); !hasFinding(findings, "embedded-marker-not-structured") {
		t.Fatalf("embedded marker was accepted: %+v", findings)
	}
}

func TestListAnnotationsReportsMissingSubjectAndUnknownKind(t *testing.T) {
	items := []Annotation{
		{ID: "n1", Kind: "note", State: "open"},
		{ID: "n2", Subject: "Data.Order", Kind: "sms", State: "new"},
	}
	_, findings := ListAnnotations(items)
	for _, code := range []string{"annotation-without-subject", "unknown-annotation-kind", "unknown-annotation-state"} {
		if !hasFinding(findings, code) {
			t.Errorf("missing finding %q in %+v", code, findings)
		}
	}
}

func TestDeriveRevisionsAreFacts(t *testing.T) {
	facts := DeriveRevisions([]RevisionEvent{
		{Revision: "2", At: "t2", Subject: "Data.Order", Change: "added customer"},
		{Revision: "1", At: "t1", Subject: "Data.Order", Change: "initial"},
	})
	if len(facts) != 2 || facts[0].Revision != "1" {
		t.Fatalf("facts not derived in a stable order: %+v", facts)
	}
}

func TestCombineRevisionsKeepsOnlyHumanReasons(t *testing.T) {
	facts := DeriveRevisions([]RevisionEvent{
		{Revision: "1", At: "t1", Subject: "Data.Order", Change: "initial"},
	})
	history, findings := CombineRevisions(facts, []RevisionReason{
		{Revision: "1", Author: "a", Reason: "initial"},  // duplicates the derived fact
		{Revision: "9", Author: "a", Reason: "unknown"},  // no derived fact
		{Revision: "1", Author: "b", Reason: "to align"}, // a real reason
	})
	if !hasFinding(findings, "duplicated-derived-fact") {
		t.Errorf("duplicated derived fact was not reported: %+v", findings)
	}
	if !hasFinding(findings, "reason-without-revision") {
		t.Errorf("reason without a revision was not reported: %+v", findings)
	}
	if len(history.Reasons) != 1 || history.Reasons[0].Author != "b" {
		t.Fatalf("human reasons were not kept alone: %+v", history.Reasons)
	}
	if len(history.Facts) != 1 {
		t.Fatalf("derived facts changed: %+v", history.Facts)
	}
}

func TestTermIndexLeavesGuessedDefinitionsUnresolved(t *testing.T) {
	entries, findings := TermIndex([]Term{
		{ID: "t1", Symbol: "Order", Definition: "a purchase request", Accepted: true, Basis: "requirements#1"},
		{ID: "t2", Symbol: "Customer", Definition: "a buyer"},                   // not accepted
		{ID: "t3", Symbol: "Total", Definition: "sum of lines", Accepted: true}, // accepted without basis
		{ID: "t4", Symbol: "Item"}, // index only
		{ID: "t5", Definition: "no symbol"},
	})
	byID := map[string]TermEntry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	if byID["t1"].Status != "accepted" {
		t.Errorf("t1 = %+v, want accepted", byID["t1"])
	}
	if byID["t2"].Status != "unresolved" || byID["t2"].Definition != "" {
		t.Errorf("t2 = %+v, want unresolved without a definition", byID["t2"])
	}
	if byID["t3"].Status != "unresolved" {
		t.Errorf("t3 = %+v, want unresolved without a basis", byID["t3"])
	}
	if byID["t4"].Status != "index-only" {
		t.Errorf("t4 = %+v, want index-only", byID["t4"])
	}
	for _, code := range []string{"definition-not-accepted", "definition-without-basis", "term-without-symbol"} {
		if !hasFinding(findings, code) {
			t.Errorf("missing finding %q in %+v", code, findings)
		}
	}
}

func TestMeaningVersionIgnoresNonMeaningChanges(t *testing.T) {
	base := Input{
		Candidates: []Candidate{
			{ID: "Data.Order.total", AffectsExecution: true},
			{ID: "Diagram.Order.position", PresentationOnly: true},
		},
		Terms: []Term{
			{ID: "t1", Symbol: "Order", Definition: "a purchase request", Accepted: true, Basis: "requirements#1"},
		},
		Display:     []string{"zone: sales"},
		Operational: []string{"note: reviewed by a"},
		Local:       []string{"bookmark: Order"},
	}
	before := Analyze(base).MeaningVersion

	changed := base
	changed.Display = []string{"zone: sales", "zone: support"}
	changed.Operational = []string{"note: reviewed by b"}
	changed.Local = []string{"bookmark: Order", "bookmark: Customer"}
	if after := Analyze(changed).MeaningVersion; after != before {
		t.Fatalf("non-meaning change moved the meaning version: before=%s after=%s", before, after)
	}

	meaningChanged := base
	meaningChanged.Candidates = []Candidate{
		{ID: "Data.Order.total", AffectsExecution: true},
		{ID: "Data.Order.customer", AffectsExecution: true},
		{ID: "Diagram.Order.position", PresentationOnly: true},
	}
	if after := Analyze(meaningChanged).MeaningVersion; after == before {
		t.Fatalf("meaning change did not move the meaning version: %s", after)
	}
}

func TestAnalyzeIsStableAndReadOnly(t *testing.T) {
	in := Input{
		Candidates: []Candidate{
			{ID: "b", PresentationOnly: true},
			{ID: "a", AffectsExecution: true},
		},
	}
	r := Analyze(in)
	if r.Schema != Schema {
		t.Fatalf("schema = %q", r.Schema)
	}
	if r.Placements[0].ID != "a" || r.Placements[1].ID != "b" {
		t.Fatalf("placements are not in a stable order: %+v", r.Placements)
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
