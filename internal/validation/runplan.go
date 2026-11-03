package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// runPlanConcurrency and runPlanResume are the declared strategies a plan may use.
var runPlanConcurrency = map[string]bool{"singleton": true, "per-key": true}
var runPlanResume = map[string]bool{"checkpoint": true, "none": true}
var runPlanAutonomy = map[string]bool{"attended": true, "unattended": true}

// validateSemanticRunPlans validates the declared meaning of a run plan: the
// operations it runs, the key that identifies a run, the concurrency and resume
// strategies, and the autonomy. It does not check that a run happened.
func validateSemanticRunPlans(w ir.World) []Diagnostic {
	capabilities := map[core.CanonicalID]bool{}
	attributes := map[core.CanonicalID]bool{}
	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			capabilities[capability.ID] = true
		}
		for _, element := range namespace.Data {
			for _, attribute := range element.Attributes {
				attributes[attribute.ID] = true
			}
		}
	}

	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, plan := range namespace.RunPlans {
			subj := subject.Subject{Kind: "semantic-run-plan", ID: string(plan.ID)}

			if len(plan.Operations) == 0 {
				out = append(out, errFor(subj, "RUN_PLAN_NO_OPERATIONS", "run plan %s declares no operation", plan.ID))
			}
			for _, operation := range plan.Operations {
				if !capabilities[operation] {
					out = append(out, errFor(subj, "RUN_PLAN_UNKNOWN_OPERATION", "run plan %s names unknown capability %s", plan.ID, operation))
				}
			}
			if len(plan.Key) == 0 {
				out = append(out, errFor(subj, "RUN_PLAN_NO_KEY", "run plan %s declares no key, so a run cannot be identified", plan.ID))
			}
			for _, key := range plan.Key {
				if !attributes[key] {
					out = append(out, errFor(subj, "RUN_PLAN_UNKNOWN_KEY", "run plan %s names unknown key attribute %s", plan.ID, key))
				}
			}
			if plan.Concurrency != "" && !runPlanConcurrency[plan.Concurrency] {
				out = append(out, errFor(subj, "RUN_PLAN_UNKNOWN_CONCURRENCY", "run plan %s declares unknown concurrency %q", plan.ID, plan.Concurrency))
			}
			if plan.Resume != "" && !runPlanResume[plan.Resume] {
				out = append(out, errFor(subj, "RUN_PLAN_UNKNOWN_RESUME", "run plan %s declares unknown resume strategy %q", plan.ID, plan.Resume))
			}
			if plan.Autonomy != "" && !runPlanAutonomy[plan.Autonomy] {
				out = append(out, errFor(subj, "RUN_PLAN_UNKNOWN_AUTONOMY", "run plan %s declares unknown autonomy %q", plan.ID, plan.Autonomy))
			}
			// Resuming from a checkpoint without a declared output leaves nothing
			// to report the intermediate progress against.
			if plan.Resume == "checkpoint" && len(plan.Outputs) == 0 {
				out = append(out, errFor(subj, "RUN_PLAN_CHECKPOINT_WITHOUT_OUTPUT", "run plan %s resumes from a checkpoint but declares no output", plan.ID))
			}
		}
	}
	return out
}
