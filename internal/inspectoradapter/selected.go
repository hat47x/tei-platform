// Package inspectoradapter binds an already-selected authoring Inspector
// to an actual Host-owned typed implementation. It is deliberately not a
// Provider resolver, publisher/conformance authority, or application runtime.
package inspectoradapter

import (
	"errors"
	"fmt"

	"tei/internal/composition"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// Registration is supplied by a trusted Host from its actual build and
// implementation inventory, never by a renderer or an Inspector request.
// These identifiers alone do NOT attest the bytes of a linked binary.
type Registration struct {
	ProviderID     string
	ArtifactDigest string
	Inspect        func(sourceindex.Subject) (operations.SubjectInspection, error)
	// CanonicalMessages declares that Inspect forwards the canonical Operation's
	// own errors without adding or rewriting text. A transport may then show the
	// message verbatim, the way it does for the in-process builtin. It is false
	// for an implementation the Host did not verify to do that, so its text stays
	// hidden.
	CanonicalMessages bool
}

// Selected is the typed authoring-side Inspector invoked after existing
// Composition has selected an exact contract and Provider. It preserves a
// Host guard and rejects stale selection before entering implementation code.
type Selected struct {
	coordinator  *composition.Coordinator
	registration Registration
}

// Bind checks the declaration selection against the trusted implementation
// supplied by the Host. Accepted contract provenance, actual artifact bytes,
// Required Source closure and Disclosure remain separate qualifications.
// No legacy application-runtime Service Carrier is used.
func Bind(
	registry *composition.Registry,
	selection composition.Selection,
	binding composition.Binding,
	acceptedContract composition.Contract,
	registration Registration,
	currentHostGuard func(composition.Binding) error,
) (*Selected, error) {
	if selection.Scope != "authoring-tool" ||
		selection.Contract != acceptedContract ||
		acceptedContract.ID == "" || acceptedContract.Version == "" || acceptedContract.Shape == "" {
		return nil, errors.New("inspector authoring selection or contract mismatch")
	}
	if registration.ProviderID == "" ||
		registration.ProviderID != selection.ProviderID ||
		registration.ProviderID != binding.ProviderID ||
		registration.ArtifactDigest != binding.ArtifactDigest ||
		registration.Inspect == nil {
		return nil, errors.New("inspector selected and registered implementation differ")
	}
	coordinator, err := composition.NewSelectedCoordinator(
		registry, selection, binding, composition.NewScope(), currentHostGuard,
	)
	if err != nil {
		return nil, fmt.Errorf("inspector selected binding rejected: %w", err)
	}
	return &Selected{coordinator: coordinator, registration: registration}, nil
}

// Inspect satisfies the same typed Port used by CLI and Studio. A current
// Registry/Host denial rejects invocation instead of silently using a builtin.
// The runtime invocation of this function is observable, but this read-only
// adapter does not provide independently verified executable provenance.
// CanonicalInspectionMessages reports whether this selection forwards the canonical
// Operation's error text unchanged. The CLI uses it to decide whether a failure
// message may be shown verbatim or must be replaced (#3420).
func (s *Selected) CanonicalInspectionMessages() bool {
	if s == nil {
		return false
	}
	return s.registration.CanonicalMessages
}

func (s *Selected) Inspect(subject sourceindex.Subject) (operations.SubjectInspection, error) {
	if s == nil || s.coordinator == nil {
		return operations.SubjectInspection{}, errors.New("inspector selected service unavailable")
	}
	lease, err := s.coordinator.Acquire()
	if err != nil {
		return operations.SubjectInspection{}, err
	}
	defer lease.Release()
	if lease.Binding.ProviderID != s.registration.ProviderID ||
		lease.Binding.ArtifactDigest != s.registration.ArtifactDigest {
		return operations.SubjectInspection{}, errors.New("inspector actual binding differs from registration")
	}
	value, err := s.registration.Inspect(subject)
	if err != nil {
		return operations.SubjectInspection{}, err
	}
	if err := operations.ValidateInspectionProjection(subject, value); err != nil {
		return operations.SubjectInspection{}, err
	}
	return value, nil
}
