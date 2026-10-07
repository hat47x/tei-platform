package main

import (
	"encoding/json"
	"fmt"

	"tei/internal/core"
	"tei/internal/plugin"
)

// LegacyCapabilityMapping is a migration-only bridge.
// The legacy Plugin definition remains the provider declaration authority.
// The mapping only states which generic Contract the existing capability is
// being evaluated against in this Reference.
type LegacyCapabilityMapping struct {
	Kind         plugin.CapabilityKind
	CapabilityID core.CanonicalID
	Scope        AuthorityScope
	Contract     Contract
}

func projectLegacyCapabilityProvider(
	definition plugin.Definition,
	mapping LegacyCapabilityMapping,
) (Provider, error) {
	if !mapping.Kind.Valid() {
		return Provider{}, fmt.Errorf("legacy capability kind %q is invalid", mapping.Kind)
	}
	if mapping.CapabilityID == "" {
		return Provider{}, fmt.Errorf("legacy capability ID is empty")
	}
	if !mapping.Scope.Valid() {
		return Provider{}, fmt.Errorf("legacy capability projection scope %q is invalid", mapping.Scope)
	}
	if definition.ID == "" || definition.Version == "" {
		return Provider{}, fmt.Errorf("legacy Plugin identity is incomplete")
	}
	if _, err := parseVersion(definition.Version); err != nil {
		return Provider{}, fmt.Errorf("legacy Plugin version: %w", err)
	}
	contractVersion, err := parseVersion(mapping.Contract.Version)
	if err != nil {
		return Provider{}, fmt.Errorf("generic capability Contract version: %w", err)
	}
	if _, err := compatibleContracts(
		[]Contract{mapping.Contract},
		Requirement{
			ContractID:      mapping.Contract.ID,
			CompatibleMajor: contractVersion[0],
		},
	); err != nil {
		return Provider{}, err
	}
	if definition.Artifact.Digest != "" &&
		!validSHA256Fingerprint(definition.Artifact.Digest) {
		return Provider{}, fmt.Errorf("legacy Plugin artifact digest is not exact sha256")
	}

	var matches []plugin.Capability
	for _, capability := range definition.Provides {
		if capability.Kind == mapping.Kind && capability.ID == mapping.CapabilityID {
			matches = append(matches, capability)
		}
	}
	if len(matches) != 1 {
		return Provider{}, fmt.Errorf(
			"legacy Plugin %s@%s must provide exactly one %s capability %s; found %d",
			definition.ID,
			definition.Version,
			mapping.Kind,
			mapping.CapabilityID,
			len(matches),
		)
	}

	sourceDeclarationFingerprint, err := normalizedLegacyCapabilityFingerprint(matches[0])
	if err != nil {
		return Provider{}, err
	}

	return Provider{
		PluginID:         string(definition.ID),
		PluginVersion:    definition.Version,
		ImplementationID: fmt.Sprintf(
			"legacy-capability:%s:%s",
			mapping.Kind,
			mapping.CapabilityID,
		),
		ArtifactDigest:               definition.Artifact.Digest,
		SourceDeclarationFingerprint: sourceDeclarationFingerprint,
		Scope:                        mapping.Scope,
		ContractID:          mapping.Contract.ID,
		ContractVersion:     mapping.Contract.Version,
		ContractFingerprint: fingerprint(mapping.Contract),
		ConformanceProfile:  mapping.Contract.ConformanceProfile,
	}, nil
}


func normalizedLegacyCapabilityFingerprint(
	value plugin.Capability,
) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("legacy capability declaration cannot be encoded: %w", err)
	}
	var clone plugin.Capability
	if err := json.Unmarshal(encoded, &clone); err != nil {
		return "", fmt.Errorf("legacy capability declaration cannot be cloned: %w", err)
	}
	normalized := plugin.NormalizeDefinition(plugin.Definition{
		Provides: []plugin.Capability{clone},
	})
	if len(normalized.Provides) != 1 {
		return "", fmt.Errorf("legacy capability normalization lost declaration")
	}
	return fingerprint(normalized.Provides[0]), nil
}
