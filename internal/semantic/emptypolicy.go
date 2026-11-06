package semantic

// EmptyPolicy is the Owner-declared handling of an empty target set for a
// business action (#1125). It is recorded on the Action's .tei declaration and
// overrides the business-type default.
type EmptyPolicy string

const (
	// EmptyPolicyAllow treats an empty target set as satisfying the condition.
	EmptyPolicyAllow EmptyPolicy = "allow"
	// EmptyPolicyRequiresEvidence requires independent evidence that the target
	// set is empty before the condition can hold.
	EmptyPolicyRequiresEvidence EmptyPolicy = "requires-evidence"
	// EmptyPolicyDeny refuses completion on an empty target set.
	EmptyPolicyDeny EmptyPolicy = "deny"
)

// Valid reports whether the value is one of the three declared policies.
func (p EmptyPolicy) Valid() bool {
	switch p {
	case EmptyPolicyAllow, EmptyPolicyRequiresEvidence, EmptyPolicyDeny:
		return true
	default:
		return false
	}
}

// EmptyPolicyCondition is the business condition kind used to derive a default.
type EmptyPolicyCondition string

const (
	// EmptyPolicyUniversal means "every target satisfies the predicate".
	EmptyPolicyUniversal EmptyPolicyCondition = "universal"
	// EmptyPolicyAbsence means "no matching target exists".
	EmptyPolicyAbsence EmptyPolicyCondition = "absence"
	// EmptyPolicyExistence means "at least one matching target exists".
	EmptyPolicyExistence EmptyPolicyCondition = "existence"
	// EmptyPolicyAggregate means the predicate is an aggregate value.
	EmptyPolicyAggregate EmptyPolicyCondition = "aggregate"
	// EmptyPolicyHandoff means completion depends on an external handoff.
	EmptyPolicyHandoff EmptyPolicyCondition = "handoff"
)

// DefaultEmptyPolicy is the candidate default for a business condition kind.
// It is a default, not an accepted product contract: an Action declaration can
// override it, and an unknown kind has no default.
func DefaultEmptyPolicy(kind EmptyPolicyCondition) EmptyPolicy {
	switch kind {
	case EmptyPolicyUniversal:
		return EmptyPolicyRequiresEvidence
	case EmptyPolicyAbsence:
		return EmptyPolicyAllow
	case EmptyPolicyExistence:
		return EmptyPolicyDeny
	case EmptyPolicyAggregate:
		return EmptyPolicyAllow
	case EmptyPolicyHandoff:
		return EmptyPolicyRequiresEvidence
	default:
		return ""
	}
}

// EffectiveEmptyPolicy returns the declared policy, or the business-type
// default when the Action did not declare one.
func (c Capability) EffectiveEmptyPolicy(kind EmptyPolicyCondition) EmptyPolicy {
	if c.EmptyPolicy != "" {
		return c.EmptyPolicy
	}
	return DefaultEmptyPolicy(kind)
}
