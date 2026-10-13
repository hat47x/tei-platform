package main

import (
	"errors"
	"strings"

	"tei/internal/composition"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// ProjectAdvisoryContract is a reference-only service identity. It does not
// change the accepted order_entry command contract or grant host authority.
var ProjectAdvisoryContract = composition.Contract{
	ID: "tei.reference.project.customer-advice",
	Version: "v0",
	Shape: "customer-code-note-to-advisory-v0",
}

const projectAdvisoryScope = "reference-project-customer-advice"
const projectAdvisoryProviderID = "project.customer-advice.go"

// projectAdvisoryBridge exercises the existing composition Registry/Selection.
// Selection is only declaration eligibility; it is never an authorization token.
// The diagnostic host owns the Go implementation and invokes it locally.
type projectAdvisoryBridge struct {
	registry  *composition.Registry
	selection composition.Selection
	coordinator *composition.Coordinator
}

func newProjectAdvisoryBridge() (*projectAdvisoryBridge, error) {
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID:     projectAdvisoryProviderID,
		Scope:  projectAdvisoryScope,
		Offers: ProjectAdvisoryContract,
	}}, nil)
	if err != nil {
		return nil, err
	}
	if len(snapshot.Providers) != 1 || !snapshot.Providers[0].Eligible {
		return nil, errors.New("project advisory provider is not eligible")
	}
	selection, err := registry.Select(projectAdvisoryProviderID, projectAdvisoryScope)
	if err != nil {
		return nil, err
	}
	if selection.Contract != ProjectAdvisoryContract || selection.ProviderID != projectAdvisoryProviderID {
		return nil, errors.New("unexpected project advisory contract selection")
	}
	// This synthetic digest identifies the reference binary binding only; it is
	// not an artifact attestation. Production hosts must verify real bytes.
	binding := composition.Binding{
		ContractID: ProjectAdvisoryContract.ID,
		ContractVersion: ProjectAdvisoryContract.Version,
		ProviderID: projectAdvisoryProviderID,
		ArtifactDigest: "sha256:" + strings.Repeat("0", 64),
		Generation: 1,
	}
	// A production host must replace this reference guard with its current
	// permission, artifact, lease and fencing checks.
	guard := func(current composition.Binding) error {
		if current != binding {
			return errors.New("reference host binding changed")
		}
		return nil
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, binding, composition.NewScope(), guard)
	if err != nil {
		return nil, err
	}
	return &projectAdvisoryBridge{registry: registry, selection: selection, coordinator: coordinator}, nil
}

func (b *projectAdvisoryBridge) Evaluate(input customeradvice.Input) (customeradvice.Result, error) {
	if b == nil || b.registry == nil {
		return customeradvice.Result{}, errors.New("project advisory service unavailable")
	}
	current, err := b.registry.Select(projectAdvisoryProviderID, projectAdvisoryScope)
	if err != nil || current.Contract != ProjectAdvisoryContract ||
		current.ProviderID != b.selection.ProviderID || current.Epoch != b.selection.Epoch {
		return customeradvice.Result{}, errors.New("project advisory selection changed")
	}
	// The actual Go invocation remains diagnostic and host-owned.
	// Production invocation requires the Coordinator/Router guard, verified
	// artifact, permissions and lease before dispatch.
	lease, err := b.coordinator.Acquire()
	if err != nil {
		return customeradvice.Result{}, err
	}
	defer lease.Release()
	return customeradvice.Evaluate(input)
}
