package main

import (
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

	normalized := plugin.NormalizeDefinition(definition)
	var matches []plugin.Capability
	for _, capability := range normalized.Provides {
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

	return Provider{
		PluginID:         string(definition.ID),
		PluginVersion:    definition.Version,
		ImplementationID: fmt.Sprintf(
			"legacy-capability:%s:%s",
			mapping.Kind,
			mapping.CapabilityID,
		),
		ArtifactDigest:      definition.Artifact.Digest,
		Scope:               mapping.Scope,
		ContractID:          mapping.Contract.ID,
		ContractVersion:     mapping.Contract.Version,
		ContractFingerprint: fingerprint(mapping.Contract),
		ConformanceProfile:  mapping.Contract.ConformanceProfile,
	}, nil
}
