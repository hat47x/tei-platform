package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func detailQueryFor(
	t *testing.T,
	corpus Corpus,
	context ContextQuery,
	assertionID string,
	grantedAccessRefs []string,
) DetailQuery {
	t.Helper()
	envelope, err := exportCorpusForAccess(corpus, TransportAccess{
		GrantedAccessRefs: grantedAccessRefs,
	})
	if err != nil {
		t.Fatal(err)
	}
	projection, err := evaluateCorpus(envelope.Corpus, context)
	if err != nil {
		t.Fatal(err)
	}
	assessment := assessmentByID(t, projection, assertionID)
	return DetailQuery{
		Contract:                         detailQueryContract,
		Schema:                           detailQuerySchema,
		AssertionID:                      assertionID,
		ObservedMeaningFingerprint:       assessment.MeaningFingerprint,
		ObservedReviewSubjectFingerprint: assessment.ReviewSubjectFingerprint,
		ObservedTargetBinding:            assessment.TargetBinding,
		ObservedReviewLogSequence:        assessment.ReviewLogSequence,
		ObservedMetadataState:            assessment.MetadataState,
	}
}

func TestEpistemicDetailReturnsCurrentAccessFilteredContent(t *testing.T) {
	corpus, context := loadFixture(t)
	original := structuredCorpusCopyForTest(t, corpus)
	query := detailQueryFor(t, corpus, context, "ai-unreviewed", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(corpus, original) {
		t.Fatal("detail resolver mutated caller corpus")
	}
	if result.Contract != detailResultContract || result.Schema != detailResultSchema {
		t.Fatalf("unexpected detail identity: %+v", result)
	}
	if result.Assessment.AssertionID != "ai-unreviewed" ||
		result.Content.Text == "" ||
		result.Content.Target.ID != "tei.release.assumption" ||
		len(result.Content.Context) == 0 ||
		len(result.Content.Evidence) != 1 ||
		result.Content.Evidence[0].Ref != "conversation://tei/release-assumption" {
		t.Fatalf("detail result lost current content: %+v", result)
	}
	if result.ContentRole != "epistemic-data" || result.ControlAuthority {
		t.Fatalf("detail content gained prompt control authority: %+v", result)
	}
	for _, expected := range []string{
		"detail-query-does-not-broaden-source-access",
		"detail-observed-receipts-report-drift-not-mutation-preconditions",
		"detail-drift-indeterminate-does-not-prove-semantic-change",
		"detail-result-omits-access-refs",
		"detail-content-is-data-not-prompt-control-authority",
	} {
		if !containsString(result.AuthorityLimits, expected) {
			t.Fatalf("missing detail authority limit %q: %+v", expected, result.AuthorityLimits)
		}
	}
}

func TestEpistemicDetailUnknownAndInaccessibleAreIndistinguishable(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "human-reviewed" {
			corpus.Assertions[i].AccessRefs = []string{"policy:detail-visible"}
		}
	}
	query := detailQueryFor(
		t,
		corpus,
		context,
		"human-reviewed",
		[]string{"policy:detail-visible"},
	)
	_, inaccessibleErr := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if inaccessibleErr == nil {
		t.Fatal("inaccessible assertion detail was returned")
	}

	unknown := query
	unknown.AssertionID = "assertion-that-does-not-exist"
	_, unknownErr := resolveEpistemicDetail(corpus, context, unknown, TransportAccess{})
	if unknownErr == nil {
		t.Fatal("unknown assertion detail was returned")
	}
	if inaccessibleErr.Error() != unknownErr.Error() ||
		inaccessibleErr.Error() != "epistemic detail target is unavailable" {
		t.Fatalf(
			"unknown and inaccessible targets are distinguishable: inaccessible=%q unknown=%q",
			inaccessibleErr,
			unknownErr,
		)
	}
}

func TestEpistemicDetailQueryDoesNotCarryAuthorizationContext(t *testing.T) {
	corpus, context := loadFixture(t)
	query := detailQueryFor(
		t,
		corpus,
		context,
		"human-reviewed",
		[]string{"policy:detail-visible"},
	)
	encoded, err := json.Marshal(query)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "grantedAccessRefs") ||
		strings.Contains(string(encoded), "policy:detail-visible") {
		t.Fatalf("semantic detail query carried authorization context: %s", encoded)
	}
}

func TestEpistemicDetailGrantedEvidenceDoesNotExposeAccessRefs(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "ai-unreviewed" {
			continue
		}
		corpus.Assertions[i].AccessRefs = []string{"policy:assertion"}
		corpus.Assertions[i].Evidence[0].AccessRefs = []string{"policy:evidence"}
	}
	granted := []string{"policy:assertion", "policy:evidence"}
	query := detailQueryFor(t, corpus, context, "ai-unreviewed", granted)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{
		GrantedAccessRefs: granted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content.Evidence) != 1 ||
		result.Content.Evidence[0].Ref != "conversation://tei/release-assumption" {
		t.Fatalf("granted Evidence was not returned: %+v", result.Content.Evidence)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "\"accessRefs\"") ||
		strings.Contains(string(encoded), "policy:assertion") ||
		strings.Contains(string(encoded), "policy:evidence") {
		t.Fatalf("detail result exposed access refs: %s", encoded)
	}
}

func TestEpistemicDetailRedactsEvidenceAndMarksMetadataPartial(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "ai-unreviewed" {
			corpus.Assertions[i].Evidence[0].AccessRefs = []string{"policy:restricted-evidence"}
		}
	}
	query := detailQueryFor(t, corpus, context, "ai-unreviewed", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content.Evidence) != 0 {
		t.Fatalf("restricted Evidence leaked through detail: %+v", result.Content.Evidence)
	}
	if result.Assessment.MetadataState != MetadataPartial ||
		!hasReason(result.Assessment, "EPISTEMIC_METADATA_INCOMPLETE") {
		t.Fatalf("redacted Evidence did not make detail partial: %+v", result.Assessment)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "\"accessRefs\"") ||
		strings.Contains(string(encoded), "policy:restricted-evidence") {
		t.Fatalf("detail result leaked access metadata: %s", encoded)
	}
}

func TestEpistemicDetailEvidenceAccessLossReturnsPartialDetailWithIndeterminateDrift(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		corpus.Assertions[i].Evidence = []EvidenceRef{{
			Kind:       "source",
			Ref:        "repo://protected/current-main",
			Fingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			AccessRefs: []string{"policy:evidence"},
		}}
	}

	fullAccess := []string{"policy:evidence"}
	query := detailQueryFor(t, corpus, context, "human-reviewed", fullAccess)
	if query.ObservedMetadataState != MetadataComplete {
		t.Fatalf("fixture did not start from a complete observed view: %+v", query)
	}

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatalf("Evidence access loss became a hard stale error: %v", err)
	}
	if result.Assessment.MetadataState != MetadataPartial {
		t.Fatalf("Evidence access loss did not mark current view partial: %+v", result.Assessment)
	}
	if len(result.Content.Evidence) != 0 {
		t.Fatalf("hidden Evidence leaked through detail: %+v", result.Content.Evidence)
	}
	if result.Drift.Subject != "indeterminate" {
		t.Fatalf("Evidence visibility loss was misclassified as semantic change: %+v", result.Drift)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "repo://protected/current-main") ||
		strings.Contains(string(encoded), "policy:evidence") {
		t.Fatalf("hidden Evidence identity or access policy leaked: %s", encoded)
	}
}

func TestEpistemicDetailEvidenceAccessGainReturnsCurrentDetailWithIndeterminateDrift(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		corpus.Assertions[i].Evidence = []EvidenceRef{{
			Kind:       "source",
			Ref:        "repo://protected/current-main",
			Fingerprint: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			AccessRefs: []string{"policy:evidence"},
		}}
	}

	query := detailQueryFor(t, corpus, context, "human-reviewed", nil)
	if query.ObservedMetadataState != MetadataPartial {
		t.Fatalf("fixture did not start from a partial observed view: %+v", query)
	}

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{
		GrantedAccessRefs: []string{"policy:evidence"},
	})
	if err != nil {
		t.Fatalf("Evidence access gain became a hard stale error: %v", err)
	}
	if result.Assessment.MetadataState != MetadataComplete || len(result.Content.Evidence) != 1 {
		t.Fatalf("current complete detail was not returned after access gain: %+v", result)
	}
	if result.Drift.Subject != "indeterminate" {
		t.Fatalf("Evidence visibility gain was misclassified as semantic change: %+v", result.Drift)
	}
}

func TestEpistemicDetailAccessLossCanDowngradeCurrentAssessmentWithoutSemanticStale(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		switch corpus.Assertions[i].ID {
		case "conflict-a":
			corpus.Assertions[i].ConflictsWith = nil
		case "conflict-b":
			corpus.Assertions[i].ConflictsWith = []string{"conflict-a"}
			corpus.Assertions[i].AccessRefs = []string{"policy:conflict-detail"}
		}
	}

	fullAccess := []string{"policy:conflict-detail"}
	query := detailQueryFor(t, corpus, context, "conflict-a", fullAccess)
	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatalf("access loss should return current partial assessment, not semantic-stale error: %v", err)
	}
	if result.Assessment.MetadataState != MetadataPartial ||
		result.Assessment.UseState != "review-required" ||
		!hasReason(result.Assessment, "EPISTEMIC_METADATA_INCOMPLETE") {
		t.Fatalf("access loss did not downgrade current detail assessment: %+v", result.Assessment)
	}
	if result.Assessment.ReviewSubjectFingerprint != query.ObservedReviewSubjectFingerprint {
		t.Fatalf("operational access loss changed semantic subject receipt: query=%s current=%s",
			query.ObservedReviewSubjectFingerprint,
			result.Assessment.ReviewSubjectFingerprint,
		)
	}
	if result.Drift.Subject != "same" {
		t.Fatalf("relation-only access loss reported unexpected subject drift: %+v", result.Drift)
	}
}

func TestEpistemicDetailHidesRestrictedRelationsAndKeepsPartialSignal(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		switch corpus.Assertions[i].ID {
		case "conflict-a":
			corpus.Assertions[i].ConflictsWith = nil
		case "conflict-b":
			corpus.Assertions[i].ConflictsWith = []string{"conflict-a"}
			corpus.Assertions[i].AccessRefs = []string{"policy:restricted-conflict"}
		}
	}
	query := detailQueryFor(t, corpus, context, "conflict-a", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content.ConflictsWith) != 0 {
		t.Fatalf("restricted relation identity leaked: %+v", result.Content.ConflictsWith)
	}
	if result.Assessment.MetadataState != MetadataPartial {
		t.Fatalf("hidden incoming relation did not mark detail partial: %+v", result.Assessment)
	}
}

func TestEpistemicDetailReportsObservedReceiptDrift(t *testing.T) {
	corpus, context := loadFixture(t)

	t.Run("meaning", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "user-current", nil)
		query.ObservedMeaningFingerprint =
			"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
		if err != nil {
			t.Fatalf("read detail rejected an observed meaning drift: %v", err)
		}
		if result.Drift.Meaning != "changed" {
			t.Fatalf("meaning drift was not reported: %+v", result.Drift)
		}
	})

	t.Run("subject", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "user-current", nil)
		query.ObservedReviewSubjectFingerprint =
			"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
		if err != nil {
			t.Fatalf("read detail rejected an observed subject drift: %v", err)
		}
		if result.Drift.Subject != "changed" {
			t.Fatalf("complete-view subject drift was not reported as changed: %+v", result.Drift)
		}
	})

	t.Run("target", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "anchored-identifier-policy", nil)
		query.ObservedTargetBinding = "exact"
		result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
		if err != nil {
			t.Fatalf("read detail rejected an observed target drift: %v", err)
		}
		if result.Drift.TargetBinding != "changed" {
			t.Fatalf("target drift was not reported: %+v", result.Drift)
		}
	})

	t.Run("review-sequence", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "human-reviewed", nil)
		query.ObservedReviewLogSequence = 0
		result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
		if err != nil {
			t.Fatalf("read detail rejected an observed review-history drift: %v", err)
		}
		if result.Drift.ReviewHistory != "changed" {
			t.Fatalf("complete-view review drift was not reported as changed: %+v", result.Drift)
		}
	})
}

func TestEpistemicDetailRefreshesOperationalCheckedAtWithoutStaleSubject(t *testing.T) {
	corpus, context := loadFixture(t)
	context.AsOf = "2026-10-10T01:00:00Z"
	query := detailQueryFor(t, corpus, context, "user-current", nil)

	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "user-current" {
			continue
		}
		corpus.Assertions[i].CheckedAt = "2026-10-10T00:30:00Z"
		corpus.Assertions[i].CheckedSubjectFingerprint =
			freshnessSubjectFingerprint(corpus.Assertions[i])
	}

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatalf("source-check timestamp alone made detail receipt stale: %v", err)
	}
	if result.Content.CheckedAt != "2026-10-10T00:30:00Z" {
		t.Fatalf("detail did not return refreshed source check: %+v", result.Content)
	}
}

func TestEpistemicDetailDoesNotLeakFutureCheckedAt(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "user-current" {
			continue
		}
		corpus.Assertions[i].CheckedAt = "2099-01-01T00:00:00Z"
		corpus.Assertions[i].CheckedSubjectFingerprint = freshnessSubjectFingerprint(corpus.Assertions[i])
	}
	query := detailQueryFor(t, corpus, context, "user-current", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content.CheckedAt != "" {
		t.Fatalf("future checkedAt leaked through historical detail: %+v", result.Content)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "2099-01-01T00:00:00Z") {
		t.Fatalf("future checkedAt leaked through encoded detail: %s", encoded)
	}
}

func TestEpistemicDetailDoesNotPresentStaleSourceCheckAsCurrentDetail(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "user-current" {
			continue
		}
		corpus.Assertions[i].CheckedAt = "2026-10-01T00:00:00Z"
		corpus.Assertions[i].CheckedSubjectFingerprint =
			"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}
	query := detailQueryFor(t, corpus, context, "user-current", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content.CheckedAt != "" {
		t.Fatalf("stale source check was presented as current detail: %+v", result.Content)
	}
}

func TestEpistemicDetailExplainsCurrentReviewHistory(t *testing.T) {
	corpus, context := loadFixture(t)
	query := detailQueryFor(t, corpus, context, "human-reviewed", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content.ReviewHistory) != 1 {
		t.Fatalf("current review history was not returned: %+v", result.Content.ReviewHistory)
	}
	event := result.Content.ReviewHistory[0]
	if event.ID != "review-main-branch" ||
		event.Kind != ReviewEventConfirm ||
		event.Reviewer != "human" ||
		event.Method != reviewMethodHuman ||
		event.MeaningFingerprint != "sha256:3333333333333333333333333333333333333333333333333333333333333333" ||
		event.StatementKind != ModeFact ||
		event.OccurredAt != "2026-10-06T12:05:00Z" ||
		event.Binding != "current" {
		t.Fatalf("unexpected review history summary: %+v", event)
	}
	if event.TargetIdentity.Kind != "repository-fact" ||
		event.TargetIdentity.ID != "tei.main-branch" ||
		len(event.Context) != 2 {
		t.Fatalf("review scope was not preserved in detail history: %+v", event)
	}
}

func TestEpistemicDetailHidesRestrictedReviewHistory(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		corpus.Assertions[i].ReviewEvents[0].AccessRefs = []string{"policy:review-history"}
	}
	fullAccess := []string{"policy:review-history"}
	query := detailQueryFor(t, corpus, context, "human-reviewed", fullAccess)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content.ReviewHistory) != 0 {
		t.Fatalf("restricted review history leaked: %+v", result.Content.ReviewHistory)
	}
	if result.Assessment.MetadataState != MetadataPartial ||
		result.Assessment.ConfirmationState != "unreviewed" ||
		result.Assessment.UseState != "review-required" {
		t.Fatalf("hidden review history did not downgrade current detail: %+v", result.Assessment)
	}
	if result.Drift.ReviewHistory != "indeterminate" {
		t.Fatalf("hidden review history did not become indeterminate drift: %+v", result.Drift)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "review-main-branch") ||
		strings.Contains(string(encoded), "policy:review-history") {
		t.Fatalf("hidden review identity leaked through detail output: %s", encoded)
	}
}

func TestEpistemicDetailMarksStaleVisibleReviewHistory(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		corpus.Assertions[i].ReviewEvents = append(corpus.Assertions[i].ReviewEvents, ReviewEvent{
			Sequence:           2,
			ID:                 "stale-review-history",
			Kind:               ReviewEventConfirm,
			Reviewer:           "human",
			Method:             reviewMethodHuman,
			MeaningFingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			StatementKind:      corpus.Assertions[i].Mode,
			Context:            append([]ScopeBinding{}, corpus.Assertions[i].Context...),
			TargetIdentity: ReviewTargetIdentity{
				Kind: corpus.Assertions[i].Target.Kind,
				ID:   corpus.Assertions[i].Target.ID,
			},
			OccurredAt: "2026-10-06T12:06:00Z",
		})
	}
	query := detailQueryFor(t, corpus, context, "human-reviewed", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content.ReviewHistory) != 2 {
		t.Fatalf("visible stale review history was dropped: %+v", result.Content.ReviewHistory)
	}
	var stale *DetailReviewEvent
	for i := range result.Content.ReviewHistory {
		if result.Content.ReviewHistory[i].ID == "stale-review-history" {
			stale = &result.Content.ReviewHistory[i]
			break
		}
	}
	if stale == nil || stale.Binding != "stale" ||
		stale.MeaningFingerprint != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" ||
		stale.StatementKind != ModeFact {
		t.Fatalf("stale review event did not preserve reviewed subject identity: %+v", result.Content.ReviewHistory)
	}
}

func TestEpistemicDetailDoesNotLeakFutureReviewHistory(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		original := corpus.Assertions[i].ReviewEvents[0]
		corpus.Assertions[i].ReviewEvents = append(corpus.Assertions[i].ReviewEvents, ReviewEvent{
			Sequence:           2,
			ID:                 "future-review-event",
			Kind:               ReviewEventReject,
			Reviewer:           "future-reviewer",
			Method:             reviewMethodHuman,
			MeaningFingerprint: corpus.Assertions[i].MeaningFingerprint,
			StatementKind:      corpus.Assertions[i].Mode,
			Context:            append([]ScopeBinding{}, original.Context...),
			TargetIdentity:     original.TargetIdentity,
			OccurredAt:         "2099-01-01T00:00:00Z",
		})
	}
	query := detailQueryFor(t, corpus, context, "human-reviewed", nil)

	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Assessment.ReviewLogSequence != 1 ||
		result.Assessment.LatestReviewEventID != "review-main-branch" {
		t.Fatalf("future review changed as-of detail: %+v", result.Assessment)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "future-review-event") ||
		strings.Contains(string(encoded), "future-reviewer") {
		t.Fatalf("future review leaked through detail output: %s", encoded)
	}
}

func TestEpistemicDetailInstructionRemainsDataNotPromptControl(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "user-current" {
			corpus.Assertions[i].Mode = ModeInstruction
		}
	}
	query := detailQueryFor(t, corpus, context, "user-current", nil)
	result, err := resolveEpistemicDetail(corpus, context, query, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Assessment.StatementKind != ModeInstruction ||
		result.ContentRole != "epistemic-data" ||
		result.ControlAuthority {
		t.Fatalf("instruction-kind content became prompt control: %+v", result)
	}
}

func TestCLIDetailAccessRequiresDetailQuery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"-corpus", filepath.Join("testdata", "epistemic-fixture-v0.json"),
			"-context", filepath.Join("testdata", "context-current.json"),
			"-detail-access", "access.json",
		},
		&stdout,
		&stderr,
	)
	if code != 2 || !strings.Contains(stderr.String(), "detail-access requires detail-query") {
		t.Fatalf("orphan detail access context was accepted: code=%d stderr=%s", code, stderr.String())
	}
}

func TestCLIDetailAndReviewModesAreMutuallyExclusive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"-corpus", filepath.Join("testdata", "epistemic-fixture-v0.json"),
			"-context", filepath.Join("testdata", "context-current.json"),
			"-detail-query", "detail.json",
			"-review-command", "review.json",
		},
		&stdout,
		&stderr,
	)
	if code != 2 || !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("detail and review modes were not rejected together: code=%d stderr=%s", code, stderr.String())
	}
}

func TestCLIInjectsDetailAccessSeparatelyFromSemanticQuery(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID != "human-reviewed" {
			continue
		}
		corpus.Assertions[i].AccessRefs = []string{"policy:detail-visible"}
	}

	temp := t.TempDir()
	corpusPath := filepath.Join(temp, "corpus.json")
	contextPath := filepath.Join(temp, "context.json")
	queryPath := filepath.Join(temp, "detail-query.json")
	accessPath := filepath.Join(temp, "detail-access.json")

	query := detailQueryFor(
		t,
		corpus,
		context,
		"human-reviewed",
		[]string{"policy:detail-visible"},
	)
	for path, value := range map[string]any{
		corpusPath: corpus,
		contextPath: context,
		queryPath: query,
		accessPath: TransportAccess{GrantedAccessRefs: []string{"policy:detail-visible"}},
	} {
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"-corpus", corpusPath,
			"-context", contextPath,
			"-detail-query", queryPath,
			"-detail-access", accessPath,
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var result DetailResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Assessment.AssertionID != "human-reviewed" {
		t.Fatalf("separate detail access context did not resolve target: %+v", result)
	}
}

func TestCLIEmitsEpistemicDetailWithoutWritingCorpus(t *testing.T) {
	corpus, context := loadFixture(t)
	query := detailQueryFor(t, corpus, context, "ai-unreviewed", nil)

	temp := t.TempDir()
	queryPath := filepath.Join(temp, "detail-query.json")
	encodedQuery, err := json.MarshalIndent(query, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(queryPath, encodedQuery, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{
			"-corpus", filepath.Join("testdata", "epistemic-fixture-v0.json"),
			"-context", filepath.Join("testdata", "context-current.json"),
			"-detail-query", queryPath,
		},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var result DetailResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Contract != detailResultContract ||
		result.Assessment.AssertionID != "ai-unreviewed" ||
		result.Content.Text == "" {
		t.Fatalf("unexpected detail CLI output: %+v", result)
	}
}
