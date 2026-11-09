package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/resource"
)

func legacyReferenceDefinition(
	permissions []permission.Spec,
) plugin.Definition {
	return plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.document_render"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{
			Kind:    plugin.RuntimeProcess,
			Command: "document-render-worker",
			Sandbox: plugin.SandboxRequired,
		},
		Artifact: plugin.Artifact{
			Digest: "sha256:" + strings.Repeat("a", 64),
		},
		Provides: []plugin.Capability{{
			Kind:        plugin.CapabilityDeploymentExecutor,
			ID:          core.MustCanonicalID("legacy.document_render"),
			Permissions: append([]permission.Spec(nil), permissions...),
		}},
	}
}

func legacyReferenceEnvironment(
	perm permission.Spec,
) environment.Environment {
	return environment.Environment{
		ID: core.MustCanonicalID("legacy_runtime"),
		PermissionGrants: []environment.PermissionGrant{{
			CapabilityKind: string(plugin.CapabilityDeploymentExecutor),
			CapabilityID:   core.MustCanonicalID("legacy.document_render"),
			Permission:     perm,
		}},
		ResourceBindings: []environment.ResourceBinding{
			referenceResourceBinding(),
			{
				Kind:       permission.KindNetwork,
				Scope:      "unrelated-network",
				ResourceID: core.MustCanonicalID("resource.unrelated_network"),
			},
		},
		LeasePolicies: []environment.CapabilityLeasePolicy{{
			CapabilityKind: string(plugin.CapabilityDeploymentExecutor),
			CapabilityID:   core.MustCanonicalID("legacy.document_render"),
			Policy: resource.LeasePolicy{
				MaxLifetime:    "20m",
				RenewBefore:    "3m",
				RevokeTimeout:  "2s",
				RenewalFailure: resource.RenewalFailClosed,
			},
		}},
	}
}

func TestLegacyCapabilityProjectsOneWayIntoRuntimeCapabilityRef(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	projection, ref, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		candidate,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	if projection.RuntimeCapabilityFingerprint != ref.Fingerprint ||
		projection.Plugin.ID != definition.ID ||
		projection.SourceDeclarationFingerprint == "" {
		t.Fatalf("legacy Runtime projection=%+v ref=%+v", projection, ref)
	}
	if err := verifyLegacyRuntimeProjection(
		projection, ref, definition, candidate, profile,
	); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyRuntimeProjectionDetectsAuthoritativeDeclarationDrift(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	projection, ref, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		candidate,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}

	drifted := definition
	drifted.Provides = append([]plugin.Capability(nil), definition.Provides...)
	drifted.Provides[0].Features = []environment.Feature{"legacy.render_feature"}
	if err := verifyLegacyRuntimeProjection(
		projection, ref, drifted, candidate, profile,
	); err == nil || !strings.Contains(err.Error(), "current authoritative declaration") {
		t.Fatalf("legacy declaration drift was accepted: %v", err)
	}
}

func TestLegacyRuntimeProjectionRejectsArtifactAndPermissionBroadening(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	artifactDrift := candidate
	artifactDrift.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	if _, _, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		artifactDrift,
		profile,
	); err == nil || !strings.Contains(err.Error(), "implementation artifact") {
		t.Fatalf("legacy artifact mismatch was accepted: %v", err)
	}

	broadProfile := referenceExecutionProfile(
		t,
		"web-browser",
		[]permission.Spec{
			perm,
			{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "ambient-network"},
		},
	)
	broadCandidate := referenceRuntimeCandidate(broadProfile)
	if _, _, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		broadCandidate,
		broadProfile,
	); err == nil || !strings.Contains(err.Error(), "Permission declaration") {
		t.Fatalf("generic profile broadened legacy Permission declaration: %v", err)
	}
}

func TestLegacyEnvironmentAuthorityProjectsIntoGenericAuthorityCandidate(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	runtimeProjection, ref, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		candidate,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	env := legacyReferenceEnvironment(perm)
	environmentProjection, err := projectLegacyEnvironmentAuthority(
		env,
		runtimeProjection,
		ref,
		profile,
		fingerprint("accepted-legacy-migration-execution-authority"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(environmentProjection.Authority.LegacyClaims) != 0 {
		t.Fatalf("legacy Authority was copied as a second Authority: %+v", environmentProjection.Authority.LegacyClaims)
	}
	if len(environmentProjection.Authority.PermissionGrants) != 1 ||
		len(environmentProjection.Authority.ResourceBindings) != 1 ||
		len(environmentProjection.Authority.LeasePolicies) != 1 {
		t.Fatalf("legacy Environment projection=%+v", environmentProjection.Authority)
	}
	plan, err := buildRuntimeEnvironmentPlan(
		ref, profile, environmentProjection.Authority,
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Environment != env.ID || plan.RuntimeCapability.Fingerprint != ref.Fingerprint {
		t.Fatalf("projected Runtime Environment Plan=%+v", plan)
	}
	if err := verifyLegacyEnvironmentAuthorityProjection(
		environmentProjection, env, runtimeProjection, ref, profile,
	); err != nil {
		t.Fatal(err)
	}
	if err := verifyLegacyRuntimeMigration(
		runtimeProjection,
		environmentProjection,
		ref,
		definition,
		candidate,
		profile,
		env,
	); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyEnvironmentProjectionIgnoresUnrelatedAuthorityGrowth(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	runtimeProjection, ref, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		candidate,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	env := legacyReferenceEnvironment(perm)
	projection, err := projectLegacyEnvironmentAuthority(
		env, runtimeProjection, ref, profile, fingerprint("migration-authority"),
	)
	if err != nil {
		t.Fatal(err)
	}

	env.PermissionGrants = append(env.PermissionGrants, environment.PermissionGrant{
		CapabilityKind: string(plugin.CapabilityEvidenceProvider),
		CapabilityID:   core.MustCanonicalID("unrelated.evidence"),
		Permission: permission.Spec{
			Kind:   permission.KindNetwork,
			Access: permission.AccessConnect,
			Scope:  "unrelated-network",
		},
	})
	env.LeasePolicies = append(env.LeasePolicies, environment.CapabilityLeasePolicy{
		CapabilityKind: string(plugin.CapabilityEvidenceProvider),
		CapabilityID:   core.MustCanonicalID("unrelated.evidence"),
		Policy:         resource.DefaultLeasePolicy(),
	})
	if err := verifyLegacyEnvironmentAuthorityProjection(
		projection, env, runtimeProjection, ref, profile,
	); err != nil {
		t.Fatalf("unrelated legacy Authority growth made projection stale: %v", err)
	}
}

func TestLegacyEnvironmentProjectionDetectsRelevantAuthorityDrift(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	runtimeProjection, ref, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		candidate,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	env := legacyReferenceEnvironment(perm)
	projection, err := projectLegacyEnvironmentAuthority(
		env, runtimeProjection, ref, profile, fingerprint("migration-authority"),
	)
	if err != nil {
		t.Fatal(err)
	}

	env.PermissionGrants[0].Permission.Access = permission.AccessReadWrite
	if err := verifyLegacyEnvironmentAuthorityProjection(
		projection, env, runtimeProjection, ref, profile,
	); err == nil || !strings.Contains(err.Error(), "Permission Authority") {
		t.Fatalf("relevant legacy Environment Authority drift was accepted: %v", err)
	}
}

func TestLegacyEnvironmentMigrationRequiresAcceptedExecutionAuthority(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	runtimeProjection, ref, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		candidate,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectLegacyEnvironmentAuthority(
		legacyReferenceEnvironment(perm),
		runtimeProjection,
		ref,
		profile,
		"",
	); err == nil || !strings.Contains(err.Error(), "accepted Execution Authority Evidence") {
		t.Fatalf("legacy Environment migration manufactured Execution Authority: %v", err)
	}
}

func TestProjectedLegacyAuthorityCannotCoexistWithIndependentLegacyClaim(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	definition := legacyReferenceDefinition([]permission.Spec{perm})
	candidate := referenceRuntimeCandidate(profile)
	runtimeProjection, ref, err := projectLegacyRuntimeCapabilityRef(
		definition,
		plugin.CapabilityDeploymentExecutor,
		core.MustCanonicalID("legacy.document_render"),
		candidate,
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := projectLegacyEnvironmentAuthority(
		legacyReferenceEnvironment(perm),
		runtimeProjection,
		ref,
		profile,
		fingerprint("migration-authority"),
	)
	if err != nil {
		t.Fatal(err)
	}
	authority := projection.Authority
	authority.LegacyClaims = []LegacyAuthorityClaim{{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		CapabilityKind:               string(runtimeProjection.LegacyCapabilityKind),
		CapabilityID:                 runtimeProjection.LegacyCapabilityID,
	}}
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, authority); err == nil ||
		!strings.Contains(err.Error(), "both generic and legacy Authority") {
		t.Fatalf("projected generic Authority coexisted with independent legacy claim: %v", err)
	}
}
