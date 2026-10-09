package knowledgecontext

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/index"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestBuildBuilderReviewHandoffQualifiesCurrentContextBeforeDisclosure(t *testing.T) {
	ctx, sources := builderReviewHandoffApplicationFixture("sha256:pattern", "sha256:dependency")

	got, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
			RequiredDependencies: []DependencyRequirement{
				{
					Direction: "outgoing",
					Relation:  "contract-query",
					Kind:      "database-contract-query",
					ID:        "master_data.customer/find_by_id",
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildBuilderReviewPlan(ctx)
	if got.ProjectionRevision != builderReviewHandoffProjectionRevision ||
		got.Purpose != "design-change" ||
		got.Subject != ctx.Subject ||
		got.ReviewPlanProjectionRevision != plan.ProjectionRevision ||
		got.ReviewPlanFingerprint != plan.Fingerprint ||
		got.RoutingScope != plan.RoutingScope ||
		got.Status != "qualified-for-disclosure" ||
		got.Fingerprint == "" {
		t.Fatalf("builder review handoff identity drifted: %+v", got)
	}
	if !reflect.DeepEqual(got.SelectedSections, plan.RecommendedDisclosureSections) {
		t.Fatalf("handoff selection drifted: got=%+v want=%+v", got.SelectedSections, plan.RecommendedDisclosureSections)
	}
	if got.Qualification.Status != QualificationApplicable ||
		len(got.Qualification.Evidence) != 1 ||
		len(got.Qualification.DependencyEvidence) != 1 {
		t.Fatalf("qualified handoff lost exact Evidence: %+v", got.Qualification)
	}
	for _, limit := range []string{
		"builder-review-handoff-is-derived-not-semantic-authority",
		"selected-section-is-not-disclosed-knowledge",
		"qualification-applicable-is-not-acceptance",
		"qualification-applicable-is-not-write-or-runtime-authority",
		"qualified-for-disclosure-does-not-prove-selected-knowledge-complete",
		"qualified-for-disclosure-does-not-prove-section-local-evidence-complete",
		"handoff-does-not-materialize-knowledge",
		"current-disclosure-must-be-built-after-handoff",
		"disclosure-fingerprint-not-handoff-fingerprint-governs-disclosed-content-reuse",
	} {
		if !containsBuilderReviewString(got.AuthorityLimits, limit) {
			t.Fatalf("builder review handoff authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
	}
}

func TestBuildBuilderReviewHandoffKeepsUnknownQualificationBlocked(t *testing.T) {
	ctx := Context{
		Subject:    subject.Subject{Kind: "semantic-data", ID: "missing.example"},
		Unresolved: []string{"source-reference-unavailable"},
	}
	got, err := BuildBuilderReviewHandoff(
		ctx,
		sourceindex.Index{},
		QualificationRequirements{ExpectedKind: "semantic-data"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "blocked-unresolved" ||
		got.Qualification.Status != QualificationUnknown ||
		len(got.Qualification.Reasons) == 0 {
		t.Fatalf("unknown qualification became usable: %+v", got)
	}
}

func TestBuildBuilderReviewHandoffKeepsIncompatibleQualificationBlocked(t *testing.T) {
	ctx, sources := builderReviewHandoffApplicationFixture("sha256:pattern", "sha256:dependency")
	got, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "repair_intake",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "blocked-incompatible" ||
		got.Qualification.Status != QualificationIncompatible {
		t.Fatalf("incompatible qualification became usable: %+v", got)
	}
}

func TestBuilderReviewHandoffCompositionRejectsUnsubstantiatedApplicableQualification(t *testing.T) {
	ctx, _ := builderReviewHandoffApplicationFixture("sha256:pattern", "sha256:dependency")
	plan := BuildBuilderReviewPlan(ctx)

	_, err := buildBuilderReviewHandoffFromQualification(plan, QualificationResult{
		Candidate: plan.Subject,
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "application-pattern",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "without authoritative Evidence") {
		t.Fatalf("applicable qualification without Evidence was not rejected: %v", err)
	}

	_, err = buildBuilderReviewHandoffFromQualification(plan, QualificationResult{
		Candidate: plan.Subject,
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "application-pattern",
			RequiredDependencies: []DependencyRequirement{
				{
					Direction: "outgoing",
					Relation:  "contract-query",
					Kind:      "database-contract-query",
					ID:        "master_data.customer/find_by_id",
				},
			},
		},
		Evidence: []QualificationEvidence{
			{
				SubjectKind:       "application-pattern",
				Subject:           "order_entry.customer_reference",
				Source:            "applications/order-entry/patterns/customer-reference.tei",
				SourceFingerprint: "sha256:pattern",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "without dependency Evidence") {
		t.Fatalf("applicable qualification without dependency Evidence was not rejected: %v", err)
	}
}

func TestBuilderReviewHandoffCompositionRejectsMismatchedOrModifiedPlan(t *testing.T) {
	ctx, _ := builderReviewHandoffApplicationFixture("sha256:pattern", "sha256:dependency")
	plan := BuildBuilderReviewPlan(ctx)

	_, err := buildBuilderReviewHandoffFromQualification(plan, QualificationResult{
		Candidate: subject.Subject{Kind: "application-pattern", ID: "other.pattern"},
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "application-pattern",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "subject mismatch") {
		t.Fatalf("mismatched qualification subject was not rejected: %v", err)
	}

	modified := plan
	modified.ReviewSequence = append([]string(nil), plan.ReviewSequence...)
	modified.ReviewSequence[0] = "modified"
	_, err = buildBuilderReviewHandoffFromQualification(modified, QualificationResult{
		Candidate: modified.Subject,
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "application-pattern",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "modified Builder Review Plan") {
		t.Fatalf("modified review plan was not rejected: %v", err)
	}
}

func TestBuildBuilderReviewHandoffRejectsMixedContextAndSourceIndexRevisions(t *testing.T) {
	ctx, sources := builderReviewHandoffApplicationFixture("sha256:context-pattern", "sha256:dependency")
	for i := range sources.Artifacts {
		if sources.Artifacts[i].Path == "applications/order-entry/patterns/customer-reference.tei" {
			sources.Artifacts[i].Fingerprint = "sha256:source-index-pattern"
		}
	}

	_, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "mixed Context and Source Index revisions") {
		t.Fatalf("mixed Context / Source Index revisions were not rejected: %v", err)
	}
}

func TestBuilderReviewHandoffFingerprintTracksQualificationEvidenceRevision(t *testing.T) {
	ctx, firstSources := builderReviewHandoffApplicationFixture("sha256:pattern", "sha256:dependency-first")
	requirements := QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []DependencyRequirement{
			{
				Direction: "outgoing",
				Relation:  "contract-query",
				Kind:      "database-contract-query",
				ID:        "master_data.customer/find_by_id",
			},
		},
	}
	first, err := BuildBuilderReviewHandoff(ctx, firstSources, requirements)
	if err != nil {
		t.Fatal(err)
	}

	_, secondSources := builderReviewHandoffApplicationFixture("sha256:pattern", "sha256:dependency-second")
	second, err := BuildBuilderReviewHandoff(ctx, secondSources, requirements)
	if err != nil {
		t.Fatal(err)
	}
	if first.ReviewPlanFingerprint != second.ReviewPlanFingerprint {
		t.Fatalf("dependency Evidence revision changed routing identity: first=%s second=%s", first.ReviewPlanFingerprint, second.ReviewPlanFingerprint)
	}
	if first.Fingerprint == second.Fingerprint {
		t.Fatalf("dependency Evidence revision did not change handoff fingerprint: %s", first.Fingerprint)
	}
}

func TestBuilderReviewHandoffIsNotAUniversalContextPacket(t *testing.T) {
	typ := reflect.TypeOf(BuilderReviewHandoff{})
	for _, forbidden := range []string{
		"Payload",
		"Prompt",
		"AllKnowledge",
		"Disclosed",
		"CanonicalMeaning",
		"AcceptedDecision",
		"Authorized",
		"CanExecute",
		"NextAction",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("Builder Review Handoff introduced forbidden universal/authority field %q", forbidden)
		}
	}
}

func builderReviewHandoffApplicationFixture(
	patternFingerprint,
	dependencyFingerprint string,
) (Context, sourceindex.Index) {
	pattern := subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"}
	dependency := subject.Subject{
		Kind: "database-contract-query",
		ID:   "master_data.customer/find_by_id",
	}
	patternPath := "applications/order-entry/patterns/customer-reference.tei"
	dependencyPath := "database-contracts/master_data/customer.tei"

	ctx := Context{
		Subject: pattern,
		References: []sourceindex.Reference{
			{
				Subject:      pattern,
				Role:         sourceindex.RoleDefinition,
				ArtifactPath: patternPath,
			},
		},
		Artifacts: []sourceindex.Artifact{
			{Path: patternPath, Fingerprint: patternFingerprint},
		},
		ApplicationPattern: &index.ApplicationPatternIndexEntry{
			Application: "order_entry",
		},
		Outgoing: []Edge{
			{
				From:     pattern,
				Relation: "contract-query",
				To:       dependency,
			},
		},
	}
	sources := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{
			{Path: patternPath, Fingerprint: patternFingerprint},
			{Path: dependencyPath, Fingerprint: dependencyFingerprint},
		},
		References: []sourceindex.Reference{
			{
				Subject:      pattern,
				Role:         sourceindex.RoleDefinition,
				ArtifactPath: patternPath,
			},
			{
				Subject:      dependency,
				Role:         sourceindex.RoleDefinition,
				ArtifactPath: dependencyPath,
			},
		},
	}
	return ctx, sources
}
