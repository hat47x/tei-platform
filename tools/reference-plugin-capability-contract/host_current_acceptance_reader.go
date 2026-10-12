package main

import (
	"context"
	"fmt"
	"strings"

	pruntime "tei/internal/plugin/runtime"
	authoritytrust "tei/tools/reference-authority-evidence-trust/trust"
)

const (
	serviceContractPublicationAuthorityClass = "service-contract-publisher"
	serviceProviderConformanceAuthorityClass = "service-provider-conformance-issuer"
)

// HostCurrentAcceptanceSnapshot is supplied by a Host-owned source, NEVER by
// a plugin or a request. The source must authenticate current issuer grants,
// publication/acceptance records, conformance and revocation before returning
// these values. This Reference only verifies their exact bindings; neither a
// fingerprint nor an in-memory Grant authenticates the Host source.
type HostCurrentAcceptanceSnapshot struct {
	Contract                 Contract
	Provider                 Provider
	ProviderEvidence         ConformanceEvidence
	ContractPublication      authoritytrust.Statement
	ProviderConformance      authoritytrust.Statement
	CurrentTrustGrants       []authoritytrust.Grant
	ContractAccepted         bool
	ProviderAccepted         bool
	ContractAcceptanceRef    string
	ProviderConformanceRef   string
	AuthorityRevision        string
}

// HostCurrentAcceptanceSource reads the Host's CURRENT accepted publication,
// Provider conformance and issuer/namespace grants for an exact selection.
// The Host, not this adapter, owns the acceptance records and trust roots.
type HostCurrentAcceptanceSource interface {
	ReadCurrentHostAcceptance(context.Context, pruntime.ServiceProviderSelection) (HostCurrentAcceptanceSnapshot, error)
}

// ReferenceHostCurrentAcceptanceReader is a proof of the existing Authority
// trust verifier serving the production Runtime's read-only acceptance port.
// It is NOT a production Trust Root, Conformance issuer or second Resolver.
type ReferenceHostCurrentAcceptanceReader struct {
	Source HostCurrentAcceptanceSource
}

var _ pruntime.ServiceAcceptanceReader = (*ReferenceHostCurrentAcceptanceReader)(nil)

func (r *ReferenceHostCurrentAcceptanceReader) ReadCurrentAcceptedService(
	ctx context.Context, selected pruntime.ServiceProviderSelection,
) (pruntime.ServiceAcceptanceRecord, error) {
	if r == nil || r.Source == nil {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host current acceptance source is missing")
	}
	if err := ctx.Err(); err != nil {
		return pruntime.ServiceAcceptanceRecord{}, err
	}
	snapshot, err := r.Source.ReadCurrentHostAcceptance(ctx, selected)
	if err != nil {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host current acceptance source denied: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return pruntime.ServiceAcceptanceRecord{}, err
	}
	if !snapshot.ContractAccepted || !snapshot.ProviderAccepted {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host current Contract or Provider acceptance is inactive")
	}
	contract := snapshot.Contract
	if contract.ID != selected.ContractID ||
		contract.Version != selected.ContractVersion ||
		fingerprint(contract) != selected.ContractFingerprint {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host accepted Contract differs from selected exact Contract")
	}
	provider := snapshot.Provider
	if provider.PluginID != string(selected.Provider.ID) ||
		provider.PluginVersion != selected.Provider.Version ||
		provider.ImplementationID != selected.ImplementationID ||
		provider.ArtifactDigest != selected.ArtifactDigest ||
		provider.Scope != AuthorityScope(selected.Scope) ||
		provider.ContractID != selected.ContractID ||
		provider.ContractVersion != selected.ContractVersion ||
		provider.ContractFingerprint != selected.ContractFingerprint ||
		provider.ConformanceProfile != contract.ConformanceProfile {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host accepted Provider differs from selected exact implementation")
	}
	evidence := snapshot.ProviderEvidence
	if evidence.PluginID != provider.PluginID ||
		evidence.PluginVersion != provider.PluginVersion ||
		evidence.ImplementationID != provider.ImplementationID ||
		evidence.ArtifactDigest != provider.ArtifactDigest ||
		evidence.ContractFingerprint != selected.ContractFingerprint ||
		evidence.ConformanceProfile != contract.ConformanceProfile ||
		strings.TrimSpace(evidence.EvidenceFingerprint) == "" {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host accepted Provider conformance evidence differs from selected implementation")
	}
	publication := snapshot.ContractPublication
	if publication.AuthorityClass != serviceContractPublicationAuthorityClass ||
		publication.Scope != "contract:"+selected.ContractID ||
		publication.StatementFingerprint != selected.ContractFingerprint {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host Contract publisher statement has wrong authority or exact Contract")
	}
	conformance := snapshot.ProviderConformance
	if conformance.AuthorityClass != serviceProviderConformanceAuthorityClass ||
		conformance.Scope != "contract:"+selected.ContractID+"/provider:"+provider.PluginID+"@"+provider.PluginVersion ||
		conformance.StatementFingerprint != fingerprint(evidence) {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host Provider conformance statement has wrong authority or evidence")
	}
	// This reuses the existing Reference Authority trust primitive, including
	// exact issuer/scope/class matching, revoked and ambiguous grants. The
	// caller-supplied selected Provider cannot provide its own grant.
	contractTrust, err := authoritytrust.Verify(publication, snapshot.CurrentTrustGrants)
	if err != nil {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("current Contract publisher trust denied: %w", err)
	}
	providerTrust, err := authoritytrust.Verify(conformance, snapshot.CurrentTrustGrants)
	if err != nil {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("current Provider conformance issuer trust denied: %w", err)
	}
	if strings.TrimSpace(snapshot.ContractAcceptanceRef) == "" ||
		strings.TrimSpace(snapshot.ProviderConformanceRef) == "" ||
		strings.TrimSpace(snapshot.AuthorityRevision) == "" ||
		contractTrust.QualificationFingerprint == "" ||
		providerTrust.QualificationFingerprint == "" {
		return pruntime.ServiceAcceptanceRecord{}, fmt.Errorf("Host current acceptance references or authority revision are missing")
	}
	return pruntime.ServiceAcceptanceRecord{
		Selection:              selected,
		ContractAcceptanceRef:  snapshot.ContractAcceptanceRef,
		ProviderConformanceRef: snapshot.ProviderConformanceRef,
		AuthorityRevision:      snapshot.AuthorityRevision,
	}, nil
}
