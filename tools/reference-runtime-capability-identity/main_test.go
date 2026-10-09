package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

func referenceExecutionProfile(
	t *testing.T,
	target string,
	permissions []permission.Spec,
) ExecutionProfile {
	t.Helper()
	profile, err := newExecutionProfile(
		target,
		RuntimeIsolationHostEnforcedSandbox,
		permissions,
	)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func referenceRuntimeCandidate(
	profile ExecutionProfile,
) AcceptedRuntimeCandidate {
	return AcceptedRuntimeCandidate{
		ContractID: "application.runtime.document-render",
		ContractVersion: "1.0.0",
		ContractFingerprint: fingerprint("document-render-contract-v1"),
		Plugin: plugin.Identity{
			ID: core.MustCanonicalID("tei.reference.document_render"),
			Version: "1.0.0",
		},
		ImplementationID: "document-render.process",
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
		RealizationTarget: profile.RealizationTarget,
		ExecutionProfileFingerprint: profile.Fingerprint,
		CapabilityResolutionLockFingerprint: fingerprint("accepted-resolution-lock"),
	}
}

func referenceRuntimeRef(
	t *testing.T,
	profile ExecutionProfile,
) RuntimeCapabilityRef {
	t.Helper()
	ref, err := deriveRuntimeCapabilityRef(
		referenceRuntimeCandidate(profile),
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func referencePermission() permission.Spec {
	return permission.Spec{
		Kind: permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope: "workspace-source",
	}
}

func referenceResourceBinding() resource.Binding {
	return resource.Binding{
		Kind: permission.KindFilesystem,
		Scope: "workspace-source",
		ResourceID: core.MustCanonicalID("resource.workspace_source"),
	}
}

func referenceAuthority(
	ref RuntimeCapabilityRef,
	permissions []permission.Spec,
) RuntimeEnvironmentAuthority {
	authority := RuntimeEnvironmentAuthority{
		Environment: core.MustCanonicalID("reference_runtime"),
		ExecutionGrants: []RuntimeExecutionGrant{{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			ExecutionProfileFingerprint: ref.ExecutionProfileFingerprint,
			AuthorityEvidenceFingerprint: fingerprint("reference-environment-execution-authority"),
		}},
	}
	for _, p := range permissions {
		authority.PermissionGrants = append(
			authority.PermissionGrants,
			RuntimePermissionGrant{
				RuntimeCapabilityFingerprint: ref.Fingerprint,
				Permission: p,
			},
		)
	}
	if len(permissions) > 0 {
		authority.ResourceBindings = []RuntimeResourceBinding{{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			Binding: referenceResourceBinding(),
		}}
		authority.LeasePolicies = []RuntimeLeasePolicy{{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			Policy: resource.LeasePolicy{
				MaxLifetime: "20m",
				RenewBefore: "3m",
				RevokeTimeout: "2s",
				RenewalFailure: resource.RenewalFailClosed,
			},
		}}
	}
	return authority
}

func TestRuntimeCapabilityRefDoesNotRequireCoreCapabilityKind(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	ref := referenceRuntimeRef(t, profile)
	if plugin.CapabilityKind(ref.ContractID).Valid() {
		t.Fatalf("test Contract unexpectedly became a Core CapabilityKind: %q", ref.ContractID)
	}
	if err := validateRuntimeCapabilityRef(ref); err != nil {
		t.Fatal(err)
	}
	if ref.ContractID != "application.runtime.document-render" ||
		ref.PluginID != core.MustCanonicalID("tei.reference.document_render") ||
		ref.ExecutionProfileFingerprint != profile.Fingerprint {
		t.Fatalf("runtime Capability Ref=%+v", ref)
	}
}

func TestArbitraryStringOnlyRuntimeCapabilityRefFailsClosed(t *testing.T) {
	ref := RuntimeCapabilityRef{
		Schema: runtimeCapabilityRefSchema,
		ContractID: "arbitrary.rpc",
		ContractVersion: "1.0.0",
		PluginID: core.MustCanonicalID("tei.reference.arbitrary"),
		PluginVersion: "1.0.0",
		ImplementationID: "arbitrary",
		ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
		RealizationTarget: "server",
	}
	ref.Fingerprint = runtimeCapabilityRefFingerprint(ref)
	if err := validateRuntimeCapabilityRef(ref); err == nil {
		t.Fatal("string-only Runtime Capability Ref was accepted without accepted fingerprints")
	}
}

func TestRuntimeCapabilityRefRejectsExecutionProfileDrift(t *testing.T) {
	original := referenceExecutionProfile(t, "web-browser", nil)
	candidate := referenceRuntimeCandidate(original)
	drifted := referenceExecutionProfile(t, "server-linux-amd64", nil)
	if _, err := deriveRuntimeCapabilityRef(candidate, drifted); err == nil ||
		!strings.Contains(err.Error(), "realization Target") {
		t.Fatalf("execution profile drift was accepted: %v", err)
	}
}

func TestRuntimeCapabilityRefIdentityChangesWithAcceptedRuntimeInputs(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	baseCandidate := referenceRuntimeCandidate(profile)
	base, err := deriveRuntimeCapabilityRef(baseCandidate, profile)
	if err != nil {
		t.Fatal(err)
	}

	artifactCandidate := baseCandidate
	artifactCandidate.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	artifact, err := deriveRuntimeCapabilityRef(artifactCandidate, profile)
	if err != nil {
		t.Fatal(err)
	}

	contractCandidate := baseCandidate
	contractCandidate.ContractFingerprint = fingerprint("document-render-contract-v1-revised")
	contract, err := deriveRuntimeCapabilityRef(contractCandidate, profile)
	if err != nil {
		t.Fatal(err)
	}

	targetProfile := referenceExecutionProfile(t, "server-linux-amd64", nil)
	targetCandidate := baseCandidate
	targetCandidate.RealizationTarget = targetProfile.RealizationTarget
	targetCandidate.ExecutionProfileFingerprint = targetProfile.Fingerprint
	target, err := deriveRuntimeCapabilityRef(targetCandidate, targetProfile)
	if err != nil {
		t.Fatal(err)
	}

	permissionProfile := referenceExecutionProfile(
		t, "web-browser", []permission.Spec{referencePermission()},
	)
	profileCandidate := baseCandidate
	profileCandidate.ExecutionProfileFingerprint = permissionProfile.Fingerprint
	profileRef, err := deriveRuntimeCapabilityRef(profileCandidate, permissionProfile)
	if err != nil {
		t.Fatal(err)
	}

	for name, got := range map[string]RuntimeCapabilityRef{
		"artifact": artifact,
		"contract": contract,
		"target": target,
		"profile": profileRef,
	} {
		if got.Fingerprint == base.Fingerprint {
			t.Fatalf("%s drift did not change Runtime Capability identity", name)
		}
	}
}

func TestRuntimeCapabilityRefDoesNotAcquireExecutionAuthorityByExistence(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	ref := referenceRuntimeRef(t, profile)
	authority := RuntimeEnvironmentAuthority{
		Environment: core.MustCanonicalID("reference_runtime"),
	}
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, authority); err == nil ||
		!strings.Contains(err.Error(), "Execution Grant") {
		t.Fatalf("Runtime Capability Ref acquired execution Authority implicitly: %v", err)
	}
}

func TestEnvironmentAuthorityMustExactlyMatchExecutionProfile(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	ref := referenceRuntimeRef(t, profile)

	missing := referenceAuthority(ref, nil)
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, missing); err == nil ||
		!strings.Contains(err.Error(), "Permission Grant") {
		t.Fatalf("missing Environment Permission Grant was accepted: %v", err)
	}

	extraPermission := permission.Spec{
		Kind: permission.KindNetwork,
		Access: permission.AccessConnect,
		Scope: "unexpected-network",
	}
	extra := referenceAuthority(ref, []permission.Spec{perm})
	extra.PermissionGrants = append(extra.PermissionGrants, RuntimePermissionGrant{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Permission: extraPermission,
	})
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, extra); err == nil ||
		!strings.Contains(err.Error(), "Permission Grant") {
		t.Fatalf("ambient extra Permission was accepted: %v", err)
	}

	exact := referenceAuthority(ref, []permission.Spec{perm})
	plan, err := buildRuntimeEnvironmentPlan(ref, profile, exact)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Permissions) != 1 || len(plan.ResourceBindings) != 1 {
		t.Fatalf("exact Environment Authority was not preserved: %+v", plan)
	}
}

func TestEnvironmentAuthorityRejectsLegacyAndGenericDoubleAuthority(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	ref := referenceRuntimeRef(t, profile)
	authority := referenceAuthority(ref, nil)
	authority.LegacyClaims = []LegacyAuthorityClaim{{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		CapabilityKind: string(plugin.CapabilityDeploymentExecutor),
		CapabilityID: core.MustCanonicalID("legacy.document_render"),
	}}
	if _, err := buildRuntimeEnvironmentPlan(ref, profile, authority); err == nil ||
		!strings.Contains(err.Error(), "both generic and legacy Authority") {
		t.Fatalf("double Authority was accepted: %v", err)
	}
}

func TestUnrelatedEnvironmentAuthorityDoesNotStaleAcceptedRuntimeRef(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	ref := referenceRuntimeRef(t, profile)
	authority := referenceAuthority(ref, nil)
	base, err := buildRuntimeEnvironmentPlan(ref, profile, authority)
	if err != nil {
		t.Fatal(err)
	}

	unrelated := fingerprint("unrelated-runtime-capability")
	authority.PermissionGrants = append(authority.PermissionGrants, RuntimePermissionGrant{
		RuntimeCapabilityFingerprint: unrelated,
		Permission: referencePermission(),
	})
	authority.ResourceBindings = append(authority.ResourceBindings, RuntimeResourceBinding{
		RuntimeCapabilityFingerprint: unrelated,
		Binding: referenceResourceBinding(),
	})
	authority.LeasePolicies = append(authority.LeasePolicies, RuntimeLeasePolicy{
		RuntimeCapabilityFingerprint: unrelated,
		Policy: resource.DefaultLeasePolicy(),
	})
	after, err := buildRuntimeEnvironmentPlan(ref, profile, authority)
	if err != nil {
		t.Fatal(err)
	}
	if base.RuntimePolicyFingerprint != after.RuntimePolicyFingerprint {
		t.Fatal("unrelated Runtime Authority changed accepted Runtime Environment Plan")
	}
	if err := verifyRuntimeEnvironmentPlan(base, ref, profile, authority); err != nil {
		t.Fatalf("unrelated Runtime Authority made accepted Plan stale: %v", err)
	}
}

func TestEnvironmentExecutionAuthorityRefreshStalesPlanButNotRuntimeRef(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	ref := referenceRuntimeRef(t, profile)
	authority := referenceAuthority(ref, nil)
	plan, err := buildRuntimeEnvironmentPlan(ref, profile, authority)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeEnvironmentPlan(plan, ref, profile, authority); err != nil {
		t.Fatal(err)
	}

	authority.ExecutionGrants[0].AuthorityEvidenceFingerprint =
		fingerprint("refreshed-environment-execution-authority")
	if ref.Fingerprint != referenceRuntimeRef(t, profile).Fingerprint {
		t.Fatal("Environment Authority refresh changed Runtime Capability identity")
	}
	if err := verifyRuntimeEnvironmentPlan(plan, ref, profile, authority); err == nil ||
		!strings.Contains(err.Error(), "differs from current Environment Authority") {
		t.Fatalf("stale Runtime Environment Plan survived Authority refresh: %v", err)
	}
}

func TestRuntimeCapabilityRefDoesNotContainConcreteResourceAuthority(t *testing.T) {
	profile := referenceExecutionProfile(
		t, "web-browser", []permission.Spec{referencePermission()},
	)
	ref := referenceRuntimeRef(t, profile)
	raw, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"resourceId", "mountPath", "leasePolicy", "secretTarget",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("Runtime Capability Ref leaked concrete Resource Authority %q: %s", forbidden, raw)
		}
	}
}

type recordingBroker struct {
	AcquireCalls int
	ReleaseCalls int
	LastAcquire resource.AcquireRequest
}

func (b *recordingBroker) Acquire(
	ctx context.Context,
	req resource.AcquireRequest,
) (resource.Lease, error) {
	if err := ctx.Err(); err != nil {
		return resource.Lease{}, err
	}
	b.AcquireCalls++
	b.LastAcquire = req
	if len(req.Permissions) != 1 || len(req.Bindings) != 1 {
		return resource.Lease{}, fmt.Errorf("reference broker expects one Permission and Binding")
	}
	now := time.Date(2026, 10, 9, 4, 0, 0, 0, time.UTC)
	perm := req.Permissions[0]
	binding := req.Bindings[0]
	return resource.Lease{
		ID: "generic-runtime-lease",
		Revision: 1,
		IssuedAt: now.Format(time.RFC3339Nano),
		ExpiresAt: now.Add(10 * time.Minute).Format(time.RFC3339Nano),
		RuntimePolicyFingerprint: req.RuntimePolicyFingerprint,
		Resources: []resource.Resolved{{
			Kind: perm.Kind,
			Access: perm.Access,
			Scope: perm.Scope,
			ResourceID: binding.ResourceID,
			MountSourceHandle: "workspace-source-handle",
			MountPath: "/workspace/source",
		}},
	}, nil
}

func (b *recordingBroker) Release(
	ctx context.Context,
	lease resource.Lease,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease.ID != "generic-runtime-lease" {
		return fmt.Errorf("unexpected lease %q", lease.ID)
	}
	b.ReleaseCalls++
	return nil
}

type recordingNetworkEnforcer struct {
	Calls int
	Policy pruntime.NetworkPolicy
}

func (e *recordingNetworkEnforcer) ApplyNetworkPolicy(
	ctx context.Context,
	policy pruntime.NetworkPolicy,
) (pruntime.Cleanup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.Calls++
	e.Policy = policy
	return func(context.Context) error { return nil }, nil
}

type recordingFilesystemEnforcer struct {
	Calls int
	Policy pruntime.FilesystemPolicy
}

func (e *recordingFilesystemEnforcer) ApplyFilesystemPolicy(
	ctx context.Context,
	policy pruntime.FilesystemPolicy,
) (pruntime.Cleanup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.Calls++
	e.Policy = policy
	return func(context.Context) error { return nil }, nil
}

type recordingSandboxRunner struct {
	Calls int
	Request pruntime.Request
}

func (r *recordingSandboxRunner) InvokeSandboxed(
	ctx context.Context,
	command pruntime.ProcessCommand,
	req pruntime.Request,
) (pruntime.Response, error) {
	if err := ctx.Err(); err != nil {
		return pruntime.Response{}, err
	}
	if command.Executable == "" {
		return pruntime.Response{}, fmt.Errorf("sandbox runner received empty executable")
	}
	r.Calls++
	r.Request = req
	return pruntime.Response{Status: pruntime.ResponseOK}, nil
}

func TestExistingResourceBrokerAndHostSandboxExecuteGenericRuntimeRef(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	ref := referenceRuntimeRef(t, profile)
	plan, err := buildRuntimeEnvironmentPlan(
		ref, profile, referenceAuthority(ref, []permission.Spec{perm}),
	)
	if err != nil {
		t.Fatal(err)
	}
	broker := &recordingBroker{}
	network := &recordingNetworkEnforcer{}
	filesystem := &recordingFilesystemEnforcer{}
	runner := &recordingSandboxRunner{}
	host := pruntime.HostSandboxLauncher{
		Network: network,
		Filesystem: filesystem,
		Runner: runner,
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin: plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact: plugin.Artifact{Digest: ref.ArtifactDigest},
		Command: pruntime.ProcessCommand{Executable: "document-render-worker"},
	}
	prov, err := invokeWithExistingHostBoundary(
		context.Background(),
		plan,
		realization,
		broker,
		host,
		"render",
		"request-generic-1",
		map[string]any{"document": "sample"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if broker.AcquireCalls != 1 || broker.ReleaseCalls != 1 {
		t.Fatalf("Resource Broker calls acquire=%d release=%d", broker.AcquireCalls, broker.ReleaseCalls)
	}
	expectedTransportID, err := genericTransportCapabilityID(ref)
	if err != nil {
		t.Fatal(err)
	}
	if broker.LastAcquire.CapabilityKind != genericResourceTransportKind ||
		broker.LastAcquire.CapabilityID != expectedTransportID ||
		broker.LastAcquire.RuntimePolicyFingerprint != plan.RuntimePolicyFingerprint {
		t.Fatalf("Resource Broker transport identity=%+v", broker.LastAcquire)
	}
	if runner.Calls != 1 ||
		runner.Request.CapabilityKind != plugin.CapabilityKind(genericResourceTransportKind) ||
		runner.Request.CapabilityID != expectedTransportID {
		t.Fatalf("sandbox runner request=%+v", runner.Request)
	}
	if network.Calls != 1 || !network.Policy.DenyByDefault || len(network.Policy.Allowlist) != 0 {
		t.Fatalf("network policy=%+v", network.Policy)
	}
	if filesystem.Calls != 1 ||
		!filesystem.Policy.DenyByDefault ||
		len(filesystem.Policy.Mounts) != 1 ||
		filesystem.Policy.Mounts[0].Target != "/workspace/source" ||
		filesystem.Policy.Mounts[0].Access != permission.AccessRead {
		t.Fatalf("filesystem policy=%+v", filesystem.Policy)
	}
	if prov.RuntimeCapability.Fingerprint != ref.Fingerprint ||
		prov.RuntimeCapability.ContractFingerprint != ref.ContractFingerprint ||
		prov.RuntimeCapability.PluginID != ref.PluginID ||
		prov.RuntimeCapability.ImplementationID != ref.ImplementationID ||
		prov.ResourceLeaseID != "generic-runtime-lease" ||
		!prov.ResourceLeaseReleased ||
		len(prov.Resources) != 1 {
		t.Fatalf("generic invocation provenance=%+v", prov)
	}
}

func TestPermissionedGenericRuntimeRefRequiresExistingResourceBroker(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	ref := referenceRuntimeRef(t, profile)
	plan, err := buildRuntimeEnvironmentPlan(
		ref, profile, referenceAuthority(ref, []permission.Spec{perm}),
	)
	if err != nil {
		t.Fatal(err)
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin: plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact: plugin.Artifact{Digest: ref.ArtifactDigest},
		Command: pruntime.ProcessCommand{Executable: "document-render-worker"},
	}
	host := pruntime.HostSandboxLauncher{
		Network: &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner: &recordingSandboxRunner{},
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization, nil, host, "render", "request-2", nil,
	); err == nil || !strings.Contains(err.Error(), "no Resource Broker") {
		t.Fatalf("generic runtime invocation bypassed missing Resource Broker: %v", err)
	}
}

func TestProcessRealizationDriftFailsBeforeResourceAuthority(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	ref := referenceRuntimeRef(t, profile)
	plan, err := buildRuntimeEnvironmentPlan(
		ref, profile, referenceAuthority(ref, []permission.Spec{perm}),
	)
	if err != nil {
		t.Fatal(err)
	}
	broker := &recordingBroker{}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin: plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact: plugin.Artifact{Digest: "sha256:" + strings.Repeat("b", 64)},
		Command: pruntime.ProcessCommand{Executable: "document-render-worker"},
	}
	host := pruntime.HostSandboxLauncher{
		Network: &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner: &recordingSandboxRunner{},
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization, broker, host, "render", "request-3", nil,
	); err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("artifact drift was accepted: %v", err)
	}
	if broker.AcquireCalls != 0 {
		t.Fatal("Resource Authority was acquired before process realization identity validation")
	}
}

func TestHostTrustedProfileCannotUseSandboxOnlyInvocationPath(t *testing.T) {
	profile, err := newExecutionProfile(
		"server-linux-amd64", RuntimeIsolationHostTrusted, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	ref := referenceRuntimeRef(t, profile)
	plan, err := buildRuntimeEnvironmentPlan(ref, profile, referenceAuthority(ref, nil))
	if err != nil {
		t.Fatal(err)
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin: plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact: plugin.Artifact{Digest: ref.ArtifactDigest},
		Command: pruntime.ProcessCommand{Executable: "worker"},
	}
	host := pruntime.HostSandboxLauncher{
		Network: &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner: &recordingSandboxRunner{},
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization, nil, host, "render", "request-4", nil,
	); err == nil || !strings.Contains(err.Error(), "host-enforced-sandbox") {
		t.Fatalf("host-trusted profile was silently routed through sandbox-only path: %v", err)
	}
}

func TestRuntimeEnvironmentPlanValidationRejectsRewrittenPermissionAuthority(t *testing.T) {
	perm := referencePermission()
	profile := referenceExecutionProfile(t, "web-browser", []permission.Spec{perm})
	ref := referenceRuntimeRef(t, profile)
	plan, err := buildRuntimeEnvironmentPlan(
		ref, profile, referenceAuthority(ref, []permission.Spec{perm}),
	)
	if err != nil {
		t.Fatal(err)
	}
	tampered := plan
	tampered.Permissions = append(tampered.Permissions, permission.Spec{
		Kind: permission.KindNetwork,
		Access: permission.AccessConnect,
		Scope: "ambient-network",
	})
	tampered.RuntimePolicyFingerprint = runtimeEnvironmentPlanFingerprint(tampered)
	if err := validateRuntimeEnvironmentPlan(tampered); err == nil {
		t.Fatal("self-consistent fingerprint hid rewritten Permission Authority")
	}
}
