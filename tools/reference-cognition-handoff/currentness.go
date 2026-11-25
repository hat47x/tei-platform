// Currentness check for a handed-off change candidate.
//
// A change candidate produced from a cognition proposal is derived from a
// context that may already be old by the time a consumer offers it. #5808
// (Studio background preparation) and #5809 (typed cognition in a delivered
// application) both require that a candidate produced from an older revision, or
// under an older judgement basis, never reaches the current selection, and that
// the fact be reported separately from a wrong answer.
//
// This file is the handoff-side counterpart of the port's RequireCurrent: the
// port judges a proposal against the current request, and this judges a derived
// change candidate against the binding it carries. Neither accepts, applies, or
// grants authority.
package cognitionhandoff

import "fmt"

// Currentness outcomes. They are distinct facts, not one boolean.
const (
	// Current: the candidate's binding matches the caller's current binding.
	Current = "current"
	// Unbound: the candidate carries no binding, so currentness cannot be
	// established. A caller may still show it as a historical artifact, but not
	// as a candidate for the current selection.
	Unbound = "unbound"
	// StaleRevision: the subject moved to another revision.
	StaleRevision = "stale-revision"
	// StaleBasis: the judgement basis changed (a revised criteria table, an
	// added or removed option).
	StaleBasis = "stale-basis"
	// ForeignSubject: the candidate is about another subject.
	ForeignSubject = "foreign-subject"
)

// CurrentnessResult reports why a candidate may or may not be offered against
// the caller's current binding.
type CurrentnessResult struct {
	State       string           `json:"state"`
	Diagnostic  string           `json:"diagnostic,omitempty"`
	CandidateID string           `json:"candidateId"`
	Proposed    *ProposalBinding `json:"proposed,omitempty"`
	Current     ProposalBinding  `json:"current"`
	// Acceptable and GrantsAuthority stay false: being current is the
	// precondition for offering a candidate, not acceptance of its meaning.
	Acceptable      bool `json:"acceptable"`
	GrantsAuthority bool `json:"grantsAuthority"`
}

// RequireCurrentAtHandoff decides whether a handed-off candidate may be offered
// against the binding that is current now.
//
// The order is subject, then revision, then basis. Reporting another subject as
// "stale" would invite a retry against the wrong subject, which is why the
// subject is decided first.
func RequireCurrentAtHandoff(candidate Candidate, current ProposalBinding) CurrentnessResult {
	result := CurrentnessResult{
		CandidateID:     candidate.ID,
		Current:         current,
		Acceptable:      false,
		GrantsAuthority: false,
		State:           Current,
	}
	if candidate.Binding == nil {
		result.State = Unbound
		result.Diagnostic = "candidate carries no proposal binding; currentness is not established"
		return result
	}
	proposed := *candidate.Binding
	result.Proposed = &proposed

	if proposed.Subject == "" || current.Subject == "" || proposed.Subject != current.Subject {
		result.State = ForeignSubject
		result.Diagnostic = fmt.Sprintf("candidate subject %q is not the current subject %q", proposed.Subject, current.Subject)
		return result
	}
	if proposed.Revision != current.Revision {
		result.State = StaleRevision
		result.Diagnostic = fmt.Sprintf("candidate was bound to revision %q, current revision is %q", proposed.Revision, current.Revision)
		return result
	}
	if proposed.Basis != current.Basis {
		result.State = StaleBasis
		result.Diagnostic = fmt.Sprintf("candidate was bound to basis %q, current basis is %q", proposed.Basis, current.Basis)
		return result
	}
	return result
}
