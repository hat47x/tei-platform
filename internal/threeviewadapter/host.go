// Package threeviewadapter binds a Host-selected three-view navigation
// Provider to the function that is actually entered, and lets the Host
// observe each call.
//
// It follows tei/internal/inspectoradapter: the existing composition Registry
// selects the Provider, the selection is bound to the Host's own compiled
// inventory, and no renderer or request names a Provider. It is not a plugin
// discovery mechanism, a Conformance authority or an authorization service.
//
// The two adapters repeat each other's selection and binding code on purpose
// for now; folding them into one generic Host is tracked as follow-up work so
// that this first three-view slice does not change the Inspector path.
package threeviewadapter

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
	"tei/internal/workspace"
)

// ErrWorkspaceOriginUnverified is returned when the Workspace a Provider says
// it read is not the root this Host selected.
var ErrWorkspaceOriginUnverified = errors.New("workspace-source-unverified: the Provider's Workspace is not the Host-selected root")

// Provider is one three-view implementation compiled into this Host. Navigate
// must be a named Go function and Entry its linked symbol. The Host reads the
// symbol of the function value it is about to call, so an inventory that
// labels one Provider's code with another Provider's ID is refused.
//
// The symbol is an observation of this process's own function value. It is
// NOT an attestation of artifact bytes, a signature or Conformance.
type Provider struct {
	ID       string
	Entry    string
	Navigate func(trustedRoot string, request operations.ThreeViewRequest) (operations.ThreeViewNavigation, error)
}

// Invocation is what the Host observed about one call. Each field names its
// own source; none of them is a grant.
type Invocation struct {
	// SelectedProviderID is the Registry's answer for the Host's selection.
	SelectedProviderID string
	// RegisteredProviderID is the inventory key the Host bound to that answer.
	RegisteredProviderID string
	// ObservedEntry is the linked symbol of the function the Host entered.
	ObservedEntry string
	// Request is the request the Host handed to the Provider.
	Request operations.ThreeViewRequest
	// HostRoot is the root the Host bound; HostManifest is the Host's own read
	// of that root's manifest after the Provider returned.
	HostRoot     string
	HostManifest workspace.ManifestIdentity
	// ReturnedWorkspace is what the Provider reported. It is a claim.
	ReturnedWorkspace *workspace.WorkspaceIdentity
	// OriginVerified is true only when the two manifest identities are equal.
	// The source file's own bytes are not part of this comparison.
	OriginVerified bool
	// Err is the error the Host returned to its caller, if any.
	Err error
}

// Host is a static composition of three-view Providers for a developer tool.
type Host struct {
	registry  *composition.Registry
	selection composition.Selection
	contract  composition.Contract
	digest    string
	inventory map[string]Provider
	observer  func(Invocation)
}

// Contract is the composition contract a three-view Provider offers.
func Contract() composition.Contract {
	return composition.Contract{ID: "tei.operations.three-view-navigation", Version: "0", Shape: "bounded-reference-v0"}
}

// NewHostSelecting records the bytes of the running executable and selects
// providerID from inventory through the composition Registry. An ID that is
// not in the inventory fails closed; there is no default Provider.
//
// The digest is a local file observation. It does not prove which Provider
// symbol was linked or that any meaning or authorization was accepted.
func NewHostSelecting(providerID string, inventory []Provider) (*Host, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, errors.New("three-view host executable is unavailable")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("three-view host executable cannot be read")
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.New("three-view host executable cannot be hashed")
	}
	return newHost("sha256:"+hex.EncodeToString(h.Sum(nil)), providerID, inventory)
}

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

func newHost(digest, providerID string, inventory []Provider) (*Host, error) {
	contract := Contract()
	entries := make(map[string]Provider, len(inventory))
	offers := make([]composition.Provider, 0, len(inventory))
	for _, p := range inventory {
		if p.ID == "" || p.Entry == "" || p.Navigate == nil {
			return nil, errors.New("three-view inventory entry is incomplete")
		}
		if _, dup := entries[p.ID]; dup {
			return nil, fmt.Errorf("three-view inventory repeats %s", p.ID)
		}
		// Compare the declaration with the function this process would call.
		if got := observedEntry(p.Navigate); got != p.Entry {
			return nil, fmt.Errorf("three-view Provider %s declares entry %s but links %s", p.ID, p.Entry, got)
		}
		entries[p.ID] = p
		offers = append(offers, composition.Provider{ID: p.ID, Scope: "authoring-tool", Offers: contract})
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, offers, nil); err != nil {
		return nil, fmt.Errorf("register three-view Provider: %w", err)
	}
	selection, err := registry.Select(providerID, "authoring-tool")
	if err != nil {
		return nil, fmt.Errorf("select three-view Provider: %w", err)
	}
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: selection.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	if !binding.Valid() {
		return nil, errors.New("three-view host digest is invalid")
	}
	return &Host{
		registry: registry, selection: selection, contract: contract, digest: digest, inventory: entries,
	}, nil
}

// Observe installs a Host-owned receiver of every Invocation made through
// services bound afterwards. No renderer or Provider can set it.
func (h *Host) Observe(receiver func(Invocation)) {
	if h != nil {
		h.observer = receiver
	}
}

// Selected is the bound service. It satisfies studiohost.ThreeViewService.
type Selected struct {
	coordinator *composition.Coordinator
	providerID  string
	digest      string
	navigate    func(operations.ThreeViewRequest) (operations.ThreeViewNavigation, error)
}

// Bind installs the selected implementation for a Host-trusted Workspace
// root. The entry is looked up by the Registry's selection, never by a second
// caller-supplied ID.
func (h *Host) Bind(trustedRoot string) (*Selected, error) {
	if h == nil || h.registry == nil || trustedRoot == "" {
		return nil, errors.New("three-view Host or workspace root is missing")
	}
	entry, ok := h.inventory[h.selection.ProviderID]
	if !ok {
		return nil, errors.New("selected three-view Provider is not in the Host inventory")
	}
	if h.selection.Scope != "authoring-tool" || h.selection.Contract != h.contract {
		return nil, errors.New("three-view authoring selection or contract mismatch")
	}
	binding := composition.Binding{
		ContractID: h.contract.ID, ContractVersion: h.contract.Version,
		ProviderID: h.selection.ProviderID, ArtifactDigest: h.digest, Generation: 1,
	}
	guard := func(current composition.Binding) error {
		if current != binding {
			return errors.New("three-view registration was changed")
		}
		return nil
	}
	coordinator, err := composition.NewSelectedCoordinator(h.registry, h.selection, binding, composition.NewScope(), guard)
	if err != nil {
		return nil, fmt.Errorf("three-view selected binding rejected: %w", err)
	}
	observer := h.observer
	selectedID := h.selection.ProviderID
	return &Selected{
		coordinator: coordinator, providerID: entry.ID, digest: h.digest,
		navigate: func(request operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
			seen := Invocation{
				SelectedProviderID:   selectedID,
				RegisteredProviderID: entry.ID,
				ObservedEntry:        observedEntry(entry.Navigate),
				Request:              request,
				HostRoot:             trustedRoot,
			}
			value, err := entry.Navigate(trustedRoot, request)
			if err == nil {
				seen.ReturnedWorkspace = value.Workspace
				// The Host reads its own root. It does not ask the Provider
				// where the answer came from and then believe it.
				manifest, observeErr := workspace.ObserveManifestIdentity(trustedRoot)
				seen.HostManifest = manifest
				seen.OriginVerified = observeErr == nil && value.Workspace != nil &&
					value.Workspace.Manifest == manifest
				if !seen.OriginVerified {
					value, err = operations.ThreeViewNavigation{}, ErrWorkspaceOriginUnverified
				}
			}
			seen.Err = err
			if observer != nil {
				observer(seen)
			}
			return value, err
		},
	}, nil
}

// Navigate enters the selected Provider after rechecking the current binding,
// and checks the answer against the Operation contract before returning it.
// A Registry or Host denial rejects the call; nothing else is tried instead.
func (s *Selected) Navigate(request operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	if s == nil || s.coordinator == nil {
		return operations.ThreeViewNavigation{}, errors.New("three-view selected service unavailable")
	}
	lease, err := s.coordinator.Acquire()
	if err != nil {
		return operations.ThreeViewNavigation{}, err
	}
	defer lease.Release()
	if lease.Binding.ProviderID != s.providerID || lease.Binding.ArtifactDigest != s.digest {
		return operations.ThreeViewNavigation{}, errors.New("three-view actual binding differs from registration")
	}
	value, err := s.navigate(request)
	if err != nil {
		return operations.ThreeViewNavigation{}, err
	}
	if err := operations.ValidateThreeViewNavigation(request, value); err != nil {
		return operations.ThreeViewNavigation{}, err
	}
	return value, nil
}
