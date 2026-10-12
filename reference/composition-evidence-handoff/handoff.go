// Package handoff reconciles design-time provider targets with references to
// external conformance results. It neither validates nor grants trust.
package handoff

import (
	"sort"
	"strings"

	"tei/internal/composition"
)

// EvidenceReference is a locator to a separately validated #951 resolution.
// It is untrusted input, NOT a conformance assertion. Subject is the exact
// Registry declaration to which the external resolver's result is attributed.
// A trusted caller must retrieve and check the external evidence independently.
type EvidenceReference struct {
	Subject               composition.VerificationTarget
	AuthorityScope        string
	PluginID              string
	PluginVersion         string
	ImplementationID      string
	ArtifactDigest        string
	ContractFingerprint   string
	ConformanceProfile    string
	EvidenceFingerprint   string
	ResolutionFingerprint string
}

type Diagnostic string

const (
	ReferenceLinked     Diagnostic = "reference-linked-unverified"
	ReferenceMissing    Diagnostic = "reference-missing"
	ReferenceStale      Diagnostic = "reference-stale"
	ReferenceMismatch   Diagnostic = "reference-target-mismatch"
	ReferenceIncomplete Diagnostic = "reference-incomplete"
	ReferenceAmbiguous  Diagnostic = "reference-ambiguous"
)

// SubjectCheck links a structural target to at most one reference. Linked
// explicitly means only that exact declaration metadata was reconciled.
type SubjectCheck struct {
	Target    composition.VerificationTarget
	State     Diagnostic
	Reference *EvidenceReference
}

type HandoffReport struct {
	Revision uint64
	Scope    string
	// Blocked means the structural proposal itself was not eligible.
	Blocked  bool
	Subjects []SubjectCheck
}

// Reconcile matches an already-obtained Registry proposal to external evidence
// locators without importing a second Resolver into the Composition Kernel.
// It cannot infer the complete contract fingerprint from Contract.Shape, or
// validate issuer authority, evidence signatures, revocation or host rights.
func Reconcile(set composition.WiringSetFinding, refs []EvidenceReference) HandoffReport {
	out := HandoffReport{Revision: set.Revision, Scope: set.Scope, Blocked: true}
	if set.Verdict != composition.ProposalCandidate || set.Scope == "" || set.Revision == 0 || len(set.Findings) == 0 {
		return out
	}
	targets := map[string]composition.VerificationTarget{}
	for _, f := range set.Findings {
		if f.Verdict != composition.ProposalCandidate || f.Selected == nil || f.Selected.Revision != set.Revision || f.Selected.Scope != set.Scope || len(f.VerificationTargets) == 0 {
			return out
		}
		if f.VerificationTargets[0].ProviderID != f.Selected.ProviderID || f.VerificationTargets[0].Epoch != f.Selected.Epoch || f.VerificationTargets[0].Contract != f.Selected.Contract || len(f.VerificationTargets) != len(f.Selected.DependencyEpochs)+1 {
			return out
		}
		expected := make(map[string]uint64, len(f.Selected.DependencyEpochs))
		for _, dep := range f.Selected.DependencyEpochs {
			if dep.ProviderID == "" || dep.Epoch == 0 || dep.ProviderID == f.Selected.ProviderID {
				return out
			}
			expected[dep.ProviderID] = dep.Epoch
		}
		if len(expected) != len(f.Selected.DependencyEpochs) {
			return out
		}
		seenDependencies := map[string]bool{}
		for i, target := range f.VerificationTargets {
			if i > 0 {
				if target.ProviderID == f.Selected.ProviderID || seenDependencies[target.ProviderID] || expected[target.ProviderID] != target.Epoch {
					return out
				}
				seenDependencies[target.ProviderID] = true
			}

			if target.ProviderID == "" || target.Scope != set.Scope || target.Epoch == 0 || target.Contract.ID == "" || target.Contract.Version == "" || target.Contract.Shape == "" {
				return out
			}
			if old, seen := targets[target.ProviderID]; seen && old != target {
				return out
			}
			targets[target.ProviderID] = target
		}
		if len(seenDependencies) != len(expected) {
			return out
		}
	}
	ids := make([]string, 0, len(targets))
	for id := range targets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	// Retain only locators for providers actually requested by the proposal.
	// Untrusted, unrelated tenants and providers must not expand the index
	// even if the caller supplies a very large catalog of evidence references.
	type subjectKey struct{ providerID, scope string }
	bySubject := make(map[subjectKey][]EvidenceReference, len(targets))
	for _, ref := range refs {
		target, needed := targets[ref.Subject.ProviderID]
		if !needed || target.Scope != ref.Subject.Scope {
			continue
		}
		key := subjectKey{ref.Subject.ProviderID, ref.Subject.Scope}
		bySubject[key] = append(bySubject[key], ref)
	}
	out.Subjects = make([]SubjectCheck, 0, len(ids))
	for _, id := range ids {
		target := targets[id]
		check := SubjectCheck{Target: target, State: ReferenceMissing}
		var exact []EvidenceReference
		hasStale, hasMismatch := false, false
		for _, ref := range bySubject[subjectKey{target.ProviderID, target.Scope}] {
			if ref.Subject.Contract != target.Contract {
				hasMismatch = true
				continue
			}
			if ref.Subject.Epoch != target.Epoch {
				hasStale = true
				continue
			}
			exact = append(exact, ref)
		}
		switch {
		case len(exact) > 1:
			check.State = ReferenceAmbiguous
		case len(exact) == 1:
			if incomplete(exact[0]) {
				check.State = ReferenceIncomplete
			} else {
				check.State = ReferenceLinked
				copyOfRef := exact[0]
				check.Reference = &copyOfRef
			}
		case hasMismatch:
			check.State = ReferenceMismatch
		case hasStale:
			check.State = ReferenceStale
		}
		out.Subjects = append(out.Subjects, check)
	}
	out.Blocked = false
	return out
}

func incomplete(ref EvidenceReference) bool {
	for _, value := range []string{
		ref.AuthorityScope, ref.PluginID, ref.PluginVersion,
		ref.ImplementationID, ref.ContractFingerprint,
		ref.ConformanceProfile, ref.EvidenceFingerprint, ref.ResolutionFingerprint,
	} {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	digest := ref.ArtifactDigest
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
		return true
	}
	for _, c := range digest[len("sha256:"):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return true
		}
	}
	return false
}
