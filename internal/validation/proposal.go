package validation

import (
	"fmt"

	"tei/internal/subject"
)

// Proposal is a candidate correction for one diagnostic. Creating a proposal
// never accepts it: Accepted stays false until Accept is given the same
// accepted revision the diagnostic was computed from. A proposal carries no
// patch, only the diagnostic it answers and an author-facing hint, because a
// correction of meaning is the author's decision and not a mechanical rewrite.
type Proposal struct {
	Code       string           `json:"code"`
	Severity   Severity         `json:"severity"`
	Subject    *subject.Subject `json:"subject,omitempty"`
	Guidance   string           `json:"guidance,omitempty"`
	Accepted   bool             `json:"accepted"`
	Diagnostic string           `json:"diagnostic"`
	// Revision is the world fingerprint the diagnostic was read from. Accept
	// refuses a proposal whose revision no longer matches the accepted source.
	Revision string `json:"revision"`
}

// Propose turns diagnostics into proposals. Each proposal is unanswered until a
// human accepts it; the function never sets Accepted.
func Propose(diagnostics []Diagnostic, revision string) []Proposal {
	out := make([]Proposal, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, Proposal{
			Code:       diagnostic.Code,
			Severity:   diagnostic.Severity,
			Subject:    diagnostic.Subject,
			Diagnostic: diagnostic.Message,
			Revision:   revision,
			Accepted:   false,
		})
	}
	return out
}

// RevisionMismatchError reports that an accepted revision changed after the
// diagnostic was read, so the proposal no longer describes the current source.
type RevisionMismatchError struct {
	Proposed string
	Accepted string
}

func (e *RevisionMismatchError) Error() string {
	return fmt.Sprintf("proposal was read from revision %q but the accepted revision is %q", e.Proposed, e.Accepted)
}

// Accept marks one proposal accepted. It requires an accepted revision: the
// revision must be present and equal to the one the diagnostic was read from.
// Accepting twice is refused, so a caller cannot widen a decision by repeating
// the same proposal.
func Accept(proposal *Proposal, acceptedRevision string) error {
	if proposal == nil {
		return fmt.Errorf("proposal is required")
	}
	if proposal.Accepted {
		return fmt.Errorf("proposal %s is already accepted", proposal.Code)
	}
	if proposal.Revision == "" {
		return fmt.Errorf("proposal %s has no revision", proposal.Code)
	}
	if acceptedRevision == "" {
		return fmt.Errorf("accepted revision is required to accept a proposal")
	}
	if proposal.Revision != acceptedRevision {
		return &RevisionMismatchError{Proposed: proposal.Revision, Accepted: acceptedRevision}
	}
	proposal.Accepted = true
	return nil
}
