package main

import (
	"encoding/json"
	"fmt"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/resource"
)

const (
	legacyRuntimeProjectionSchema = "tei.reference.legacy-runtime-capability-projection/v0"
	legacyEnvironmentProjectionSchema = "tei.reference.legacy-runtime-environment-projection/v0"
)

type LegacyRuntimeProjection struct {
	Schema string `json:"schema"`
	LegacyCapabilityKind plugin.CapabilityKind `json:"legacyCapabilityKind"`
	LegacyCapabilityID core.CanonicalID `json:"legacyCapabilityId"`
	Plugin plugin.Identity `json:"plugin"`
	SourceDeclarationFingerprint string `json:"sourceDeclarationFingerprint"`
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	ProjectionFingerprint string `json:"projectionFingerprint"`
}

func projectLegacyRuntimeCapabilityRef(
	definition plugin.Definition,
	kind plugin.CapabilityKind,
	id core.CanonicalID,
	candidate AcceptedRuntimeCandidate,
	profile ExecutionProfile,
) (LegacyRuntimeProjection, RuntimeCapabilityRef, error) {
	if !kind.Valid() || id == "" {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, fmt.Errorf(
			"legacy Runtime projection requires exact CapabilityKind and ID",
		)
	}
	if definition.ID != candidate.Plugin.ID ||
		definition.Version != candidate.Plugin.Version {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, fmt.Errorf(
			"legacy Plugin definition differs from accepted Provider identity",
		)
	}
	if definition.Artifact.Digest == "" ||
		definition.Artifact.Digest != candidate.ArtifactDigest {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, fmt.Errorf(
			"legacy Plugin definition does not identify the accepted implementation artifact",
		)
	}
	capability, err := exactLegacyCapability(definition, kind, id)
	if err != nil {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, err
	}
	legacyPermissions, err := normalizePermissionSet(capability.Permissions)
	if err != nil {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, err
	}
	if !samePermissionSet(legacyPermissions, profile.Permissions) {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, fmt.Errorf(
			"legacy Capability Permission declaration differs from generic Execution Profile",
		)
	}
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, err
	}
	sourceFingerprint, err := legacyCapabilityDeclarationFingerprint(capability)
	if err != nil {
		return LegacyRuntimeProjection{}, RuntimeCapabilityRef{}, err
	}
	projection := LegacyRuntimeProjection{
		Schema: legacyRuntimeProjectionSchema,
		LegacyCapabilityKind: kind,
		LegacyCapabilityID: id,
		Plugin: plugin.Identity{ID: definition.ID, Version: definition.Version},
		SourceDeclarationFingerprint: sourceFingerprint,
		RuntimeCapabilityFingerprint: ref.Fingerprint,
	}
	projection.ProjectionFingerprint = legacyRuntimeProjectionFingerprint(projection)
	return projection, ref, nil
}

func validateLegacyRuntimeProjection(
	projection LegacyRuntimeProjection,
) error {
	if projection.Schema != legacyRuntimeProjectionSchema ||
		!projection.LegacyCapabilityKind.Valid() ||
		projection.LegacyCapabilityID == "" ||
		projection.Plugin.ID == "" ||
		!validSHA256Fingerprint(projection.SourceDeclarationFingerprint) ||
		!validSHA256Fingerprint(projection.RuntimeCapabilityFingerprint) ||
		!validSHA256Fingerprint(projection.ProjectionFingerprint) ||
		legacyRuntimeProjectionFingerprint(projection) != projection.ProjectionFingerprint {
		return fmt.Errorf("legacy Runtime projection is invalid or stale")
	}
	if _, err := plugin.ParseVersion(projection.Plugin.Version); err != nil {
		return fmt.Errorf("legacy Runtime projection Plugin version: %w", err)
	}
	return nil
}

func verifyLegacyRuntimeProjection(
	projection LegacyRuntimeProjection,
	ref RuntimeCapabilityRef,
	definition plugin.Definition,
	candidate AcceptedRuntimeCandidate,
	profile ExecutionProfile,
) error {
	if err := validateLegacyRuntimeProjection(projection); err != nil {
		return err
	}
	currentProjection, currentRef, err := projectLegacyRuntimeCapabilityRef(
		definition,
		projection.LegacyCapabilityKind,
		projection.LegacyCapabilityID,
		candidate,
		profile,
	)
	if err != nil {
		return err
	}
	if projection != currentProjection || ref != currentRef {
		return fmt.Errorf("legacy Runtime projection differs from current authoritative declaration")
	}
	return nil
}

func legacyRuntimeProjectionFingerprint(projection LegacyRuntimeProjection) string {
	projection.ProjectionFingerprint = ""
	return fingerprint(projection)
}

func exactLegacyCapability(
	definition plugin.Definition,
	kind plugin.CapabilityKind,
	id core.CanonicalID,
) (plugin.Capability, error) {
	var matches []plugin.Capability
	for _, capability := range definition.Provides {
		if capability.Kind == kind && capability.ID == id {
			matches = append(matches, capability)
		}
	}
	if len(matches) != 1 {
		return plugin.Capability{}, fmt.Errorf(
			"legacy Plugin %s@%s requires exactly one Capability %s/%s; found %d",
			definition.ID,
			definition.Version,
			kind,
			id,
			len(matches),
		)
	}
	return matches[0], nil
}

func legacyCapabilityDeclarationFingerprint(
	capability plugin.Capability,
) (string, error) {
	raw, err := json.Marshal(capability)
	if err != nil {
		return "", err
	}
	var clone plugin.Capability
	if err := json.Unmarshal(raw, &clone); err != nil {
		return "", err
	}
	normalized := plugin.NormalizeDefinition(plugin.Definition{
		Provides: []plugin.Capability{clone},
	})
	if len(normalized.Provides) != 1 {
		return "", fmt.Errorf("legacy Capability normalization lost declaration")
	}
	return fingerprint(normalized.Provides[0]), nil
}

type LegacyEnvironmentAuthorityProjection struct {
	Schema string `json:"schema"`
	Environment core.CanonicalID `json:"environment"`
	LegacyCapabilityKind plugin.CapabilityKind `json:"legacyCapabilityKind"`
	LegacyCapabilityID core.CanonicalID `json:"legacyCapabilityId"`
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	LegacyAuthorityFingerprint string `json:"legacyAuthorityFingerprint"`
	ExecutionAuthorityEvidenceFingerprint string `json:"executionAuthorityEvidenceFingerprint"`
	Authority RuntimeEnvironmentAuthority `json:"authority"`
	ProjectionFingerprint string `json:"projectionFingerprint"`
}

func projectLegacyEnvironmentAuthority(
	env environment.Environment,
	runtimeProjection LegacyRuntimeProjection,
	ref RuntimeCapabilityRef,
	profile ExecutionProfile,
	executionAuthorityEvidenceFingerprint string,
) (LegacyEnvironmentAuthorityProjection, error) {
	if err := validateLegacyRuntimeProjection(runtimeProjection); err != nil {
		return LegacyEnvironmentAuthorityProjection{}, err
	}
	if env.ID == "" {
		return LegacyEnvironmentAuthorityProjection{}, fmt.Errorf("legacy Environment ID is empty")
	}
	if runtimeProjection.RuntimeCapabilityFingerprint != ref.Fingerprint {
		return LegacyEnvironmentAuthorityProjection{}, fmt.Errorf(
			"legacy Runtime projection targets a different Runtime Capability Ref",
		)
	}
	if !validSHA256Fingerprint(executionAuthorityEvidenceFingerprint) {
		return LegacyEnvironmentAuthorityProjection{}, fmt.Errorf(
			"legacy Environment migration requires accepted Execution Authority Evidence",
		)
	}

	var grants []permission.Spec
	for _, grant := range env.PermissionGrants {
		if grant.CapabilityKind == string(runtimeProjection.LegacyCapabilityKind) &&
			grant.CapabilityID == runtimeProjection.LegacyCapabilityID {
			grants = append(grants, grant.Permission)
		}
	}
	normalizedGrants, err := normalizePermissionSet(grants)
	if err != nil {
		return LegacyEnvironmentAuthorityProjection{}, err
	}
	if !samePermissionSet(normalizedGrants, profile.Permissions) {
		return LegacyEnvironmentAuthorityProjection{}, fmt.Errorf(
			"legacy Environment Permission Authority differs from generic Execution Profile",
		)
	}

	var bindings []resource.Binding
	if len(normalizedGrants) > 0 {
		bindings, err = resource.MatchBindings(normalizedGrants, env.ResourceBindings)
		if err != nil {
			return LegacyEnvironmentAuthorityProjection{}, err
		}
	}

	var policies []resource.LeasePolicy
	for _, lease := range env.LeasePolicies {
		if lease.CapabilityKind == string(runtimeProjection.LegacyCapabilityKind) &&
			lease.CapabilityID == runtimeProjection.LegacyCapabilityID {
			policies = append(policies, lease.Policy)
		}
	}
	if len(policies) > 1 {
		return LegacyEnvironmentAuthorityProjection{}, fmt.Errorf(
			"legacy Environment has multiple Lease Policies for Capability %s/%s",
			runtimeProjection.LegacyCapabilityKind,
			runtimeProjection.LegacyCapabilityID,
		)
	}
	if len(policies) == 1 {
		policies[0] = policies[0].WithDefaults()
		if err := policies[0].Validate(); err != nil {
			return LegacyEnvironmentAuthorityProjection{}, err
		}
	}

	authority := RuntimeEnvironmentAuthority{
		Environment: env.ID,
		ExecutionGrants: []RuntimeExecutionGrant{{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			ExecutionProfileFingerprint: profile.Fingerprint,
			AuthorityEvidenceFingerprint: executionAuthorityEvidenceFingerprint,
		}},
	}
	for _, grant := range normalizedGrants {
		authority.PermissionGrants = append(authority.PermissionGrants, RuntimePermissionGrant{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			Permission: grant,
		})
	}
	for _, binding := range bindings {
		authority.ResourceBindings = append(authority.ResourceBindings, RuntimeResourceBinding{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			Binding: binding,
		})
	}
	if len(policies) == 1 {
		authority.LeasePolicies = []RuntimeLeasePolicy{{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			Policy: policies[0],
		}}
	}

	if _, err := buildRuntimeEnvironmentPlan(ref, profile, authority); err != nil {
		return LegacyEnvironmentAuthorityProjection{}, err
	}
	legacyAuthorityFingerprint := fingerprint(struct {
		Environment core.CanonicalID `json:"environment"`
		CapabilityKind plugin.CapabilityKind `json:"capabilityKind"`
		CapabilityID core.CanonicalID `json:"capabilityId"`
		Permissions []permission.Spec `json:"permissions,omitempty"`
		Bindings []resource.Binding `json:"bindings,omitempty"`
		LeasePolicies []resource.LeasePolicy `json:"leasePolicies,omitempty"`
	}{
		Environment: env.ID,
		CapabilityKind: runtimeProjection.LegacyCapabilityKind,
		CapabilityID: runtimeProjection.LegacyCapabilityID,
		Permissions: normalizedGrants,
		Bindings: resource.NormalizeBindings(bindings),
		LeasePolicies: policies,
	})
	projection := LegacyEnvironmentAuthorityProjection{
		Schema: legacyEnvironmentProjectionSchema,
		Environment: env.ID,
		LegacyCapabilityKind: runtimeProjection.LegacyCapabilityKind,
		LegacyCapabilityID: runtimeProjection.LegacyCapabilityID,
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		LegacyAuthorityFingerprint: legacyAuthorityFingerprint,
		ExecutionAuthorityEvidenceFingerprint: executionAuthorityEvidenceFingerprint,
		Authority: authority,
	}
	projection.ProjectionFingerprint = legacyEnvironmentProjectionFingerprint(projection)
	return projection, nil
}

func verifyLegacyEnvironmentAuthorityProjection(
	projection LegacyEnvironmentAuthorityProjection,
	env environment.Environment,
	runtimeProjection LegacyRuntimeProjection,
	ref RuntimeCapabilityRef,
	profile ExecutionProfile,
) error {
	if projection.Schema != legacyEnvironmentProjectionSchema ||
		!validSHA256Fingerprint(projection.ProjectionFingerprint) ||
		legacyEnvironmentProjectionFingerprint(projection) != projection.ProjectionFingerprint {
		return fmt.Errorf("legacy Environment Authority projection is invalid or stale")
	}
	current, err := projectLegacyEnvironmentAuthority(
		env,
		runtimeProjection,
		ref,
		profile,
		projection.ExecutionAuthorityEvidenceFingerprint,
	)
	if err != nil {
		return err
	}
	if projection.ProjectionFingerprint != current.ProjectionFingerprint {
		return fmt.Errorf(
			"legacy Environment Authority projection differs from current authoritative Environment",
		)
	}
	return nil
}

func verifyLegacyRuntimeMigration(
	runtimeProjection LegacyRuntimeProjection,
	environmentProjection LegacyEnvironmentAuthorityProjection,
	ref RuntimeCapabilityRef,
	definition plugin.Definition,
	candidate AcceptedRuntimeCandidate,
	profile ExecutionProfile,
	env environment.Environment,
) error {
	if err := verifyLegacyRuntimeProjection(
		runtimeProjection,
		ref,
		definition,
		candidate,
		profile,
	); err != nil {
		return err
	}
	if err := verifyLegacyEnvironmentAuthorityProjection(
		environmentProjection,
		env,
		runtimeProjection,
		ref,
		profile,
	); err != nil {
		return err
	}
	return nil
}

func legacyEnvironmentProjectionFingerprint(
	projection LegacyEnvironmentAuthorityProjection,
) string {
	projection.ProjectionFingerprint = ""
	return fingerprint(projection)
}
