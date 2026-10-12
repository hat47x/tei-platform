package composition

// ProposalCurrent checks that a design-time proposal is still consistent with
// the same Registry. It does not assert contract conformance, artifact trust,
// live process readiness or Host authority, and it does not reserve the state.
// The caller must perform a fresh Host admission check before any invocation.
func (r *Registry) ProposalCurrent(set WiringSetFinding) bool {
	if r == nil || set.Verdict != ProposalCandidate || set.Scope == "" ||
		set.Revision == 0 || len(set.Findings) == 0 {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if set.Revision > r.snapshot.Revision {
		return false
	}
	for _, finding := range set.Findings {
		if finding.Verdict != ProposalCandidate || finding.Selected == nil {
			return false
		}
		selected := *finding.Selected
		if selected.Revision != set.Revision || selected.Scope != set.Scope ||
			len(finding.VerificationTargets) != len(selected.DependencyEpochs)+1 {
			return false
		}
		// This uses the same selection gate that Host routing uses for the
		// declaration snapshot; supplying the identity-only Binding here
		// never grants permission to execute that Binding.
		identity := Binding{
			ProviderID: selected.ProviderID,
			ContractID: selected.Contract.ID,
			ContractVersion: selected.Contract.Version,
		}
		if r.validateSelectionLocked(selected, identity) != nil {
			return false
		}
		root := finding.VerificationTargets[0]
		if root.ProviderID != selected.ProviderID || root.Scope != set.Scope ||
			root.Contract != selected.Contract || root.Epoch != selected.Epoch {
			return false
		}
		seen := map[string]bool{root.ProviderID: true}
		for i, expected := range selected.DependencyEpochs {
			target := finding.VerificationTargets[i+1]
			if target.ProviderID != expected.ProviderID || target.Epoch != expected.Epoch ||
				target.Scope != set.Scope || seen[target.ProviderID] {
				return false
			}
			declared, ok := r.providers[target.ProviderID]
			if !ok || declared.Scope != target.Scope || declared.Offers != target.Contract {
				return false
			}
			seen[target.ProviderID] = true
		}
	}
	return true
}
