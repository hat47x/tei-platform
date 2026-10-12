package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

// ServiceProviderSelection is the exact logical selection supplied by the
// future Contract resolver. It is not itself Accepted Contract evidence.
type ServiceProviderSelection struct {
	ContractID          string          `json:"contractId"`
	ContractVersion     string          `json:"contractVersion"`
	ContractFingerprint string          `json:"contractFingerprint"`
	Scope               string          `json:"scope"`
	Provider            plugin.Identity `json:"provider"`
	ImplementationID    string          `json:"implementationId"`
	ArtifactDigest      string          `json:"artifactDigest"`
}

// ServiceCarrierBinding belongs to a Host-selected EnvironmentPlan. It maps
// an exact Service selection to one EXISTING runtime carrier, without creating
// a second runtime authority or treating legacy CapabilityKind as the
// meaning of the Service Contract.
//
// This is a transitional read-only route. The Host must independently
// accept the Contract and Provider conformance before publishing a binding.
type ServiceCarrierBinding struct {
	ServiceProviderSelection
	CarrierKind plugin.CapabilityKind `json:"carrierKind"`
	CarrierID   core.CanonicalID      `json:"carrierId"`
	Operation   string                `json:"operation"`
}

// ServiceCarrierInvocation keeps semantic routing identity distinct from
// existing Runtime provenance. Neither one certifies Contract publication,
// shape validity or Provider conformance merely by being present.
type ServiceCarrierInvocation struct {
	Selection       ServiceProviderSelection `json:"selection"`
	Runtime         Invocation               `json:"runtime"`
	AuthorityLimits []string                 `json:"authorityLimits"`
}

func validateServiceSelection(selection ServiceProviderSelection) error {
	if strings.TrimSpace(selection.ContractID) == "" ||
		!core.ValidNumericVersion(selection.ContractVersion) {
		return fmt.Errorf("service selection needs an exact Contract ID and numeric version")
	}
	if err := validateRuntimeSHA256("service contract fingerprint", selection.ContractFingerprint); err != nil {
		return err
	}
	if selection.Scope != "application-runtime" {
		return fmt.Errorf("service carrier only supports application-runtime scope")
	}
	if selection.Provider.ID == "" ||
		!core.ValidNumericVersion(selection.Provider.Version) ||
		strings.TrimSpace(selection.ImplementationID) == "" {
		return fmt.Errorf("service selection needs an exact Provider and implementation")
	}
	if err := validateRuntimeSHA256("service artifact digest", selection.ArtifactDigest); err != nil {
		return err
	}
	return nil
}

// InvokeSelectedServiceCarrier bridges a pre-selected Service to existing
// Registry.InvokeWithProvenance. It DOES NOT select providers or validate
// domain request/result shapes. The trusted Host is responsible for sourcing
// the EnvironmentPlan and the Contract/Conformance evidence. In particular,
// a caller-supplied plan fingerprint is not an issuer trust root.
//
// All comparisons run BEFORE Runtime acquires a resource lease or launches a
// process. The legacy carrier kind/ID are selected solely from the Host plan,
// never from the caller's opaque business request.
func (r Registry) InvokeSelectedServiceCarrier(
	ctx context.Context, plan EnvironmentPlan,
	selection ServiceProviderSelection, requestID string, payload json.RawMessage,
) (ServiceCarrierInvocation, error) {
	if err := validateServiceSelection(selection); err != nil {
		return ServiceCarrierInvocation{}, err
	}
	if plan.RuntimePolicyFingerprint == "" ||
		plan.RuntimePolicyFingerprint != PolicyFingerprint(plan) {
		return ServiceCarrierInvocation{}, fmt.Errorf("service carrier Environment Policy fingerprint is missing or stale")
	}
	var binding *ServiceCarrierBinding
	for i := range plan.ServiceBindings {
		item := &plan.ServiceBindings[i]
		if item.ContractID != selection.ContractID ||
			item.Scope != selection.Scope ||
			item.ContractVersion != selection.ContractVersion {
			continue
		}
		if binding != nil {
			return ServiceCarrierInvocation{}, fmt.Errorf("service carrier has ambiguous Host bindings")
		}
		binding = item
	}
	if binding == nil {
		return ServiceCarrierInvocation{}, fmt.Errorf("service carrier has no Host binding")
	}
	if binding.ServiceProviderSelection != selection {
		return ServiceCarrierInvocation{}, fmt.Errorf("service selection differs from Host binding")
	}
	if binding.CarrierKind == "" || binding.CarrierID == "" ||
		strings.TrimSpace(binding.Operation) == "" {
		return ServiceCarrierInvocation{}, fmt.Errorf("service carrier Host binding is incomplete")
	}
	capability, ok := plan.Capability(binding.CarrierKind, binding.CarrierID)
	if !ok || capability.Plugin != selection.Provider {
		return ServiceCarrierInvocation{}, fmt.Errorf("service carrier does not match unique Runtime capability")
	}
	plugins := 0
	for _, installed := range plan.Plugins {
		if installed.Plugin == selection.Provider {
			plugins++
			if installed.Artifact.Digest != selection.ArtifactDigest {
				return ServiceCarrierInvocation{}, fmt.Errorf("service Provider artifact differs from Environment Plan")
			}
		}
	}
	if plugins != 1 {
		return ServiceCarrierInvocation{}, fmt.Errorf("service carrier does not match a unique installed Provider")
	}
	if len(payload) == 0 || !json.Valid(payload) {
		return ServiceCarrierInvocation{}, fmt.Errorf("service request must be a complete JSON payload")
	}
	invocation, err := r.InvokeWithProvenance(
		ctx, plan, binding.CarrierKind, binding.CarrierID,
		binding.Operation, requestID, payload,
	)
	if err != nil {
		return ServiceCarrierInvocation{}, err
	}
	if invocation.Provenance.Plugin != selection.Provider ||
		invocation.Provenance.Artifact.Digest != selection.ArtifactDigest ||
		invocation.Provenance.CapabilityKind != binding.CarrierKind ||
		invocation.Provenance.CapabilityID != binding.CarrierID {
		return ServiceCarrierInvocation{}, fmt.Errorf("service carrier Runtime provenance differs from Host binding")
	}
	return ServiceCarrierInvocation{
		Selection: selection,
		Runtime:   invocation,
		AuthorityLimits: []string{
			"service-binding-is-not-Contract-publication-authority",
			"service-binding-is-not-Provider-conformance-evidence",
			"request-and-result-shape-are-not-validated-by-carrier",
			"legacy-carrier-is-not-universal-service-ABI",
		},
	}, nil
}
