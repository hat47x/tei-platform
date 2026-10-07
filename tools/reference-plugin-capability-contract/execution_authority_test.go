package main

import (
	"strings"
	"testing"

	"tei/internal/permission"
)

func executionAuthorityFixture(
	t *testing.T,
	profile CapabilityExecutionProfile,
) (ResolveInput, CapabilityResolutionLock) {
	t.Helper()
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	var err error
	provider, err = bindProviderExecutionProfile(provider, profile)
	if err != nil {
		t.Fatal(err)
	}
	input := ResolveInput{
		Scope:             ScopeAuthoringTool,
		RealizationTarget: profile.RealizationTarget,
		Requirement:       Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:         []Contract{contract},
		Providers:         []Provider{provider},
		Evidence:          []ConformanceEvidence{evidenceFor(provider, contract, "execution-profile-conformance")},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	return input, lock
}

func TestExecutionProfileNormalizesPermissionOrdering(t *testing.T) {
	profile, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"desktop",
		CapabilityIsolationHostEnforcedSandbox,
		[]permission.Spec{
			{
				Kind:   permission.KindNetwork,
				Access: permission.AccessConnect,
				Scope:  "metadata-service",
			},
			{
				Kind:   permission.KindFilesystem,
				Access: permission.AccessRead,
				Scope:  "workspace-source",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !validSHA256Fingerprint(profile.ProfileFingerprint) {
		t.Fatalf("profile fingerprint=%q", profile.ProfileFingerprint)
	}
	if len(profile.Permissions) != 2 ||
		profile.Permissions[0].Kind != permission.KindFilesystem ||
		profile.Permissions[1].Kind != permission.KindNetwork {
		t.Fatalf("permissions were not normalized: %+v", profile.Permissions)
	}
}

func TestExecutionAuthorityBindsExactPermissionsWithoutCapabilityKind(t *testing.T) {
	required := []permission.Spec{{
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}}
	profile, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"desktop",
		CapabilityIsolationHostEnforcedSandbox,
		required,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, lock := executionAuthorityFixture(t, profile)

	grant := CapabilityExecutionGrant{
		CapabilityLockFingerprint:    lock.LockFingerprint,
		ExecutionProfileFingerprint:  profile.ProfileFingerprint,
		Permissions:                  append([]permission.Spec(nil), required...),
		AuthorityEvidenceFingerprint: fingerprint("authoring-execution-authority"),
	}
	binding, err := verifyCapabilityExecutionAuthority(lock, profile, grant)
	if err != nil {
		t.Fatal(err)
	}
	if binding.CapabilityLockFingerprint != lock.LockFingerprint ||
		binding.ExecutionProfileFingerprint != profile.ProfileFingerprint ||
		binding.IsolationPolicy != CapabilityIsolationHostEnforcedSandbox ||
		len(binding.Permissions) != 1 ||
		!validSHA256Fingerprint(binding.BindingFingerprint) {
		t.Fatalf("execution authority binding=%+v", binding)
	}
}

func TestExecutionAuthorityRejectsMissingAndExtraPermissions(t *testing.T) {
	required := []permission.Spec{{
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}}
	profile, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"desktop",
		CapabilityIsolationHostEnforcedSandbox,
		required,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, lock := executionAuthorityFixture(t, profile)

	base := CapabilityExecutionGrant{
		CapabilityLockFingerprint:    lock.LockFingerprint,
		ExecutionProfileFingerprint:  profile.ProfileFingerprint,
		AuthorityEvidenceFingerprint: fingerprint("execution-authority"),
	}

	if _, err := verifyCapabilityExecutionAuthority(lock, profile, base); err == nil ||
		!strings.Contains(err.Error(), "permissions differ") {
		t.Fatalf("missing Permission was accepted: %v", err)
	}

	extra := base
	extra.Permissions = []permission.Spec{
		required[0],
		{
			Kind:   permission.KindNetwork,
			Access: permission.AccessConnect,
			Scope:  "unexpected-network",
		},
	}
	if _, err := verifyCapabilityExecutionAuthority(lock, profile, extra); err == nil ||
		!strings.Contains(err.Error(), "permissions differ") {
		t.Fatalf("extra Permission was accepted: %v", err)
	}
}

func TestExecutionProfileDriftChangesResolutionIdentityAndInvalidatesReplay(t *testing.T) {
	base, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"desktop",
		CapabilityIsolationHostTrusted,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	input, lock := executionAuthorityFixture(t, base)

	changed, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"desktop",
		CapabilityIsolationHostEnforcedSandbox,
		[]permission.Spec{{
			Kind:   permission.KindFilesystem,
			Access: permission.AccessRead,
			Scope:  "workspace-source",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	changedProvider, err := bindProviderExecutionProfile(input.Providers[0], changed)
	if err != nil {
		t.Fatal(err)
	}
	changedInput := input
	changedInput.Providers = []Provider{changedProvider}
	changedInput.Evidence = []ConformanceEvidence{
		evidenceFor(changedProvider, sourceAdapterContract(), "execution-profile-conformance"),
	}

	changedResolution, err := resolveCapability(changedInput)
	if err != nil {
		t.Fatal(err)
	}
	if changedResolution.ResolutionFingerprint == lock.ResolutionFingerprint {
		t.Fatal("execution profile drift did not change capability resolution identity")
	}
	if err := verifyCapabilityResolutionLockReplay(lock, changedInput); err == nil {
		t.Fatal("execution profile drift was accepted by old Capability Resolution Lock")
	}
}

func TestExecutionAuthorityCanAuthorizeZeroAmbientPermissions(t *testing.T) {
	profile, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"",
		CapabilityIsolationHostTrusted,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, lock := executionAuthorityFixture(t, profile)
	grant := CapabilityExecutionGrant{
		CapabilityLockFingerprint:    lock.LockFingerprint,
		ExecutionProfileFingerprint:  profile.ProfileFingerprint,
		AuthorityEvidenceFingerprint: fingerprint("zero-permission-authority"),
	}
	binding, err := verifyCapabilityExecutionAuthority(lock, profile, grant)
	if err != nil {
		t.Fatal(err)
	}
	if len(binding.Permissions) != 0 {
		t.Fatalf("zero-permission execution gained ambient authority: %+v", binding.Permissions)
	}
}

func TestExecutionAuthorityEvidenceRefreshDoesNotChangeCapabilityLock(t *testing.T) {
	profile, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"",
		CapabilityIsolationHostTrusted,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, lock := executionAuthorityFixture(t, profile)
	lockFingerprint := lock.LockFingerprint

	left, err := verifyCapabilityExecutionAuthority(
		lock,
		profile,
		CapabilityExecutionGrant{
			CapabilityLockFingerprint:    lock.LockFingerprint,
			ExecutionProfileFingerprint:  profile.ProfileFingerprint,
			AuthorityEvidenceFingerprint: fingerprint("authority-left"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	right, err := verifyCapabilityExecutionAuthority(
		lock,
		profile,
		CapabilityExecutionGrant{
			CapabilityLockFingerprint:    lock.LockFingerprint,
			ExecutionProfileFingerprint:  profile.ProfileFingerprint,
			AuthorityEvidenceFingerprint: fingerprint("authority-right"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if lock.LockFingerprint != lockFingerprint {
		t.Fatal("execution Authority Evidence refresh changed Capability Lock identity")
	}
	if left.BindingFingerprint == right.BindingFingerprint {
		t.Fatal("execution Authority Evidence refresh was invisible")
	}
}

func TestExecutionProfileRejectsDuplicatePermission(t *testing.T) {
	permissionSpec := permission.Spec{
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}
	_, err := newCapabilityExecutionProfile(
		ScopeAuthoringTool,
		"",
		CapabilityIsolationHostEnforcedSandbox,
		[]permission.Spec{permissionSpec, permissionSpec},
	)
	if err == nil || !strings.Contains(err.Error(), "declared more than once") {
		t.Fatalf("duplicate Permission was accepted: %v", err)
	}
}
