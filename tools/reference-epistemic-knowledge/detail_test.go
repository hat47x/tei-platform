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
		ExpectedMeaningFingerprint:       assessment.MeaningFingerprint,
		ExpectedReviewSubjectFingerprint: assessment.ReviewSubjectFingerprint,
		ExpectedTargetBinding:            assessment.TargetBinding,
		ExpectedReviewLogSequence:        assessment.ReviewLogSequence,
		GrantedAccessRefs:                append([]string{}, grantedAccessRefs...),
	}
}

func TestEpistemicDetailReturnsCurrentAccessFilteredContent(t *testing.T) {
	corpus, context := loadFixture(t)
	original := structuredCorpusCopyForTest(t, corpus)
	query := detailQueryFor(t, corpus, context, "ai-unreviewed", nil)

	result, err := resolveEpistemicDetail(corpus, context, query)
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
		"detail-query-requires-current-binding-receipts",
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
	query.GrantedAccessRefs = nil

	_, inaccessibleErr := resolveEpistemicDetail(corpus, context, query)
	if inaccessibleErr == nil {
		t.Fatal("inaccessible assertion detail was returned")
	}

	unknown := query
	unknown.AssertionID = "assertion-that-does-not-exist"
	_, unknownErr := resolveEpistemicDetail(corpus, context, unknown)
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

	result, err := resolveEpistemicDetail(corpus, context, query)
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

	result, err := resolveEpistemicDetail(corpus, context, query)
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

	result, err := resolveEpistemicDetail(corpus, context, query)
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

func TestEpistemicDetailRejectsStaleReceipts(t *testing.T) {
	corpus, context := loadFixture(t)

	t.Run("meaning", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "user-current", nil)
		query.ExpectedMeaningFingerprint = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		_, err := resolveEpistemicDetail(corpus, context, query)
		if err == nil || !strings.Contains(err.Error(), "meaning binding is stale") {
			t.Fatalf("stale meaning receipt was accepted: %v", err)
		}
	})

	t.Run("subject", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "user-current", nil)
		query.ExpectedReviewSubjectFingerprint = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		_, err := resolveEpistemicDetail(corpus, context, query)
		if err == nil || !strings.Contains(err.Error(), "subject binding is stale") {
			t.Fatalf("stale subject receipt was accepted: %v", err)
		}
	})

	t.Run("target", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "anchored-identifier-policy", nil)
		query.ExpectedTargetBinding = "exact"
		_, err := resolveEpistemicDetail(corpus, context, query)
		if err == nil || !strings.Contains(err.Error(), "target binding is stale") {
			t.Fatalf("stale target receipt was accepted: %v", err)
		}
	})

	t.Run("review-sequence", func(t *testing.T) {
		query := detailQueryFor(t, corpus, context, "human-reviewed", nil)
		query.ExpectedReviewLogSequence = 0
		_, err := resolveEpistemicDetail(corpus, context, query)
		if err == nil || !strings.Contains(err.Error(), "review sequence is stale") {
			t.Fatalf("stale review sequence was accepted: %v", err)
		}
	})
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

	result, err := resolveEpistemicDetail(corpus, context, query)
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

	result, err := resolveEpistemicDetail(corpus, context, query)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content.CheckedAt != "" {
		t.Fatalf("stale source check was presented as current detail: %+v", result.Content)
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

	result, err := resolveEpistemicDetail(corpus, context, query)
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
	result, err := resolveEpistemicDetail(corpus, context, query)
	if err != nil {
		t.Fatal(err)
	}
	if result.Assessment.StatementKind != ModeInstruction ||
		result.ContentRole != "epistemic-data" ||
		result.ControlAuthority {
		t.Fatalf("instruction-kind content became prompt control: %+v", result)
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
