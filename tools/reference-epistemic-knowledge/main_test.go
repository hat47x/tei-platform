package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) (Corpus, ContextQuery) {
	t.Helper()
	corpusData, err := os.ReadFile(filepath.Join("testdata", "epistemic-fixture-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	contextData, err := os.ReadFile(filepath.Join("testdata", "context-current.json"))
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := decodeStrict[Corpus](corpusData)
	if err != nil {
		t.Fatal(err)
	}
	context, err := decodeStrict[ContextQuery](contextData)
	if err != nil {
		t.Fatal(err)
	}
	return corpus, context
}

func assessmentByID(t *testing.T, projection Projection, id string) Assessment {
	t.Helper()
	for _, assessment := range projection.Assessments {
		if assessment.AssertionID == id {
			return assessment
		}
	}
	t.Fatalf("assessment %s not found", id)
	return Assessment{}
}

func hasReason(assessment Assessment, code string) bool {
	for _, reason := range assessment.Reasons {
		if reason == code {
			return true
		}
	}
	return false
}

func TestReferenceSeparatesOriginReviewContextFreshnessAndConflict(t *testing.T) {
	corpus, context := loadFixture(t)
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}

	user := assessmentByID(t, projection, "user-current")
	if user.UseState != "premise" || user.ConfirmationState != "unreviewed" ||
		user.ReviewBinding != "none" || user.ContextCompatibility != "compatible" ||
		!hasReason(user, "DIRECT_USER_WORKING_PREMISE") {
		t.Fatalf("user current-context fact was not preserved as an unconfirmed scoped working premise: %+v", user)
	}

	ai := assessmentByID(t, projection, "ai-unreviewed")
	if ai.UseState != "candidate-only" || ai.ConfirmationState != "unreviewed" ||
		!hasReason(ai, "ASSERTION_UNREVIEWED") {
		t.Fatalf("AI assertion was silently promoted: %+v", ai)
	}

	conditional := assessmentByID(t, projection, "conditional-production")
	if conditional.UseState != "premise" || conditional.ConfirmationState != "confirmed" ||
		conditional.ReviewBinding != "current" || len(conditional.ReviewConditions) != 1 ||
		conditional.ContextCompatibility != "compatible" {
		t.Fatalf("conditional reviewed fact was not usable in matching scope: %+v", conditional)
	}

	stale := assessmentByID(t, projection, "stale-critical")
	if stale.UseState != "blocked" || stale.Freshness != "stale" ||
		!hasReason(stale, "ASSERTION_STALE") {
		t.Fatalf("stale assertion was not blocked: %+v", stale)
	}

	superseded := assessmentByID(t, projection, "superseded-design")
	if superseded.UseState != "blocked" || superseded.LifecycleState != "superseded" ||
		superseded.ConfirmationState != "confirmed" ||
		len(superseded.SupersededBy) != 1 || superseded.SupersededBy[0] != "superseding-design" ||
		!hasReason(superseded, "SUPERSEDED_BY_RELATION") {
		t.Fatalf("superseded assertion remained usable or lost relation provenance: %+v", superseded)
	}

	rejected := assessmentByID(t, projection, "rejected-claim")
	if rejected.UseState != "blocked" || rejected.ConfirmationState != "rejected" ||
		rejected.LatestReviewEventKind != ReviewEventReject {
		t.Fatalf("rejected assertion remained usable: %+v", rejected)
	}

	for _, id := range []string{"conflict-a", "conflict-b"} {
		conflict := assessmentByID(t, projection, id)
		if conflict.Conflict != "present" || conflict.UseState != "review-required" ||
			!hasReason(conflict, "CONFLICT_PRESENT") {
			t.Fatalf("reviewed conflict was collapsed for %s: %+v", id, conflict)
		}
	}

	inferred := assessmentByID(t, projection, "human-reviewed")
	if !inferred.InferredContextUsed || !hasReason(inferred, "INFERRED_CONTEXT_USED") {
		t.Fatalf("inferred context became indistinguishable from explicit scope: %+v", inferred)
	}
}

func TestNarrowContextDoesNotGeneralizeAcrossProject(t *testing.T) {
	corpus, _ := loadFixture(t)
	contextData, err := os.ReadFile(filepath.Join("testdata", "context-other-project.json"))
	if err != nil {
		t.Fatal(err)
	}
	context, err := decodeStrict[ContextQuery](contextData)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	user := assessmentByID(t, projection, "user-current")
	if user.ContextCompatibility != "incompatible" || user.UseState != "blocked" ||
		!hasReason(user, "CONTEXT_INCOMPATIBLE") {
		t.Fatalf("project-scoped user fact generalized to another project: %+v", user)
	}
}

func TestSemanticEditInvalidatesReviewBindingWithoutChangingOrigin(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "human-reviewed" {
			corpus.Assertions[i].MeaningFingerprint =
				"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}
	}
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	edited := assessmentByID(t, projection, "human-reviewed")
	if edited.ConfirmationState != "unreviewed" || edited.ReviewBinding != "stale" ||
		edited.UseState != "review-required" ||
		!hasReason(edited, "REVIEW_BINDING_STALE") {
		t.Fatalf("semantic edit inherited old review: %+v", edited)
	}
}

func TestKnowledgeHealthWeightsCriticalDependenciesInsteadOfReviewRatio(t *testing.T) {
	corpus, context := loadFixture(t)

	for i := 0; i < 40; i++ {
		id := fmt.Sprintf("low-memo-%02d", i)
		corpus.Assertions = append(corpus.Assertions, Assertion{
			ID:                 id,
			Target:             TargetRef{Kind: "memo", ID: id},
			Text:               "low importance unreviewed memo",
			MeaningFingerprint: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Origin:             OriginAI,
			IngestedBy:         IngestedByAI,
			Mode:               ModeFact,
			Context: []ScopeBinding{{
				Dimension: "project", Value: "tei", Provenance: ScopeDeclared,
			}},
		})
	}
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Health.Level != "critical" {
		t.Fatalf("critical dependency was diluted by low-importance volume: %+v", projection.Health)
	}
	if len(projection.Health.Issues) == 0 ||
		projection.Health.Issues[0].AssertionID != "ai-unreviewed" ||
		projection.Health.Issues[0].Criticality != CriticalityCritical {
		t.Fatalf("critical dependency is not the leading health issue: %+v", projection.Health.Issues)
	}
}

func TestReviewRequestPreservesImpactAndDoesNotGrantAuthority(t *testing.T) {
	corpus, context := loadFixture(t)
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	var found *ReviewRequest
	for i := range projection.ReviewRequests {
		if projection.ReviewRequests[i].AssertionID == "ai-unreviewed" {
			found = &projection.ReviewRequests[i]
			break
		}
	}
	if found == nil {
		t.Fatal("critical unreviewed assertion did not become review request")
	}
	if len(found.DownstreamImpact) != 1 ||
		!strings.Contains(found.DownstreamImpact[0], "release-decision") ||
		len(found.Evidence) != 1 ||
		found.Evidence[0].Ref != "conversation://tei/release-assumption" ||
		found.AuthorityLimit != "reviewer-candidate-is-not-review-authority" {
		t.Fatalf("review request lost impact, evidence, or authority boundary: %+v", *found)
	}
}

func TestProjectionIsDeterministic(t *testing.T) {
	corpus, context := loadFixture(t)
	first, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	second, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("projection changed across identical evaluation:\nfirst=%+v\nsecond=%+v", first, second)
	}
}

func TestCLIEmitsMachineReadableProjection(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"-corpus", filepath.Join("testdata", "epistemic-fixture-v0.json"),
			"-context", filepath.Join("testdata", "context-current.json"),
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var projection Projection
	if err := json.Unmarshal(stdout.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	if projection.Schema != projectionSchema || projection.Health.Level != "critical" {
		t.Fatalf("unexpected projection: %+v", projection)
	}
}

func TestAIOriginIngestedByUserDoesNotBecomeDirectUserPremise(t *testing.T) {
	corpus, context := loadFixture(t)
	corpus.Assertions = append(corpus.Assertions, Assertion{
		ID:                 "pasted-ai-without-review",
		Target:             TargetRef{Kind: "claim", ID: "pasted-ai-without-review"},
		Text:               "AI output pasted by the user remains AI-origin content.",
		MeaningFingerprint: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Origin:             OriginAI,
		IngestedBy:         IngestedByUser,
		Mode:               ModeFact,
		Context: []ScopeBinding{{
			Dimension: "project", Value: "tei", Provenance: ScopeDeclared,
		}},
	})
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "pasted-ai-without-review")
	if got.UseState != "candidate-only" || got.ConfirmationState != "unreviewed" ||
		hasReason(got, "DIRECT_USER_WORKING_PREMISE") {
		t.Fatalf("AI-origin content ingested by user became a direct-user premise: %+v", got)
	}
}

func TestMissingContextDimensionDoesNotGeneralize(t *testing.T) {
	corpus, context := loadFixture(t)
	context.Bindings = []ScopeBinding{{
		Dimension: "project",
		Value: "tei",
		Provenance: ScopeDeclared,
	}}

	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	conditional := assessmentByID(t, projection, "conditional-production")
	if conditional.ContextCompatibility != "unknown" ||
		conditional.UseState != "blocked" ||
		!hasReason(conditional, "CONTEXT_COMPATIBILITY_UNKNOWN") {
		t.Fatalf("missing environment scope was generalized silently: %+v", conditional)
	}
}

func TestReviewEventRequiresHumanReviewMethod(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "human-reviewed" && len(corpus.Assertions[i].ReviewEvents) > 0 {
			corpus.Assertions[i].ReviewEvents[0].Method = "ai-self-review"
		}
	}
	if _, err := evaluateCorpus(corpus, context); err == nil ||
		!strings.Contains(err.Error(), "human-review") {
		t.Fatalf("non-human review method was accepted: %v", err)
	}
}

func TestProjectionStatesMeaningFingerprintIsNotSemanticProof(t *testing.T) {
	corpus, context := loadFixture(t)
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, limit := range projection.AuthorityLimits {
		if limit == "meaning-fingerprint-is-input-not-semantic-proof" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("meaning fingerprint authority limit is missing: %+v", projection.AuthorityLimits)
	}
}


func TestUserDecisionAndConstraintCanBeWorkingPremisesWithoutBecomingCanonicalAuthority(t *testing.T) {
	corpus, context := loadFixture(t)
	for _, tc := range []struct {
		id   string
		mode AssertionMode
		text string
	}{
		{id: "user-decision", mode: ModeDecision, text: "Use the plugin-first path for this work."},
		{id: "user-constraint", mode: ModeConstraint, text: "Do not add a Core fixed enum."},
	} {
		corpus.Assertions = append(corpus.Assertions, Assertion{
			ID:                 tc.id,
			Target:             TargetRef{Kind: "working-premise", ID: tc.id},
			Text:               tc.text,
			MeaningFingerprint: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			Origin:             OriginUser,
			IngestedBy:         IngestedByUser,
			Mode:               tc.mode,
			Context: []ScopeBinding{{
				Dimension: "project", Value: "tei", Provenance: ScopeDeclared,
			}},
		})
	}
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"user-decision", "user-constraint"} {
		got := assessmentByID(t, projection, id)
		if got.UseState != "premise" || got.ContentOrigin != OriginUser || got.IngestedBy != IngestedByUser {
			t.Fatalf("%s was not preserved as a scoped working premise: %+v", id, got)
		}
	}
}

func TestUserPastedAIContentRetainsAIProvenance(t *testing.T) {
	corpus, context := loadFixture(t)
	corpus.Assertions = append(corpus.Assertions, Assertion{
		ID:                 "pasted-ai-content",
		Target:             TargetRef{Kind: "quoted-ai-output", ID: "pasted-ai-content"},
		Text:               "This sentence was generated by AI and pasted by the user.",
		MeaningFingerprint: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		Origin:             OriginAI,
		IngestedBy:         IngestedByUser,
		Mode:               ModeFact,
		Context: []ScopeBinding{{
			Dimension: "project", Value: "tei", Provenance: ScopeDeclared,
		}},
	})
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "pasted-ai-content")
	if got.ContentOrigin != OriginAI || got.IngestedBy != IngestedByUser ||
		got.ConfirmationState != "unreviewed" || got.UseState != "candidate-only" {
		t.Fatalf("pasted AI content lost source provenance or was promoted: %+v", got)
	}
}

func TestContextProvenanceDistinguishesObservedDeclaredAndInferred(t *testing.T) {
	corpus, context := loadFixture(t)
	if context.Bindings[0].Provenance != ScopeObserved {
		t.Fatalf("current runtime context should be observed: %+v", context.Bindings)
	}
	foundDeclared := false
	foundInferred := false
	for _, assertion := range corpus.Assertions {
		for _, binding := range assertion.Context {
			switch binding.Provenance {
			case ScopeDeclared:
				foundDeclared = true
			case ScopeInferred:
				foundInferred = true
			}
		}
	}
	if !foundDeclared || !foundInferred {
		t.Fatalf("fixture must preserve declared and inferred context provenance")
	}
}

func TestReviewHistoryDerivesCurrentConfirmationWithoutOverwritingEvents(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		assertion := &corpus.Assertions[i]
		assertion.ReviewEvents = append(assertion.ReviewEvents,
			ReviewEvent{
				Sequence: 2,
				ID: "reject-main-branch",
				Kind: ReviewEventReject,
				Reviewer: "reviewer-b",
				Method: reviewMethodHuman,
				MeaningFingerprint: assertion.MeaningFingerprint,
				OccurredAt: "2026-10-06T13:00:00Z",
			},
			ReviewEvent{
				Sequence: 3,
				ID: "reconfirm-main-branch",
				Kind: ReviewEventConfirm,
				Reviewer: "reviewer-c",
				Method: reviewMethodHuman,
				MeaningFingerprint: assertion.MeaningFingerprint,
				OccurredAt: "2026-10-06T14:00:00Z",
			},
		)
	}
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "human-reviewed")
	if got.ConfirmationState != "confirmed" ||
		got.ReviewEventCount != 3 ||
		got.LatestReviewEventID != "reconfirm-main-branch" ||
		got.LatestReviewEventKind != ReviewEventConfirm ||
		got.UseState != "premise" {
		t.Fatalf("current confirmation was not derived from full review history: %+v", got)
	}
}

func TestConfirmationWithdrawalPreservesHistoryAndReturnsExternalFactToCandidate(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		assertion := &corpus.Assertions[i]
		assertion.ReviewEvents = append(assertion.ReviewEvents, ReviewEvent{
			Sequence: 2,
			ID: "withdraw-main-branch-confirmation",
			Kind: ReviewEventWithdraw,
			Reviewer: "reviewer-b",
			Method: reviewMethodHuman,
			MeaningFingerprint: assertion.MeaningFingerprint,
			OccurredAt: "2026-10-06T13:00:00Z",
		})
	}
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "human-reviewed")
	if got.ConfirmationState != "unreviewed" ||
		got.ReviewBinding != "current" ||
		got.LatestReviewEventKind != ReviewEventWithdraw ||
		got.ReviewEventCount != 2 ||
		got.UseState != "candidate-only" ||
		!hasReason(got, "CONFIRMATION_WITHDRAWN") {
		t.Fatalf("confirmation withdrawal was not preserved as history: %+v", got)
	}
}

func TestReviewHistoryRejectsDuplicateSequence(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		assertion := &corpus.Assertions[i]
		assertion.ReviewEvents = append(assertion.ReviewEvents, ReviewEvent{
			Sequence: 1,
			ID: "duplicate-sequence",
			Kind: ReviewEventReject,
			Reviewer: "reviewer-b",
			Method: reviewMethodHuman,
			MeaningFingerprint: assertion.MeaningFingerprint,
			OccurredAt: "2026-10-06T13:00:00Z",
		})
	}
	if _, err := evaluateCorpus(corpus, context); err == nil ||
		!strings.Contains(err.Error(), "duplicate review event sequence") {
		t.Fatalf("duplicate review event sequence was accepted: %v", err)
	}
}

func TestDirectUserPremiseIsNotMisreportedAsConfirmed(t *testing.T) {
	corpus, context := loadFixture(t)
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "user-current")
	if got.UseState != "premise" ||
		got.ConfirmationState != "unreviewed" ||
		got.ReviewEventCount != 0 ||
		!hasReason(got, "DIRECT_USER_WORKING_PREMISE") {
		t.Fatalf("direct user premise was conflated with human confirmation: %+v", got)
	}
}

func TestReviewHistoryRejectsBackwardChronology(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		assertion := &corpus.Assertions[i]
		assertion.ReviewEvents = append(assertion.ReviewEvents, ReviewEvent{
			Sequence:           2,
			ID:                 "backdated-reject",
			Kind:               ReviewEventReject,
			Reviewer:           "reviewer-b",
			Method:             reviewMethodHuman,
			MeaningFingerprint: assertion.MeaningFingerprint,
			OccurredAt:         "2026-10-06T11:00:00Z",
		})
	}
	if _, err := evaluateCorpus(corpus, context); err == nil ||
		!strings.Contains(err.Error(), "must not move backward") {
		t.Fatalf("backward review chronology was accepted: %v", err)
	}
}
