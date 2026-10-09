package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

const genericProcessHelperEnv = "TEI_GENERIC_RUNTIME_PROCESS_HELPER"

func TestMain(m *testing.M) {
	if mode := os.Getenv(genericProcessHelperEnv); mode != "" {
		os.Exit(runGenericRuntimeProcessHelper(mode))
	}
	os.Exit(m.Run())
}

func runGenericRuntimeProcessHelper(mode string) int {
	var req pruntime.Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return 3
	}
	renderID, err := processDispatchID(
		"application.runtime.document-render",
		"1.0.0",
		fingerprint("document-render-contract-v1"),
		"document-render.process",
	)
	if err != nil {
		return 4
	}
	summaryID, err := processDispatchID(
		"application.runtime.document-summary",
		"1.0.0",
		fingerprint("document-summary-contract-v1"),
		"document-summary.process",
	)
	if err != nil {
		return 5
	}
	if req.CapabilityKind != plugin.CapabilityKind(genericProcessDispatchKind) {
		writeGenericProcessHelperError(req, "unexpected generic process dispatch kind")
		return 0
	}

	dispatch := ""
	switch req.CapabilityID {
	case renderID:
		if req.Operation != "render" {
			writeGenericProcessHelperError(req, "render dispatch received wrong operation")
			return 0
		}
		dispatch = "render"
	case summaryID:
		if req.Operation != "summarize" {
			writeGenericProcessHelperError(req, "summary dispatch received wrong operation")
			return 0
		}
		dispatch = "summary"
	default:
		writeGenericProcessHelperError(req, "unknown generic process dispatch id")
		return 0
	}

	payload, err := json.Marshal(map[string]string{"dispatch": dispatch})
	if err != nil {
		return 6
	}
	resp := pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseOK,
		Payload:               payload,
	}
	if mode == "forge" {
		resp.CapabilityID = core.MustCanonicalID("runtime_dispatch.forged")
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		return 7
	}
	return 0
}

func writeGenericProcessHelperError(req pruntime.Request, message string) {
	_ = json.NewEncoder(os.Stdout).Encode(pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseError,
		Error:                 message,
	})
}

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
		ContractID:          "application.runtime.document-render",
		ContractVersion:     "1.0.0",
		ContractFingerprint: fingerprint("document-render-contract-v1"),
		Plugin: plugin.Identity{
			ID:      core.MustCanonicalID("tei.reference.document_render"),
			Version: "1.0.0",
		},
		ImplementationID:                    "document-render.process",
		ArtifactDigest:                      "sha256:" + strings.Repeat("a", 64),
		RealizationTarget:                   profile.RealizationTarget,
		ExecutionProfileFingerprint:         profile.Fingerprint,
		CapabilityResolutionLockFingerprint: fingerprint("accepted-resolution-lock"),
	}
}

func summaryRuntimeCandidate(
	profile ExecutionProfile,
) AcceptedRuntimeCandidate {
	candidate := referenceRuntimeCandidate(profile)
	candidate.ContractID = "application.runtime.document-summary"
	candidate.ContractFingerprint = fingerprint("document-summary-contract-v1")
	candidate.ImplementationID = "document-summary.process"
	candidate.CapabilityResolutionLockFingerprint =
		fingerprint("accepted-summary-resolution-lock")
	return candidate
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
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}
}

func referenceResourceBinding() resource.Binding {
	return resource.Binding{
		Kind:       permission.KindFilesystem,
		Scope:      "workspace-source",
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
			ExecutionProfileFingerprint:  ref.ExecutionProfileFingerprint,
			AuthorityEvidenceFingerprint: fingerprint("reference-environment-execution-authority"),
		}},
	}
	for _, p := range permissions {
		authority.PermissionGrants = append(
			authority.PermissionGrants,
			RuntimePermissionGrant{
				RuntimeCapabilityFingerprint: ref.Fingerprint,
				Permission:                   p,
			},
		)
	}
	if len(permissions) > 0 {
		authority.ResourceBindings = []RuntimeResourceBinding{{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			Binding:                      referenceResourceBinding(),
		}}
		authority.LeasePolicies = []RuntimeLeasePolicy{{
			RuntimeCapabilityFingerprint: ref.Fingerprint,
			Policy: resource.LeasePolicy{
				MaxLifetime:    "20m",
				RenewBefore:    "3m",
				RevokeTimeout:  "2s",
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
		Schema:            runtimeCapabilityRefSchema,
		ContractID:        "arbitrary.rpc",
		ContractVersion:   "1.0.0",
		PluginID:          core.MustCanonicalID("tei.reference.arbitrary"),
		PluginVersion:     "1.0.0",
		ImplementationID:  "arbitrary",
		ArtifactDigest:    "sha256:" + strings.Repeat("a", 64),
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

func TestProcessDispatchIdentityIsStableAcrossExecutionProfileChanges(t *testing.T) {
	baseProfile := referenceExecutionProfile(t, "web-browser", nil)
	baseCandidate := referenceRuntimeCandidate(baseProfile)
	baseRef, err := deriveRuntimeCapabilityRef(baseCandidate, baseProfile)
	if err != nil {
		t.Fatal(err)
	}
	baseResourceID, err := genericResourceAuthoritySubjectID(baseRef)
	if err != nil {
		t.Fatal(err)
	}
	baseDispatchID, err := genericProcessDispatchID(baseRef)
	if err != nil {
		t.Fatal(err)
	}

	permissionProfile := referenceExecutionProfile(
		t, "web-browser", []permission.Spec{referencePermission()},
	)
	permissionCandidate := baseCandidate
	permissionCandidate.ExecutionProfileFingerprint = permissionProfile.Fingerprint
	permissionRef, err := deriveRuntimeCapabilityRef(
		permissionCandidate,
		permissionProfile,
	)
	if err != nil {
		t.Fatal(err)
	}
	permissionResourceID, err :=
		genericResourceAuthoritySubjectID(permissionRef)
	if err != nil {
		t.Fatal(err)
	}
	permissionDispatchID, err := genericProcessDispatchID(permissionRef)
	if err != nil {
		t.Fatal(err)
	}
	if permissionRef.Fingerprint == baseRef.Fingerprint ||
		permissionResourceID == baseResourceID {
		t.Fatal("Execution Profile change did not change Runtime Authority identity")
	}
	if permissionDispatchID != baseDispatchID {
		t.Fatalf(
			"Execution Profile change altered process dispatch identity: %s != %s",
			permissionDispatchID,
			baseDispatchID,
		)
	}

	contractCandidate := baseCandidate
	contractCandidate.ContractFingerprint = fingerprint("changed-contract")
	contractRef, err := deriveRuntimeCapabilityRef(contractCandidate, baseProfile)
	if err != nil {
		t.Fatal(err)
	}
	contractDispatchID, err := genericProcessDispatchID(contractRef)
	if err != nil {
		t.Fatal(err)
	}
	if contractDispatchID == baseDispatchID {
		t.Fatal("Contract semantic drift did not change process dispatch identity")
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
		"target":   target,
		"profile":  profileRef,
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
		Kind:   permission.KindNetwork,
		Access: permission.AccessConnect,
		Scope:  "unexpected-network",
	}
	extra := referenceAuthority(ref, []permission.Spec{perm})
	extra.PermissionGrants = append(extra.PermissionGrants, RuntimePermissionGrant{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Permission:                   extraPermission,
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
		CapabilityKind:               string(plugin.CapabilityDeploymentExecutor),
		CapabilityID:                 core.MustCanonicalID("legacy.document_render"),
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
		Permission:                   referencePermission(),
	})
	authority.ResourceBindings = append(authority.ResourceBindings, RuntimeResourceBinding{
		RuntimeCapabilityFingerprint: unrelated,
		Binding:                      referenceResourceBinding(),
	})
	authority.LeasePolicies = append(authority.LeasePolicies, RuntimeLeasePolicy{
		RuntimeCapabilityFingerprint: unrelated,
		Policy:                       resource.DefaultLeasePolicy(),
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

type recordingArtifactVerifier struct {
	Calls          int
	DigestVerified bool
}

func (v *recordingArtifactVerifier) Verify(
	ctx context.Context,
	identity plugin.Identity,
	artifact plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	if err := ctx.Err(); err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	v.Calls++
	return pruntime.ArtifactVerification{
		VerifierID:        "reference-artifact-verifier",
		DigestVerified:    v.DigestVerified,
		SignatureVerified: artifact.SignatureRef == "",
		ProvenanceChecked: artifact.Provenance == "",
	}, nil
}

type recordingBroker struct {
	AcquireCalls int
	ReleaseCalls int
	LastAcquire  resource.AcquireRequest
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
		ID:                       "generic-runtime-lease",
		Revision:                 1,
		IssuedAt:                 now.Format(time.RFC3339Nano),
		ExpiresAt:                now.Add(10 * time.Minute).Format(time.RFC3339Nano),
		RuntimePolicyFingerprint: req.RuntimePolicyFingerprint,
		Resources: []resource.Resolved{{
			Kind:              perm.Kind,
			Access:            perm.Access,
			Scope:             perm.Scope,
			ResourceID:        binding.ResourceID,
			MountSourceHandle: "workspace-source-handle",
			MountPath:         "/workspace/source",
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
	Calls  int
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
	Calls  int
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
	Calls   int
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
	return pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseOK,
		Payload:               req.Payload,
	}, nil
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
	verifier := &recordingArtifactVerifier{DigestVerified: true}
	network := &recordingNetworkEnforcer{}
	filesystem := &recordingFilesystemEnforcer{}
	runner := &recordingSandboxRunner{}
	host := pruntime.HostSandboxLauncher{
		Network:    network,
		Filesystem: filesystem,
		Runner:     runner,
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin:                       plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: ref.ArtifactDigest},
		Command:                      pruntime.ProcessCommand{Executable: "document-render-worker"},
	}
	inv, err := invokeWithExistingHostBoundary(
		context.Background(),
		plan,
		realization,
		verifier,
		broker,
		host,
		"render",
		"request-generic-1",
		map[string]any{"document": "sample"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if verifier.Calls != 1 ||
		inv.Provenance.ArtifactVerification == nil ||
		!inv.Provenance.ArtifactVerification.DigestVerified {
		t.Fatalf("artifact verification=%+v calls=%d", inv.Provenance.ArtifactVerification, verifier.Calls)
	}
	if broker.AcquireCalls != 1 || broker.ReleaseCalls != 1 {
		t.Fatalf("Resource Broker calls acquire=%d release=%d", broker.AcquireCalls, broker.ReleaseCalls)
	}
	expectedResourceSubjectID, err :=
		genericResourceAuthoritySubjectID(ref)
	if err != nil {
		t.Fatal(err)
	}
	expectedProcessDispatchID, err := genericProcessDispatchID(ref)
	if err != nil {
		t.Fatal(err)
	}
	if broker.LastAcquire.CapabilityKind != genericResourceAuthorityKind ||
		broker.LastAcquire.CapabilityID != expectedResourceSubjectID ||
		broker.LastAcquire.RuntimePolicyFingerprint != plan.RuntimePolicyFingerprint {
		t.Fatalf("Resource Broker Authority identity=%+v", broker.LastAcquire)
	}
	if runner.Calls != 1 ||
		runner.Request.CapabilityKind != plugin.CapabilityKind(genericProcessDispatchKind) ||
		runner.Request.CapabilityID != expectedProcessDispatchID {
		t.Fatalf("sandbox runner dispatch request=%+v", runner.Request)
	}
	if inv.Provenance.ResourceAuthoritySubjectID != expectedResourceSubjectID ||
		inv.Provenance.ProcessDispatchID != expectedProcessDispatchID {
		t.Fatalf("generic invocation transport provenance=%+v", inv.Provenance)
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
	if inv.Provenance.RuntimeCapability.Fingerprint != ref.Fingerprint ||
		inv.Provenance.RuntimeCapability.ContractFingerprint != ref.ContractFingerprint ||
		inv.Provenance.RuntimeCapability.PluginID != ref.PluginID ||
		inv.Provenance.RuntimeCapability.ImplementationID != ref.ImplementationID ||
		inv.Provenance.ResourceLeaseID != "generic-runtime-lease" ||
		!inv.Provenance.ResourceLeaseReleased ||
		len(inv.Provenance.Resources) != 1 {
		t.Fatalf("generic invocation provenance=%+v", inv.Provenance)
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
		Plugin:                       plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: ref.ArtifactDigest},
		Command:                      pruntime.ProcessCommand{Executable: "document-render-worker"},
	}
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     &recordingSandboxRunner{},
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization,
		&recordingArtifactVerifier{DigestVerified: true},
		nil, host, "render", "request-2", nil,
	); err == nil || !strings.Contains(err.Error(), "no Resource Broker") {
		t.Fatalf("generic runtime invocation bypassed missing Resource Broker: %v", err)
	}
}

func TestGenericRuntimeInvocationRequiresArtifactVerifierBeforeResourceAuthority(t *testing.T) {
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
		Plugin:                       plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: ref.ArtifactDigest},
		Command:                      pruntime.ProcessCommand{Executable: "document-render-worker"},
	}
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     &recordingSandboxRunner{},
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(),
		plan,
		realization,
		nil,
		broker,
		host,
		"render",
		"request-verifier-1",
		nil,
	); err == nil || !strings.Contains(err.Error(), "Artifact Verifier") {
		t.Fatalf("generic runtime invocation bypassed missing Artifact Verifier: %v", err)
	}
	if broker.AcquireCalls != 0 {
		t.Fatal("Resource Authority was acquired before artifact verification")
	}

	verifier := &recordingArtifactVerifier{DigestVerified: false}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(),
		plan,
		realization,
		verifier,
		broker,
		host,
		"render",
		"request-verifier-2",
		nil,
	); err == nil || !strings.Contains(err.Error(), "digest was not verified") {
		t.Fatalf("unverified artifact reached Resource Authority: %v", err)
	}
	if broker.AcquireCalls != 0 {
		t.Fatal("Resource Authority was acquired after failed artifact verification")
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
		Plugin:                       plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: "sha256:" + strings.Repeat("b", 64)},
		Command:                      pruntime.ProcessCommand{Executable: "document-render-worker"},
	}
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     &recordingSandboxRunner{},
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization,
		&recordingArtifactVerifier{DigestVerified: true},
		broker, host, "render", "request-3", nil,
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
		Plugin:                       plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: ref.ArtifactDigest},
		Command:                      pruntime.ProcessCommand{Executable: "worker"},
	}
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     &recordingSandboxRunner{},
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization,
		&recordingArtifactVerifier{DigestVerified: true},
		nil, host, "render", "request-4", nil,
	); err == nil || !strings.Contains(err.Error(), "host-enforced-sandbox") {
		t.Fatalf("host-trusted profile was silently routed through sandbox-only path: %v", err)
	}
}

func genericProcessLauncherForTest(
	t *testing.T,
	mode string,
) (pruntime.StdioProcessLauncher, pruntime.ProcessCommand) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := append([]string(nil), os.Environ()...)
	env = append(env, genericProcessHelperEnv+"="+mode)
	return pruntime.StdioProcessLauncher{Env: env},
		pruntime.ProcessCommand{Executable: executable}
}

func TestDerivedDispatchIdentityRoutesMultipleGenericCapabilitiesInRealProcess(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	renderRef := referenceRuntimeRef(t, profile)
	summaryRef, err := deriveRuntimeCapabilityRef(
		summaryRuntimeCandidate(profile),
		profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	renderDispatch, err := genericProcessDispatchID(renderRef)
	if err != nil {
		t.Fatal(err)
	}
	summaryDispatch, err := genericProcessDispatchID(summaryRef)
	if err != nil {
		t.Fatal(err)
	}
	if renderDispatch == summaryDispatch {
		t.Fatal("different Contract / implementation pairs collapsed to one dispatch ID")
	}

	renderPlan, err := buildRuntimeEnvironmentPlan(
		renderRef, profile, referenceAuthority(renderRef, nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	summaryPlan, err := buildRuntimeEnvironmentPlan(
		summaryRef, profile, referenceAuthority(summaryRef, nil),
	)
	if err != nil {
		t.Fatal(err)
	}

	launcher, command := genericProcessLauncherForTest(t, "dispatch")
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     launcher,
	}
	verifier := &recordingArtifactVerifier{DigestVerified: true}
	renderRealization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: renderRef.Fingerprint,
		Plugin:                       plugin.Identity{ID: renderRef.PluginID, Version: renderRef.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: renderRef.ArtifactDigest},
		Command:                      command,
	}
	summaryRealization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: summaryRef.Fingerprint,
		Plugin:                       plugin.Identity{ID: summaryRef.PluginID, Version: summaryRef.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: summaryRef.ArtifactDigest},
		Command:                      command,
	}

	renderInvocation, err := invokeWithExistingHostBoundary(
		context.Background(),
		renderPlan,
		renderRealization,
		verifier,
		nil,
		host,
		"render",
		"real-process-render",
		map[string]string{"document": "one"},
	)
	if err != nil {
		t.Fatal(err)
	}
	summaryInvocation, err := invokeWithExistingHostBoundary(
		context.Background(),
		summaryPlan,
		summaryRealization,
		verifier,
		nil,
		host,
		"summarize",
		"real-process-summary",
		map[string]string{"document": "two"},
	)
	if err != nil {
		t.Fatal(err)
	}

	var renderPayload map[string]string
	if err := json.Unmarshal(renderInvocation.Response.Payload, &renderPayload); err != nil {
		t.Fatal(err)
	}
	var summaryPayload map[string]string
	if err := json.Unmarshal(summaryInvocation.Response.Payload, &summaryPayload); err != nil {
		t.Fatal(err)
	}
	if renderPayload["dispatch"] != "render" ||
		summaryPayload["dispatch"] != "summary" {
		t.Fatalf(
			"real process dispatch render=%v summary=%v",
			renderPayload,
			summaryPayload,
		)
	}
	if renderInvocation.Provenance.ProcessDispatchID != renderDispatch ||
		summaryInvocation.Provenance.ProcessDispatchID != summaryDispatch {
		t.Fatal("real process provenance lost derived dispatch identity")
	}
}

func TestGenericProcessResponseCannotForgeDerivedDispatchIdentity(t *testing.T) {
	profile := referenceExecutionProfile(t, "web-browser", nil)
	ref := referenceRuntimeRef(t, profile)
	plan, err := buildRuntimeEnvironmentPlan(
		ref, profile, referenceAuthority(ref, nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	launcher, command := genericProcessLauncherForTest(t, "forge")
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     launcher,
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin:                       plugin.Identity{ID: ref.PluginID, Version: ref.PluginVersion},
		Artifact:                     plugin.Artifact{Digest: ref.ArtifactDigest},
		Command:                      command,
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(),
		plan,
		realization,
		&recordingArtifactVerifier{DigestVerified: true},
		nil,
		host,
		"render",
		"real-process-forged",
		nil,
	); err == nil || !strings.Contains(err.Error(), "response identity") {
		t.Fatalf("forged generic process response identity was accepted: %v", err)
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
		Kind:   permission.KindNetwork,
		Access: permission.AccessConnect,
		Scope:  "ambient-network",
	})
	tampered.RuntimePolicyFingerprint = runtimeEnvironmentPlanFingerprint(tampered)
	if err := validateRuntimeEnvironmentPlan(tampered); err == nil {
		t.Fatal("self-consistent fingerprint hid rewritten Permission Authority")
	}
}
