package composition

import "fmt"

// ProposalVerdict describes structural feasibility only. In particular,
// a candidate is not proof of conformance, artifact trust or host authority.
type ProposalVerdict string

const (
	ProposalCandidate ProposalVerdict = "candidate"
	ProposalConflict  ProposalVerdict = "conflict"
	ProposalUnknown   ProposalVerdict = "unknown"
)

// ProposalReason is stable machine-readable diagnostic information. A missing
// candidate in a partial registry is Unknown, never proof that code is needed.
type ProposalReason string

const (
	ReasonStructurallyEligible ProposalReason = "structurally-eligible"
	ReasonInvalidRequirement   ProposalReason = "invalid-requirement"
	ReasonNoKnownProvider      ProposalReason = "no-known-provider"
	ReasonAmbiguousProviders   ProposalReason = "ambiguous-providers"
	ReasonExplicitMismatch     ProposalReason = "explicit-provider-mismatch"
	ReasonDependencyIneligible ProposalReason = "dependency-ineligible"
	ReasonMixedScopes          ProposalReason = "mixed-scopes"
	ReasonInconsistentSlot     ProposalReason = "inconsistent-slot"
)

// WiringRequirement is a design-time request, not a runtime invocation. When
// ProviderID is empty, an unambiguous match may be proposed by this registry.
// SlotID identifies a consumer's binding point. An omitted SlotID uses the
// exact Contract as an implicit shared slot within this batch.
type WiringRequirement struct {
	Contract   Contract
	Scope      string
	ProviderID string
	SlotID     string
}

// WiringFinding records a candidate derived from the current Registry or the
// reason why no candidate can be justified. A returned Selection is copied;
// it may become stale before actual Host admission and must be revalidated.
type WiringFinding struct {
	Verdict    ProposalVerdict
	Reason     ProposalReason
	Candidates []string
	Selected   *Selection
	Detail     string
}

// WiringSetFinding records one consistent design-time Registry view. This is
// not proof that the selections can run together or that external conditions
// (contract conformance, artifact trust, host authority) are satisfied.
type WiringSetFinding struct {
	Revision uint64
	Scope    string
	Verdict  ProposalVerdict
	Findings []WiringFinding
}

// InspectWiring answers a design-time structural question without executing
// or authorizing a provider. It uses the same Registry as runtime selection.
func (r *Registry) InspectWiring(req WiringRequirement) WiringFinding {
	return r.InspectWiringSet([]WiringRequirement{req}).Findings[0]
}

// InspectWiringSet assesses a group from one atomic Registry snapshot. All
// requirements must share one scope; multiple scopes require independent
// scoped authorization and must not be silently merged by an AI assistant.
// Conflict takes precedence over Unknown to surface explicit contradictions.
func (r *Registry) InspectWiringSet(reqs []WiringRequirement) WiringSetFinding {
	out := WiringSetFinding{Verdict: ProposalUnknown}
	if len(reqs) == 0 {
		return out
	}
	out.Findings = make([]WiringFinding, len(reqs))
	for i := range out.Findings {
		out.Findings[i] = WiringFinding{Verdict: ProposalUnknown, Reason: ReasonInvalidRequirement}
	}
	if r == nil || reqs[0].Scope == "" {
		return out
	}
	out.Scope = reqs[0].Scope
	for i, req := range reqs {
		if req.Scope != out.Scope {
			for j := range out.Findings {
				out.Findings[j].Reason = ReasonMixedScopes
			}
			return out
		}
		if !req.Contract.valid() {
			out.Findings[i].Reason = ReasonInvalidRequirement
		}
	}
	// A proposal for one binding point must not contain competing providers
	// or incompatible Contracts. Independent consumers may use distinct SlotIDs.
	slots := make(map[string][]int, len(reqs))
	for i, req := range reqs {
		key := fmt.Sprintf("slot:%q", req.SlotID)
		if req.SlotID == "" {
			key = fmt.Sprintf("contract:%q/%q/%q", req.Contract.ID, req.Contract.Version, req.Contract.Shape)
		}
		slots[key] = append(slots[key], i)
	}
	contradicted := make(map[int]bool)
	for _, indices := range slots {
		first := reqs[indices[0]]
		for _, i := range indices[1:] {
			if reqs[i].Contract != first.Contract || reqs[i].ProviderID != first.ProviderID {
				for _, j := range indices {
					contradicted[j] = true
				}
				break
			}
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out.Revision = r.snapshot.Revision
	out.Verdict = ProposalCandidate
	for i, req := range reqs {
		if contradicted[i] {
			out.Findings[i] = WiringFinding{Verdict: ProposalConflict, Reason: ReasonInconsistentSlot}
		} else {
			out.Findings[i] = r.inspectWiringLocked(req)
		}
		switch out.Findings[i].Verdict {
		case ProposalConflict:
			out.Verdict = ProposalConflict
		case ProposalUnknown:
			if out.Verdict == ProposalCandidate {
				out.Verdict = ProposalUnknown
			}
		}
	}
	return out
}

// inspectWiringLocked examines current registry state without changing it.
// The caller must hold r.mu. Selection is a defensive copy.
func (r *Registry) inspectWiringLocked(req WiringRequirement) WiringFinding {
	if req.Scope == "" || !req.Contract.valid() {
		return WiringFinding{Verdict: ProposalUnknown, Reason: ReasonInvalidRequirement}
	}
	var matches []string
	for _, id := range sortedKeys(r.providers) {
		p := r.providers[id]
		if p.Scope == req.Scope && p.Offers == req.Contract {
			matches = append(matches, id)
		}
	}
	finding := WiringFinding{Candidates: matches}
	chosen := req.ProviderID
	if chosen != "" {
		p, exists := r.providers[chosen]
		if exists && p.Scope != req.Scope {
			finding.Verdict, finding.Reason = ProposalUnknown, ReasonNoKnownProvider
			return finding
		}
		if exists && p.Offers != req.Contract {
			finding.Verdict, finding.Reason = ProposalConflict, ReasonExplicitMismatch
			return finding
		}
		if !exists {
			finding.Verdict, finding.Reason = ProposalUnknown, ReasonNoKnownProvider
			return finding
		}
	} else {
		switch len(matches) {
		case 0:
			finding.Verdict, finding.Reason = ProposalUnknown, ReasonNoKnownProvider
			return finding
		case 1:
			chosen = matches[0]
		default:
			finding.Verdict, finding.Reason = ProposalConflict, ReasonAmbiguousProviders
			return finding
		}
	}
	assessment, ok := r.snapshot.Assessment(chosen)
	if !ok || !assessment.Eligible {
		finding.Verdict, finding.Reason = ProposalUnknown, ReasonDependencyIneligible
		if ok {
			finding.Detail = assessment.Reason
		}
		return finding
	}
	selected := r.selectionLocked(r.providers[chosen], assessment)
	finding.Verdict, finding.Reason, finding.Selected = ProposalCandidate, ReasonStructurallyEligible, &selected
	return finding
}
