package main

import (
	"fmt"
	"sort"
	"strings"
)

const capabilityContractAuthorityBindingSchema =
	"tei.reference.capability-contract-authority-binding/v0"

type CapabilityContractAuthorityGrant struct {
	Namespace                    string `json:"namespace"`
	Publisher                    string `json:"publisher"`
	AuthorityEvidenceFingerprint string `json:"authorityEvidenceFingerprint"`
}

type CapabilityContractPublication struct {
	ContractID                   string `json:"contractId"`
	ContractVersion              string `json:"contractVersion"`
	ContractFingerprint          string `json:"contractFingerprint"`
	Publisher                    string `json:"publisher"`
	AuthorityEvidenceFingerprint string `json:"authorityEvidenceFingerprint"`
}

type CapabilityContractAuthorityBinding struct {
	Schema                       string `json:"schema"`
	ContractID                   string `json:"contractId"`
	ContractVersion              string `json:"contractVersion"`
	ContractFingerprint          string `json:"contractFingerprint"`
	Namespace                    string `json:"namespace"`
	Publisher                    string `json:"publisher"`
	AuthorityEvidenceFingerprint string `json:"authorityEvidenceFingerprint"`
	BindingFingerprint           string `json:"bindingFingerprint"`
}

func resolveCapabilityWithAcceptedContractAuthority(
	input ResolveInput,
	grants []CapabilityContractAuthorityGrant,
	publications []CapabilityContractPublication,
) (Resolution, []CapabilityContractAuthorityBinding, error) {
	contracts, bindings, err := acceptedCapabilityContracts(
		input.Requirement,
		input.Contracts,
		grants,
		publications,
	)
	if err != nil {
		return Resolution{}, nil, err
	}
	authorizedInput := input
	authorizedInput.Contracts = contracts
	resolution, err := resolveCapability(authorizedInput)
	if err != nil {
		return Resolution{}, nil, err
	}
	return resolution, bindings, nil
}

func acceptedCapabilityContracts(
	requirement Requirement,
	contracts []Contract,
	grants []CapabilityContractAuthorityGrant,
	publications []CapabilityContractPublication,
) ([]Contract, []CapabilityContractAuthorityBinding, error) {
	normalizedGrants, err := normalizeCapabilityContractAuthorityGrants(grants)
	if err != nil {
		return nil, nil, err
	}
	normalizedPublications, err := normalizeCapabilityContractPublications(publications)
	if err != nil {
		return nil, nil, err
	}

	seenContracts := map[string]struct{}{}
	var accepted []Contract
	var bindings []CapabilityContractAuthorityBinding
	for _, contract := range contracts {
		version, err := parseVersion(contract.Version)
		if err != nil {
			return nil, nil, fmt.Errorf("contract %s: %w", contract.ID, err)
		}
		key := contract.ID + "@" + contract.Version
		if _, duplicate := seenContracts[key]; duplicate {
			return nil, nil, fmt.Errorf("contract %s is declared more than once", key)
		}
		seenContracts[key] = struct{}{}

		if contract.ID != requirement.ContractID ||
			version[0] != requirement.CompatibleMajor {
			continue
		}

		grant, ok, err := capabilityContractOwner(contract.ID, normalizedGrants)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, fmt.Errorf(
				"contract %s has no accepted definition Authority",
				key,
			)
		}

		publication, ok := normalizedPublications[key]
		if !ok {
			return nil, nil, fmt.Errorf(
				"contract %s has no accepted publication",
				key,
			)
		}
		contractFingerprint := fingerprint(contract)
		if publication.ContractFingerprint != contractFingerprint {
			return nil, nil, fmt.Errorf(
				"contract %s publication fingerprint is stale",
				key,
			)
		}
		if publication.Publisher != grant.Publisher {
			return nil, nil, fmt.Errorf(
				"contract %s publisher %q does not own namespace %s",
				key,
				publication.Publisher,
				grant.Namespace,
			)
		}
		if publication.AuthorityEvidenceFingerprint !=
			grant.AuthorityEvidenceFingerprint {
			return nil, nil, fmt.Errorf(
				"contract %s publication references stale Authority Evidence",
				key,
			)
		}

		binding := CapabilityContractAuthorityBinding{
			Schema:                       capabilityContractAuthorityBindingSchema,
			ContractID:                   contract.ID,
			ContractVersion:              contract.Version,
			ContractFingerprint:          contractFingerprint,
			Namespace:                    grant.Namespace,
			Publisher:                    grant.Publisher,
			AuthorityEvidenceFingerprint: grant.AuthorityEvidenceFingerprint,
		}
		binding.BindingFingerprint =
			capabilityContractAuthorityBindingFingerprint(binding)
		accepted = append(accepted, contract)
		bindings = append(bindings, binding)
	}
	if len(accepted) == 0 {
		return nil, nil, fmt.Errorf(
			"no authorized capability contract %s compatible with major %d",
			requirement.ContractID,
			requirement.CompatibleMajor,
		)
	}

	sort.Slice(accepted, func(i, j int) bool {
		return compareVersion(accepted[i].Version, accepted[j].Version) > 0
	})
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].ContractID != bindings[j].ContractID {
			return bindings[i].ContractID < bindings[j].ContractID
		}
		return compareVersion(
			bindings[i].ContractVersion,
			bindings[j].ContractVersion,
		) > 0
	})
	return accepted, bindings, nil
}

func normalizeCapabilityContractAuthorityGrants(
	values []CapabilityContractAuthorityGrant,
) (map[string]CapabilityContractAuthorityGrant, error) {
	out := map[string]CapabilityContractAuthorityGrant{}
	for i, value := range values {
		value.Namespace = strings.Trim(strings.TrimSpace(value.Namespace), ".")
		value.Publisher = strings.TrimSpace(value.Publisher)
		value.AuthorityEvidenceFingerprint =
			strings.TrimSpace(value.AuthorityEvidenceFingerprint)
		if value.Namespace == "" ||
			value.Publisher == "" ||
			!validSHA256Fingerprint(value.AuthorityEvidenceFingerprint) {
			return nil, fmt.Errorf(
				"capability contract authority grant[%d] is incomplete or invalid",
				i,
			)
		}
		if existing, ok := out[value.Namespace]; ok && existing != value {
			return nil, fmt.Errorf(
				"capability contract namespace %s has conflicting Authorities",
				value.Namespace,
			)
		}
		out[value.Namespace] = value
	}
	return out, nil
}

func normalizeCapabilityContractPublications(
	values []CapabilityContractPublication,
) (map[string]CapabilityContractPublication, error) {
	out := map[string]CapabilityContractPublication{}
	for i, value := range values {
		value.ContractID = strings.TrimSpace(value.ContractID)
		value.ContractVersion = strings.TrimSpace(value.ContractVersion)
		value.ContractFingerprint = strings.TrimSpace(value.ContractFingerprint)
		value.Publisher = strings.TrimSpace(value.Publisher)
		value.AuthorityEvidenceFingerprint =
			strings.TrimSpace(value.AuthorityEvidenceFingerprint)
		if value.ContractID == "" ||
			value.ContractVersion == "" ||
			!validSHA256Fingerprint(value.ContractFingerprint) ||
			value.Publisher == "" ||
			!validSHA256Fingerprint(value.AuthorityEvidenceFingerprint) {
			return nil, fmt.Errorf(
				"capability contract publication[%d] is incomplete or invalid",
				i,
			)
		}
		if _, err := parseVersion(value.ContractVersion); err != nil {
			return nil, fmt.Errorf(
				"capability contract publication[%d] version: %w",
				i,
				err,
			)
		}
		key := value.ContractID + "@" + value.ContractVersion
		if existing, ok := out[key]; ok && existing != value {
			return nil, fmt.Errorf(
				"capability contract publication %s is conflicting",
				key,
			)
		}
		out[key] = value
	}
	return out, nil
}

func capabilityContractOwner(
	contractID string,
	grants map[string]CapabilityContractAuthorityGrant,
) (CapabilityContractAuthorityGrant, bool, error) {
	var matches []CapabilityContractAuthorityGrant
	longest := -1
	for namespace, grant := range grants {
		if contractID != namespace &&
			!strings.HasPrefix(contractID, namespace+".") {
			continue
		}
		if len(namespace) > longest {
			longest = len(namespace)
			matches = []CapabilityContractAuthorityGrant{grant}
			continue
		}
		if len(namespace) == longest {
			matches = append(matches, grant)
		}
	}
	if len(matches) == 0 {
		return CapabilityContractAuthorityGrant{}, false, nil
	}
	if len(matches) > 1 {
		return CapabilityContractAuthorityGrant{}, false, fmt.Errorf(
			"contract %s has ambiguous definition Authority",
			contractID,
		)
	}
	return matches[0], true, nil
}

func capabilityContractAuthorityBindingFingerprint(
	binding CapabilityContractAuthorityBinding,
) string {
	binding.BindingFingerprint = ""
	return fingerprint(binding)
}
