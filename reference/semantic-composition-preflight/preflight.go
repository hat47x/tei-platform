// Package semanticcompositionpreflight connects the existing design-time
// Composition Registry to already-locked, data-only semantic exports.
// This is an isolated reference, not a production selector or authorization.
package semanticcompositionpreflight

import (
	"errors"
	"fmt"

	"tei/internal/composition"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/semantic"
	"tei/internal/stdlib"
	"tei/internal/validation"
	"tei/internal/workspace"
	semanticdistribution "tei/reference/semantic-distribution"
)

var (
	ErrSelectionUnavailable = errors.New("semantic asset selection is not a structural candidate")
	ErrProposalChanged      = errors.New("semantic asset selection changed during inspection")
	ErrInvalidSemanticView  = errors.New("selected semantic view is invalid")
)

// SelectedAsset is a caller-owned, exact design-time selection of one
// namespace. ProviderID identifies the Registry's retrieval service, while
// Expected.Package identifies a separate build artifact package. Neither
// identity implies authority over the other; trusted publication and artifact
// authorization must be verified independently by the actual host.
type SelectedAsset struct {
	ProviderID string
	Contract   composition.Contract
	Lock       plugin.BuildArtifactLock
	Build      plugin.LockedBuild
	Expected   semanticdistribution.Expectation
	Fetch      func(string) ([]byte, error)
}

// Report holds derived observations only. A candidate Registry proposal,
// a matching artifact digest and successful Semantic validation do not
// establish Publisher trust, current Host permission or Project Acceptance.
type Report struct {
	Wiring      composition.WiringSetFinding
	References  semanticdistribution.SelectedNamespaceSetAudit
	Diagnostics []validation.Diagnostic
}

// Inspect verifies explicitly chosen retrieval services from one Registry
// snapshot, then checks each exact locked data-only artifact with the existing
// semantic distribution adapter. It inspects the local and imported Semantic
// namespaces together using the existing validator and dependency audit.
//
// The service coeffect graph and business Semantic reference graph are
// deliberately independent. In particular, reciprocal Data.Relations do not
// become cyclic service dependencies. No provider is discovered, no plugin
// is activated, and neither the caller's local model nor Source Index changes.
//
// This is a time-of-check observation, not a reservation. The Host must
// independently verify current selection, trust and authorization at use time.
func Inspect(registry *composition.Registry, scope string, local []semantic.Namespace, assets []SelectedAsset) (Report, error) {
	if registry == nil || scope == "" || len(assets) == 0 {
		return Report{}, errors.New("registry, scope and at least one explicit asset selection are required")
	}
	seenNamespaces := make(map[string]bool, len(assets))
	requirements := make([]composition.WiringRequirement, 0, len(assets))
	for _, asset := range assets {
		namespace := asset.Expected.NamespaceID
		if namespace == "" || asset.ProviderID == "" || seenNamespaces[namespace] {
			return Report{}, fmt.Errorf("missing, duplicate or ambiguous exact semantic selection %q", namespace)
		}
		seenNamespaces[namespace] = true
		requirements = append(requirements, composition.WiringRequirement{
			Contract: asset.Contract, Scope: scope, ProviderID: asset.ProviderID,
			SlotID: "semantic:" + namespace,
		})
	}
	wiring := registry.InspectWiringSet(requirements)
	report := Report{Wiring: wiring}
	if wiring.Verdict != composition.ProposalCandidate || len(wiring.Findings) != len(assets) {
		return report, ErrSelectionUnavailable
	}
	for i, finding := range wiring.Findings {
		if finding.Verdict != composition.ProposalCandidate || finding.Selected == nil ||
			finding.Selected.ProviderID != assets[i].ProviderID ||
			finding.Selected.Scope != scope || finding.Selected.Contract != assets[i].Contract {
			return report, ErrSelectionUnavailable
		}
	}
	if !registry.ProposalCurrent(wiring) {
		return report, ErrProposalChanged
	}
	available := make([]semantic.Namespace, 0, len(local)+len(assets))
	available = append(available, local...)
	for _, asset := range assets {
		// The exact BuildArtifactLock is the existing build-output contract.
		// The Registry has no knowledge of its artifact digest or publisher.
		model, err := semanticdistribution.InspectLockedSemanticAsset(
			asset.Lock, asset.Build, asset.Expected, asset.Fetch,
		)
		if err != nil {
			return report, fmt.Errorf("selected namespace %s: %w", asset.Expected.NamespaceID, err)
		}
		available = append(available, model)
	}
	audit, err := semanticdistribution.AuditSelectedNamespaceSet(available)
	if err != nil {
		return report, fmt.Errorf("selected namespace set: %w", err)
	}
	report.References = audit
	if len(audit.Unmatched) != 0 {
		return report, fmt.Errorf("%w: %d unmatched semantic references", ErrInvalidSemanticView, len(audit.Unmatched))
	}

	// Validate only the Semantic view, not a fabricated complete Application
	// World. The standard prelude is supplied by TEI itself, never by an
	// untrusted semantic export. Clone the slice before normalization.
	model := ir.World{Semantics: make([]semantic.Namespace, 0, 1+len(available))}
	model.Semantics = append(model.Semantics, stdlib.SemanticNamespace())
	model.Semantics = append(model.Semantics, available...)
	workspace.Normalize(&model)
	report.Diagnostics = validation.Validate(model)
	if validation.HasErrors(report.Diagnostics) {
		return report, ErrInvalidSemanticView
	}
	// A Registry modification while fetching or validating invalidates the
	// preview; this does not constitute a Host invocation admission.
	if !registry.ProposalCurrent(wiring) {
		return report, ErrProposalChanged
	}
	return report, nil
}
