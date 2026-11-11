package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

// periodClosureEntry is what a contradiction check needs from one period: the
// operations it refuses while closed, and the capability it names for corrections.
type periodClosureEntry struct {
	ID            core.CanonicalID
	ClosedRejects []string
	Correction    core.CanonicalID
}

// validateClosedRejectsAgainstCorrection reports a period whose two declarations
// contradict each other.
//
// A period can declare both:
//
//	closed-rejects update      -- a change of this kind is refused while closed
//	correction via sales.correct -- a change goes through this capability
//
// If the correction capability's own effect is one of the refused kinds, the two
// declarations cannot both hold: the correction is the only way through, and the
// closure refuses the kind of change it makes. At run time the correction is
// exempted from the closure (see internal/command's checkPeriod), so the
// declarations resolve rather than deadlock — but a reader of the source, and any
// other host, sees a contradiction that only that one host knows how to resolve.
//
// Reporting it here keeps the meaning honest: the author should say what they meant
// (either the correction is not an update, or update is not refused). The run-time
// exemption stays, because it is the safe resolution of an already-written
// declaration, but it should not be the only place the contradiction exists.
func validateClosedRejectsAgainstCorrectionOf(w ir.World) []Diagnostic {
	var periods []periodClosureEntry
	// correctionEffects maps a correction capability to its effect kind and data,
	// which is what decides whether it collides with a refused operation.
	correctionEffects := map[core.CanonicalID]semantic.DataEffect{}
	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			if capability.Effect == nil {
				continue
			}
			correctionEffects[capability.ID] = *capability.Effect
		}
		for _, period := range namespace.Periods {
			periods = append(periods, periodClosureEntry{
				ID:            period.ID,
				ClosedRejects: period.ClosedRejects,
				Correction:    period.Correction,
			})
		}
	}

	var out []Diagnostic
	for _, period := range periods {
		if period.Correction == "" {
			continue
		}
		effect, ok := correctionEffects[period.Correction]
		if !ok {
			// A correction capability whose effect is not declared says nothing
			// about which operation it performs, so there is no collision to
			// report. The unknown-capability check reports a missing capability.
			continue
		}
		kind := string(effect.Kind)
		for _, rejected := range period.ClosedRejects {
			if rejected != kind {
				continue
			}
			out = append(out, errFor(
				subject.Subject{Kind: "semantic-period", ID: string(period.ID)},
				"PERIOD_CORRECTION_REFUSED_BY_CLOSURE",
				"period %s refuses %s while closed but names %s as its correction, whose effect is also %s; the correction could never run",
				period.ID, kind, period.Correction, kind))
		}
	}
	return out
}
