package knowledgecontext

import (
	"encoding/json"
	"fmt"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const builderReviewHandoffProjectionRevision = "tei.knowledge.builder-review-handoff/v1"

type BuilderReviewHandoff struct {
	ProjectionRevision    string              `json:"projectionRevision"`
	Fingerprint           string              `json:"fingerprint"`
	Purpose               string              `json:"purpose"`
	Subject               subject.Subject     `json:"subject"`
	ReviewPlanFingerprint string              `json:"reviewPlanFingerprint"`
	SelectedSections      []string            `json:"selectedSections,omitempty"`
	Status                string              `json:"status"`
	Qualification         QualificationResult `json:"qualification"`
	AuthorityLimits       []string            `json:"authorityLimits"`
}

func BuildBuilderReviewHandoff(
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
		ReviewPlanFingerprint: plan.Fingerprint,
		SelectedSections:      append([]string(nil), plan.RecommendedDisclosureSections...),
		Status:                status,
		Qualification:         qualification,
		AuthorityLimits: []string{
			"builder-review-handoff-is-derived-not-semantic-authority",
			"selected-section-is-not-disclosed-knowledge",
			"qualification-applicable-is-not-acceptance",
			"qualification-applicable-is-not-write-or-runtime-authority",
			"qualified-for-disclosure-does-not-prove-selected-knowledge-complete",
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
