package knowledgecontext

import (
	"encoding/json"
	"fmt"
	"reflect"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const builderReviewHandoffProjectionRevision = "tei.knowledge.builder-review-handoff/v1"

type BuilderReviewHandoff struct {
	ProjectionRevision    string              `json:"projectionRevision"`
	Fingerprint           string              `json:"fingerprint"`
	Purpose               string              `json:"purpose"`
	Subject               subject.Subject     `json:"subject"`
	ReviewPlanProjectionRevision string              `json:"reviewPlanProjectionRevision"`
	ReviewPlanFingerprint          string              `json:"reviewPlanFingerprint"`
	RoutingScope                   string              `json:"routingScope"`
	SelectedSections      []string            `json:"selectedSections,omitempty"`
	Status                string              `json:"status"`
	Qualification         QualificationResult `json:"qualification"`
	AuthorityLimits       []string            `json:"authorityLimits"`
}

func ValidateBuilderReviewHandoff(handoff BuilderReviewHandoff) error {
	if handoff.ProjectionRevision != builderReviewHandoffProjectionRevision {
		return fmt.Errorf(
			"builder review handoff projection revision %q is unsupported",
			handoff.ProjectionRevision,
		)
	}
	if handoff.Purpose != "design-change" {
		return fmt.Errorf("builder review handoff purpose %q is unsupported", handoff.Purpose)
	}
	if handoff.RoutingScope != "current-knowledge-context" {
		return fmt.Errorf(
			"builder review handoff routing scope %q is unsupported",
			handoff.RoutingScope,
		)
	}
	if handoff.Subject.Kind == "" || handoff.Subject.ID == "" {
		return fmt.Errorf("builder review handoff subject is incomplete")
	}
	if handoff.ReviewPlanProjectionRevision != builderReviewPlanProjectionRevision ||
		handoff.ReviewPlanFingerprint == "" {
		return fmt.Errorf("builder review handoff review-plan identity is incomplete")
	}
	if handoff.Fingerprint == "" {
		return fmt.Errorf("builder review handoff fingerprint is empty")
	}
	if builderReviewHandoffFingerprint(handoff) != handoff.Fingerprint {
		return fmt.Errorf("builder review handoff fingerprint does not match content")
	}
	return nil
}

func BuildBuilderReviewHandoff(
	ctx Context,
	sources sourceindex.Index,
	requirements QualificationRequirements,
) (BuilderReviewHandoff, error) {
	plan := BuildBuilderReviewPlan(ctx)
	qualification, err := QualifyWithSources(ctx, sources, requirements)
	if err != nil {
		return BuilderReviewHandoff{}, err
	}
	if qualification.Status == QualificationApplicable {
		currentEvidence := authoritativeEvidenceForSubject(sources, ctx.Subject)
		if !reflect.DeepEqual(qualification.Evidence, currentEvidence) {
			return BuilderReviewHandoff{}, fmt.Errorf(
				"builder review handoff rejects mixed Context and Source Index revisions",
			)
		}
	}
	return buildBuilderReviewHandoffFromQualification(plan, qualification)
}

func buildBuilderReviewHandoffFromQualification(
	plan BuilderReviewPlan,
	qualification QualificationResult,
) (BuilderReviewHandoff, error) {
	if plan.ProjectionRevision != builderReviewPlanProjectionRevision ||
		plan.Purpose != "design-change" ||
		plan.RoutingScope != "current-knowledge-context" ||
		plan.Fingerprint == "" {
		return BuilderReviewHandoff{}, fmt.Errorf("builder review handoff requires a current Builder Review Plan")
	}
	if builderReviewPlanFingerprint(plan) != plan.Fingerprint {
		return BuilderReviewHandoff{}, fmt.Errorf("builder review handoff rejects a modified Builder Review Plan")
	}
	if qualification.Candidate != plan.Subject {
		return BuilderReviewHandoff{}, fmt.Errorf(
			"builder review handoff subject mismatch: plan=%s:%s qualification=%s:%s",
			plan.Subject.Kind,
			plan.Subject.ID,
			qualification.Candidate.Kind,
			qualification.Candidate.ID,
		)
	}

	status := ""
	switch qualification.Status {
	case QualificationApplicable:
		if qualification.Requirements.ExpectedKind != plan.Subject.Kind {
			return BuilderReviewHandoff{}, fmt.Errorf(
				"builder review handoff rejects applicable qualification for a different expected kind",
			)
		}
		if len(qualification.Evidence) == 0 {
			return BuilderReviewHandoff{}, fmt.Errorf(
				"builder review handoff rejects applicable qualification without authoritative Evidence",
			)
		}
		if len(qualification.Requirements.RequiredDependencies) != 0 &&
			len(qualification.DependencyEvidence) == 0 {
			return BuilderReviewHandoff{}, fmt.Errorf(
				"builder review handoff rejects applicable qualification without dependency Evidence",
			)
		}
		if len(plan.RecommendedDisclosureSections) == 0 {
			return BuilderReviewHandoff{}, fmt.Errorf(
				"builder review handoff rejects applicable qualification without selected Knowledge",
			)
		}
		status = "qualified-for-disclosure"
	case QualificationIncompatible:
		status = "blocked-incompatible"
	case QualificationUnknown:
		status = "blocked-unresolved"
	default:
		return BuilderReviewHandoff{}, fmt.Errorf(
			"builder review handoff rejects unknown qualification status %q",
			qualification.Status,
		)
	}

	result := BuilderReviewHandoff{
		ProjectionRevision:    builderReviewHandoffProjectionRevision,
		Purpose:               plan.Purpose,
		Subject:               plan.Subject,
		ReviewPlanProjectionRevision: plan.ProjectionRevision,
		ReviewPlanFingerprint:          plan.Fingerprint,
		RoutingScope:                   plan.RoutingScope,
		SelectedSections:      append([]string(nil), plan.RecommendedDisclosureSections...),
		Status:                status,
		Qualification:         qualification,
		AuthorityLimits: []string{
			"builder-review-handoff-is-derived-not-semantic-authority",
			"builder-review-handoff-is-not-access-control-or-redaction",
			"selected-section-is-not-disclosed-knowledge",
			"qualification-applicable-is-not-acceptance",
			"qualification-applicable-is-not-write-or-runtime-authority",
			"qualified-for-disclosure-does-not-prove-selected-knowledge-complete",
			"qualified-for-disclosure-does-not-prove-section-local-evidence-complete",
			"handoff-does-not-materialize-knowledge",
			"current-disclosure-must-be-built-after-handoff",
			"disclosure-fingerprint-not-handoff-fingerprint-governs-disclosed-content-reuse",
		},
	}
	result.Fingerprint = builderReviewHandoffFingerprint(result)
	return result, nil
}

func builderReviewHandoffFingerprint(value BuilderReviewHandoff) string {
	value.Fingerprint = ""
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("Builder Review Handoff contains non-JSON state: " + err.Error())
	}
	return sourceindex.Fingerprint(encoded)
}
