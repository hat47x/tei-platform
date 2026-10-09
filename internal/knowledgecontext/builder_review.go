package knowledgecontext

import (
	"encoding/json"
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const builderReviewPlanProjectionRevision = "tei.knowledge.builder-review-plan/v1"

type BuilderReviewPlan struct {
	ProjectionRevision            string          `json:"projectionRevision"`
	Fingerprint                   string          `json:"fingerprint"`
	Purpose                       string          `json:"purpose"`
	RoutingScope                  string          `json:"routingScope"`
	Subject                       subject.Subject `json:"subject"`
	RecommendedDisclosureSections []string        `json:"recommendedDisclosureSections,omitempty"`
	ReviewSequence                []string        `json:"reviewSequence"`
	Unresolved                    []string        `json:"unresolved,omitempty"`
	StopRules                     []string        `json:"stopRules"`
	AuthorityLimits               []string        `json:"authorityLimits"`
}

var builderReviewSectionOrder = []string{
	"source",
	"definition",
	"data-design",
	"dependencies",
}

func BuildBuilderReviewPlan(ctx Context) BuilderReviewPlan {
	available := availableDisclosureSections(ctx)
	availableSet := make(map[string]struct{}, len(available))
	for _, section := range available {
		availableSet[section] = struct{}{}
	}

	recommended := make([]string, 0, len(available))
	for _, section := range builderReviewSectionOrder {
		if _, ok := availableSet[section]; ok {
			recommended = append(recommended, section)
		}
	}

	reviewSequence := []string{
		"review-current-source-and-evidence",
	}
	if _, ok := availableSet["definition"]; ok {
		reviewSequence = append(reviewSequence, "review-current-definition")
	}
	if _, ok := availableSet["data-design"]; ok {
		reviewSequence = append(reviewSequence, "review-data-design-and-its-review-protocol")
	}
	if _, ok := availableSet["dependencies"]; ok {
		reviewSequence = append(reviewSequence, "review-dependencies-and-impact")
	}
	reviewSequence = append(
		reviewSequence,
		"apply-required-knowledge-qualification-before-semantic-use",
		"inspect-selected-section-local-resolution-and-evidence-status",
		"inspect-unresolved-and-authority-limits",
		"form-minimum-semantic-delta-proposal-without-inventing-missing-meaning",
		"route-authoritative-change-through-shared-acceptance",
	)

	unresolved := append([]string(nil), ctx.Unresolved...)
	sort.Strings(unresolved)

	result := BuilderReviewPlan{
		ProjectionRevision:            builderReviewPlanProjectionRevision,
		Purpose:                       "design-change",
		RoutingScope:                  "current-knowledge-context",
		Subject:                       ctx.Subject,
		RecommendedDisclosureSections: recommended,
		ReviewSequence:                reviewSequence,
		Unresolved:                    unresolved,
		StopRules: []string{
			"stop-when-required-evidence-or-authority-is-unresolved",
			"do-not-treat-omitted-or-unavailable-knowledge-as-negative-evidence",
			"do-not-promote-derived-candidate-or-guidance-to-accepted-meaning",
			"do-not-copy-disclosed-knowledge-snapshot-into-proposal-meaning",
			"do-not-treat-recommended-section-as-qualified-or-disclosed-knowledge",
			"do-not-treat-context-unresolved-as-exhaustive-of-section-local-uncertainty",
			"stop-semantic-use-when-selected-section-local-status-is-unresolved",
			"do-not-infer-global-knowledge-absence-from-current-context-section-absence",
			"refresh-review-plan-before-reusing-cached-routing",
		},
		AuthorityLimits: []string{
			"builder-review-plan-is-derived-routing-guidance-not-semantic-authority",
			"recommended-section-is-not-semantic-applicability",
			"recommended-section-is-not-knowledge-qualification",
			"review-order-does-not-grant-write-or-runtime-authority",
			"review-plan-does-not-replace-qualification-or-shared-acceptance",
			"proposal-delta-remains-separate-from-knowledge-evidence",
			"recommended-sections-cover-current-knowledge-context-not-all-builder-knowledge",
			"review-plan-fingerprint-is-routing-identity-not-knowledge-freshness-proof",
			"review-plan-fingerprint-is-not-selected-knowledge-fingerprint",
		},
	}
	result.Fingerprint = builderReviewPlanFingerprint(result)
	return result
}

func builderReviewPlanFingerprint(value BuilderReviewPlan) string {
	value.Fingerprint = ""
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("Builder Review Plan contains non-JSON state: " + err.Error())
	}
	return sourceindex.Fingerprint(encoded)
}
