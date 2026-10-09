package knowledgecontext

import "sort"

const builderReviewPlanProjectionRevision = "tei.knowledge.builder-review-plan/v1"

type BuilderReviewPlan struct {
	ProjectionRevision            string   `json:"projectionRevision"`
	Purpose                       string   `json:"purpose"`
	RecommendedDisclosureSections []string `json:"recommendedDisclosureSections,omitempty"`
	ReviewSequence                []string `json:"reviewSequence"`
	StopRules                     []string `json:"stopRules"`
	AuthorityLimits               []string `json:"authorityLimits"`
}

var builderReviewSectionOrder = []string{
	"source",
	"definition",
	"data-design",
	"dependencies",
}

func buildBuilderReviewPlan(ctx Context) BuilderReviewPlan {
	available := availableKnowledgeSections(ctx)
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
		"inspect-unresolved-and-authority-limits",
		"form-proposal-without-inventing-missing-meaning",
		"route-authoritative-change-through-shared-acceptance",
	)

	return BuilderReviewPlan{
		ProjectionRevision:            builderReviewPlanProjectionRevision,
		Purpose:                       "design-change",
		RecommendedDisclosureSections: recommended,
		ReviewSequence:                reviewSequence,
		StopRules: []string{
			"stop-when-required-evidence-or-authority-is-unresolved",
			"do-not-treat-omitted-or-unavailable-knowledge-as-negative-evidence",
			"do-not-promote-derived-candidate-or-guidance-to-accepted-meaning",
			"refresh-selected-knowledge-before-reusing-cached-design-context",
		},
		AuthorityLimits: []string{
			"builder-review-plan-is-derived-routing-guidance-not-semantic-authority",
			"recommended-section-is-not-semantic-applicability",
			"review-order-does-not-grant-write-or-runtime-authority",
			"review-plan-does-not-replace-qualification-or-shared-acceptance",
		},
	}
}

func availableKnowledgeSections(ctx Context) []string {
	var result []string
	if len(ctx.Artifacts) != 0 || len(ctx.References) != 0 {
		result = append(result, "source")
	}
	if ctx.SemanticElement != nil || ctx.ApplicationPattern != nil {
		result = append(result, "definition")
	}
	if ctx.DataDesign != nil {
		result = append(result, "data-design")
	}
	if len(ctx.Outgoing) != 0 || len(ctx.Incoming) != 0 {
		result = append(result, "dependencies")
	}
	sort.Strings(result)
	return result
}

func cloneBuilderReviewPlan(value BuilderReviewPlan) BuilderReviewPlan {
	value.RecommendedDisclosureSections = append([]string(nil), value.RecommendedDisclosureSections...)
	value.ReviewSequence = append([]string(nil), value.ReviewSequence...)
	value.StopRules = append([]string(nil), value.StopRules...)
	value.AuthorityLimits = append([]string(nil), value.AuthorityLimits...)
	return value
}
