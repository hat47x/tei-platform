package languagereference

import (
	"errors"
	"fmt"

	"tei/internal/semantic"
)

var (
	ErrMatchPolicyInvalid  = errors.New("relation match policy is incomplete or invalid")
	ErrMatchKeyUnavailable = errors.New("relation key value cannot be evaluated")
	ErrMatchNullRejected   = errors.New("NULL relation key rejected by policy")
	ErrMatchCardinality    = errors.New("relation match violates declared cardinality")
	ErrMatchUnmatched      = errors.New("relation has no matching target")
)

// UnmatchedPolicy distinguishes an absent target from a present row with a
// nullable attribute. No default is inferred from semantic cardinality.
type UnmatchedPolicy string

const (
	UnmatchedReject       UnmatchedPolicy = "reject"
	UnmatchedDiscard      UnmatchedPolicy = "discard"
	UnmatchedRetainAbsent UnmatchedPolicy = "retain-absence"
)

// NullKeyPolicy controls a known NULL key. An unavailable key is always an
// evaluation error, never NULL or a nonmatching key.
type NullKeyPolicy string

const (
	NullKeyReject  NullKeyPolicy = "reject"
	NullKeyNoMatch NullKeyPolicy = "no-match"
)

// RelationMatchPolicy must be supplied explicitly for an executable expansion
// implementation. This standalone probe does not authorize any implementation.
type RelationMatchPolicy struct {
	Unmatched UnmatchedPolicy
	NullKey   NullKeyPolicy
}

func (p RelationMatchPolicy) Validate() error {
	if p.Unmatched != UnmatchedReject && p.Unmatched != UnmatchedDiscard && p.Unmatched != UnmatchedRetainAbsent {
		return fmt.Errorf("%w: unmatched %q", ErrMatchPolicyInvalid, p.Unmatched)
	}
	if p.NullKey != NullKeyReject && p.NullKey != NullKeyNoMatch {
		return fmt.Errorf("%w: null key %q", ErrMatchPolicyInvalid, p.NullKey)
	}
	return nil
}

type ProbeValueKind string

const (
	ProbeValue       ProbeValueKind = "value"
	ProbeNull        ProbeValueKind = "null"
	ProbeUnavailable ProbeValueKind = "unavailable"
)

// ProbeKeyPart uses the caller's canonical, typed value encoding. The probe
// does not infer types, equality rules, collation, or numeric precision.
// A value with Canonical=="" is distinct from NULL and unavailable.
type ProbeKeyPart struct {
	Kind      ProbeValueKind
	Canonical string
}

// RelationMatchState preserves why there is no matched target. In particular,
// a retained absence is never represented by a synthetic all-NULL record.
type RelationMatchState string

const (
	MatchFound          RelationMatchState = "matched"
	MatchDiscarded      RelationMatchState = "discarded"
	MatchRetainedAbsent RelationMatchState = "retained-absent"
)

type RelationMatchProbeResult struct {
	State         RelationMatchState
	TargetIndices []int // input fixture positions; not a result-order guarantee
}

// ProbePinnedExpansion evaluates only a keyed, in-memory fixture for one
// already-inspected candidate. It is a conformance oracle, not a relation
// executor or an authorization boundary. Keys must be in authored mapping
// order. A cardinality-one relation rejects more than one matching target.
// Distinct matching rows are never silently coalesced or arbitrarily chosen.
func ProbePinnedExpansion(candidate RelationExpansionCandidate, policy RelationMatchPolicy, source []ProbeKeyPart, targets [][]ProbeKeyPart) (RelationMatchProbeResult, error) {
	if err := policy.Validate(); err != nil {
		return RelationMatchProbeResult{}, err
	}
	if candidate.SourceID == "" || candidate.RelationID == "" || candidate.TargetID == "" ||
		candidate.Operator.ID != "std.op.expand" || candidate.Operator.Version == "" ||
		candidate.Operator.Fingerprint == "" || candidate.Operator.Effect != "pure" ||
		!expandTypesMatch(candidate.Operator, candidate.SourceID, candidate.TargetID) ||
		candidate.OutputType != "rows/"+string(candidate.TargetID) ||
		!candidate.Cardinality.Valid() || len(candidate.Keys) == 0 {
		return RelationMatchProbeResult{}, fmt.Errorf("%w: invalid expansion candidate", ErrMatchPolicyInvalid)
	}
	width := len(candidate.Keys)
	if err := validateProbeKey(source, width, policy.NullKey, "source"); err != nil {
		return RelationMatchProbeResult{}, err
	}
	for i, target := range targets {
		if err := validateProbeKey(target, width, policy.NullKey, fmt.Sprintf("target %d", i)); err != nil {
			return RelationMatchProbeResult{}, err
		}
	}

	indices := make([]int, 0)
	if !probeHasNull(source) {
		for i, target := range targets {
			if probeHasNull(target) {
				continue
			}
			matched := true
			for j := range source {
				if source[j].Canonical != target[j].Canonical {
					matched = false
					break
				}
			}
			if matched {
				indices = append(indices, i)
				if candidate.Cardinality == semantic.CardinalityOne && len(indices) > 1 {
					return RelationMatchProbeResult{}, fmt.Errorf("%w: relation %s has multiple targets", ErrMatchCardinality, candidate.RelationID)
				}
			}
		}
	}
	if len(indices) > 0 {
		return RelationMatchProbeResult{State: MatchFound, TargetIndices: indices}, nil
	}
	switch policy.Unmatched {
	case UnmatchedReject:
		return RelationMatchProbeResult{}, fmt.Errorf("%w: %s", ErrMatchUnmatched, candidate.RelationID)
	case UnmatchedDiscard:
		return RelationMatchProbeResult{State: MatchDiscarded}, nil
	case UnmatchedRetainAbsent:
		return RelationMatchProbeResult{State: MatchRetainedAbsent}, nil
	default:
		return RelationMatchProbeResult{}, ErrMatchPolicyInvalid
	}
}

func validateProbeKey(parts []ProbeKeyPart, width int, nullPolicy NullKeyPolicy, label string) error {
	if len(parts) != width {
		return fmt.Errorf("%w: %s has %d key parts, expected %d", ErrMatchKeyUnavailable, label, len(parts), width)
	}
	for i, part := range parts {
		switch part.Kind {
		case ProbeValue:
			// Empty strings are valid values in the fixture's canonical encoding.
		case ProbeNull:
			if part.Canonical != "" {
				return fmt.Errorf("%w: %s key %d: NULL has a value", ErrMatchKeyUnavailable, label, i)
			}
			if nullPolicy == NullKeyReject {
				return fmt.Errorf("%w: %s key %d is NULL", ErrMatchNullRejected, label, i)
			}
		case ProbeUnavailable:
			return fmt.Errorf("%w: %s key %d is unavailable", ErrMatchKeyUnavailable, label, i)
		default:
			return fmt.Errorf("%w: %s key %d has unknown state %q", ErrMatchKeyUnavailable, label, i, part.Kind)
		}
	}
	return nil
}

func probeHasNull(parts []ProbeKeyPart) bool {
	for _, part := range parts {
		if part.Kind == ProbeNull {
			return true
		}
	}
	return false
}
