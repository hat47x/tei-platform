package knowledgecontext

import (
	"reflect"
	"testing"

	"tei/internal/index"
	"tei/internal/subject"
)

func TestBuildBuilderReviewPlanRoutesAvailableKnowledgeInReviewOrder(t *testing.T) {
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

	got := BuildBuilderReviewPlan(ctx)

	if got.ProjectionRevision != builderReviewPlanProjectionRevision ||
		got.Purpose != "design-change" {
		t.Fatalf("builder review plan identity drifted: %+v", got)
	}
	if got.Subject != ctx.Subject {
		t.Fatalf("builder review subject drifted: got=%+v want=%+v", got.Subject, ctx.Subject)
	}
	if got.Fingerprint == "" {
		t.Fatalf("builder review plan fingerprint is missing: %+v", got)
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
		"refresh-review-plan-before-reusing-cached-routing",
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
		"review-plan-fingerprint-is-routing-identity-not-knowledge-freshness-proof",
		"review-plan-fingerprint-is-not-selected-knowledge-fingerprint",
	} {
		if !containsBuilderReviewString(got.AuthorityLimits, limit) {
			t.Fatalf("builder review authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
	}
}

func TestBuildBuilderReviewPlanKeepsSourceGapVisible(t *testing.T) {
	got := BuildBuilderReviewPlan(Context{
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
	if !reflect.DeepEqual(got.Unresolved, []string{"source-reference-unavailable"}) {
		t.Fatalf("source gap was lost from review plan: %+v", got.Unresolved)
	}
}

func TestBuilderReviewPlanFingerprintTracksRoutingNotKnowledgeContent(t *testing.T) {
	base := Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
		DataDesign: &DataDesignContext{
			ProjectionRevision: dataDesignProjectionRevision,
			Data:               "sales.order_line",
		},
		Unresolved: []string{"z-gap", "a-gap"},
	}
	first := BuildBuilderReviewPlan(base)

	reordered := base
	reordered.Unresolved = []string{"a-gap", "z-gap"}
	second := BuildBuilderReviewPlan(reordered)
	if first.Fingerprint != second.Fingerprint {
		t.Fatalf("unresolved ordering changed routing identity: first=%s second=%s", first.Fingerprint, second.Fingerprint)
	}

	changedContent := base
	changedContent.DataDesign = &DataDesignContext{
		ProjectionRevision: dataDesignProjectionRevision,
		Data:               "sales.changed",
	}
	third := BuildBuilderReviewPlan(changedContent)
	if first.Fingerprint != third.Fingerprint {
		t.Fatalf("selected knowledge content changed routing identity: first=%s third=%s", first.Fingerprint, third.Fingerprint)
	}

	changedAvailability := base
	changedAvailability.SemanticElement = &index.SemanticElementEntry{
		Kind: "data",
		ID:   "sales.order_line",
	}
	fourth := BuildBuilderReviewPlan(changedAvailability)
	if first.Fingerprint == fourth.Fingerprint {
		t.Fatalf("new review section did not change routing identity: %s", fourth.Fingerprint)
	}

	changedGap := base
	changedGap.Unresolved = []string{"a-gap", "different-gap"}
	fifth := BuildBuilderReviewPlan(changedGap)
	if first.Fingerprint == fifth.Fingerprint {
		t.Fatalf("changed unresolved routing state kept fingerprint: %s", fifth.Fingerprint)
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
