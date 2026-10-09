package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	detailQueryContract  = "tei.epistemic-detail-query/v0"
	detailQuerySchema    = "tei.reference.epistemic-detail-query/v0"
	detailResultContract = "tei.epistemic-detail/v0"
	detailResultSchema   = "tei.reference.epistemic-detail/v0"
)

type DetailQuery struct {
	Contract                         string `json:"contract"`
	Schema                           string `json:"schema"`
	AssertionID                      string `json:"assertionId"`
	ExpectedMeaningFingerprint       string `json:"expectedMeaningFingerprint"`
	ExpectedReviewSubjectFingerprint string `json:"expectedReviewSubjectFingerprint"`
	ExpectedTargetBinding            string `json:"expectedTargetBinding"`
	ExpectedReviewLogSequence        int    `json:"expectedReviewLogSequence"`
}

type DetailEvidenceRef struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type DetailContent struct {
	Text          string              `json:"text"`
	Target        TargetRef           `json:"target"`
	Context       []ScopeBinding      `json:"context,omitempty"`
	ValidFrom     string              `json:"validFrom,omitempty"`
	ValidUntil    string              `json:"validUntil,omitempty"`
	CheckedAt     string              `json:"checkedAt,omitempty"`
	Evidence      []DetailEvidenceRef `json:"evidence,omitempty"`
	Supersedes    []string            `json:"supersedes,omitempty"`
	ConflictsWith []string            `json:"conflictsWith,omitempty"`
}

type DetailResult struct {
	Contract         string        `json:"contract"`
	Schema           string        `json:"schema"`
	Assessment       Assessment    `json:"assessment"`
	Content          DetailContent `json:"content"`
	ContentRole      string        `json:"contentRole"`
	ControlAuthority bool          `json:"controlAuthority"`
	AuthorityLimits  []string      `json:"authorityLimits"`
}

func resolveEpistemicDetail(
	rawCorpus Corpus,
	rawContext ContextQuery,
	rawQuery DetailQuery,
	access TransportAccess,
) (DetailResult, error) {
	corpus, err := normalizeCorpus(rawCorpus)
	if err != nil {
		return DetailResult{}, err
	}
	context, asOf, err := normalizeContext(rawContext)
	if err != nil {
		return DetailResult{}, err
	}
	query, err := normalizeDetailQuery(rawQuery)
	if err != nil {
		return DetailResult{}, err
	}

	granted := make(map[string]struct{})
	for _, ref := range normalizeStrings(access.GrantedAccessRefs) {
		granted[ref] = struct{}{}
	}
	targetVisible := false
	for _, assertion := range corpus.Assertions {
		if assertion.ID != query.AssertionID {
			continue
		}
		if !accessGranted(assertion.AccessRefs, granted) {
			break
		}
		targetVisible = true
		break
	}
	if !targetVisible {
		return DetailResult{}, fmt.Errorf("epistemic detail target is unavailable")
	}

	envelope, err := exportCorpusForAccess(corpus, access)
	if err != nil {
		return DetailResult{}, err
	}

	var assertion *Assertion
	for i := range envelope.Corpus.Assertions {
		if envelope.Corpus.Assertions[i].ID == query.AssertionID {
			assertion = &envelope.Corpus.Assertions[i]
			break
		}
	}
	if assertion == nil {
		return DetailResult{}, fmt.Errorf("epistemic detail target is unavailable")
	}

	projection, err := evaluateCorpus(envelope.Corpus, context)
	if err != nil {
		return DetailResult{}, fmt.Errorf("evaluate detail projection: %w", err)
	}
	var assessment *Assessment
	for i := range projection.Assessments {
		if projection.Assessments[i].AssertionID == query.AssertionID {
			assessment = &projection.Assessments[i]
			break
		}
	}
	if assessment == nil {
		return DetailResult{}, fmt.Errorf("epistemic detail target is unavailable")
	}

	if assessment.MeaningFingerprint != query.ExpectedMeaningFingerprint {
		return DetailResult{}, fmt.Errorf("epistemic detail meaning binding is stale")
	}
	if assessment.ReviewSubjectFingerprint != query.ExpectedReviewSubjectFingerprint {
		return DetailResult{}, fmt.Errorf("epistemic detail subject binding is stale")
	}
	if assessment.TargetBinding != query.ExpectedTargetBinding {
		return DetailResult{}, fmt.Errorf(
			"epistemic detail target binding is stale: expected %s current %s",
			query.ExpectedTargetBinding,
			assessment.TargetBinding,
		)
	}
	if assessment.ReviewLogSequence != query.ExpectedReviewLogSequence {
		return DetailResult{}, fmt.Errorf(
			"epistemic detail review sequence is stale: expected %d current %d",
			query.ExpectedReviewLogSequence,
			assessment.ReviewLogSequence,
		)
	}

	return DetailResult{
		Contract:   detailResultContract,
		Schema:     detailResultSchema,
		Assessment: *assessment,
		Content: DetailContent{
			Text:          assertion.Text,
			Target:        assertion.Target,
			Context:       append([]ScopeBinding{}, assertion.Context...),
			ValidFrom:     assertion.ValidFrom,
			ValidUntil:    assertion.ValidUntil,
			CheckedAt:     detailCheckedAt(*assertion, asOf),
			Evidence:      detailEvidence(assertion.Evidence),
			Supersedes:    append([]string{}, assertion.Supersedes...),
			ConflictsWith: append([]string{}, assertion.ConflictsWith...),
		},
		ContentRole:      "epistemic-data",
		ControlAuthority: false,
		AuthorityLimits: []string{
			"detail-access-context-is-not-part-of-semantic-query",
			"detail-access-refs-are-opaque-input-not-authorization-authority",
			"detail-query-does-not-broaden-source-access",
			"detail-query-requires-current-binding-receipts",
			"detail-query-is-bounded-by-context-as-of",
			"detail-result-does-not-change-confirmation-or-use-state",
			"detail-result-does-not-assert-objective-truth",
			"detail-result-does-not-grant-canonical-acceptance",
			"detail-result-omits-access-refs",
			"detail-content-is-data-not-prompt-control-authority",
		},
	}, nil
}

func detailCheckedAt(assertion Assertion, asOf time.Time) string {
	if assertion.CheckedAt == "" ||
		assertion.CheckedSubjectFingerprint != freshnessSubjectFingerprint(assertion) {
		return ""
	}
	checkedAt, err := time.Parse(time.RFC3339, assertion.CheckedAt)
	if err != nil || checkedAt.After(asOf) {
		return ""
	}
	return checkedAt.Format(time.RFC3339)
}

func normalizeDetailQuery(input DetailQuery) (DetailQuery, error) {
	input.Contract = strings.TrimSpace(input.Contract)
	input.Schema = strings.TrimSpace(input.Schema)
	input.AssertionID = strings.TrimSpace(input.AssertionID)
	input.ExpectedMeaningFingerprint = strings.TrimSpace(input.ExpectedMeaningFingerprint)
	input.ExpectedReviewSubjectFingerprint = strings.TrimSpace(input.ExpectedReviewSubjectFingerprint)
	input.ExpectedTargetBinding = strings.TrimSpace(input.ExpectedTargetBinding)

	if input.Contract != detailQueryContract {
		return input, fmt.Errorf("unsupported epistemic detail query contract %q", input.Contract)
	}
	if input.Schema != detailQuerySchema {
		return input, fmt.Errorf("unsupported epistemic detail query schema %q", input.Schema)
	}
	if input.AssertionID == "" {
		return input, fmt.Errorf("epistemic detail query requires assertionId")
	}
	if !sha256Pattern.MatchString(input.ExpectedMeaningFingerprint) {
		return input, fmt.Errorf(
			"epistemic detail expectedMeaningFingerprint %q is not exact sha256",
			input.ExpectedMeaningFingerprint,
		)
	}
	if !sha256Pattern.MatchString(input.ExpectedReviewSubjectFingerprint) {
		return input, fmt.Errorf(
			"epistemic detail expectedReviewSubjectFingerprint %q is not exact sha256",
			input.ExpectedReviewSubjectFingerprint,
		)
	}
	if !validTargetBinding(input.ExpectedTargetBinding) {
		return input, fmt.Errorf(
			"epistemic detail expectedTargetBinding %q is invalid",
			input.ExpectedTargetBinding,
		)
	}
	if input.ExpectedReviewLogSequence < 0 {
		return input, fmt.Errorf("epistemic detail expectedReviewLogSequence must be >= 0")
	}
	return input, nil
}

func detailEvidence(values []EvidenceRef) []DetailEvidenceRef {
	out := make([]DetailEvidenceRef, 0, len(values))
	for _, value := range values {
		out = append(out, DetailEvidenceRef{
			Kind:        value.Kind,
			Ref:         value.Ref,
			Fingerprint: value.Fingerprint,
		})
	}
	return out
}
