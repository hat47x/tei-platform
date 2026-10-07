package main

import (
	"reflect"
	"strings"
	"testing"
)

func reviewCommandFor(
	t *testing.T,
	corpus Corpus,
	context ContextQuery,
	assertionID string,
	operation ReviewEventKind,
	eventID string,
) ReviewCommand {
	t.Helper()
	normalized, err := normalizeCorpus(corpus)
	if err != nil {
		t.Fatal(err)
	}
	normalizedContext, _, err := normalizeContext(context)
	if err != nil {
		t.Fatal(err)
	}
	for _, assertion := range normalized.Assertions {
		if assertion.ID != assertionID {
			continue
		}
		lastSequence := 0
		for _, event := range assertion.ReviewEvents {
			if event.Sequence > lastSequence {
				lastSequence = event.Sequence
			}
		}
		return ReviewCommand{
			Contract:                   reviewCommandContract,
			Schema:                     reviewCommandSchema,
			AssertionID:                assertion.ID,
			Operation:                  operation,
			Reviewer:                   "user:reviewer",
			EventID:                    eventID,
			OccurredAt:                 normalizedContext.AsOf,
			ExpectedMeaningFingerprint: assertion.MeaningFingerprint,
			ExpectedReviewSubjectFingerprint: reviewSubjectFingerprint(assertion),
			ExpectedTargetBinding:      targetBinding(assertion.Target, normalizedContext),
			ExpectedLastSequence:       lastSequence,
		}
	}
	t.Fatalf("assertion %s not found", assertionID)
	return ReviewCommand{}
}

func TestReviewCommandConfirmsCurrentMeaningWithoutCanonicalMutation(t *testing.T) {
	corpus, context := loadFixture(t)
	original := structuredCorpusCopyForTest(t, corpus)
	command := reviewCommandFor(t, corpus, context, "ai-unreviewed", ReviewEventConfirm, "review-command-confirm")

	result, err := applyReviewCommand(corpus, context, command)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(corpus, original) {
		t.Fatal("applyReviewCommand mutated caller corpus")
	}
	if result.Contract != reviewResultContract || result.Schema != reviewResultSchema {
		t.Fatalf("unexpected mutation result identity: %+v", result)
	}
	if result.AppendedEvent.Kind != ReviewEventConfirm ||
		result.AppendedEvent.MeaningFingerprint != command.ExpectedMeaningFingerprint ||
		result.AppendedEvent.Sequence != command.ExpectedLastSequence+1 {
		t.Fatalf("unexpected appended event: %+v", result.AppendedEvent)
	}
	got := assessmentByID(t, result.Projection, "ai-unreviewed")
	if got.ConfirmationState != "confirmed" || got.UseState != "premise" {
		t.Fatalf("confirmed current meaning did not become usable premise: %+v", got)
	}
	if got.MeaningFingerprint != command.ExpectedMeaningFingerprint {
		t.Fatalf("projection lost meaning binding: %+v", got)
	}
}

func TestReviewCommandRejectsStaleMeaningBinding(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "ai-unreviewed", ReviewEventConfirm, "stale-meaning")
	command.ExpectedMeaningFingerprint = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "meaning binding is stale") {
		t.Fatalf("stale meaning command was accepted: %v", err)
	}
}

func TestReviewCommandRejectsStaleSequence(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "human-reviewed", ReviewEventConfirm, "stale-sequence")
	command.ExpectedLastSequence = 0

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "sequence is stale") {
		t.Fatalf("stale sequence command was accepted: %v", err)
	}
}

func TestReviewCommandRejectsStaleReviewSubjectBinding(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "user-current", ReviewEventConfirm, "stale-subject")

	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "user-current" {
			corpus.Assertions[i].Mode = ModeHypothesis
		}
	}

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "subject binding is stale") {
		t.Fatalf("stale epistemic subject command was accepted: %v", err)
	}
}

func TestReviewCommandRejectsExactToExactTargetReplacement(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "user-current", ReviewEventConfirm, "replaced-target")
	if command.ExpectedTargetBinding != "not-required" {
		t.Fatalf("unexpected target binding: %+v", command)
	}

	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "user-current" {
			corpus.Assertions[i].Target.ID = "tei.database-lifecycle-replaced"
		}
	}

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "subject binding is stale") {
		t.Fatalf("exact-to-exact target replacement was not detected: %v", err)
	}
}

func TestReviewCommandRejectsChangedTargetBinding(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "anchored-identifier-policy", ReviewEventConfirm, "stale-target")
	if command.ExpectedTargetBinding != "reanchored" {
		t.Fatalf("fixture no longer exercises reanchoring: %+v", command)
	}
	command.ExpectedTargetBinding = "exact"

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "target binding is stale") {
		t.Fatalf("stale target binding was accepted: %v", err)
	}
}

func TestReviewCommandRefusesAmbiguousTargetConfirmation(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range context.TargetSurfaces {
		if context.TargetSurfaces[i].SourceRef != "repo://plan/identifier-policy.md" {
			continue
		}
		context.TargetSurfaces[i].Segments = append(context.TargetSurfaces[i].Segments, TargetSegment{
			StructuralPath: "/archive/naming/paragraph[9]",
			Text:           "Rule: Identifiers may contain hyphens. Applies to application identifiers.",
		})
	}
	command := reviewCommandFor(t, corpus, context, "anchored-identifier-policy", ReviewEventConfirm, "ambiguous-target")
	if command.ExpectedTargetBinding != "ambiguous" {
		t.Fatalf("fixture did not become ambiguous: %+v", command)
	}

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "targetBinding=ambiguous") {
		t.Fatalf("ambiguous target was confirmed: %v", err)
	}
}

func TestReviewCommandRefusesPartialMetadataConfirmation(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "ai-unreviewed" {
			corpus.Assertions[i].MetadataState = MetadataPartial
		}
	}
	command := reviewCommandFor(t, corpus, context, "ai-unreviewed", ReviewEventConfirm, "partial-metadata")

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "metadataState=partial") {
		t.Fatalf("partial metadata was confirmed: %v", err)
	}
}

func TestReviewCommandRejectsEventAfterContextAsOf(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "ai-unreviewed", ReviewEventConfirm, "future-review")
	command.OccurredAt = "2099-01-01T00:00:00Z"

	_, err := applyReviewCommand(corpus, context, command)
	if err == nil || !strings.Contains(err.Error(), "after context asOf") {
		t.Fatalf("future review event was accepted into past projection: %v", err)
	}
}

func TestReviewCommandWithdrawPreservesHistoryAndRemovesCurrentConfirmation(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "human-reviewed", ReviewEventWithdraw, "withdraw-current")

	result, err := applyReviewCommand(corpus, context, command)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, result.Projection, "human-reviewed")
	if got.ConfirmationState != "unreviewed" ||
		got.LatestReviewEventKind != ReviewEventWithdraw ||
		got.ReviewEventCount < 2 {
		t.Fatalf("withdraw did not preserve history and remove confirmation: %+v", got)
	}
}

func TestReviewCommandWithConditionsDoesNotCreateUnconditionalPremise(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "ai-unreviewed", ReviewEventConfirm, "conditional-confirm")
	command.Conditions = []string{"Only after external owner approval"}

	result, err := applyReviewCommand(corpus, context, command)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, result.Projection, "ai-unreviewed")
	if got.ConfirmationState != "confirmed" ||
		got.UseState != "review-required" ||
		!hasReason(got, "REVIEW_CONDITIONS_UNRESOLVED") {
		t.Fatalf("conditional confirmation became unconditional premise: %+v", got)
	}
}

func TestReviewCommandAuthorityLimitsRemainExplicit(t *testing.T) {
	corpus, context := loadFixture(t)
	command := reviewCommandFor(t, corpus, context, "ai-unreviewed", ReviewEventConfirm, "authority-limits")
	result, err := applyReviewCommand(corpus, context, command)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"review-command-does-not-grant-reviewer-authority",
		"review-command-does-not-grant-canonical-acceptance",
		"review-command-does-not-change-canonical-content",
	} {
		if !containsString(result.AuthorityLimits, expected) {
			t.Fatalf("missing mutation authority limit %q: %+v", expected, result.AuthorityLimits)
		}
	}
}

func structuredCorpusCopyForTest(t *testing.T, corpus Corpus) Corpus {
	t.Helper()
	copy, err := cloneCorpus(corpus)
	if err != nil {
		t.Fatal(err)
	}
	return copy
}
