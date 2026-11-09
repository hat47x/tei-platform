package main

import (
	"errors"
)

// This file adds the plan-level cross-check that §4.8 left as the remaining
// step between L2-a (hand-off) and L2-b (interpretation): the entry compares
// the resolver child's planFingerprint with the planFingerprint the builder
// child reports about the plan it actually built.
//
// It still does not establish L2-b. The builder's fingerprint is a self-report
// from a separate process; agreement shows the two children named the same
// plan, not that the builder derived it from the same declaration bytes and
// not that the artifact is that plan's output. The record says exactly that.

const (
	planCrossCheckMatched     = "matched"
	planCrossCheckMismatched  = "mismatched"
	planCrossCheckUnperformed = "unperformed"
)

// planCrossCheck is the §4.8 third observation: both sides' plan fingerprints,
// and whether they agree. A missing side is recorded as unperformed with the
// reason, never as a match.
type planCrossCheck struct {
	State                          string `json:"state"`
	ResolverPlanFingerprint        string `json:"resolverPlanFingerprint,omitempty"`
	BuilderReportedPlanFingerprint string `json:"builderReportedPlanFingerprint,omitempty"`
	Matched                        bool   `json:"matched"`
	L2BClaimed                     bool   `json:"l2bClaimed"`
	Boundary                       string `json:"boundary"`
	UnperformedReason              string `json:"unperformedReason,omitempty"`
}

const planCrossCheckBoundary = "the resolver child and the builder child named the same plan fingerprint; this is agreement of two self-reports from separate processes, not proof that the builder derived that plan from the declaration this entry held, and not proof that the artifact is that plan's output"

// crossCheckPlan compares the resolver's plan with the builder's reported plan.
// It runs only after both children have finished, so a mismatch is a recorded
// finding rather than a precondition that could have stopped the build.
func crossCheckPlan(resolverFingerprint, builderFingerprint string) planCrossCheck {
	record := planCrossCheck{
		L2BClaimed:                     false,
		Boundary:                       planCrossCheckBoundary,
		ResolverPlanFingerprint:        resolverFingerprint,
		BuilderReportedPlanFingerprint: builderFingerprint,
	}
	if resolverFingerprint == "" {
		record.State = planCrossCheckUnperformed
		record.UnperformedReason = "no resolver receipt was produced, so there is no resolved plan to compare"
		return record
	}
	if builderFingerprint == "" {
		record.State = planCrossCheckUnperformed
		record.UnperformedReason = "the builder child did not report the plan fingerprint it built; pass the reporting argument through -child-arg"
		return record
	}
	if resolverFingerprint == builderFingerprint {
		record.State = planCrossCheckMatched
		record.Matched = true
		return record
	}
	record.State = planCrossCheckMismatched
	return record
}

// planMismatchCause states the mismatch in the record's own words so a reader
// cannot mistake a recorded mismatch for a failed precondition.
func planMismatchCause() error {
	return errors.New("the builder child reported a plan fingerprint that differs from the resolver's plan")
}

// describePoolPlan keeps the unperformed reason for a resolver that never ran
// distinguishable from an unperformed check after a resolver ran fine.
