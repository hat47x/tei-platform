package inspectoradapter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"tei/internal/composition"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// LocalInspectorHost is a deliberately narrow static reference composition for
// a developer tool. It uses the existing Registry and selected call path.
// It is NOT a plugin discovery mechanism, a Conformance publisher, or an
// authorization service. Dynamic selection belongs to the existing Profile
// and Resolver owners (#951/#1016).
type LocalInspectorHost struct {
	registry  *composition.Registry
	selection composition.Selection
	contract  composition.Contract
	digest    string
}

const LocalInspectorProviderID = "tei.reference.inspector.local"

// NewLocalInspectorHost records the bytes read from the running host's
// executable path and chooses the explicitly compiled reference Provider.
// This digest is a local file observation, NOT proof that a particular
// Provider symbol was linked, that the OS loaded these exact bytes, or that
// the Provider's meaning or authorization was accepted.
func NewLocalInspectorHost() (*LocalInspectorHost, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, errors.New("inspector host executable is unavailable")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("inspector host executable cannot be read")
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.New("inspector host executable cannot be hashed")
	}
	return newLocalInspectorHostFromObservedDigest("sha256:" + hex.EncodeToString(h.Sum(nil)))
}

// newLocalInspectorHostFromObservedDigest is the bounded test seam for
// composed host selection. It cannot replace a verified build manifest.
func newLocalInspectorHostFromObservedDigest(digest string) (*LocalInspectorHost, error) {
	contract := composition.Contract{ID: "tei.operations.inspect", Version: "0", Shape: "bounded-reference-v0"}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{
		ID: LocalInspectorProviderID, Scope: "authoring-tool", Offers: contract,
	}}, nil); err != nil {
		return nil, fmt.Errorf("register local Inspector: %w", err)
	}
	selection, err := registry.Select(LocalInspectorProviderID, "authoring-tool")
	if err != nil {
		return nil, fmt.Errorf("select local Inspector: %w", err)
	}
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: selection.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	if !binding.Valid() {
		return nil, errors.New("local Inspector host digest is invalid")
	}
	return &LocalInspectorHost{
		registry: registry, selection: selection, contract: contract, digest: digest,
	}, nil
}

// Bind installs the reference implementation for a Host-trusted workspace
// root. CLI accepts a root from the local OS principal; Studio supplies only
// roots resolved through its startup allowlist. No renderer chooses the root.
// Host permission and Source revision remain separate from this local binding.
func (h *LocalInspectorHost) Bind(trustedRoot string) (*Selected, error) {
	if h == nil || h.registry == nil || trustedRoot == "" {
		return nil, errors.New("local Inspector Host or workspace root is missing")
	}
	binding := composition.Binding{
		ContractID: h.contract.ID, ContractVersion: h.contract.Version,
		ProviderID: h.selection.ProviderID, ArtifactDigest: h.digest, Generation: 1,
	}
	registration := Registration{
		ProviderID:     LocalInspectorProviderID,
		ArtifactDigest: h.digest,
		Inspect: func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
			return operations.InspectSubject(trustedRoot, subject)
		},
		// This registration is the canonical Operation itself, so its failure
		// messages are the ones every other transport also reports.
		CanonicalMessages: true,
	}
	// This guard enforces the Host's immutable registration binding.
	// It deliberately claims no user/tenant privilege or OS resource grant.
	guard := func(current composition.Binding) error {
		if current != binding {
			return errors.New("local Inspector registration was changed")
		}
		return nil
	}
	return Bind(h.registry, h.selection, binding, h.contract, registration, guard)
}
