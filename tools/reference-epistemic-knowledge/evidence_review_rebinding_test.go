package main

import (
	"strings"
	"testing"
)

// These tests characterize the existing Reference boundary. They deliberately
// do not prescribe whether a particular Consumer requires the current Evidence:
// a historical human review and current-use evidence qualification are separate.
func TestEvidenceReplacementAfterReviewKeepsHistoryAndInvalidatesOldCommandReceipt(t *testing.T) {
	corpus, context := loadFixture(t)
	const id = "ai-unreviewed"

	confirm := reviewCommandFor(t, corpus, context, id, ReviewEventConfirm, "evidence-probe-confirm")
	initial := evidenceProbeAssertion(t, &corpus, id)
	if len(initial.Evidence) == 0 {
		t.Fatal("fixture has no Evidence to replace")
	}
	confirm.Evidence = append([]EvidenceRef(nil), initial.Evidence...)

	confirmed, err := applyReviewCommand(corpus, context, confirm)
	if err != nil {
		t.Fatal(err)
	}
	before := assessmentByID(t, confirmed.Projection, id)
	if before.ConfirmationState != "confirmed" {
		t.Fatalf("initial review was not confirmed: %+v", before)
	}

	// A command prepared against the old subject must be rejected after E1->E2.
	oldCommand := reviewCommandFor(t, confirmed.UpdatedCorpus, context, id, ReviewEventConfirm, "stale-evidence-command")
	current := confirmed.UpdatedCorpus
	assertion := evidenceProbeAssertion(t, &current, id)
	originalRef := assertion.Evidence[0].Ref
	originalSubject := reviewSubjectFingerprint(*assertion)
	if len(assertion.ReviewEvents) != 1 || len(assertion.ReviewEvents[0].Evidence) != 1 {
		t.Fatalf("review history did not capture explicit Evidence: %+v", assertion.ReviewEvents)
	}
	assertion.Evidence[0].Ref = originalRef + "-v2"
	if reviewSubjectFingerprint(*assertion) == originalSubject {
		t.Fatal("material Evidence replacement did not change review subject fingerprint")
	}
	if assertion.ReviewEvents[0].Evidence[0].Ref != originalRef {
		t.Fatal("current Evidence replacement retroactively rewrote review history")
	}

	if _, err := applyReviewCommand(current, context, oldCommand); err == nil ||
		!strings.Contains(err.Error(), "subject binding is stale") {
		t.Fatalf("old command receipt accepted after Evidence changed: %v", err)
	}
	projection, err := evaluateCorpus(current, context)
	if err != nil {
		t.Fatal(err)
	}
	after := assessmentByID(t, projection, id)
	if after.ReviewSubjectFingerprint != reviewSubjectFingerprint(*assertion) {
		t.Fatal("Projection did not report the current Evidence-bound review subject")
	}
	if after.MeaningFingerprint != before.MeaningFingerprint ||
		after.ReviewEventCount != before.ReviewEventCount {
		t.Fatalf("Evidence replacement changed meaning or review event history: before=%+v after=%+v", before, after)
	}

	// Diagnostic observation only, NOT an assertion that the current Evidence
	// suffices for every Consumer or that Owner business acceptance was granted.
	t.Logf("E1->E2 without new review: confirmation=%s reviewBinding=%s useState=%s; old command rejected",
		after.ConfirmationState, after.ReviewBinding, after.UseState)
}

func TestEvidenceWithdrawAndReturnDoesNotResurrectWithdrawnReview(t *testing.T) {
	corpus, context := loadFixture(t)
	const id = "ai-unreviewed"
	confirm := reviewCommandFor(t, corpus, context, id, ReviewEventConfirm, "withdraw-probe-confirm")
	initial := evidenceProbeAssertion(t, &corpus, id)
	if len(initial.Evidence) == 0 {
		t.Fatal("fixture has no Evidence")
	}
	confirm.Evidence = append([]EvidenceRef(nil), initial.Evidence...)
	confirmed, err := applyReviewCommand(corpus, context, confirm)
	if err != nil {
		t.Fatal(err)
	}

	replaced := confirmed.UpdatedCorpus
	assertion := evidenceProbeAssertion(t, &replaced, id)
	originalRef := assertion.Evidence[0].Ref
	assertion.Evidence[0].Ref = originalRef + "-v2"

	withdraw := reviewCommandFor(t, replaced, context, id, ReviewEventWithdraw, "withdraw-probe-e2")
	withdraw.Evidence = append([]EvidenceRef(nil), assertion.Evidence...)
	withdrawn, err := applyReviewCommand(replaced, context, withdraw)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, withdrawn.Projection, id)
	if got.ConfirmationState != "unreviewed" || got.LatestReviewEventKind != ReviewEventWithdraw ||
		got.ReviewEventCount != 2 {
		t.Fatalf("withdraw did not preserve history and remove confirmation: %+v", got)
	}

	returned := withdrawn.UpdatedCorpus
	assertion = evidenceProbeAssertion(t, &returned, id)
	assertion.Evidence[0].Ref = originalRef
	returnedProjection, err := evaluateCorpus(returned, context)
	if err != nil {
		t.Fatal(err)
	}
	again := assessmentByID(t, returnedProjection, id)
	if again.ConfirmationState != "unreviewed" || again.LatestReviewEventKind != ReviewEventWithdraw ||
		again.ReviewEventCount != 2 {
		t.Fatalf("E1->E2->withdraw->E1 incorrectly resurrected historical confirmation: %+v", again)
	}
}

func evidenceProbeAssertion(t *testing.T, corpus *Corpus, id string) *Assertion {
	t.Helper()
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == id {
			return &corpus.Assertions[i]
		}
	}
	t.Fatalf("fixture assertion %q not found", id)
	return nil
}
