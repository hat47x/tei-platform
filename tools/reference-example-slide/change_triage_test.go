package exampleslide

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestChangeTriageRoutesCorrectionWithoutPatch(t *testing.T) {
	in, candidate := reviewAnswerFixture(t)
	cases := []struct {
		disposition, route, reason string
	}{
		{"clarify-example", "example-review", "correction-applies-to-illustrative-case-not-canonical-source"},
		{"retain-model", "confirmation-review", "no-canonical-edit-proposed"},
		{"investigate", "investigation", "insufficient-determination"},
		{"propose-model-change", "semantic-change-review", "cause-and-typed-delta-not-qualified"},
	}
	for _, tc := range cases {
		t.Run(tc.disposition, func(t *testing.T) {
			candidate.Answers[0].Disposition = tc.disposition
			triage, err := TriageAnsweredReview(in, candidate)
			if err != nil {
				t.Fatal(err)
			}
			if len(triage.Items) != 1 {
				t.Fatalf("incorrect triage: %+v", triage)
			}
			item := triage.Items[0]
			if item.CandidateReady || item.Route != tc.route || item.Reason != tc.reason {
				t.Fatalf("unqualified source change route: %+v", item)
			}
			if len(item.NextValidation) == 0 || triage.AnswerFingerprint == "" ||
				!strings.HasPrefix(TriageFingerprint(triage), "sha256:") {
				t.Fatalf("triage evidence missing: %+v", triage)
			}
		})
	}
}

func TestExistingRelationKeyCannotBeReplacedByAddRelation(t *testing.T) {
	in, candidate := reviewAnswerFixture(t)
	in.Catalog.Relations[0].Keys = nil
	review, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Issues) != 1 || review.Issues[0].Reason != "relation-keys-not-declared" {
		t.Fatalf("negative control not keyless: %+v", review.Issues)
	}
	candidate.ReviewFingerprint = review.Fingerprint
	candidate.Answers[0].Question.ID = review.Issues[0].ID
	candidate.Answers[0].Disposition = "propose-model-change"
	triage, err := TriageAnsweredReview(in, candidate)
	if err != nil {
		t.Fatal(err)
	}
	item := triage.Items[0]
	if item.CandidateReady ||
		item.Reason != "existing-relation-key-edit-not-supported-by-current-typed-operations" {
		t.Fatalf("unsupported existing relation edit was promoted to Patch: %+v", item)
	}
}

func TestContradictoryExampleStillRequiresCauseDetermination(t *testing.T) {
	in, candidate := reviewAnswerFixture(t)
	in.Example.CompleteTables = nil
	duplicate := copyRecord(in.Example.Base[0])
	duplicate.ID = "another-parent"
	in.Example.Base = append(in.Example.Base, duplicate)
	review, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Issues) != 1 || review.Issues[0].Reason != "multiple-targets-for-one" {
		t.Fatalf("negative control not a contradiction: %+v", review.Issues)
	}
	candidate.ReviewFingerprint = review.Fingerprint
	candidate.Answers[0].Question.ID = review.Issues[0].ID
	candidate.Answers[0].Disposition = "propose-model-change"
	triage, err := TriageAnsweredReview(in, candidate)
	if err != nil {
		t.Fatal(err)
	}
	item := triage.Items[0]
	if item.CandidateReady || item.Reason != "model-versus-example-cause-not-determined" {
		t.Fatalf("model and example causes were conflated: %+v", item)
	}
}

func TestStaleAnswerCannotBeTriaged(t *testing.T) {
	in, a := reviewAnswerFixture(t)
	in.Example.Base[0].Values["number"] = []byte("\"new\"")
	if _, err := TriageAnsweredReview(in, a); err == nil {
		t.Fatal("stale answer review was routed to semantic change")
	}
}

func TestChangeTriageRetainsExactSourceReviewEvidence(t *testing.T) {
	in, a := reviewAnswerFixture(t)
	in.Example.Base = append(in.Example.Base, Record{
		ID: "maybe-parent",
		Table: "parent",
		Values: map[string]json.RawMessage{"number": json.RawMessage("\"A\"")},
	})
	review, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Issues) != 1 || review.Issues[0].Reason != "unresolved-target-key" {
		t.Fatalf("invalid test setup: %+v", review)
	}
	a.ReviewFingerprint = review.Fingerprint
	a.Answers[0].Question.ID = review.Issues[0].ID
	a.Answers[0].Disposition = "propose-model-change"
	handoff, err := BindAnswers(in, a)
	if err != nil {
		t.Fatal(err)
	}
	triage, err := TriageAnsweredReview(in, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(triage.Items) != 1 ||
		triage.Items[0].Route != "semantic-change-review" ||
		triage.Items[0].CandidateReady ||
		triage.Items[0].SourceIssue == nil ||
		!reflect.DeepEqual(*triage.Items[0].SourceIssue, review.Issues[0]) ||
		!reflect.DeepEqual(triage.Items[0].SourceIssue.PossibleTargetIDs, []string{"maybe-parent"}) ||
		!reflect.DeepEqual(triage.Items[0].SourceIssue.MatchedTargetIDs, []string{"a"}) ||
		!reflect.DeepEqual(triage.Items[0].SourceIssue, handoff.Answers[0].SourceIssue) {
		t.Fatalf("triage lost provenance or granted edit authority: %+v", triage)
	}
}

func TestAuthoredChangeTriageDoesNotInventDerivedIssue(t *testing.T) {
	in := fixture(t)
	review, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	a := AnswerInput{
		Schema: answerSchema,
		CaseID: review.CaseID,
		ModelRevision: review.ModelRevision,
		ReviewFingerprint: review.Fingerprint,
		Answers: []AnswerEntry{{
			Question: QuestionRef{Kind: "authored", ID: "residual"},
			Response: "保留", Disposition: "investigate",
			Context: "検収", ClaimedBy: AnswerActor{Kind: "ai", ID: "test"},
		}},
	}
	triage, err := TriageAnsweredReview(in, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(triage.Items) != 1 || triage.Items[0].SourceIssue != nil ||
		triage.Items[0].Route != "investigation" || triage.Items[0].CandidateReady {
		t.Fatalf("fabricated canonical provenance: %+v", triage)
	}
}
