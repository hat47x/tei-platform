package semantic

import "fmt"

// CompletionState is the outcome of evaluating an Owner completion claim.
type CompletionState string

const (
	// CompletionHolds means the declared condition is satisfied with the
	// evidence required by the empty-set policy.
	CompletionHolds CompletionState = "holds"
	// CompletionFails means a definite counterexample was observed, or the
	// empty-set policy forbids completion on an empty target set.
	CompletionFails CompletionState = "fails"
	// CompletionUnverified means the evidence needed to decide is missing.
	CompletionUnverified CompletionState = "unverified"
	// CompletionContradiction means the supplied evidence conflicts with
	// itself and cannot be used to decide.
	CompletionContradiction CompletionState = "contradiction"
)

// Coverage is how much of the Owner-declared target set the observed results
// cover. It is evidence supplied by the caller, not something this package can
// derive from the predicate.
type Coverage string

const (
	// CoverageComplete means the observation covers the whole target set L for
	// the declared scope and version.
	CoverageComplete Coverage = "complete"
	// CoveragePartial means some targets are observed but coverage is not
	// established.
	CoveragePartial Coverage = "partial"
	// CoverageUnestablished means coverage was not measured.
	CoverageUnestablished Coverage = "unestablished"
)

// TargetResult is one target's predicate result.
type TargetResult struct {
	ID    string
	Holds bool
}

// CompletionEvidence is the observed evidence for one completion claim. The
// evaluator never derives coverage or emptiness from the predicate shape.
type CompletionEvidence struct {
	// Targets are the observed targets for the declared scope.
	Targets []TargetResult
	// Coverage reports whether Targets covers the Owner-declared target set.
	Coverage Coverage
	// EmptyJustified reports independent evidence that the target set is
	// empty (for example, a complete same-scope/provider read).
	EmptyJustified bool
	// HandoffConfirmed reports the external or manual confirmation required by
	// a handoff business condition.
	HandoffConfirmed bool
	// Conflicting reports that two evidence sources disagree about the same
	// target, scope or version.
	Conflicting bool
}

// CompletionDecision explains one completion evaluation.
type CompletionDecision struct {
	State        CompletionState
	Reason       string
	EmptyPolicy  EmptyPolicy
	PolicySource string
	Targets      int
	Violations   []string
}

// EvaluateCompletion evaluates the (TargetSet L, EmptyPolicy, Predicate P)
// triple for one business condition kind. It is fail-closed: missing coverage,
// missing emptiness evidence, and a missing policy all produce
// CompletionUnverified rather than success.
//
// The caller supplies the observed targets and whether they cover L. This
// function does not infer L from the predicate or from the result count.
func EvaluateCompletion(capability Capability, kind EmptyPolicyCondition, evidence CompletionEvidence) CompletionDecision {
	policy := capability.EffectiveEmptyPolicy(kind)
	decision := CompletionDecision{
		EmptyPolicy:  policy,
		PolicySource: "default",
		Targets:      len(evidence.Targets),
	}
	if capability.EmptyPolicy != "" {
		decision.PolicySource = "declared"
	}
	if !policy.Valid() {
		decision.State = CompletionUnverified
		decision.Reason = "no empty-set policy is declared or derivable for this business condition"
		return decision
	}
	if evidence.Conflicting {
		decision.State = CompletionContradiction
		decision.Reason = "the supplied evidence conflicts and cannot decide the claim"
		return decision
	}

	// A definite violation is a counterexample even when coverage is unknown.
	for _, target := range evidence.Targets {
		if !target.Holds {
			decision.Violations = append(decision.Violations, target.ID)
		}
	}
	if len(decision.Violations) > 0 {
		decision.State = CompletionFails
		decision.Reason = fmt.Sprintf("%d observed target(s) do not satisfy the predicate", len(decision.Violations))
		return decision
	}

	if len(evidence.Targets) == 0 {
		switch policy {
		case EmptyPolicyDeny:
			// No observed target is not proof that the Owner-declared target set is empty.
			// Deny applies only after a complete read establishes the empty set.
			if evidence.Coverage != CoverageComplete {
				decision.State = CompletionUnverified
				decision.Reason = "the empty-set policy forbids completion, but target-set emptiness is not established"
				return decision
			}
			decision.State = CompletionFails
			decision.Reason = "the empty-set policy forbids completion without a target"
		case EmptyPolicyAllow:
			if evidence.Coverage != CoverageComplete {
				decision.State = CompletionUnverified
				decision.Reason = "an empty target set is allowed, but emptiness is not established"
				return decision
			}
			decision.State = CompletionHolds
			decision.Reason = "the target set is empty and the empty-set policy allows completion"
		case EmptyPolicyRequiresEvidence:
			if !evidence.EmptyJustified {
				decision.State = CompletionUnverified
				decision.Reason = "an empty target set requires independent emptiness evidence"
				return decision
			}
			decision.State = CompletionHolds
			decision.Reason = "the target set is empty and its emptiness is independently justified"
		}
		return decision
	}

	if evidence.Coverage != CoverageComplete {
		decision.State = CompletionUnverified
		decision.Reason = "the observed targets do not cover the Owner-declared target set"
		return decision
	}
	if kind == EmptyPolicyHandoff && !evidence.HandoffConfirmed {
		decision.State = CompletionUnverified
		decision.Reason = "the handoff condition is not confirmed"
		return decision
	}
	decision.State = CompletionHolds
	decision.Reason = "every observed target satisfies the predicate and the target set is covered"
	return decision
}
