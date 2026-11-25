// Currentness re-check for cognition proposals.
//
// A proposal is produced against one captured context. Between the request and
// the moment a consumer could show it, the selection, the source revision, or
// the judgement basis may have changed. #5808 (Studio background preparation)
// and #5809 (typed cognition in a delivered application) both require that a
// candidate produced from an older observation never reaches the current
// selection. This file is the shared, provider-independent part of that rule:
// it decides from the receipt and the *current* request only, and it never
// accepts meaning, grants authority, or applies a change.
//
// It is a reference primitive, not a new Host, store, or scheduler.
package cognitionport

import (
	"encoding/json"
	"fmt"
)

// Recheck outcomes. Each is a distinct fact about why a proposal may or may not
// be shown; none of them means the candidate is correct.
const (
	// RecheckCurrent: the receipt was produced from the same binding that is
	// current now, so a consumer may show it as a candidate.
	RecheckCurrent = "current"
	// RecheckStaleRevision: the subject's revision moved since the proposal.
	RecheckStaleRevision = "stale-revision"
	// RecheckStaleBasis: the judgement basis the candidate cites is no longer
	// present in the current request (a deleted option, a revised criteria
	// table, an observation that no longer exists).
	RecheckStaleBasis = "stale-basis"
	// RecheckForeignSubject: the candidate's subject is not the subject the
	// current request asks about.
	RecheckForeignSubject = "foreign-subject"
	// RecheckNoProposal: the receipt carries no proposal to re-check.
	RecheckNoProposal = "no-proposal"
)

// Binding is the identity a proposal is bound to: which subject, at which
// revision, under which judgement basis. A candidate is showable only while
// all of it still holds.
type Binding struct {
	// Subject is the observation or business row the request asked about.
	Subject string `json:"subject"`
	// Revision is the caller's version marker for that subject (a workspace
	// revision, a row version, or a source digest). It is opaque here.
	Revision string `json:"revision"`
	// Basis is the version marker of the judgement basis in force: an option
	// set, a criteria table, or a classification standard.
	Basis string `json:"basis,omitempty"`
}

// RecheckResult states whether a receipt may be shown against the current
// request, and separates the reasons rather than collapsing them into a boolean.
type RecheckResult struct {
	// State is one of the Recheck* constants.
	State string `json:"state"`
	// Diagnostic explains a non-current state; empty when current.
	Diagnostic string `json:"diagnostic,omitempty"`
	// Current is the binding that must be satisfied by any later proposal for
	// the same subject. It is reported even when the receipt is discarded, so a
	// caller can re-prepare against the value that is now current.
	Current Binding `json:"current"`
	// Proposed is the binding the receipt was produced under.
	Proposed Binding `json:"proposed"`
	// GrantsAuthority and Acceptable are always false: re-checking currentness
	// is not acceptance, and a current candidate is still only a candidate.
	GrantsAuthority bool `json:"grantsAuthority"`
	Acceptable      bool `json:"acceptable"`
}

// BindingOf reports the binding a request asks about. A request without a
// choice question has no single subject, so its subject is the empty string and
// a caller that needs one must supply the business key itself.
func BindingOf(request Request, basis string) Binding {
	return Binding{
		Subject:  RequestSubject(request),
		Revision: request.Context.WorkspaceRevision,
		Basis:    basis,
	}
}

// RequestSubject reports the subject a request is about.
func RequestSubject(request Request) string {
	if request.Context.Choice != nil {
		return request.Context.Choice.ObservationID
	}
	if len(request.Context.Observations) == 1 {
		return request.Context.Observations[0].ID
	}
	return ""
}

// BasisDigest fingerprints the judgement basis a request carries: the option
// set and the stated criteria. It answers "which criteria produced this
// candidate" without adding a version field to the request, and it changes
// exactly when the option set or a criterion changes — a revised classification
// standard, a removed option, or a reordered option list.
func BasisDigest(request Request) (string, error) {
	payload := struct {
		Constraints []string        `json:"constraints"`
		Choice      *ChoiceQuestion `json:"choice"`
	}{Constraints: request.Context.Constraints, Choice: request.Context.Choice}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return digest(raw), nil
}

// RequireCurrent decides whether a receipt may be shown against the current
// request. The check is deliberately conservative: an unchanged revision is not
// enough when the candidate cites a basis the current request no longer
// contains, and a candidate for another subject is never current.
//
// The caller supplies the current request — typically re-read after the
// inference returned, not the one that produced the proposal — and the basis
// marker in force now. The function never observes the provider, opens a
// network, or writes anything.
func RequireCurrent(receipt Receipt, current Request, currentBasis string) RecheckResult {
	proposed := receipt.Binding
	now := BindingOf(current, currentBasis)
	result := RecheckResult{
		State:           RecheckCurrent,
		Current:         now,
		Proposed:        proposed,
		GrantsAuthority: false,
		Acceptable:      false,
	}
	if receipt.Proposal == nil {
		result.State = RecheckNoProposal
		result.Diagnostic = "receipt carries no proposal"
		return result
	}
	// Subject first: a candidate about another subject is not a version of the
	// current question at all, and reporting it as merely stale would invite a
	// retry against the wrong subject.
	if now.Subject == "" || proposed.Subject != now.Subject {
		result.State = RecheckForeignSubject
		result.Diagnostic = fmt.Sprintf("proposal subject %q is not the current subject %q", proposed.Subject, now.Subject)
		return result
	}
	if proposed.Revision != now.Revision {
		result.State = RecheckStaleRevision
		result.Diagnostic = fmt.Sprintf("proposal was bound to revision %q, current revision is %q", proposed.Revision, now.Revision)
		return result
	}
	if proposed.Basis != now.Basis {
		result.State = RecheckStaleBasis
		result.Diagnostic = fmt.Sprintf("proposal was bound to basis %q, current basis is %q", proposed.Basis, now.Basis)
		return result
	}
	for _, candidate := range receipt.Proposal.Candidates {
		if candidate.Subject != now.Subject {
			result.State = RecheckForeignSubject
			result.Diagnostic = fmt.Sprintf("candidate %s has subject %q, current subject is %q", candidate.ID, candidate.Subject, now.Subject)
			return result
		}
		for _, basis := range candidate.Basis {
			if !basisIsPresent(basis, current) {
				result.State = RecheckStaleBasis
				result.Diagnostic = fmt.Sprintf("candidate %s cites basis %q, which the current request does not contain", candidate.ID, basis)
				return result
			}
		}
	}
	return result
}
