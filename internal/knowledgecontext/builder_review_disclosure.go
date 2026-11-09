package knowledgecontext

import (
	"fmt"

	"tei/internal/sourceindex"
)

func BuildBuilderReviewDisclosure(
	ctx Context,
	sources sourceindex.Index,
	handoff BuilderReviewHandoff,
) (DisclosureProjection, error) {
	if err := ValidateBuilderReviewHandoff(handoff); err != nil {
		return DisclosureProjection{}, fmt.Errorf(
			"builder review disclosure rejects invalid handoff: %w",
			err,
		)
	}
	if handoff.Subject != ctx.Subject {
		return DisclosureProjection{}, fmt.Errorf(
			"builder review disclosure subject mismatch: handoff=%s:%s context=%s:%s",
			handoff.Subject.Kind,
			handoff.Subject.ID,
			ctx.Subject.Kind,
			ctx.Subject.ID,
		)
	}

	current, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		handoff.Qualification.Requirements,
	)
	if err != nil {
		return DisclosureProjection{}, err
	}
	if current.Fingerprint != handoff.Fingerprint {
		return DisclosureProjection{}, fmt.Errorf(
			"builder review disclosure rejects stale handoff: supplied=%s current=%s",
			handoff.Fingerprint,
			current.Fingerprint,
		)
	}
	if current.Status != "qualified-for-disclosure" {
		return DisclosureProjection{}, fmt.Errorf(
			"builder review disclosure blocked by handoff status %q",
			current.Status,
		)
	}

	disclosure, err := BuildDisclosure(ctx, current.SelectedSections)
	if err != nil {
		return DisclosureProjection{}, err
	}
	if disclosure.Status != "complete" {
		return DisclosureProjection{}, fmt.Errorf(
			"builder review disclosure requires complete selected Knowledge; status=%q",
			disclosure.Status,
		)
	}
	return disclosure, nil
}
