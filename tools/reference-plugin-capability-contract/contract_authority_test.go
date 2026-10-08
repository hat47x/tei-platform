package main

import (
	"strings"
	"testing"
)

func contractAuthorityGrant(
	namespace string,
	publisher string,
	marker string,
) CapabilityContractAuthorityGrant {
	return CapabilityContractAuthorityGrant{
		Namespace:                    namespace,
		Publisher:                    publisher,
		AuthorityEvidenceFingerprint: fingerprint(marker),
	}
}

func contractPublication(
	contract Contract,
	grant CapabilityContractAuthorityGrant,
) CapabilityContractPublication {
	return CapabilityContractPublication{
		ContractID:                   contract.ID,
		ContractVersion:              contract.Version,
		ContractFingerprint:          fingerprint(contract),
		Publisher:                    grant.Publisher,
		AuthorityEvidenceFingerprint: grant.AuthorityEvidenceFingerprint,
	}
}

func contractAuthorityInput(
	contracts []Contract,
	providers []Provider,
	evidence []ConformanceEvidence,
) ResolveInput {
	return ResolveInput{
		Scope: ScopeAuthoringTool,
		Requirement: Requirement{
			ContractID:      contracts[0].ID,
			CompatibleMajor: 1,
		},
		Contracts: contracts,
		Providers: providers,
		Evidence:  evidence,
	}
}

func TestCapabilityContractResolutionRequiresDefinitionAuthority(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := contractAuthorityInput(
		[]Contract{contract},
		[]Provider{provider},
		[]ConformanceEvidence{evidenceFor(provider, contract, "evidence")},
	)
	grant := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.reference",
		"contract-authority",
	)

	resolution, bindings, err :=
		resolveCapabilityWithAcceptedContractAuthority(
			input,
			[]CapabilityContractAuthorityGrant{grant},
			[]CapabilityContractPublication{contractPublication(contract, grant)},
		)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Contract != contract {
		t.Fatalf("resolution contract=%+v", resolution.Contract)
	}
	if len(bindings) != 1 ||
		bindings[0].Publisher != grant.Publisher ||
		bindings[0].Namespace != grant.Namespace ||
		!validSHA256Fingerprint(bindings[0].BindingFingerprint) {
		t.Fatalf("contract authority binding=%+v", bindings)
	}
}

func TestUnauthorizedHigherCompatibleContractCannotPreemptAcceptedContract(t *testing.T) {
	v1 := sourceAdapterContract()
	v11 := sourceAdapterContract()
	v11.Version = "1.1.0"

	providerV1 := sourceAdapterProvider(
		v1,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v1",
	)
	providerV11 := sourceAdapterProvider(
		v11,
		ScopeAuthoringTool,
		"third.party.xead.source",
		"xead.reader.v11",
	)
	input := contractAuthorityInput(
		[]Contract{v11, v1},
		[]Provider{providerV11, providerV1},
		[]ConformanceEvidence{
			evidenceFor(providerV11, v11, "evidence-v11"),
			evidenceFor(providerV1, v1, "evidence-v1"),
		},
	)

	accepted := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.reference",
		"accepted-authority",
	)
	attacker := contractAuthorityGrant(
		"third.party",
		"third.party.publisher",
		"attacker-authority",
	)
	_, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{accepted, attacker},
		[]CapabilityContractPublication{
			contractPublication(v1, accepted),
			contractPublication(v11, attacker),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not own namespace") {
		t.Fatalf("unauthorized higher Contract was not rejected: %v", err)
	}

	resolution, _, err := resolveCapabilityWithAcceptedContractAuthority(
		contractAuthorityInput(
			[]Contract{v1},
			[]Provider{providerV1},
			[]ConformanceEvidence{evidenceFor(providerV1, v1, "evidence-v1")},
		),
		[]CapabilityContractAuthorityGrant{accepted},
		[]CapabilityContractPublication{contractPublication(v1, accepted)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Contract.Version != "1.0.0" {
		t.Fatalf("accepted fallback contract=%s", resolution.Contract.Version)
	}
}

func TestAuthorizedHigherCompatibleContractCanAdvanceFreshResolution(t *testing.T) {
	v1 := sourceAdapterContract()
	v11 := sourceAdapterContract()
	v11.Version = "1.1.0"

	providerV1 := sourceAdapterProvider(
		v1,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v1",
	)
	providerV11 := sourceAdapterProvider(
		v11,
		ScopeAuthoringTool,
		"reference.compat.xead.source.next",
		"xead.reader.v11",
	)
	grant := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.reference",
		"accepted-authority",
	)
	input := contractAuthorityInput(
		[]Contract{v1, v11},
		[]Provider{providerV1, providerV11},
		[]ConformanceEvidence{
			evidenceFor(providerV1, v1, "evidence-v1"),
			evidenceFor(providerV11, v11, "evidence-v11"),
		},
	)
	resolution, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{grant},
		[]CapabilityContractPublication{
			contractPublication(v1, grant),
			contractPublication(v11, grant),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Contract.Version != "1.1.0" {
		t.Fatalf("authorized higher Contract was not selected: %s", resolution.Contract.Version)
	}
}

func TestContractAuthorityRotationDoesNotChangeContractMeaningIdentity(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := contractAuthorityInput(
		[]Contract{contract},
		[]Provider{provider},
		[]ConformanceEvidence{evidenceFor(provider, contract, "evidence")},
	)

	leftGrant := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.reference",
		"authority-left",
	)
	leftResolution, leftBindings, err :=
		resolveCapabilityWithAcceptedContractAuthority(
			input,
			[]CapabilityContractAuthorityGrant{leftGrant},
			[]CapabilityContractPublication{
				contractPublication(contract, leftGrant),
			},
		)
	if err != nil {
		t.Fatal(err)
	}

	rightGrant := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.reference",
		"authority-right",
	)
	if _, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{rightGrant},
		[]CapabilityContractPublication{
			contractPublication(contract, leftGrant),
		},
	); err == nil || !strings.Contains(err.Error(), "stale Authority Evidence") {
		t.Fatalf("stale Contract publication survived Authority rotation: %v", err)
	}

	rightResolution, rightBindings, err :=
		resolveCapabilityWithAcceptedContractAuthority(
			input,
			[]CapabilityContractAuthorityGrant{rightGrant},
			[]CapabilityContractPublication{
				contractPublication(contract, rightGrant),
			},
		)
	if err != nil {
		t.Fatal(err)
	}
	if leftResolution.ResolutionFingerprint != rightResolution.ResolutionFingerprint {
		t.Fatal("Contract Authority rotation changed Contract/Provider resolution identity")
	}
	if leftBindings[0].BindingFingerprint == rightBindings[0].BindingFingerprint {
		t.Fatal("Contract Authority rotation was invisible in Authority binding")
	}
}

func TestMostSpecificContractNamespaceAuthorityOwnsChildContract(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := contractAuthorityInput(
		[]Contract{contract},
		[]Provider{provider},
		[]ConformanceEvidence{evidenceFor(provider, contract, "evidence")},
	)
	root := contractAuthorityGrant(
		"authoring",
		"tei.contract.root",
		"root-authority",
	)
	specific := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.source-adapter",
		"specific-authority",
	)

	if _, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{root, specific},
		[]CapabilityContractPublication{
			contractPublication(contract, root),
		},
	); err == nil || !strings.Contains(err.Error(), "does not own namespace") {
		t.Fatalf("broader namespace publisher overrode specific owner: %v", err)
	}

	if _, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{root, specific},
		[]CapabilityContractPublication{
			contractPublication(contract, specific),
		},
	); err != nil {
		t.Fatalf("specific namespace owner was rejected: %v", err)
	}
}

func TestStandardContractNamespaceCannotBeClaimedByThirdPartyPublication(t *testing.T) {
	contract := sourceAdapterContract()
	contract.ID = "std.contract-query"
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"third.party.query",
		"query.reader",
	)
	input := contractAuthorityInput(
		[]Contract{contract},
		[]Provider{provider},
		[]ConformanceEvidence{evidenceFor(provider, contract, "evidence")},
	)
	standard := contractAuthorityGrant(
		"std",
		"tei.standard.contracts",
		"std-authority",
	)
	thirdParty := contractAuthorityGrant(
		"third.party",
		"third.party.publisher",
		"third-party-authority",
	)
	_, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{standard, thirdParty},
		[]CapabilityContractPublication{
			contractPublication(contract, thirdParty),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not own namespace") {
		t.Fatalf("third-party publication claimed std namespace: %v", err)
	}
}

func TestConflictingContractNamespaceAuthoritiesFailClosed(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := contractAuthorityInput(
		[]Contract{contract},
		[]Provider{provider},
		[]ConformanceEvidence{evidenceFor(provider, contract, "evidence")},
	)
	left := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.left",
		"left-authority",
	)
	right := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.right",
		"right-authority",
	)
	_, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{left, right},
		[]CapabilityContractPublication{
			contractPublication(contract, left),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "conflicting Authorities") {
		t.Fatalf("conflicting Contract namespace Authorities were accepted: %v", err)
	}
}

func TestLockedContractAuthorityIsRevalidatedSeparatelyFromLockReplay(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := contractAuthorityInput(
		[]Contract{contract},
		[]Provider{provider},
		[]ConformanceEvidence{evidenceFor(provider, contract, "evidence")},
	)
	leftGrant := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.reference",
		"authority-left",
	)
	resolution, _, err := resolveCapabilityWithAcceptedContractAuthority(
		input,
		[]CapabilityContractAuthorityGrant{leftGrant},
		[]CapabilityContractPublication{
			contractPublication(contract, leftGrant),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	leftBinding, err := verifyCapabilityResolutionLockAcceptedContractAuthority(
		lock,
		[]CapabilityContractAuthorityGrant{leftGrant},
		[]CapabilityContractPublication{
			contractPublication(contract, leftGrant),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	rightGrant := contractAuthorityGrant(
		"authoring.source-adapter",
		"tei.contract.reference",
		"authority-right",
	)
	if err := verifyCapabilityResolutionLockReplay(lock, input); err != nil {
		t.Fatalf("Contract Authority rotation changed declaration replay: %v", err)
	}
	if _, err := verifyCapabilityResolutionLockAcceptedContractAuthority(
		lock,
		[]CapabilityContractAuthorityGrant{rightGrant},
		[]CapabilityContractPublication{
			contractPublication(contract, leftGrant),
		},
	); err == nil || !strings.Contains(err.Error(), "stale Authority Evidence") {
		t.Fatalf("stale locked Contract Authority was accepted: %v", err)
	}

	rightBinding, err := verifyCapabilityResolutionLockAcceptedContractAuthority(
		lock,
		[]CapabilityContractAuthorityGrant{rightGrant},
		[]CapabilityContractPublication{
			contractPublication(contract, rightGrant),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !validSHA256Fingerprint(lock.LockFingerprint) {
		t.Fatalf("lock fingerprint=%q", lock.LockFingerprint)
	}
	if leftBinding.BindingFingerprint == rightBinding.BindingFingerprint {
		t.Fatal("current Contract Authority refresh was invisible")
	}
	if leftBinding.ContractFingerprint != rightBinding.ContractFingerprint ||
		leftBinding.ContractFingerprint != lock.ContractFingerprint {
		t.Fatal("Contract Authority refresh changed locked Contract meaning")
	}
}
