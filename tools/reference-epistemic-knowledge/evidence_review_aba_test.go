package main

import "testing"

// This characterizes the v0 content-derived Review Subject receipt's ABA
// limitation; it does NOT assert uninterrupted Source history or current-use
// Evidence suitability. A version-bound implementation can revise this test.
func TestEvidenceABARestoresReceiptWithoutProvingContinuity(t *testing.T) {
	corpus, context := loadFixture(t)
	const id = "ai-unreviewed"
	prepared := reviewCommandFor(t, corpus, context, id, ReviewEventConfirm, "evidence-aba-old-command")
	assertion := evidenceProbeAssertion(t, &corpus, id)
	if len(assertion.Evidence) == 0 {
		t.Fatal("fixture has no Evidence")
	}
	originalRef := assertion.Evidence[0].Ref
	originalReceipt := reviewSubjectFingerprint(*assertion)

	assertion.Evidence[0].Ref = originalRef + "-v2"
	if reviewSubjectFingerprint(*assertion) == originalReceipt {
		t.Fatal("E1->E2 did not change review subject fingerprint")
	}
	assertion.Evidence[0].Ref = originalRef
	if got := reviewSubjectFingerprint(*assertion); got != originalReceipt {
		t.Fatalf("E1->E2->E1 did not restore content receipt: %s", got)
	}

	// No Review event occurred while Evidence was E2; therefore the current
	// implementation has no monotonic evidence epoch to distinguish the path.
	result, err := applyReviewCommand(corpus, context, prepared)
	if err != nil {
		t.Fatalf("v0 ABA characterization changed: %v", err)
	}
	if result.AppendedEvent.ID != prepared.EventID {
		t.Fatalf("old prepared command not appended: %+v", result.AppendedEvent)
	}
	t.Log("v0 accepted an E1-prepared command after E1->E2->E1; receipt equality alone does not prove continuity")
}
