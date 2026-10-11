package exampleslide

import (
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
