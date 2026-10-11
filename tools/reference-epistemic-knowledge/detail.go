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
	Contract                         string        `json:"contract"`
	Schema                           string        `json:"schema"`
	AssertionID                      string        `json:"assertionId"`
	ExpectedMeaningFingerprint       string        `json:"expectedMeaningFingerprint"`
	ExpectedReviewSubjectFingerprint string        `json:"expectedReviewSubjectFingerprint"`
	ExpectedTargetBinding            string        `json:"expectedTargetBinding"`
	ExpectedReviewLogSequence        int           `json:"expectedReviewLogSequence"`
	ObservedMeaningFingerprint       string        `json:"observedMeaningFingerprint"`
	ObservedReviewSubjectFingerprint string        `json:"observedReviewSubjectFingerprint"`
	ObservedTargetBinding            string        `json:"observedTargetBinding"`
	ObservedReviewLogSequence        int           `json:"observedReviewLogSequence"`
	ObservedMetadataState            MetadataState `json:"observedMetadataState"`
}

type DetailEvidenceRef struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type DetailReviewEvent struct {
	Sequence           int                  `json:"sequence"`
	ID                 string               `json:"id"`
	Kind               ReviewEventKind      `json:"kind"`
	Reviewer           string               `json:"reviewer"`
	Method             string               `json:"method"`
	MeaningFingerprint string               `json:"meaningFingerprint"`
	StatementKind      AssertionMode        `json:"statementKind"`
	OccurredAt         string               `json:"occurredAt"`
	Binding            string               `json:"binding"`
	Context            []ScopeBinding       `json:"context,omitempty"`
	TargetIdentity     ReviewTargetIdentity `json:"targetIdentity"`
	ValidFrom          string               `json:"validFrom,omitempty"`
	ValidUntil         string               `json:"validUntil,omitempty"`
	Evidence           []DetailEvidenceRef  `json:"evidence,omitempty"`
	Conditions         []string             `json:"conditions,omitempty"`
}

type DetailContent struct {
	Text          string              `json:"text"`
	Target        TargetRef           `json:"target"`
	Context       []ScopeBinding      `json:"context,omitempty"`
	ValidFrom     string              `json:"validFrom,omitempty"`
	ValidUntil    string              `json:"validUntil,omitempty"`
	CheckedAt     string              `json:"checkedAt,omitempty"`
	Evidence      []DetailEvidenceRef `json:"evidence,omitempty"`
	ReviewHistory []DetailReviewEvent `json:"reviewHistory,omitempty"`
	Supersedes    []string            `json:"supersedes,omitempty"`
	ConflictsWith []string            `json:"conflictsWith,omitempty"`
}

type DetailDrift struct {
	Meaning       string `json:"meaning"`
	Subject       string `json:"subject"`
	ReviewHistory string `json:"reviewHistory"`
	TargetBinding string `json:"targetBinding"`
}

type DetailResult struct {
	Contract         string        `json:"contract"`
	Schema           string        `json:"schema"`
	Assessment       Assessment    `json:"assessment"`
	Drift            DetailDrift   `json:"drift"`
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

	drift := detailDrift(query, *assessment)

	return DetailResult{
		Contract:   detailResultContract,
		Schema:     detailResultSchema,
		Assessment: *assessment,
		Drift:      drift,
		Content: DetailContent{
			Text:          assertion.Text,
			Target:        assertion.Target,
			Context:       append([]ScopeBinding{}, assertion.Context...),
			ValidFrom:     assertion.ValidFrom,
			ValidUntil:    assertion.ValidUntil,
			CheckedAt:     detailCheckedAt(*assertion, asOf),
			Evidence:      detailEvidence(assertion.Evidence),
			ReviewHistory: detailReviewHistory(*assertion, asOf),
			Supersedes:    append([]string{}, assertion.Supersedes...),
			ConflictsWith: append([]string{}, assertion.ConflictsWith...),
		},
		ContentRole:      "epistemic-data",
		ControlAuthority: false,
		AuthorityLimits: []string{
			"detail-access-context-is-not-part-of-semantic-query",
			"detail-access-refs-are-opaque-input-not-authorization-authority",
			"detail-query-does-not-broaden-source-access",
			"detail-observed-receipts-report-drift-not-mutation-preconditions",
			"detail-drift-indeterminate-does-not-prove-semantic-change",
			"detail-query-is-bounded-by-context-as-of",
			"detail-result-does-not-change-confirmation-or-use-state",
			"detail-result-does-not-assert-objective-truth",
			"detail-result-does-not-grant-canonical-acceptance",
			"detail-result-omits-access-refs",
			"detail-content-is-data-not-prompt-control-authority",
		},
	}, nil
}

func detailDrift(query DetailQuery, assessment Assessment) DetailDrift {
	meaning := "same"
	if assessment.MeaningFingerprint != query.ObservedMeaningFingerprint {
		meaning = "changed"
	}

	subject := detailMetadataAwareDrift(
		assessment.ReviewSubjectFingerprint == query.ObservedReviewSubjectFingerprint,
		query.ObservedMetadataState,
		assessment.MetadataState,
	)
	reviewHistory := detailMetadataAwareDrift(
		assessment.ReviewLogSequence == query.ObservedReviewLogSequence,
		query.ObservedMetadataState,
		assessment.MetadataState,
	)

	targetBinding := "same"
	if assessment.TargetBinding != query.ObservedTargetBinding {
		targetBinding = "changed"
	}

	return DetailDrift{
		Meaning:       meaning,
		Subject:       subject,
		ReviewHistory: reviewHistory,
		TargetBinding: targetBinding,
	}
}

func detailMetadataAwareDrift(
	equal bool,
	observed MetadataState,
	current MetadataState,
) string {
	if equal {
		return "same"
	}
	if observed == MetadataComplete && current == MetadataComplete {
		return "changed"
	}
	return "indeterminate"
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
	input.ObservedMeaningFingerprint = strings.TrimSpace(input.ObservedMeaningFingerprint)
	input.ObservedReviewSubjectFingerprint = strings.TrimSpace(input.ObservedReviewSubjectFingerprint)
	input.ObservedTargetBinding = strings.TrimSpace(input.ObservedTargetBinding)

	if input.Contract != detailQueryContract {
		return input, fmt.Errorf("unsupported epistemic detail query contract %q", input.Contract)
	}
	if input.Schema != detailQuerySchema {
		return input, fmt.Errorf("unsupported epistemic detail query schema %q", input.Schema)
	}
	if input.AssertionID == "" {
		return input, fmt.Errorf("epistemic detail query requires assertionId")
	}
	if !sha256Pattern.MatchString(input.ObservedMeaningFingerprint) {
		return input, fmt.Errorf(
			"epistemic detail observedMeaningFingerprint %q is not exact sha256",
			input.ObservedMeaningFingerprint,
		)
	}
	if !sha256Pattern.MatchString(input.ObservedReviewSubjectFingerprint) {
		return input, fmt.Errorf(
			"epistemic detail observedReviewSubjectFingerprint %q is not exact sha256",
			input.ObservedReviewSubjectFingerprint,
		)
	}
	if !validTargetBinding(input.ObservedTargetBinding) {
		return input, fmt.Errorf(
			"epistemic detail observedTargetBinding %q is invalid",
			input.ObservedTargetBinding,
		)
	}
	if input.ObservedReviewLogSequence < 0 {
		return input, fmt.Errorf("epistemic detail observedReviewLogSequence must be >= 0")
	}
	if !input.ObservedMetadataState.valid() {
		return input, fmt.Errorf(
			"epistemic detail observedMetadataState %q is invalid",
			input.ObservedMetadataState,
		)
	}
	return input, nil
}

func detailReviewHistory(assertion Assertion, asOf time.Time) []DetailReviewEvent {
	out := make([]DetailReviewEvent, 0, len(assertion.ReviewEvents))
	for _, event := range assertion.ReviewEvents {
		occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
		if err != nil || occurredAt.After(asOf) {
			continue
		}
		binding := "stale"
		if reviewEventMatchesAssertion(event, assertion) {
			binding = "current"
		}
		out = append(out, DetailReviewEvent{
			Sequence:           event.Sequence,
			ID:                 event.ID,
			Kind:               event.Kind,
			Reviewer:           event.Reviewer,
			Method:             event.Method,
			MeaningFingerprint: event.MeaningFingerprint,
			StatementKind:      event.StatementKind,
			OccurredAt:         event.OccurredAt,
			Binding:            binding,
			Context:            append([]ScopeBinding{}, event.Context...),
			TargetIdentity:     event.TargetIdentity,
			ValidFrom:          event.ValidFrom,
			ValidUntil:         event.ValidUntil,
			Evidence:           detailEvidence(event.Evidence),
			Conditions:         append([]string{}, event.Conditions...),
		})
	}
	return out
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
