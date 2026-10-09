package knowledgecontext

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestBuildBuilderReviewHandoffRequiresApplicableQualificationBeforeDisclosure(t *testing.T) {
	plan := BuildBuilderReviewPlan(Context{
		Subject: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		References: []sourceindex.Reference{
			{
				Subject:      subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
				Role:         sourceindex.RoleDefinition,
				ArtifactPath: "applications/order-entry/patterns/customer-reference.tei",
			},
		},
		Artifacts: []sourceindex.Artifact{
			{
				Path:        "applications/order-entry/patterns/customer-reference.tei",
				Fingerprint: "sha256:pattern",
			},
		},
	})
	qualification := QualificationResult{
		Candidate: plan.Subject,
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
		Evidence: []QualificationEvidence{
			{
				SubjectKind:       "application-pattern",
				Subject:           "order_entry.customer_reference",
				Source:            "applications/order-entry/patterns/customer-reference.tei",
				SourceFingerprint: "sha256:pattern",
			},
		},
		AuthorityLimits: []string{
			"qualification-is-not-acceptance",
		},
	}

	got, err := BuildBuilderReviewHandoff(plan, qualification)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectionRevision != builderReviewHandoffProjectionRevision ||
		got.Purpose != "design-change" ||
		got.Subject != plan.Subject ||
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
	for _, limit := range []string{
		"builder-review-handoff-is-derived-not-semantic-authority",
		"selected-section-is-not-disclosed-knowledge",
		"qualification-applicable-is-not-acceptance",
		"qualification-applicable-is-not-write-or-runtime-authority",
		"qualified-for-disclosure-does-not-prove-selected-knowledge-complete",
		"handoff-does-not-materialize-knowledge",
		"current-disclosure-must-be-built-after-handoff",
		"disclosure-fingerprint-not-handoff-fingerprint-governs-disclosed-content-reuse",
	} {
		if !containsBuilderReviewString(got.AuthorityLimits, limit) {
			t.Fatalf("builder review handoff authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
	}
}

func TestBuildBuilderReviewHandoffBlocksUnknownAndIncompatibleQualification(t *testing.T) {
	plan := BuildBuilderReviewPlan(Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	})

	for _, tc := range []struct {
		status QualificationStatus
		want   string
	}{
		{status: QualificationUnknown, want: "blocked-unresolved"},
		{status: QualificationIncompatible, want: "blocked-incompatible"},
	} {
		t.Run(string(tc.status), func(t *testing.T) {
			got, err := BuildBuilderReviewHandoff(plan, QualificationResult{
				Candidate: plan.Subject,
				Status:    tc.status,
				Requirements: QualificationRequirements{
					ExpectedKind: "semantic-data",
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.want {
				t.Fatalf("handoff status=%q want=%q", got.Status, tc.want)
			}
		})
	}
}

func TestBuildBuilderReviewHandoffRejectsUnsubstantiatedApplicableQualification(t *testing.T) {
	plan := BuildBuilderReviewPlan(Context{
		Subject: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		References: []sourceindex.Reference{
			{
				Subject:      subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
				Role:         sourceindex.RoleDefinition,
				ArtifactPath: "applications/order-entry/patterns/customer-reference.tei",
			},
		},
		Artifacts: []sourceindex.Artifact{
			{
				Path:        "applications/order-entry/patterns/customer-reference.tei",
				Fingerprint: "sha256:pattern",
			},
		},
	})

	_, err := BuildBuilderReviewHandoff(plan, QualificationResult{
		Candidate: plan.Subject,
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "application-pattern",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "without authoritative Evidence") {
		t.Fatalf("applicable qualification without Evidence was not rejected: %v", err)
	}

	_, err = BuildBuilderReviewHandoff(plan, QualificationResult{
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

func TestBuildBuilderReviewHandoffRejectsMismatchedOrModifiedPlan(t *testing.T) {
	plan := BuildBuilderReviewPlan(Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	})

	_, err := BuildBuilderReviewHandoff(plan, QualificationResult{
		Candidate: subject.Subject{Kind: "semantic-data", ID: "sales.order"},
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "semantic-data",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "subject mismatch") {
		t.Fatalf("mismatched qualification subject was not rejected: %v", err)
	}

	modified := plan
	modified.ReviewSequence = append([]string(nil), plan.ReviewSequence...)
	modified.ReviewSequence[0] = "modified"
	_, err = BuildBuilderReviewHandoff(modified, QualificationResult{
		Candidate: modified.Subject,
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "semantic-data",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "modified Builder Review Plan") {
		t.Fatalf("modified review plan was not rejected: %v", err)
	}
}

func TestBuilderReviewHandoffFingerprintTracksQualificationEvidence(t *testing.T) {
	plan := BuildBuilderReviewPlan(Context{
		Subject: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		References: []sourceindex.Reference{
			{
				Subject:      subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
				Role:         sourceindex.RoleDefinition,
				ArtifactPath: "applications/order-entry/patterns/customer-reference.tei",
			},
		},
		Artifacts: []sourceindex.Artifact{
			{
				Path:        "applications/order-entry/patterns/customer-reference.tei",
				Fingerprint: "sha256:pattern",
			},
		},
	})
	base := QualificationResult{
		Candidate: plan.Subject,
		Status:    QualificationApplicable,
		Requirements: QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
		Evidence: []QualificationEvidence{
			{
				SubjectKind:       "application-pattern",
				Subject:           "order_entry.customer_reference",
				Source:            "applications/order-entry/patterns/customer-reference.tei",
				SourceFingerprint: "sha256:first",
			},
		},
	}
	first, err := BuildBuilderReviewHandoff(plan, base)
	if err != nil {
		t.Fatal(err)
	}

	changedEvidence := base
	changedEvidence.Evidence = append([]QualificationEvidence(nil), base.Evidence...)
	changedEvidence.Evidence[0].SourceFingerprint = "sha256:second"
	second, err := BuildBuilderReviewHandoff(plan, changedEvidence)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint == second.Fingerprint {
		t.Fatalf("qualification Evidence revision did not change handoff fingerprint: %s", first.Fingerprint)
	}

	changedRequirement := base
	changedRequirement.Requirements.Application = "repair_intake"
	third, err := BuildBuilderReviewHandoff(plan, changedRequirement)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint == third.Fingerprint {
		t.Fatalf("qualification requirement change did not change handoff fingerprint: %s", first.Fingerprint)
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
