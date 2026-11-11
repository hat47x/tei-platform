package inspectoradapter

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"

	"tei/internal/composition"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// LocalInspectorHost is a deliberately narrow static composition for a
// developer tool. It uses the existing Registry and selected call path.
// It is NOT a plugin discovery mechanism, a Conformance publisher, or an
// authorization service. Which Provider a Project wants belongs to the
// existing Profile and Resolver owners (#951/#1016); this Host only accepts
// that answer as a Provider ID and binds it to its own compiled inventory.
type LocalInspectorHost struct {
	registry  *composition.Registry
	selection composition.Selection
	contract  composition.Contract
	digest    string
	inventory map[string]LocalProvider
	observer  func(Invocation)
}

const LocalInspectorProviderID = "tei.reference.inspector.local"

// ErrWorkspaceOriginUnverified is returned when the Workspace a Provider says
// it read is not the Workspace this Host selected, or when the Host could not
// observe its own root. A well-formed answer about another Workspace is not
// accepted because its Subject ID and fingerprints are valid.
var ErrWorkspaceOriginUnverified = errors.New("workspace-source-unverified: the Provider's Workspace is not the Host-selected root")

// LocalProvider is one authoring Inspector implementation compiled into this
// Host. Inspect must be a named Go function, and Entry must be its linked
// symbol. The Host reads the symbol of the function value it is about to
// call and refuses an inventory whose declaration and linked function differ,
// so "Provider B is selected, A's code runs" cannot hide behind an ID label
// and an identical answer.
//
// The symbol is an observation of this process's own function value. It is
// NOT an attestation of artifact bytes, a signature, or Conformance (#943,
// #3267, #951).
type LocalProvider struct {
	ID      string
	Entry   string
	Inspect func(trustedRoot string, subject sourceindex.Subject) (operations.SubjectInspection, error)
	// CanonicalMessages has the meaning of Registration.CanonicalMessages.
	CanonicalMessages bool
}

// Invocation is what the Host observed about one call. Each field names its
// source, because they are separate facts: a Provider ID is a declaration, an
// entry symbol is what the Host read from the function it called, and
// ReturnedWorkspace is the Provider's own statement. Nothing here is a grant.
type Invocation struct {
	// SelectedProviderID is the Registry's answer for the requested selection.
	SelectedProviderID string
	// RegisteredProviderID is the inventory key the Host bound to that answer.
	RegisteredProviderID string
	// ObservedEntry is the linked symbol of the function the Host entered.
	ObservedEntry string
	// HostRoot is the root the Host bound; HostManifest is the Host's own read
	// of that root's manifest bytes after the Provider returned.
	HostRoot     string
	HostManifest workspace.ManifestIdentity
	// ReturnedWorkspace is what the Provider reported. It is a claim.
	ReturnedWorkspace *workspace.WorkspaceIdentity
	// OriginVerified is true only when the Host's manifest observation equals
	// the Provider's reported manifest identity. Semantic Source bytes are not
	// part of this comparison.
	OriginVerified bool
	// Err is the error the Host returned to its caller, if any.
	Err error
}

// ReferenceInventory is the inventory of the normal CLI and Studio: the one
// explicitly compiled reference Inspector, which is the canonical Operation.
func ReferenceInventory() []LocalProvider {
	return []LocalProvider{{
		ID:                LocalInspectorProviderID,
		Entry:             "tei/internal/operations.InspectSubject",
		Inspect:           operations.InspectSubject,
		CanonicalMessages: true,
	}}
}

// NewLocalInspectorHost selects the reference Provider from the reference
// inventory. See NewLocalInspectorHostSelecting.
func NewLocalInspectorHost() (*LocalInspectorHost, error) {
	return NewLocalInspectorHostSelecting(LocalInspectorProviderID, ReferenceInventory())
}

// NewLocalInspectorHostSelecting records the bytes read from the running
// host's executable path and selects providerID from inventory through the
// existing Registry. An ID that is not in the inventory fails closed.
//
// The digest is a local file observation, NOT proof that a particular
// Provider symbol was linked, that the OS loaded these exact bytes, or that
// the Provider's meaning or authorization was accepted.
func NewLocalInspectorHostSelecting(providerID string, inventory []LocalProvider) (*LocalInspectorHost, error) {
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
	return newLocalInspectorHost("sha256:"+hex.EncodeToString(h.Sum(nil)), providerID, inventory)
}

// newLocalInspectorHostFromObservedDigest is the bounded test seam for
// composed host selection. It cannot replace a verified build manifest.
func newLocalInspectorHostFromObservedDigest(digest string) (*LocalInspectorHost, error) {
	return newLocalInspectorHost(digest, LocalInspectorProviderID, ReferenceInventory())
}

// observedEntry reads the linked symbol of a function value. It is empty for
// a nil or non-function value.
func observedEntry(fn any) string {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func || v.IsNil() {
		return ""
	}
	f := runtime.FuncForPC(v.Pointer())
	if f == nil {
		return ""
	}
	return f.Name()
}

func newLocalInspectorHost(digest, providerID string, inventory []LocalProvider) (*LocalInspectorHost, error) {
	contract := composition.Contract{ID: "tei.operations.inspect", Version: "0", Shape: "bounded-reference-v0"}
	entries := make(map[string]LocalProvider, len(inventory))
	offers := make([]composition.Provider, 0, len(inventory))
	for _, p := range inventory {
		if p.ID == "" || p.Entry == "" || p.Inspect == nil {
			return nil, errors.New("local Inspector inventory entry is incomplete")
		}
		if _, dup := entries[p.ID]; dup {
			return nil, fmt.Errorf("local Inspector inventory repeats %s", p.ID)
		}
		// Compare the declaration with the function this process would call.
		// The Provider ID is not consulted: it is a label on the entry.
		if got := observedEntry(p.Inspect); got != p.Entry {
			return nil, fmt.Errorf("local Inspector %s declares entry %s but links %s", p.ID, p.Entry, got)
		}
		entries[p.ID] = p
		offers = append(offers, composition.Provider{ID: p.ID, Scope: "authoring-tool", Offers: contract})
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, offers, nil); err != nil {
		return nil, fmt.Errorf("register local Inspector: %w", err)
	}
	selection, err := registry.Select(providerID, "authoring-tool")
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
		inventory: entries,
	}, nil
}

// Observe installs a Host-owned receiver of every Invocation made through
// Selected services bound afterwards. It is for the composing Host and its
// tests; no renderer or Provider can set it. A nil receiver disables it.
func (h *LocalInspectorHost) Observe(receiver func(Invocation)) {
	if h != nil {
		h.observer = receiver
	}
}

// Bind installs the selected implementation for a Host-trusted workspace
// root. CLI accepts a root from the local OS principal; Studio supplies only
// roots resolved through its startup allowlist. No renderer chooses the root.
// Host permission and Source revision remain separate from this local binding.
func (h *LocalInspectorHost) Bind(trustedRoot string) (*Selected, error) {
	if h == nil || h.registry == nil || trustedRoot == "" {
		return nil, errors.New("local Inspector Host or workspace root is missing")
	}
	// The entry is looked up by the Registry's selection, never by a second
	// caller-supplied ID, so the registered label cannot name another entry.
	entry, ok := h.inventory[h.selection.ProviderID]
	if !ok {
		return nil, errors.New("selected local Inspector is not in the Host inventory")
	}
	binding := composition.Binding{
		ContractID: h.contract.ID, ContractVersion: h.contract.Version,
		ProviderID: h.selection.ProviderID, ArtifactDigest: h.digest, Generation: 1,
	}
	observer := h.observer
	selectedID := h.selection.ProviderID
	registration := Registration{
		ProviderID:     entry.ID,
		ArtifactDigest: h.digest,
		Inspect: func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
			seen := Invocation{
				SelectedProviderID:   selectedID,
				RegisteredProviderID: entry.ID,
				ObservedEntry:        observedEntry(entry.Inspect),
				HostRoot:             trustedRoot,
			}
			value, err := entry.Inspect(trustedRoot, subject)
			if err == nil {
				seen.ReturnedWorkspace = value.Workspace
				// The Host reads its own root. It does not ask the Provider
				// where the answer came from and then believe it.
				manifest, observeErr := workspace.ObserveManifestIdentity(trustedRoot)
				seen.HostManifest = manifest
				seen.OriginVerified = observeErr == nil && value.Workspace != nil &&
					value.Workspace.Manifest == manifest
				if !seen.OriginVerified {
					value, err = operations.SubjectInspection{}, ErrWorkspaceOriginUnverified
				}
			}
			seen.Err = err
			if observer != nil {
				observer(seen)
			}
			return value, err
		},
		CanonicalMessages: entry.CanonicalMessages,
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
