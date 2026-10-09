package knowledgecontext

import (
	"reflect"
	"testing"

	"tei/internal/index"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestBuilderReviewPlanRoutesAvailableKnowledgeInReviewOrder(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
		SemanticElement: &index.SemanticElementEntry{
			Kind: "data",
			ID:   "sales.order_line",
		},
		DataDesign: &DataDesignContext{
			ProjectionRevision: dataDesignProjectionRevision,
			Data:               "sales.order_line",
		},
		Outgoing: []Edge{
			{
				From:     subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
				Relation: "example",
				To:       subject.Subject{Kind: "semantic-data", ID: "sales.order"},
			},
		},
	}

	got := buildBuilderReviewPlan(ctx)

	if got.ProjectionRevision != builderReviewPlanProjectionRevision ||
		got.Purpose != "design-change" {
		t.Fatalf("builder review plan identity drifted: %+v", got)
	}
	wantSections := []string{"definition", "data-design", "dependencies"}
	if !reflect.DeepEqual(got.RecommendedDisclosureSections, wantSections) {
		t.Fatalf("recommended sections=%+v want=%+v", got.RecommendedDisclosureSections, wantSections)
	}
	wantSequence := []string{
		"review-current-source-and-evidence",
		"review-current-definition",
		"review-data-design-and-its-review-protocol",
		"review-dependencies-and-impact",
		"inspect-unresolved-and-authority-limits",
		"form-proposal-without-inventing-missing-meaning",
		"route-authoritative-change-through-shared-acceptance",
	}
	if !reflect.DeepEqual(got.ReviewSequence, wantSequence) {
		t.Fatalf("review sequence=%+v want=%+v", got.ReviewSequence, wantSequence)
	}
	for _, rule := range []string{
		"stop-when-required-evidence-or-authority-is-unresolved",
		"do-not-treat-omitted-or-unavailable-knowledge-as-negative-evidence",
		"do-not-promote-derived-candidate-or-guidance-to-accepted-meaning",
		"refresh-selected-knowledge-before-reusing-cached-design-context",
	} {
		if !containsBuilderReviewString(got.StopRules, rule) {
			t.Fatalf("builder review stop rule %q is missing: %+v", rule, got.StopRules)
		}
	}
	for _, limit := range []string{
		"builder-review-plan-is-derived-routing-guidance-not-semantic-authority",
		"recommended-section-is-not-semantic-applicability",
		"review-order-does-not-grant-write-or-runtime-authority",
		"review-plan-does-not-replace-qualification-or-shared-acceptance",
	} {
		if !containsBuilderReviewString(got.AuthorityLimits, limit) {
			t.Fatalf("builder review authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
	}
}

func TestBuildAttachesBuilderReviewPlanToKnowledgeContext(t *testing.T) {
	ctx := Build(
		dataDesignRelationUniqueWorld(),
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)

	if ctx.BuilderReviewPlan.ProjectionRevision != builderReviewPlanProjectionRevision {
		t.Fatalf("builder review plan was not attached: %+v", ctx.BuilderReviewPlan)
	}
	for _, section := range []string{"definition", "data-design"} {
		if !containsBuilderReviewString(ctx.BuilderReviewPlan.RecommendedDisclosureSections, section) {
			t.Fatalf("available section %q is missing from builder review plan: %+v", section, ctx.BuilderReviewPlan)
		}
	}
	if containsBuilderReviewString(ctx.BuilderReviewPlan.RecommendedDisclosureSections, "source") {
		t.Fatalf("unavailable source section was recommended: %+v", ctx.BuilderReviewPlan)
	}
	if len(ctx.BuilderReviewPlan.ReviewSequence) == 0 ||
		ctx.BuilderReviewPlan.ReviewSequence[0] != "review-current-source-and-evidence" {
		t.Fatalf("source review did not remain the first stage: %+v", ctx.BuilderReviewPlan.ReviewSequence)
	}
}

func TestBuilderReviewPlanKeepsSourceReviewWhenSourceKnowledgeIsUnavailable(t *testing.T) {
	got := buildBuilderReviewPlan(Context{
		Subject:    subject.Subject{Kind: "semantic-capability", ID: "sales.order.submit"},
		Unresolved: []string{"source-reference-unavailable"},
	})

	if len(got.RecommendedDisclosureSections) != 0 {
		t.Fatalf("unavailable knowledge was recommended as disclosed: %+v", got.RecommendedDisclosureSections)
	}
	if len(got.ReviewSequence) == 0 ||
		got.ReviewSequence[0] != "review-current-source-and-evidence" {
		t.Fatalf("source review disappeared when source was unavailable: %+v", got.ReviewSequence)
	}
	if !containsBuilderReviewString(
		got.StopRules,
		"stop-when-required-evidence-or-authority-is-unresolved",
	) {
		t.Fatalf("missing source did not preserve fail-closed review guidance: %+v", got.StopRules)
	}
}

func TestDisclosureRederivesBuilderReviewPlanFromCurrentKnowledge(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
		DataDesign: &DataDesignContext{
			ProjectionRevision: dataDesignProjectionRevision,
			Data:               "sales.order_line",
		},
		BuilderReviewPlan: BuilderReviewPlan{
			ProjectionRevision: "stale",
			RecommendedDisclosureSections: []string{
				"source",
			},
			ReviewSequence:  []string{"stale"},
			StopRules:       []string{"stale"},
			AuthorityLimits: []string{"stale"},
		},
	}

	got, err := BuildDisclosure(ctx, []string{"data-design"})
	if err != nil {
		t.Fatal(err)
	}
	if got.BuilderReviewPlan.ProjectionRevision != builderReviewPlanProjectionRevision ||
		!reflect.DeepEqual(
			got.BuilderReviewPlan.RecommendedDisclosureSections,
			[]string{"data-design"},
		) {
		t.Fatalf("disclosure reused stale builder review plan: %+v", got.BuilderReviewPlan)
	}
	if containsBuilderReviewString(got.BuilderReviewPlan.ReviewSequence, "stale") ||
		containsBuilderReviewString(got.BuilderReviewPlan.StopRules, "stale") ||
		containsBuilderReviewString(got.BuilderReviewPlan.AuthorityLimits, "stale") {
		t.Fatalf("stale builder review metadata leaked into disclosure: %+v", got.BuilderReviewPlan)
	}
}

func TestCloneBuilderReviewPlanDetachesSlices(t *testing.T) {
	source := buildBuilderReviewPlan(Context{
		DataDesign: &DataDesignContext{
			ProjectionRevision: dataDesignProjectionRevision,
			Data:               "sales.order_line",
		},
	})
	got := cloneBuilderReviewPlan(source)

	got.RecommendedDisclosureSections[0] = "mutated"
	got.ReviewSequence[0] = "mutated"
	got.StopRules[0] = "mutated"
	got.AuthorityLimits[0] = "mutated"
	if source.RecommendedDisclosureSections[0] == "mutated" ||
		source.ReviewSequence[0] == "mutated" ||
		source.StopRules[0] == "mutated" ||
		source.AuthorityLimits[0] == "mutated" {
		t.Fatalf("cloned builder review plan aliases source slices: source=%+v clone=%+v", source, got)
	}
}

func containsBuilderReviewString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
