package procedureform

import "testing"

func sample() Input {
	return Input{
		Text: SourceText{
			ID:       "proc-1",
			Revision: "r1",
			Segments: []Segment{
				{ID: "seg1", Text: "注文を受けたら担当者が内容を確認する。"},
				{ID: "seg2", Text: "高額な注文は上長の承認後に出荷する。問題があれば差し戻す。"},
				{ID: "seg3", Text: "外部システムへ連携する。"},
			},
		},
		Candidates: []Candidate{
			{ID: "c1", Kind: KindAction, Segment: "seg1", Subject: "order", Value: "確認", Stated: true},
			{ID: "c2", Kind: KindActor, Segment: "seg1", Subject: "order", Value: "担当者", Stated: true},
			{ID: "c3", Kind: KindAction, Segment: "seg2", Subject: "high-value", Value: "出荷", Stated: true},
			{ID: "c4", Kind: KindActor, Segment: "seg2", Subject: "high-value", Value: "上長", Stated: true},
			{ID: "c5", Kind: KindCondition, Segment: "seg2", Subject: "high-value", Value: "", Stated: true},
			{ID: "c6", Kind: KindFunction, Segment: "seg3", Subject: "external", Value: "sendOrder", Stated: true},
			{ID: "c7", Kind: KindActor, Segment: "seg1", Subject: "order", Value: "多分営業担当", Stated: false},
			{ID: "c8", Kind: KindCondition, Segment: "seg2", Subject: "high-value", Value: ">=500000", Stated: true},
			{ID: "c9", Kind: KindCondition, Segment: "seg2", Subject: "high-value", Value: ">=1000000", Stated: true},
			{ID: "c10", Kind: KindAction, Segment: "seg3", Subject: "external", Value: "連携", Stated: true},
		},
		KnownFunctions: []string{"postJournal"},
	}
}

func TestAnalyzeSurfacesAmbiguityAndContradiction(t *testing.T) {
	r := Analyze(sample())
	for _, code := range []string{"missing-condition", "missing-actor", "contradictory-candidates", "unknown-function-reference"} {
		if !hasQuestion(r.Questions, code) {
			t.Errorf("missing question %q in %+v", code, r.Questions)
		}
	}
	// A missing value is never filled in.
	if !hasFinding(r.Findings, "unresolved-questions") {
		t.Errorf("missing unresolved-questions in %+v", r.Findings)
	}
}

func TestAnalyzeAcceptsOnlyStatedIndependentCandidates(t *testing.T) {
	r := Analyze(sample())
	got := map[string]bool{}
	for _, a := range r.Accepted {
		got[a.Candidate] = true
		if a.Provenance != "stated" {
			t.Errorf("accepted %q with provenance %q", a.Candidate, a.Provenance)
		}
	}
	for _, want := range []string{"c1", "c2", "c4"} {
		if !got[want] {
			t.Errorf("expected %q accepted, got %+v", want, r.Accepted)
		}
	}
	reasons := map[string]string{}
	for _, b := range r.Blocked {
		reasons[b.Candidate] = b.Reason
	}
	if reasons["c3"] != "blocked-by-question" {
		t.Errorf("c3 reason = %q", reasons["c3"])
	}
	if reasons["c7"] != "inferred-not-accepted" {
		t.Errorf("c7 reason = %q", reasons["c7"])
	}
	if reasons["c6"] != "blocked-by-question" {
		t.Errorf("c6 reason = %q", reasons["c6"])
	}
}

func TestAnswerSettlesAQuestionAndUnblocksDependents(t *testing.T) {
	in := sample()
	in.Answers = []Answer{{
		ID:       "a1",
		Question: "missing-condition:seg2:high-value",
		Segment:  "seg2",
		Revision: "r1",
		Value:    "500000円以上",
		By:       "owner",
	}}
	r := Analyze(in)
	if !hasAnswerState(r.AnswerStates, "a1", "applicable") {
		t.Fatalf("answer state = %+v", r.AnswerStates)
	}
	accepted := map[string]bool{}
	for _, a := range r.Accepted {
		accepted[a.Candidate] = true
	}
	if !accepted["c3"] || !accepted["c5"] {
		t.Fatalf("an answered condition did not unblock its dependents: %+v", r.Accepted)
	}
}

func TestEmptyAnswerDoesNotSettle(t *testing.T) {
	in := sample()
	in.Answers = []Answer{{ID: "a1", Question: "missing-condition:seg2:high-value", Segment: "seg2", Revision: "r1", Value: "  ", By: "owner"}}
	r := Analyze(in)
	for _, a := range r.Accepted {
		if a.Candidate == "c3" {
			t.Fatal("an empty answer settled the question")
		}
	}
}

func TestStaleAnswerNeedsRecheck(t *testing.T) {
	in := sample()
	in.Answers = []Answer{{ID: "a1", Question: "missing-condition:seg2:high-value", Segment: "seg2", Revision: "r0", Value: "500000円以上", By: "owner"}}
	r := Analyze(in)
	if !hasAnswerState(r.AnswerStates, "a1", "stale-revision") {
		t.Fatalf("answer state = %+v", r.AnswerStates)
	}
	if !hasFinding(r.Findings, "answer-needs-recheck:stale-revision") {
		t.Errorf("missing recheck finding in %+v", r.Findings)
	}
	for _, a := range r.Accepted {
		if a.Candidate == "c3" {
			t.Fatal("a stale answer unblocked a dependent")
		}
	}
}

func TestAnswerForMissingSegment(t *testing.T) {
	in := sample()
	in.Answers = []Answer{{ID: "a1", Question: "missing-condition:seg2:high-value", Segment: "gone", Revision: "r1", Value: "500000円以上", By: "owner"}}
	r := Analyze(in)
	if !hasAnswerState(r.AnswerStates, "a1", "source-missing") {
		t.Fatalf("answer state = %+v", r.AnswerStates)
	}
}

func TestCandidateAnchoredToUnknownSegmentIsBlocked(t *testing.T) {
	in := sample()
	in.Candidates = append(in.Candidates, Candidate{ID: "c11", Kind: KindData, Segment: "nope", Subject: "order", Value: "注文", Stated: true})
	r := Analyze(in)
	for _, b := range r.Blocked {
		if b.Candidate == "c11" {
			if b.Reason != "candidate-anchors-unknown-segment" {
				t.Fatalf("c11 reason = %q", b.Reason)
			}
			return
		}
	}
	t.Fatal("c11 was not blocked")
}

func TestAnalyzeIsDeterministic(t *testing.T) {
	a := Analyze(sample())
	b := Analyze(sample())
	if a.Fingerprint != b.Fingerprint {
		t.Fatalf("fingerprint is not reproducible: %s vs %s", a.Fingerprint, b.Fingerprint)
	}
	changed := sample()
	changed.Text.Revision = "r2"
	if Analyze(changed).Fingerprint == a.Fingerprint {
		t.Fatal("a changed revision did not change the fingerprint")
	}
}

func hasQuestion(qs []Question, code string) bool {
	for _, q := range qs {
		if q.Code == code {
			return true
		}
	}
	return false
}

func hasAnswerState(states []AnswerState, answer, state string) bool {
	for _, s := range states {
		if s.Answer == answer && s.State == state {
			return true
		}
	}
	return false
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}
