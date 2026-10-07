package main

import (
	"fmt"
	"sort"
	"time"
)

type Assessment struct {
	AssertionID          string         `json:"assertionId"`
	ContentOrigin        Origin         `json:"contentOrigin"`
	IngestedBy           IngestionActor `json:"ingestedBy"`
	StatementKind        AssertionMode  `json:"statementKind"`
	ConfirmationState    string         `json:"confirmationState"`
	ReviewBinding        string         `json:"reviewBinding"`
	ReviewEventCount     int            `json:"reviewEventCount"`
	LatestReviewEventID  string         `json:"latestReviewEventId,omitempty"`
	LatestReviewEventKind ReviewEventKind `json:"latestReviewEventKind,omitempty"`
	ReviewConditions     []string       `json:"reviewConditions,omitempty"`
	ContextCompatibility string         `json:"contextCompatibility"`
	Freshness            string         `json:"freshness"`
	LifecycleState       string         `json:"lifecycleState"`
	Conflict             string         `json:"conflict"`
	SupersededBy         []string       `json:"supersededBy,omitempty"`
	UseState             string         `json:"useState"`
	InferredContextUsed  bool           `json:"inferredContextUsed"`
	Reasons              []string       `json:"reasons"`
}

type HealthCounts struct {
	Total          int `json:"total"`
	Premise        int `json:"premise"`
	CandidateOnly  int `json:"candidateOnly"`
	ReviewRequired int `json:"reviewRequired"`
	Blocked        int `json:"blocked"`
}

type HealthIssue struct {
	AssertionID string      `json:"assertionId"`
	Consumer    string      `json:"consumer"`
	Criticality Criticality `json:"criticality"`
	UseState    string      `json:"useState"`
	Reasons     []string    `json:"reasons"`
}

type KnowledgeHealth struct {
	Level  string        `json:"level"`
	Counts HealthCounts  `json:"counts"`
	Issues []HealthIssue `json:"issues,omitempty"`
}

type ReviewRequest struct {
	AssertionID        string         `json:"assertionId"`
	Target             TargetRef      `json:"target"`
	ConfirmationState  string         `json:"confirmationState"`
	ReviewBinding      string         `json:"reviewBinding"`
	Context            []ScopeBinding `json:"context,omitempty"`
	Evidence           []EvidenceRef  `json:"evidence,omitempty"`
	Reasons            []string       `json:"reasons"`
	DownstreamImpact   []string       `json:"downstreamImpact,omitempty"`
	ReviewerCandidates []string       `json:"reviewerCandidates,omitempty"`
	AuthorityLimit     string         `json:"authorityLimit"`
}

type Projection struct {
	Schema          string          `json:"schema"`
	Context         ContextQuery    `json:"context"`
	Assessments     []Assessment    `json:"assessments"`
	Health          KnowledgeHealth `json:"health"`
	ReviewRequests  []ReviewRequest `json:"reviewRequests,omitempty"`
	AuthorityLimits []string        `json:"authorityLimits"`
}

func evaluateCorpus(raw Corpus, rawContext ContextQuery) (Projection, error) {
	corpus, err := normalizeCorpus(raw)
	if err != nil {
		return Projection{}, err
	}
	context, asOf, err := normalizeContext(rawContext)
	if err != nil {
		return Projection{}, err
	}

	assertions := make(map[string]Assertion, len(corpus.Assertions))
	assessments := make(map[string]Assessment, len(corpus.Assertions))
	supersededBy := map[string][]string{}
	for _, assertion := range corpus.Assertions {
		assertions[assertion.ID] = assertion
		assessments[assertion.ID] = baseAssessment(assertion, context, asOf)
		for _, target := range assertion.Supersedes {
			supersededBy[target] = append(supersededBy[target], assertion.ID)
		}
	}
	for target := range supersededBy {
		sort.Strings(supersededBy[target])
	}

	for _, assertion := range corpus.Assertions {
		assessment := assessments[assertion.ID]
		if replacements := supersededBy[assertion.ID]; len(replacements) > 0 {
			assessment.LifecycleState = "superseded"
			assessment.SupersededBy = append([]string{}, replacements...)
			assessment.Reasons = append(assessment.Reasons, "SUPERSEDED_BY_RELATION")
			assessments[assertion.ID] = assessment
		}
	}

	for _, assertion := range corpus.Assertions {
		assessment := assessments[assertion.ID]
		assessment.Conflict = conflictState(assertion, assessments)
		assessment.UseState, assessment.Reasons = deriveUseState(assertion, assessment)
		assessment.Reasons = normalizeStrings(assessment.Reasons)
		assessments[assertion.ID] = assessment
	}

	out := Projection{
		Schema:  projectionSchema,
		Context: context,
		AuthorityLimits: []string{
			"assessment-does-not-assert-objective-truth",
			"assessment-does-not-grant-canonical-acceptance",
			"assessment-does-not-grant-write-or-runtime-authority",
			"meaning-fingerprint-is-input-not-semantic-proof",
			"reviewer-candidate-is-not-review-authority",
		},
	}
	for _, assertion := range corpus.Assertions {
		out.Assessments = append(out.Assessments, assessments[assertion.ID])
	}
	out.Health = buildHealth(corpus, assessments)
	out.ReviewRequests = buildReviewRequests(corpus, assertions, assessments)
	return out, nil
}

func baseAssessment(assertion Assertion, context ContextQuery, asOf time.Time) Assessment {
	history := assessReviewHistory(assertion)
	assessment := Assessment{
		AssertionID:           assertion.ID,
		ContentOrigin:         assertion.Origin,
		IngestedBy:            assertion.IngestedBy,
		StatementKind:         assertion.Mode,
		ConfirmationState:     history.ConfirmationState,
		ReviewBinding:         history.ReviewBinding,
		ReviewEventCount:      len(assertion.ReviewEvents),
		LatestReviewEventID:   history.LatestEventID,
		LatestReviewEventKind: history.LatestEventKind,
		ReviewConditions:      append([]string{}, history.Conditions...),
		ContextCompatibility:  contextCompatibility(assertion.Context, context.Bindings),
		Freshness:             freshness(assertion, asOf),
		LifecycleState:        "active",
		Conflict:              "none",
	}
	if history.StaleEventCount > 0 {
		assessment.Reasons = append(assessment.Reasons, "STALE_REVIEW_HISTORY_PRESENT")
	}
	for _, binding := range assertion.Context {
		if binding.Provenance == ScopeInferred {
			assessment.InferredContextUsed = true
			assessment.Reasons = append(assessment.Reasons, "INFERRED_CONTEXT_USED")
		}
	}
	return assessment
}

type reviewHistoryAssessment struct {
	ConfirmationState string
	ReviewBinding     string
	LatestEventID     string
	LatestEventKind   ReviewEventKind
	Conditions        []string
	StaleEventCount   int
}

func assessReviewHistory(assertion Assertion) reviewHistoryAssessment {
	result := reviewHistoryAssessment{
		ConfirmationState: "unreviewed",
		ReviewBinding:     "none",
	}
	var latest *ReviewEvent
	for i := range assertion.ReviewEvents {
		event := &assertion.ReviewEvents[i]
		if event.MeaningFingerprint != assertion.MeaningFingerprint {
			result.StaleEventCount++
			continue
		}
		latest = event
	}
	if latest == nil {
		if result.StaleEventCount > 0 {
			result.ReviewBinding = "stale"
		}
		return result
	}

	result.ReviewBinding = "current"
	result.LatestEventID = latest.ID
	result.LatestEventKind = latest.Kind
	result.Conditions = append([]string{}, latest.Conditions...)
	switch latest.Kind {
	case ReviewEventConfirm:
		result.ConfirmationState = "confirmed"
	case ReviewEventReject:
		result.ConfirmationState = "rejected"
	case ReviewEventWithdraw:
		result.ConfirmationState = "unreviewed"
	}
	return result
}

func contextCompatibility(assertion, current []ScopeBinding) string {
	if len(assertion) == 0 {
		return "unknown"
	}
	currentByDimension := map[string]string{}
	for _, binding := range current {
		currentByDimension[binding.Dimension] = binding.Value
	}
	unknown := false
	for _, binding := range assertion {
		value, ok := currentByDimension[binding.Dimension]
		if !ok {
			unknown = true
			continue
		}
		if value != binding.Value {
			return "incompatible"
		}
	}
	if unknown {
		return "unknown"
	}
	return "compatible"
}

func freshness(assertion Assertion, asOf time.Time) string {
	if assertion.ValidFrom != "" {
		from, _ := time.Parse(time.RFC3339, assertion.ValidFrom)
		if asOf.Before(from) {
			return "not-yet-valid"
		}
	}
	if assertion.ValidUntil != "" {
		until, _ := time.Parse(time.RFC3339, assertion.ValidUntil)
		if asOf.After(until) {
			return "stale"
		}
		return "current"
	}
	return "unknown"
}

func conflictState(assertion Assertion, assessments map[string]Assessment) string {
	if len(assertion.ConflictsWith) == 0 {
		return "none"
	}
	active := false
	for _, id := range assertion.ConflictsWith {
		other, ok := assessments[id]
		if !ok {
			continue
		}
		if other.ContextCompatibility == "incompatible" ||
			other.ConfirmationState == "rejected" ||
			other.LifecycleState == "superseded" ||
			other.Freshness == "stale" ||
			other.Freshness == "not-yet-valid" {
			continue
		}
		active = true
	}
	if active {
		return "present"
	}
	return "inactive"
}

func deriveUseState(assertion Assertion, assessment Assessment) (string, []string) {
	reasons := append([]string{}, assessment.Reasons...)
	if assessment.ContextCompatibility == "incompatible" {
		return "blocked", append(reasons, "CONTEXT_INCOMPATIBLE")
	}
	if assessment.ContextCompatibility == "unknown" {
		return "blocked", append(reasons, "CONTEXT_COMPATIBILITY_UNKNOWN")
	}
	switch assessment.Freshness {
	case "stale":
		return "blocked", append(reasons, "ASSERTION_STALE")
	case "not-yet-valid":
		return "blocked", append(reasons, "ASSERTION_NOT_YET_VALID")
	case "unknown":
		reasons = append(reasons, "FRESHNESS_UNBOUNDED")
	}
	if assessment.LifecycleState == "superseded" {
		return "blocked", append(reasons, "ASSERTION_SUPERSEDED")
	}
	if assessment.ConfirmationState == "rejected" {
		return "blocked", append(reasons, "ASSERTION_REJECTED")
	}
	if assessment.ReviewBinding == "stale" {
		return "review-required", append(reasons, "REVIEW_BINDING_STALE")
	}
	if assessment.Conflict == "present" {
		return "review-required", append(reasons, "CONFLICT_PRESENT")
	}
	if !assertion.Mode.canBeWorkingPremise() {
		return "candidate-only", append(reasons, "STATEMENT_KIND_NOT_WORKING_PREMISE")
	}
	if assessment.ConfirmationState == "confirmed" {
		if len(assessment.ReviewConditions) > 0 {
			return "review-required", append(reasons, "REVIEW_CONDITIONS_UNRESOLVED")
		}
		return "premise", reasons
	}
	if assertion.Origin == OriginUser && assertion.IngestedBy == IngestedByUser {
		return "premise", append(reasons, "DIRECT_USER_WORKING_PREMISE")
	}
	if assessment.LatestReviewEventKind == ReviewEventWithdraw {
		reasons = append(reasons, "CONFIRMATION_WITHDRAWN")
	}
	return "candidate-only", append(reasons, "ASSERTION_UNREVIEWED")
}

func buildHealth(corpus Corpus, assessments map[string]Assessment) KnowledgeHealth {
	health := KnowledgeHealth{Level: "healthy"}
	for _, assertion := range corpus.Assertions {
		switch assessments[assertion.ID].UseState {
		case "premise":
			health.Counts.Premise++
		case "candidate-only":
			health.Counts.CandidateOnly++
		case "review-required":
			health.Counts.ReviewRequired++
		case "blocked":
			health.Counts.Blocked++
		}
		health.Counts.Total++
	}

	for _, dependency := range corpus.Dependencies {
		assessment := assessments[dependency.AssertionID]
		if assessment.UseState == "premise" {
			continue
		}
		issue := HealthIssue{
			AssertionID: dependency.AssertionID,
			Consumer:    dependency.Consumer,
			Criticality: dependency.Criticality,
			UseState:    assessment.UseState,
			Reasons:     append([]string{}, assessment.Reasons...),
		}
		health.Issues = append(health.Issues, issue)
		switch dependency.Criticality {
		case CriticalityCritical:
			health.Level = "critical"
		case CriticalityHigh:
			if health.Level != "critical" {
				health.Level = "attention"
			}
		}
	}
	sort.Slice(health.Issues, func(i, j int) bool {
		ri, rj := criticalityRank(health.Issues[i].Criticality), criticalityRank(health.Issues[j].Criticality)
		if ri != rj {
			return ri > rj
		}
		if health.Issues[i].AssertionID != health.Issues[j].AssertionID {
			return health.Issues[i].AssertionID < health.Issues[j].AssertionID
		}
		return health.Issues[i].Consumer < health.Issues[j].Consumer
	})
	return health
}

func buildReviewRequests(
	corpus Corpus,
	assertions map[string]Assertion,
	assessments map[string]Assessment,
) []ReviewRequest {
	impact := map[string][]string{}
	critical := map[string]bool{}
	for _, dependency := range corpus.Dependencies {
		impact[dependency.AssertionID] = append(impact[dependency.AssertionID], fmt.Sprintf("%s (%s)", dependency.Consumer, dependency.Criticality))
		if dependency.Criticality == CriticalityCritical || dependency.Criticality == CriticalityHigh {
			critical[dependency.AssertionID] = true
		}
	}
	var out []ReviewRequest
	for id, assessment := range assessments {
		if assessment.UseState != "review-required" &&
			!(critical[id] && assessment.UseState != "premise") {
			continue
		}
		assertion := assertions[id]
		out = append(out, ReviewRequest{
			AssertionID:        id,
			Target:             assertion.Target,
			ConfirmationState:  assessment.ConfirmationState,
			ReviewBinding:      assessment.ReviewBinding,
			Context:            append([]ScopeBinding{}, assertion.Context...),
			Evidence:           append([]EvidenceRef{}, assertion.Evidence...),
			Reasons:            append([]string{}, assessment.Reasons...),
			DownstreamImpact:   normalizeStrings(impact[id]),
			ReviewerCandidates: append([]string{}, assertion.ReviewerCandidates...),
			AuthorityLimit:     "reviewer-candidate-is-not-review-authority",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AssertionID < out[j].AssertionID })
	return out
}

func criticalityRank(value Criticality) int {
	switch value {
	case CriticalityCritical:
		return 4
	case CriticalityHigh:
		return 3
	case CriticalityNormal:
		return 2
	case CriticalityLow:
		return 1
	default:
		return 0
	}
}
