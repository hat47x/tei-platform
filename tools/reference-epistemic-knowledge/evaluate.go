package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type Assessment struct {
	AssertionID          string         `json:"assertionId"`
	MeaningFingerprint   string         `json:"meaningFingerprint"`
	ReviewSubjectFingerprint string      `json:"reviewSubjectFingerprint"`
	ContentOrigin        Origin         `json:"contentOrigin"`
	IngestedBy           IngestionActor `json:"ingestedBy"`
	MetadataState        MetadataState  `json:"metadataState"`
	StatementKind        AssertionMode  `json:"statementKind"`
	ConfirmationState    string         `json:"confirmationState"`
	ReviewBinding        string         `json:"reviewBinding"`
	ReviewEventCount     int            `json:"reviewEventCount"`
	ReviewLogSequence    int            `json:"reviewLogSequence"`
	LatestReviewEventID  string         `json:"latestReviewEventId,omitempty"`
	LatestReviewEventKind ReviewEventKind `json:"latestReviewEventKind,omitempty"`
	ReviewConditions     []string       `json:"reviewConditions,omitempty"`
	TargetBinding        string         `json:"targetBinding"`
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
	AssertionID                 string      `json:"assertionId"`
	Consumer                    string      `json:"consumer"`
	Criticality                 Criticality `json:"criticality"`
	UseState                    string      `json:"useState"`
	ConsumerFreshness           string      `json:"consumerFreshness,omitempty"`
	FreshnessRequirementSeconds int         `json:"freshnessRequirementSeconds,omitempty"`
	Reasons                     []string    `json:"reasons"`
}

type CoverageCounts struct {
	TotalAreas  int `json:"totalAreas"`
	MappedAreas int `json:"mappedAreas"`
	GapAreas    int `json:"gapAreas"`
}

type CoverageAreaAssessment struct {
	AreaID             string      `json:"areaId"`
	Target             TargetRef   `json:"target"`
	Criticality        Criticality `json:"criticality"`
	State              string      `json:"state"`
	TargetBinding      string      `json:"targetBinding"`
	MappedAssertionIDs []string    `json:"mappedAssertionIds,omitempty"`
	Reasons            []string    `json:"reasons,omitempty"`
}

type CoverageAssessment struct {
	Counts CoverageCounts           `json:"counts"`
	Areas  []CoverageAreaAssessment `json:"areas,omitempty"`
}

type ResolutionCoverage struct {
	Eligible                    int      `json:"eligible"`
	Resolved                    int      `json:"resolved"`
	Confirmed                   int      `json:"confirmed"`
	NativePremise               int      `json:"nativePremise"`
	Rejected                    int      `json:"rejected"`
	Unresolved                  int      `json:"unresolved"`
	Ratio                       float64  `json:"ratio"`
	MinimumRatio                float64  `json:"minimumRatio,omitempty"`
	MinimumEligibleAssertions   int      `json:"minimumEligibleAssertions,omitempty"`
	RequiredResolutions         int      `json:"requiredResolutions,omitempty"`
	State                       string   `json:"state"`
	Reasons                     []string `json:"reasons,omitempty"`
}

type KnowledgeHealth struct {
	Level              string             `json:"level"`
	ViewMetadataState  MetadataState      `json:"viewMetadataState"`
	Counts             HealthCounts       `json:"counts"`
	ResolutionCoverage ResolutionCoverage `json:"resolutionCoverage"`
	Issues             []HealthIssue      `json:"issues,omitempty"`
	Coverage           CoverageAssessment `json:"coverage"`
}

type ReviewRequest struct {
	AssertionID        string         `json:"assertionId"`
	Target             TargetRef      `json:"target"`
	ConfirmationState  string         `json:"confirmationState"`
	ReviewBinding      string         `json:"reviewBinding"`
	TargetBinding      string         `json:"targetBinding"`
	Context            []ScopeBinding `json:"context,omitempty"`
	Evidence           []EvidenceRef  `json:"evidence,omitempty"`
	Reasons            []string       `json:"reasons"`
	DownstreamImpact   []string       `json:"downstreamImpact,omitempty"`
	ReviewerCandidates []string       `json:"reviewerCandidates,omitempty"`
	AuthorityLimit     string         `json:"authorityLimit"`
}

type Projection struct {
	Contract        string          `json:"contract"`
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
	conflictsByID := map[string][]string{}
	for _, assertion := range corpus.Assertions {
		assertions[assertion.ID] = assertion
		assessments[assertion.ID] = baseAssessment(assertion, context, asOf)
		for _, target := range assertion.Supersedes {
			supersededBy[target] = append(supersededBy[target], assertion.ID)
		}
		for _, target := range assertion.ConflictsWith {
			conflictsByID[assertion.ID] = append(conflictsByID[assertion.ID], target)
			conflictsByID[target] = append(conflictsByID[target], assertion.ID)
		}
	}
	for target := range supersededBy {
		sort.Strings(supersededBy[target])
	}
	for id := range conflictsByID {
		conflictsByID[id] = normalizeStrings(conflictsByID[id])
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
		assessment.Conflict = conflictState(conflictsByID[assertion.ID], assessments)
		assessment.UseState, assessment.Reasons = deriveUseState(assertion, assessment)
		assessment.Reasons = normalizeStrings(assessment.Reasons)
		assessments[assertion.ID] = assessment
	}

	out := Projection{
		Contract: projectionContract,
		Schema:   projectionSchema,
		Context:  context,
		AuthorityLimits: []string{
			"assessment-does-not-assert-objective-truth",
			"assessment-does-not-grant-canonical-acceptance",
			"assessment-does-not-grant-write-or-runtime-authority",
			"meaning-fingerprint-is-input-not-semantic-proof",
			"target-anchor-resolution-does-not-prove-semantic-identity",
			"coverage-mapping-is-input-not-semantic-completeness-proof",
			"resolution-coverage-is-operational-readiness-not-truth-ratio",
			"content-fingerprint-does-not-restore-review-history",
			"partial-transport-health-is-not-project-wide-health",
			"review-history-is-bounded-by-context-as-of",
			"review-binding-includes-context-kind-and-target-identity",
			"review-binding-includes-validity-window",
			"reviewer-candidate-is-not-review-authority",
		},
	}
	for _, assertion := range corpus.Assertions {
		out.Assessments = append(out.Assessments, assessments[assertion.ID])
	}
	out.Health = buildHealth(corpus, assertions, assessments, context)
	out.ReviewRequests = buildReviewRequests(corpus, assertions, assessments, context, out.Health)
	return out, nil
}

func baseAssessment(assertion Assertion, context ContextQuery, asOf time.Time) Assessment {
	history := assessReviewHistory(assertion, asOf)
	assessment := Assessment{
		AssertionID:           assertion.ID,
		MeaningFingerprint:    assertion.MeaningFingerprint,
		ReviewSubjectFingerprint: reviewSubjectFingerprint(assertion),
		ContentOrigin:         assertion.Origin,
		IngestedBy:            assertion.IngestedBy,
		MetadataState:         assertion.MetadataState,
		StatementKind:         assertion.Mode,
		ConfirmationState:     history.ConfirmationState,
		ReviewBinding:         history.ReviewBinding,
		ReviewEventCount:      history.VisibleEventCount,
		ReviewLogSequence:     history.LastVisibleSequence,
		LatestReviewEventID:   history.LatestEventID,
		LatestReviewEventKind: history.LatestEventKind,
		ReviewConditions:      append([]string{}, history.Conditions...),
		TargetBinding:         targetBinding(assertion.Target, context),
		ContextCompatibility:  contextCompatibility(assertion.Context, context.Bindings),
		Freshness:             freshness(assertion, asOf),
		LifecycleState:        "active",
		Conflict:              "none",
	}
	if history.StaleEventCount > 0 {
		assessment.Reasons = append(assessment.Reasons, "STALE_REVIEW_HISTORY_PRESENT")
	}
	switch assertion.MetadataState {
	case MetadataPartial:
		assessment.Reasons = append(assessment.Reasons, "EPISTEMIC_METADATA_PARTIAL")
	case MetadataAbsent:
		assessment.Reasons = append(assessment.Reasons, "EPISTEMIC_METADATA_ABSENT")
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
	LatestEventID       string
	LatestEventKind     ReviewEventKind
	Conditions          []string
	VisibleEventCount   int
	LastVisibleSequence int
	StaleEventCount     int
}

func assessReviewHistory(assertion Assertion, asOf time.Time) reviewHistoryAssessment {
	result := reviewHistoryAssessment{
		ConfirmationState: "unreviewed",
		ReviewBinding:     "none",
	}
	var latest *ReviewEvent
	for i := range assertion.ReviewEvents {
		event := &assertion.ReviewEvents[i]
		occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
		if err != nil {
			continue
		}
		if occurredAt.After(asOf) {
			continue
		}
		result.VisibleEventCount++
		if event.Sequence > result.LastVisibleSequence {
			result.LastVisibleSequence = event.Sequence
		}
		if !reviewEventMatchesAssertion(*event, assertion) {
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


func reviewEventMatchesAssertion(event ReviewEvent, assertion Assertion) bool {
	if event.MeaningFingerprint != assertion.MeaningFingerprint ||
		event.StatementKind != assertion.Mode ||
		event.TargetIdentity.Kind != assertion.Target.Kind ||
		event.TargetIdentity.ID != assertion.Target.ID ||
		event.ValidFrom != assertion.ValidFrom ||
		event.ValidUntil != assertion.ValidUntil ||
		len(event.Context) != len(assertion.Context) {
		return false
	}
	for i := range event.Context {
		if event.Context[i] != assertion.Context[i] {
			return false
		}
	}
	return true
}

func targetBinding(target TargetRef, context ContextQuery) string {
	if target.Anchor == nil {
		return "not-required"
	}
	var surface *TargetSurface
	for i := range context.TargetSurfaces {
		if context.TargetSurfaces[i].SourceRef == target.Anchor.SourceRef {
			surface = &context.TargetSurfaces[i]
			break
		}
	}
	if surface == nil {
		return "detached"
	}

	anchor := target.Anchor
	if anchor.StructuralPath != "" {
		count := 0
		for _, segment := range surface.Segments {
			if segment.StructuralPath != anchor.StructuralPath {
				continue
			}
			if anchor.Quote == nil {
				count++
			} else {
				count += countTextQuoteMatches(segment.Text, *anchor.Quote)
			}
		}
		if count == 1 {
			if surface.RevisionFingerprint == anchor.SourceRevisionFingerprint {
				return "exact"
			}
			return "reanchored"
		}
		if count > 1 {
			return "ambiguous"
		}
	}

	if anchor.Quote != nil {
		count := 0
		for _, segment := range surface.Segments {
			count += countTextQuoteMatches(segment.Text, *anchor.Quote)
		}
		if count == 1 {
			if surface.RevisionFingerprint == anchor.SourceRevisionFingerprint {
				return "exact"
			}
			return "reanchored"
		}
		if count > 1 {
			return "ambiguous"
		}
	}
	return "detached"
}

func countTextQuoteMatches(text string, quote TextQuoteSelector) int {
	if quote.Exact == "" {
		return 0
	}
	count := 0
	for start := 0; start <= len(text); {
		offset := strings.Index(text[start:], quote.Exact)
		if offset < 0 {
			break
		}
		index := start + offset
		end := index + len(quote.Exact)
		prefixOK := quote.Prefix == "" || strings.HasSuffix(text[:index], quote.Prefix)
		suffixOK := quote.Suffix == "" || strings.HasPrefix(text[end:], quote.Suffix)
		if prefixOK && suffixOK {
			count++
		}
		start = index + 1
	}
	return count
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

func conflictState(conflictingIDs []string, assessments map[string]Assessment) string {
	if len(conflictingIDs) == 0 {
		return "none"
	}
	active := false
	for _, id := range conflictingIDs {
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
	switch assessment.TargetBinding {
	case "ambiguous":
		return "review-required", append(reasons, "TARGET_ANCHOR_AMBIGUOUS")
	case "detached":
		return "review-required", append(reasons, "TARGET_ANCHOR_DETACHED")
	}
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
	if assertion.MetadataState == MetadataPartial {
		return "review-required", append(reasons, "EPISTEMIC_METADATA_INCOMPLETE")
	}
	if assertion.MetadataState == MetadataAbsent {
		return "candidate-only", append(reasons, "EPISTEMIC_METADATA_INCOMPLETE")
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


func resolutionCoverageEligible(assertion Assertion, assessment Assessment) bool {
	if !assertion.Mode.canBeWorkingPremise() ||
		assertion.MetadataState != MetadataComplete ||
		assessment.LifecycleState != "active" ||
		assessment.ContextCompatibility != "compatible" ||
		assessment.Freshness == "stale" ||
		assessment.Freshness == "not-yet-valid" ||
		assessment.TargetBinding == "ambiguous" ||
		assessment.TargetBinding == "detached" {
		return false
	}
	return true
}

func resolutionCoverageResolved(assertion Assertion, assessment Assessment) bool {
	if assessment.ConfirmationState == "rejected" {
		return true
	}
	if assessment.ConfirmationState == "confirmed" && assessment.UseState == "premise" {
		return true
	}
	return assertion.Origin == OriginUser &&
		assertion.IngestedBy == IngestedByUser &&
		assessment.UseState == "premise"
}

func assessResolutionCoverage(
	corpus Corpus,
	assertions map[string]Assertion,
	assessments map[string]Assessment,
) ResolutionCoverage {
	out := ResolutionCoverage{
		MinimumRatio:              corpus.HealthPolicy.MinResolvedRatio,
		MinimumEligibleAssertions: corpus.HealthPolicy.MinimumEligibleAssertions,
		State:                     "not-configured",
	}
	for id, assertion := range assertions {
		assessment := assessments[id]
		if !resolutionCoverageEligible(assertion, assessment) {
			continue
		}
		out.Eligible++
		switch {
		case assessment.ConfirmationState == "confirmed" && assessment.UseState == "premise":
			out.Confirmed++
			out.Resolved++
		case assessment.ConfirmationState == "rejected":
			out.Rejected++
			out.Resolved++
		case assertion.Origin == OriginUser &&
			assertion.IngestedBy == IngestedByUser &&
			assessment.UseState == "premise":
			out.NativePremise++
			out.Resolved++
		default:
			out.Unresolved++
		}
	}
	if out.Eligible > 0 {
		out.Ratio = float64(out.Resolved) / float64(out.Eligible)
	}

	if corpus.MetadataState != MetadataComplete {
		out.State = "unknown"
		out.Reasons = []string{"RESOLUTION_COVERAGE_METADATA_INCOMPLETE"}
		return out
	}
	if out.MinimumRatio <= 0 {
		return out
	}
	minimum := out.MinimumEligibleAssertions
	if minimum <= 0 {
		minimum = 1
	}
	if out.Eligible < minimum {
		out.State = "insufficient-sample"
		out.Reasons = []string{"RESOLUTION_COVERAGE_SAMPLE_TOO_SMALL"}
		return out
	}
	if out.Ratio >= out.MinimumRatio {
		out.State = "healthy"
		return out
	}
	out.State = "attention"
	required := int(math.Ceil(out.MinimumRatio * float64(out.Eligible)))
	if required > out.Eligible {
		required = out.Eligible
	}
	out.RequiredResolutions = required - out.Resolved
	if out.RequiredResolutions < 0 {
		out.RequiredResolutions = 0
	}
	out.Reasons = []string{"PROJECT_RESOLUTION_RATIO_BELOW_POLICY"}
	return out
}

func highestDependencyCriticalityByAssertion(corpus Corpus) map[string]Criticality {
	out := map[string]Criticality{}
	for _, dependency := range corpus.Dependencies {
		current, ok := out[dependency.AssertionID]
		if !ok || criticalityRank(dependency.Criticality) > criticalityRank(current) {
			out[dependency.AssertionID] = dependency.Criticality
		}
	}
	return out
}

func buildHealth(
	corpus Corpus,
	assertions map[string]Assertion,
	assessments map[string]Assessment,
	context ContextQuery,
) KnowledgeHealth {
	health := KnowledgeHealth{
		Level:             "healthy",
		ViewMetadataState: corpus.MetadataState,
	}
	asOf, _ := time.Parse(time.RFC3339, context.AsOf)
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

	health.ResolutionCoverage = assessResolutionCoverage(corpus, assertions, assessments)
	if health.ResolutionCoverage.State == "attention" && health.Level == "healthy" {
		health.Level = "attention"
	}

	for _, dependency := range corpus.Dependencies {
		assessment := assessments[dependency.AssertionID]
		useState, consumerFreshness, reasons := assessDependencyUse(
			dependency,
			assertions[dependency.AssertionID],
			assessment,
			asOf,
		)
		if useState == "premise" {
			continue
		}
		issue := HealthIssue{
			AssertionID:                 dependency.AssertionID,
			Consumer:                    dependency.Consumer,
			Criticality:                 dependency.Criticality,
			UseState:                    useState,
			ConsumerFreshness:           consumerFreshness,
			FreshnessRequirementSeconds: dependency.MaxAgeSeconds,
			Reasons:                     reasons,
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
	health.Coverage = assessCoverage(corpus, assessments, context)
	for _, area := range health.Coverage.Areas {
		if area.State == "mapped" {
			continue
		}
		switch area.Criticality {
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

func assessDependencyUse(
	dependency Dependency,
	assertion Assertion,
	assessment Assessment,
	asOf time.Time,
) (string, string, []string) {
	useState := assessment.UseState
	reasons := append([]string{}, assessment.Reasons...)
	consumerFreshness := "not-required"
	if dependency.MaxAgeSeconds <= 0 {
		return useState, consumerFreshness, normalizeStrings(reasons)
	}

	if assertion.CheckedAt != "" &&
		assertion.CheckedSubjectFingerprint != freshnessSubjectFingerprint(assertion) {
		reasons = append(reasons, "CHECK_BINDING_STALE")
	}
	consumerFreshness = consumerFreshnessState(assertion, asOf, dependency.MaxAgeSeconds)
	switch consumerFreshness {
	case "stale":
		useState = "blocked"
		reasons = append(reasons, "CONSUMER_FRESHNESS_STALE")
	case "unknown":
		if useState == "premise" {
			useState = "review-required"
		}
		reasons = append(reasons, "CONSUMER_FRESHNESS_UNKNOWN")
	}
	return useState, consumerFreshness, normalizeStrings(reasons)
}

func consumerFreshnessState(assertion Assertion, asOf time.Time, maxAgeSeconds int) string {
	if maxAgeSeconds <= 0 {
		return "not-required"
	}
	var latest time.Time
	found := false
	if assertion.CheckedAt != "" &&
		assertion.CheckedSubjectFingerprint == freshnessSubjectFingerprint(assertion) {
		if checkedAt, err := time.Parse(time.RFC3339, assertion.CheckedAt); err == nil && !checkedAt.After(asOf) {
			latest = checkedAt
			found = true
		}
	}
	for _, event := range assertion.ReviewEvents {
		if event.Kind != ReviewEventConfirm ||
			!reviewEventMatchesAssertion(event, assertion) {
			continue
		}
		occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
		if err != nil || occurredAt.After(asOf) {
			continue
		}
		if !found || occurredAt.After(latest) {
			latest = occurredAt
			found = true
		}
	}
	if !found {
		return "unknown"
	}
	if asOf.Sub(latest).Seconds() > float64(maxAgeSeconds) {
		return "stale"
	}
	return "current"
}

func assessCoverage(
	corpus Corpus,
	assessments map[string]Assessment,
	context ContextQuery,
) CoverageAssessment {
	out := CoverageAssessment{}
	for _, area := range corpus.CoverageAreas {
		result := CoverageAreaAssessment{
			AreaID:             area.ID,
			Target:             area.Target,
			Criticality:        area.Criticality,
			State:              "mapped",
			TargetBinding:      targetBinding(area.Target, context),
			MappedAssertionIDs: append([]string{}, area.MappedAssertionIDs...),
		}
		out.Counts.TotalAreas++
		if result.TargetBinding == "ambiguous" {
			result.State = "gap"
			result.Reasons = append(result.Reasons, "COVERAGE_TARGET_AMBIGUOUS")
		} else if result.TargetBinding == "detached" {
			result.State = "gap"
			result.Reasons = append(result.Reasons, "COVERAGE_TARGET_DETACHED")
		} else if len(area.MappedAssertionIDs) == 0 {
			result.State = "gap"
			result.Reasons = append(result.Reasons, "COVERAGE_UNMAPPED")
		} else {
			resolved := 0
			for _, id := range area.MappedAssertionIDs {
				assessment := assessments[id]
				if assessment.TargetBinding != "ambiguous" && assessment.TargetBinding != "detached" {
					resolved++
				}
			}
			if resolved == 0 {
				result.State = "gap"
				result.Reasons = append(result.Reasons, "COVERAGE_MAPPED_TARGETS_UNRESOLVED")
			}
		}
		if result.State == "mapped" {
			out.Counts.MappedAreas++
		} else {
			out.Counts.GapAreas++
		}
		result.Reasons = normalizeStrings(result.Reasons)
		out.Areas = append(out.Areas, result)
	}
	sort.Slice(out.Areas, func(i, j int) bool {
		ri, rj := criticalityRank(out.Areas[i].Criticality), criticalityRank(out.Areas[j].Criticality)
		if ri != rj {
			return ri > rj
		}
		return out.Areas[i].AreaID < out.Areas[j].AreaID
	})
	return out
}

func buildReviewRequests(
	corpus Corpus,
	assertions map[string]Assertion,
	assessments map[string]Assessment,
	context ContextQuery,
	health KnowledgeHealth,
) []ReviewRequest {
	impact := map[string][]string{}
	critical := map[string]bool{}
	consumerReviewReasons := map[string][]string{}
	asOf, _ := time.Parse(time.RFC3339, context.AsOf)
	for _, dependency := range corpus.Dependencies {
		impact[dependency.AssertionID] = append(impact[dependency.AssertionID], fmt.Sprintf("%s (%s)", dependency.Consumer, dependency.Criticality))
		if dependency.Criticality == CriticalityCritical || dependency.Criticality == CriticalityHigh {
			critical[dependency.AssertionID] = true
			useState, _, reasons := assessDependencyUse(
				dependency,
				assertions[dependency.AssertionID],
				assessments[dependency.AssertionID],
				asOf,
			)
			if useState != "premise" {
				consumerReviewReasons[dependency.AssertionID] = append(
					consumerReviewReasons[dependency.AssertionID],
					reasons...,
				)
			}
		}
	}
	ratioRequested := map[string]bool{}
	if health.ResolutionCoverage.State == "attention" && health.ResolutionCoverage.RequiredResolutions > 0 {
		type candidate struct {
			id   string
			rank int
		}
		ranks := highestDependencyCriticalityByAssertion(corpus)
		var candidates []candidate
		for id, assessment := range assessments {
			assertion := assertions[id]
			if !resolutionCoverageEligible(assertion, assessment) ||
				resolutionCoverageResolved(assertion, assessment) {
				continue
			}
			candidates = append(candidates, candidate{
				id:   id,
				rank: criticalityRank(ranks[id]),
			})
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].rank != candidates[j].rank {
				return candidates[i].rank > candidates[j].rank
			}
			return candidates[i].id < candidates[j].id
		})
		limit := health.ResolutionCoverage.RequiredResolutions
		if limit > len(candidates) {
			limit = len(candidates)
		}
		for _, candidate := range candidates[:limit] {
			ratioRequested[candidate.id] = true
		}
	}

	var out []ReviewRequest
	for id, assessment := range assessments {
		consumerReasons := normalizeStrings(consumerReviewReasons[id])
		if assessment.UseState != "review-required" &&
			!(critical[id] && (assessment.UseState != "premise" || len(consumerReasons) > 0)) &&
			!ratioRequested[id] {
			continue
		}
		assertion := assertions[id]
		out = append(out, ReviewRequest{
			AssertionID:        id,
			Target:             assertion.Target,
			ConfirmationState:  assessment.ConfirmationState,
			ReviewBinding:      assessment.ReviewBinding,
			TargetBinding:      assessment.TargetBinding,
			Context:            append([]ScopeBinding{}, assertion.Context...),
			Evidence:           append([]EvidenceRef{}, assertion.Evidence...),
			Reasons: normalizeStrings(append(
				append(
					append([]string{}, assessment.Reasons...),
					consumerReasons...,
				),
				func() []string {
					if ratioRequested[id] {
						return []string{"PROJECT_RESOLUTION_RATIO_BELOW_POLICY"}
					}
					return nil
				}()...,
			)),
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
