package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
)

// This fixture is NOT an Accepted contract or a built Go artifact. PR #1567
// currently provides a v0 Composition declaration and a synthetic zero digest.
// This candidate represents only the additional evidence a future Host would
// need to accept before using #976's generic Runtime Capability path.
func projectAdviceCandidate(profile ExecutionProfile) AcceptedRuntimeCandidate {
	return AcceptedRuntimeCandidate{
		ContractID:                          "tei.reference.project.customer-advice",
		ContractVersion:                     "1.0.0",
		ContractFingerprint:                 fingerprint("hypothetical-project-advice-contract-shape"),
		Plugin:                              plugin.Identity{ID: core.MustCanonicalID("tei.reference.project_customer_advice"), Version: "1.0.0"},
		ImplementationID:                    "project.customer-advice.go",
		ArtifactDigest:                      "sha256:" + strings.Repeat("a", 64),
		RealizationTarget:                   profile.RealizationTarget,
		ExecutionProfileFingerprint:         profile.Fingerprint,
		CapabilityResolutionLockFingerprint: fingerprint("hypothetical-project-advice-resolution-lock"),
	}
}

func TestProjectAdviceExistingV0DeclarationCannotImplyAcceptedRuntimeIdentity(t *testing.T) {
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	if plugin.CapabilityKind(candidate.ContractID).Valid() {
		t.Fatal("project-owned contract must not require a new Core enum")
	}

	// PR #1567 declares v0 and a descriptive Shape, not an accepted semantic
	// version or the fingerprint of a reviewed request/result/failure shape.
	// A Host must not silently relabel the old declaration as an Accepted v1.
	raw := candidate
	raw.ContractVersion = "v0"
	if _, err := deriveRuntimeCapabilityRef(raw, profile); err == nil {
		t.Fatal("v0 Composition declaration was silently accepted as Runtime Contract")
	}
	raw = candidate
	raw.ContractFingerprint = "customer-code-note-to-advisory-v0"
	if _, err := deriveRuntimeCapabilityRef(raw, profile); err == nil {
		t.Fatal("descriptive Composition shape was treated as contract fingerprint")
	}
	raw = candidate
	raw.CapabilityResolutionLockFingerprint = ""
	if _, err := deriveRuntimeCapabilityRef(raw, profile); err == nil {
		t.Fatal("runtime identity without Capability Resolution Lock was accepted")
	}
	raw = candidate
	raw.ArtifactDigest = ""
	if _, err := deriveRuntimeCapabilityRef(raw, profile); err == nil {
		t.Fatal("runtime identity without exact artifact digest was accepted")
	}
}

func TestProjectAdviceGenericRuntimeNeedsExplicitEnvironmentAuthority(t *testing.T) {
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateRuntimeCapabilityRef(ref); err != nil {
		t.Fatal(err)
	}
	authority := referenceAuthority(ref, nil)
	authority.ExecutionGrants = nil
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, authority); err == nil {
		t.Fatal("registry or runtime identity alone granted Environment execution authority")
	}

	// Synthetic grant tests plan mechanics only; no issuer attestation,
	// artifact verification, runtime invocation, or production permission.
	authority = referenceAuthority(ref, nil)
	plan, err := buildRuntimeEnvironmentPlan(ref, profile, authority)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeEnvironmentPlan(plan, ref, profile, authority); err != nil {
		t.Fatal(err)
	}
	if len(plan.Permissions) != 0 || len(plan.ResourceBindings) != 0 {
		t.Fatal("project advisory unexpectedly acquired ambient resource authority")
	}

	revoked := authority
	revoked.ExecutionGrants = nil
	if err := verifyRuntimeEnvironmentPlan(plan, ref, profile, revoked); err == nil {
		t.Fatal("withdrawn Environment grant left a cached plan current")
	}
	duplicate := authority
	duplicate.ExecutionGrants = append(append([]RuntimeExecutionGrant(nil), authority.ExecutionGrants...), authority.ExecutionGrants[0])
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, duplicate); err == nil {
		t.Fatal("duplicate Environment execution grant was accepted")
	}
	dual := authority
	dual.LegacyClaims = []LegacyAuthorityClaim{{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		CapabilityKind:               "legacy-project-advice",
		CapabilityID:                 core.MustCanonicalID("legacy.project_advice"),
	}}
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, dual); err == nil {
		t.Fatal("project advisory received independent legacy and generic Authority")
	}
}

func TestProjectAdviceRuntimeAuthorityDoesNotFollowArtifactOrContractDrift(t *testing.T) {
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	original, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	authority := referenceAuthority(original, nil)
	originalPlan, err := buildRuntimeEnvironmentPlan(original, profile, authority)
	if err != nil {
		t.Fatal(err)
	}
	if originalPlan.RuntimeCapability.Fingerprint != original.Fingerprint {
		t.Fatal("plan did not retain exact runtime identity")
	}

	cases := []struct {
		name   string
		change func(*AcceptedRuntimeCandidate)
	}{
		{"artifact", func(c *AcceptedRuntimeCandidate) { c.ArtifactDigest = "sha256:" + strings.Repeat("b", 64) }},
		{"contract", func(c *AcceptedRuntimeCandidate) {
			c.ContractFingerprint = fingerprint("different-project-advice-contract-shape")
		}},
		{"resolution-lock", func(c *AcceptedRuntimeCandidate) {
			c.CapabilityResolutionLockFingerprint = fingerprint("different-project-advice-resolution-lock")
		}},
		{"provider", func(c *AcceptedRuntimeCandidate) { c.Plugin.ID = core.MustCanonicalID("tei.reference.other_project") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := candidate
			tc.change(&changed)
			next, err := deriveRuntimeCapabilityRef(changed, profile)
			if err != nil {
				t.Fatal(err)
			}
			if next.Fingerprint == original.Fingerprint {
				t.Fatal("changed accepted identity reused old Runtime Capability Ref")
			}
			if _, err := buildRuntimeEnvironmentPlan(next, profile, authority); err == nil {
				t.Fatal("old project grant authorized a different Runtime Capability Ref")
			}
		})
	}

	// An additional filesystem permission is a distinct execution profile.
	// It may not reuse either the old runtime identity or its old grant.
	privileged := referenceExecutionProfile(t, "linux-amd64", []permission.Spec{referencePermission()})
	privilegedCandidate := candidate
	privilegedCandidate.ExecutionProfileFingerprint = privileged.Fingerprint
	privilegedRef, err := deriveRuntimeCapabilityRef(privilegedCandidate, privileged)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildRuntimeEnvironmentPlan(privilegedRef, privileged, authority); err == nil {
		t.Fatal("read-only project advice silently acquired new filesystem permissions")
	}
}
