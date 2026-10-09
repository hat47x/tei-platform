package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

const (
	runtimeCapabilityRefSchema = "tei.reference.runtime-capability-ref/v0"
	runtimeEnvironmentPlanSchema = "tei.reference.runtime-capability-environment-plan/v0"
	genericResourceTransportKind = "generic-runtime-ref"
)

type RuntimeIsolationPolicy string

const (
	RuntimeIsolationHostTrusted RuntimeIsolationPolicy = "host-trusted"
	RuntimeIsolationHostEnforcedSandbox RuntimeIsolationPolicy = "host-enforced-sandbox"
)

func (p RuntimeIsolationPolicy) valid() bool {
	return p == RuntimeIsolationHostTrusted || p == RuntimeIsolationHostEnforcedSandbox
}

type ExecutionProfile struct {
	RealizationTarget string `json:"realizationTarget"`
	IsolationPolicy RuntimeIsolationPolicy `json:"isolationPolicy"`
	Permissions []permission.Spec `json:"permissions,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

func newExecutionProfile(
	target string,
	isolation RuntimeIsolationPolicy,
	permissions []permission.Spec,
) (ExecutionProfile, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return ExecutionProfile{}, fmt.Errorf("execution profile requires realization Target")
	}
	if !isolation.valid() {
		return ExecutionProfile{}, fmt.Errorf("execution profile isolation policy %q is invalid", isolation)
	}
	normalized, err := normalizePermissionSet(permissions)
	if err != nil {
		return ExecutionProfile{}, err
	}
	profile := ExecutionProfile{
		RealizationTarget: target,
		IsolationPolicy: isolation,
		Permissions: normalized,
	}
	profile.Fingerprint = executionProfileFingerprint(profile)
	return profile, nil
}

func validateExecutionProfile(profile ExecutionProfile) error {
	if strings.TrimSpace(profile.RealizationTarget) == "" ||
		profile.RealizationTarget != strings.TrimSpace(profile.RealizationTarget) {
		return fmt.Errorf("execution profile realization Target is invalid")
	}
	if !profile.IsolationPolicy.valid() {
		return fmt.Errorf("execution profile isolation policy %q is invalid", profile.IsolationPolicy)
	}
	normalized, err := normalizePermissionSet(profile.Permissions)
	if err != nil {
		return err
	}
	if !samePermissionSet(normalized, profile.Permissions) {
		return fmt.Errorf("execution profile permissions are not normalized")
	}
	if !validSHA256Fingerprint(profile.Fingerprint) ||
		executionProfileFingerprint(profile) != profile.Fingerprint {
		return fmt.Errorf("execution profile fingerprint is invalid or stale")
	}
	return nil
}

func executionProfileFingerprint(profile ExecutionProfile) string {
	profile.Fingerprint = ""
	return fingerprint(profile)
}

type AcceptedRuntimeCandidate struct {
	ContractID string `json:"contractId"`
	ContractVersion string `json:"contractVersion"`
	ContractFingerprint string `json:"contractFingerprint"`
	Plugin plugin.Identity `json:"plugin"`
	ImplementationID string `json:"implementationId"`
	ArtifactDigest string `json:"artifactDigest"`
	RealizationTarget string `json:"realizationTarget"`
	ExecutionProfileFingerprint string `json:"executionProfileFingerprint"`
	CapabilityResolutionLockFingerprint string `json:"capabilityResolutionLockFingerprint"`
}

type RuntimeCapabilityRef struct {
	Schema string `json:"schema"`
	ContractID string `json:"contractId"`
	ContractVersion string `json:"contractVersion"`
	ContractFingerprint string `json:"contractFingerprint"`
	PluginID core.CanonicalID `json:"pluginId"`
	PluginVersion string `json:"pluginVersion"`
	ImplementationID string `json:"implementationId"`
	ArtifactDigest string `json:"artifactDigest"`
	RealizationTarget string `json:"realizationTarget"`
	ExecutionProfileFingerprint string `json:"executionProfileFingerprint"`
	CapabilityResolutionLockFingerprint string `json:"capabilityResolutionLockFingerprint"`
	Fingerprint string `json:"fingerprint"`
}

func deriveRuntimeCapabilityRef(
	candidate AcceptedRuntimeCandidate,
	profile ExecutionProfile,
) (RuntimeCapabilityRef, error) {
	if err := validateExecutionProfile(profile); err != nil {
		return RuntimeCapabilityRef{}, err
	}
	if strings.TrimSpace(candidate.ContractID) == "" ||
		candidate.ContractID != strings.TrimSpace(candidate.ContractID) {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate Contract ID is invalid")
	}
	if _, err := plugin.ParseVersion(candidate.ContractVersion); err != nil {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate Contract version: %w", err)
	}
	if !validSHA256Fingerprint(candidate.ContractFingerprint) {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate Contract fingerprint is invalid")
	}
	if candidate.Plugin.ID == "" {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate Provider Plugin ID is empty")
	}
	if _, err := plugin.ParseVersion(candidate.Plugin.Version); err != nil {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate Provider Plugin version: %w", err)
	}
	if strings.TrimSpace(candidate.ImplementationID) == "" ||
		candidate.ImplementationID != strings.TrimSpace(candidate.ImplementationID) {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate implementation ID is invalid")
	}
	if candidate.ArtifactDigest == "" {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate requires exact artifact digest")
	}
	if err := (plugin.Artifact{Digest: candidate.ArtifactDigest}).Validate(); err != nil {
		return RuntimeCapabilityRef{}, err
	}
	if candidate.RealizationTarget != profile.RealizationTarget {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate realization Target differs from execution profile")
	}
	if candidate.ExecutionProfileFingerprint != profile.Fingerprint {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate execution profile fingerprint differs from current profile")
	}
	if !validSHA256Fingerprint(candidate.CapabilityResolutionLockFingerprint) {
		return RuntimeCapabilityRef{}, fmt.Errorf("runtime candidate Capability Resolution Lock fingerprint is invalid")
	}

	ref := RuntimeCapabilityRef{
		Schema: runtimeCapabilityRefSchema,
		ContractID: candidate.ContractID,
		ContractVersion: candidate.ContractVersion,
		ContractFingerprint: candidate.ContractFingerprint,
		PluginID: candidate.Plugin.ID,
		PluginVersion: candidate.Plugin.Version,
		ImplementationID: candidate.ImplementationID,
		ArtifactDigest: candidate.ArtifactDigest,
		RealizationTarget: candidate.RealizationTarget,
		ExecutionProfileFingerprint: candidate.ExecutionProfileFingerprint,
		CapabilityResolutionLockFingerprint: candidate.CapabilityResolutionLockFingerprint,
	}
	ref.Fingerprint = runtimeCapabilityRefFingerprint(ref)
	return ref, nil
}

func validateRuntimeCapabilityRef(ref RuntimeCapabilityRef) error {
	if ref.Schema != runtimeCapabilityRefSchema {
		return fmt.Errorf("runtime Capability Ref schema %q is not supported", ref.Schema)
	}
	if strings.TrimSpace(ref.ContractID) == "" ||
		ref.ContractID != strings.TrimSpace(ref.ContractID) ||
		ref.PluginID == "" ||
		strings.TrimSpace(ref.ImplementationID) == "" ||
		ref.ImplementationID != strings.TrimSpace(ref.ImplementationID) ||
		strings.TrimSpace(ref.RealizationTarget) == "" ||
		ref.RealizationTarget != strings.TrimSpace(ref.RealizationTarget) {
		return fmt.Errorf("runtime Capability Ref identity is incomplete or non-canonical")
	}
	if _, err := plugin.ParseVersion(ref.ContractVersion); err != nil {
		return fmt.Errorf("runtime Capability Ref Contract version: %w", err)
	}
	if _, err := plugin.ParseVersion(ref.PluginVersion); err != nil {
		return fmt.Errorf("runtime Capability Ref Plugin version: %w", err)
	}
	if err := (plugin.Artifact{Digest: ref.ArtifactDigest}).Validate(); err != nil || ref.ArtifactDigest == "" {
		return fmt.Errorf("runtime Capability Ref artifact digest is invalid")
	}
	for _, value := range []string{
		ref.ContractFingerprint,
		ref.ExecutionProfileFingerprint,
		ref.CapabilityResolutionLockFingerprint,
		ref.Fingerprint,
	} {
		if !validSHA256Fingerprint(value) {
			return fmt.Errorf("runtime Capability Ref fingerprint is invalid")
		}
	}
	if runtimeCapabilityRefFingerprint(ref) != ref.Fingerprint {
		return fmt.Errorf("runtime Capability Ref fingerprint is stale")
	}
	return nil
}

func runtimeCapabilityRefFingerprint(ref RuntimeCapabilityRef) string {
	ref.Fingerprint = ""
	return fingerprint(ref)
}

type RuntimeExecutionGrant struct {
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	ExecutionProfileFingerprint string `json:"executionProfileFingerprint"`
	AuthorityEvidenceFingerprint string `json:"authorityEvidenceFingerprint"`
}

type RuntimePermissionGrant struct {
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	Permission permission.Spec `json:"permission"`
}

type RuntimeResourceBinding struct {
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	Binding resource.Binding `json:"binding"`
}

type RuntimeLeasePolicy struct {
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	Policy resource.LeasePolicy `json:"policy"`
}

type LegacyAuthorityClaim struct {
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	CapabilityKind string `json:"capabilityKind"`
	CapabilityID core.CanonicalID `json:"capabilityId"`
}

type RuntimeEnvironmentAuthority struct {
	Environment core.CanonicalID `json:"environment"`
	ExecutionGrants []RuntimeExecutionGrant `json:"executionGrants,omitempty"`
	PermissionGrants []RuntimePermissionGrant `json:"permissionGrants,omitempty"`
	ResourceBindings []RuntimeResourceBinding `json:"resourceBindings,omitempty"`
	LeasePolicies []RuntimeLeasePolicy `json:"leasePolicies,omitempty"`
	LegacyClaims []LegacyAuthorityClaim `json:"legacyClaims,omitempty"`
}

type RuntimeEnvironmentPlan struct {
	Schema string `json:"schema"`
	Environment core.CanonicalID `json:"environment"`
	RuntimeCapability RuntimeCapabilityRef `json:"runtimeCapability"`
	ExecutionProfileFingerprint string `json:"executionProfileFingerprint"`
	ExecutionAuthorityEvidenceFingerprint string `json:"executionAuthorityEvidenceFingerprint"`
	IsolationPolicy RuntimeIsolationPolicy `json:"isolationPolicy"`
	Permissions []permission.Spec `json:"permissions,omitempty"`
	ResourceBindings []resource.Binding `json:"resourceBindings,omitempty"`
	LeasePolicy resource.LeasePolicy `json:"leasePolicy,omitempty"`
	RuntimePolicyFingerprint string `json:"runtimePolicyFingerprint"`
}

func buildRuntimeEnvironmentPlan(
	ref RuntimeCapabilityRef,
	profile ExecutionProfile,
	authority RuntimeEnvironmentAuthority,
) (RuntimeEnvironmentPlan, error) {
	if err := validateRuntimeCapabilityRef(ref); err != nil {
		return RuntimeEnvironmentPlan{}, err
	}
	if err := validateExecutionProfile(profile); err != nil {
		return RuntimeEnvironmentPlan{}, err
	}
	if authority.Environment == "" {
		return RuntimeEnvironmentPlan{}, fmt.Errorf("runtime Environment Authority requires Environment")
	}
	if ref.ExecutionProfileFingerprint != profile.Fingerprint ||
		ref.RealizationTarget != profile.RealizationTarget {
		return RuntimeEnvironmentPlan{}, fmt.Errorf("runtime Capability Ref differs from execution profile")
	}
	var executionGrants []RuntimeExecutionGrant
	for _, grant := range authority.ExecutionGrants {
		if grant.RuntimeCapabilityFingerprint == ref.Fingerprint {
			executionGrants = append(executionGrants, grant)
		}
	}
	if len(executionGrants) != 1 {
		return RuntimeEnvironmentPlan{}, fmt.Errorf(
			"runtime Capability Ref requires exactly one Environment Execution Grant; found %d",
			len(executionGrants),
		)
	}
	executionGrant := executionGrants[0]
	if executionGrant.ExecutionProfileFingerprint != profile.Fingerprint {
		return RuntimeEnvironmentPlan{}, fmt.Errorf(
			"Environment Execution Grant targets a different execution profile",
		)
	}
	if !validSHA256Fingerprint(executionGrant.AuthorityEvidenceFingerprint) {
		return RuntimeEnvironmentPlan{}, fmt.Errorf(
			"Environment Execution Grant Authority Evidence fingerprint is invalid",
		)
	}
	for _, legacy := range authority.LegacyClaims {
		if legacy.RuntimeCapabilityFingerprint == ref.Fingerprint {
			return RuntimeEnvironmentPlan{}, fmt.Errorf(
				"runtime Capability Ref %s has both generic and legacy Authority claim %s/%s",
				ref.Fingerprint, legacy.CapabilityKind, legacy.CapabilityID,
			)
		}
	}

	var grants []permission.Spec
	for _, grant := range authority.PermissionGrants {
		if grant.RuntimeCapabilityFingerprint != ref.Fingerprint {
			continue
		}
		if err := grant.Permission.Validate(); err != nil {
			return RuntimeEnvironmentPlan{}, err
		}
		grants = append(grants, grant.Permission)
	}
	normalizedGrants, err := normalizePermissionSet(grants)
	if err != nil {
		return RuntimeEnvironmentPlan{}, err
	}
	if !samePermissionSet(normalizedGrants, profile.Permissions) {
		return RuntimeEnvironmentPlan{}, fmt.Errorf("Environment Permission Grant differs from execution profile")
	}

	var declaredBindings []resource.Binding
	for _, item := range authority.ResourceBindings {
		if item.RuntimeCapabilityFingerprint != ref.Fingerprint {
			continue
		}
		if err := item.Binding.Validate(); err != nil {
			return RuntimeEnvironmentPlan{}, err
		}
		declaredBindings = append(declaredBindings, item.Binding)
	}
	var matchedBindings []resource.Binding
	if len(normalizedGrants) > 0 {
		matchedBindings, err = resource.MatchBindings(normalizedGrants, declaredBindings)
		if err != nil {
			return RuntimeEnvironmentPlan{}, err
		}
		if len(matchedBindings) != len(declaredBindings) {
			return RuntimeEnvironmentPlan{}, fmt.Errorf("Environment declares unused Resource Binding for runtime Capability Ref")
		}
	} else if len(declaredBindings) > 0 {
		return RuntimeEnvironmentPlan{}, fmt.Errorf("zero-Permission runtime Capability Ref may not carry Resource Binding")
	}

	var policies []resource.LeasePolicy
	for _, item := range authority.LeasePolicies {
		if item.RuntimeCapabilityFingerprint == ref.Fingerprint {
			policies = append(policies, item.Policy)
		}
	}
	if len(policies) > 1 {
		return RuntimeEnvironmentPlan{}, fmt.Errorf("runtime Capability Ref has multiple Lease Policies")
	}
	leasePolicy := resource.LeasePolicy{}
	if len(normalizedGrants) > 0 {
		leasePolicy = resource.DefaultLeasePolicy()
		if len(policies) == 1 {
			leasePolicy = policies[0].WithDefaults()
		}
		if err := leasePolicy.Validate(); err != nil {
			return RuntimeEnvironmentPlan{}, err
		}
	} else if len(policies) > 0 {
		return RuntimeEnvironmentPlan{}, fmt.Errorf("zero-Permission runtime Capability Ref may not carry Lease Policy")
	}

	plan := RuntimeEnvironmentPlan{
		Schema: runtimeEnvironmentPlanSchema,
		Environment: authority.Environment,
		RuntimeCapability: ref,
		ExecutionProfileFingerprint: profile.Fingerprint,
		ExecutionAuthorityEvidenceFingerprint: executionGrant.AuthorityEvidenceFingerprint,
		IsolationPolicy: profile.IsolationPolicy,
		Permissions: normalizedGrants,
		ResourceBindings: resource.NormalizeBindings(matchedBindings),
		LeasePolicy: leasePolicy,
	}
	plan.RuntimePolicyFingerprint = runtimeEnvironmentPlanFingerprint(plan)
	return plan, nil
}

func runtimeEnvironmentPlanFingerprint(plan RuntimeEnvironmentPlan) string {
	plan.RuntimePolicyFingerprint = ""
	return fingerprint(plan)
}

func validateRuntimeEnvironmentPlan(plan RuntimeEnvironmentPlan) error {
	if plan.Schema != runtimeEnvironmentPlanSchema || plan.Environment == "" {
		return fmt.Errorf("runtime Environment Plan schema / Environment is invalid")
	}
	if err := validateRuntimeCapabilityRef(plan.RuntimeCapability); err != nil {
		return err
	}
	if plan.ExecutionProfileFingerprint != plan.RuntimeCapability.ExecutionProfileFingerprint {
		return fmt.Errorf("runtime Environment Plan execution profile differs from runtime Capability Ref")
	}
	profileSnapshot := ExecutionProfile{
		RealizationTarget: plan.RuntimeCapability.RealizationTarget,
		IsolationPolicy: plan.IsolationPolicy,
		Permissions: append([]permission.Spec(nil), plan.Permissions...),
		Fingerprint: plan.ExecutionProfileFingerprint,
	}
	if err := validateExecutionProfile(profileSnapshot); err != nil {
		return fmt.Errorf("runtime Environment Plan Execution Profile snapshot: %w", err)
	}
	if !validSHA256Fingerprint(plan.ExecutionAuthorityEvidenceFingerprint) {
		return fmt.Errorf("runtime Environment Plan Execution Authority Evidence fingerprint is invalid")
	}
	if !plan.IsolationPolicy.valid() {
		return fmt.Errorf("runtime Environment Plan isolation policy %q is invalid", plan.IsolationPolicy)
	}
	normalizedPermissions, err := normalizePermissionSet(plan.Permissions)
	if err != nil {
		return err
	}
	if !samePermissionSet(normalizedPermissions, plan.Permissions) {
		return fmt.Errorf("runtime Environment Plan permissions are not normalized")
	}
	normalizedBindings := resource.NormalizeBindings(plan.ResourceBindings)
	if len(normalizedBindings) != len(plan.ResourceBindings) {
		return fmt.Errorf("runtime Environment Plan Resource Bindings are invalid")
	}
	for i := range normalizedBindings {
		if err := normalizedBindings[i].Validate(); err != nil {
			return err
		}
		if normalizedBindings[i] != plan.ResourceBindings[i] {
			return fmt.Errorf("runtime Environment Plan Resource Bindings are not normalized")
		}
	}
	if len(plan.Permissions) == 0 {
		if len(plan.ResourceBindings) > 0 || plan.LeasePolicy != (resource.LeasePolicy{}) {
			return fmt.Errorf("zero-Permission runtime Environment Plan may not carry Resource Authority")
		}
	} else {
		matched, err := resource.MatchBindings(plan.Permissions, plan.ResourceBindings)
		if err != nil {
			return err
		}
		if len(matched) != len(plan.ResourceBindings) {
			return fmt.Errorf("runtime Environment Plan carries unused Resource Binding")
		}
		if err := plan.LeasePolicy.Validate(); err != nil {
			return err
		}
	}
	if !validSHA256Fingerprint(plan.RuntimePolicyFingerprint) ||
		runtimeEnvironmentPlanFingerprint(plan) != plan.RuntimePolicyFingerprint {
		return fmt.Errorf("runtime Environment Plan fingerprint is invalid or stale")
	}
	return nil
}

func verifyRuntimeEnvironmentPlan(
	plan RuntimeEnvironmentPlan,
	ref RuntimeCapabilityRef,
	profile ExecutionProfile,
	authority RuntimeEnvironmentAuthority,
) error {
	if err := validateRuntimeEnvironmentPlan(plan); err != nil {
		return err
	}
	current, err := buildRuntimeEnvironmentPlan(ref, profile, authority)
	if err != nil {
		return err
	}
	if plan.RuntimePolicyFingerprint != current.RuntimePolicyFingerprint {
		return fmt.Errorf(
			"runtime Environment Plan differs from current Environment Authority",
		)
	}
	return nil
}

func genericTransportCapabilityID(ref RuntimeCapabilityRef) (core.CanonicalID, error) {
	if err := validateRuntimeCapabilityRef(ref); err != nil {
		return "", err
	}
	hexPart := strings.TrimPrefix(ref.Fingerprint, "sha256:")
	return core.ParseCanonicalID("runtime_capability.r_" + hexPart)
}

func resourceAcquireRequest(plan RuntimeEnvironmentPlan) (resource.AcquireRequest, error) {
	if err := validateRuntimeEnvironmentPlan(plan); err != nil {
		return resource.AcquireRequest{}, err
	}
	capabilityID, err := genericTransportCapabilityID(plan.RuntimeCapability)
	if err != nil {
		return resource.AcquireRequest{}, err
	}
	return resource.AcquireRequest{
		Environment: plan.Environment,
		CapabilityKind: genericResourceTransportKind,
		CapabilityID: capabilityID,
		RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint,
		Permissions: append([]permission.Spec(nil), plan.Permissions...),
		Bindings: append([]resource.Binding(nil), plan.ResourceBindings...),
		LeasePolicy: plan.LeasePolicy,
	}, nil
}

type RuntimeProcessRealization struct {
	RuntimeCapabilityFingerprint string `json:"runtimeCapabilityFingerprint"`
	Plugin plugin.Identity `json:"plugin"`
	Artifact plugin.Artifact `json:"artifact"`
	Command pruntime.ProcessCommand `json:"command"`
}

type GenericInvocationProvenance struct {
	Environment core.CanonicalID `json:"environment"`
	RuntimeCapability RuntimeCapabilityRef `json:"runtimeCapability"`
	RuntimePolicyFingerprint string `json:"runtimePolicyFingerprint"`
	ExecutionAuthorityEvidenceFingerprint string `json:"executionAuthorityEvidenceFingerprint"`
	TransportCapabilityKind string `json:"transportCapabilityKind"`
	TransportCapabilityID core.CanonicalID `json:"transportCapabilityId"`
	Sandboxed bool `json:"sandboxed"`
	ResourceLeaseID string `json:"resourceLeaseId,omitempty"`
	ResourceLeaseReleased bool `json:"resourceLeaseReleased,omitempty"`
	Resources []resource.Record `json:"resources,omitempty"`
}

func invokeWithExistingHostBoundary(
	ctx context.Context,
	plan RuntimeEnvironmentPlan,
	realization RuntimeProcessRealization,
	broker resource.Broker,
	sandbox pruntime.SandboxLauncher,
	operation string,
	requestID string,
	payload any,
) (prov GenericInvocationProvenance, retErr error) {
	if err := validateRuntimeEnvironmentPlan(plan); err != nil {
		return prov, err
	}
	if sandbox == nil {
		return prov, fmt.Errorf("generic runtime invocation requires existing Host Sandbox")
	}
	if plan.IsolationPolicy != RuntimeIsolationHostEnforcedSandbox {
		return prov, fmt.Errorf(
			"Host Sandbox invocation requires host-enforced-sandbox isolation, got %q",
			plan.IsolationPolicy,
		)
	}
	if strings.TrimSpace(operation) == "" || strings.TrimSpace(requestID) == "" {
		return prov, fmt.Errorf("generic runtime invocation requires operation and request ID")
	}
	if plan.RuntimeCapability.Fingerprint != realization.RuntimeCapabilityFingerprint {
		return prov, fmt.Errorf("process realization belongs to a different runtime Capability Ref")
	}
	if realization.Plugin.ID != plan.RuntimeCapability.PluginID ||
		realization.Plugin.Version != plan.RuntimeCapability.PluginVersion {
		return prov, fmt.Errorf("process realization Plugin differs from runtime Capability Ref")
	}
	if realization.Artifact.Digest != plan.RuntimeCapability.ArtifactDigest {
		return prov, fmt.Errorf("process realization artifact differs from runtime Capability Ref")
	}
	if strings.TrimSpace(realization.Command.Executable) == "" {
		return prov, fmt.Errorf("process realization executable is empty")
	}
	capabilityID, err := genericTransportCapabilityID(plan.RuntimeCapability)
	if err != nil {
		return prov, err
	}
	prov = GenericInvocationProvenance{
		Environment: plan.Environment,
		RuntimeCapability: plan.RuntimeCapability,
		RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint,
		ExecutionAuthorityEvidenceFingerprint: plan.ExecutionAuthorityEvidenceFingerprint,
		TransportCapabilityKind: genericResourceTransportKind,
		TransportCapabilityID: capabilityID,
		Sandboxed: true,
	}

	var lease *resource.Lease
	if len(plan.Permissions) > 0 {
		if broker == nil {
			return prov, fmt.Errorf("runtime Capability Ref requires resources but host has no Resource Broker")
		}
		acquire, err := resourceAcquireRequest(plan)
		if err != nil {
			return prov, err
		}
		got, err := broker.Acquire(ctx, acquire)
		if err != nil {
			return prov, fmt.Errorf("Resource Broker acquire failed: %w", err)
		}
		if err := resource.ValidateLease(acquire, got); err != nil {
			_ = broker.Release(context.WithoutCancel(ctx), got)
			return prov, fmt.Errorf("Resource Broker returned invalid lease: %w", err)
		}
		lease = &got
		prov.ResourceLeaseID = got.ID
		prov.Resources = resource.Records(got.Resources)
		defer func() {
			if err := broker.Release(context.WithoutCancel(ctx), *lease); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("Resource Broker release failed: %w", err))
				return
			}
			prov.ResourceLeaseReleased = true
		}()
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return prov, err
	}
	req := pruntime.Request{
		ProtocolVersion: pruntime.ProtocolVersion,
		RequestID: requestID,
		Plugin: realization.Plugin,
		CapabilityKind: plugin.CapabilityKind(genericResourceTransportKind),
		CapabilityID: capabilityID,
		Permissions: append([]permission.Spec(nil), plan.Permissions...),
		ResourceLease: lease,
		Operation: operation,
		Payload: raw,
	}
	resp, err := sandbox.Invoke(ctx, pruntime.SandboxSpec{
		Command: realization.Command,
		Plugin: realization.Plugin,
		Artifact: realization.Artifact,
		Permissions: append([]permission.Spec(nil), plan.Permissions...),
		ResourceLease: lease,
		LeasePolicy: plan.LeasePolicy,
		Profile: pruntime.SandboxProfileForLease(lease),
	}, req)
	if err != nil {
		return prov, err
	}
	if resp.Status != pruntime.ResponseOK {
		return prov, fmt.Errorf("sandboxed runtime Capability returned status %q", resp.Status)
	}
	return prov, nil
}

func normalizePermissionSet(in []permission.Spec) ([]permission.Spec, error) {
	out := permission.Normalize(in)
	for i, value := range out {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		if i > 0 && out[i-1].Key() == value.Key() {
			return nil, fmt.Errorf("duplicate Permission %s", value.Key())
		}
	}
	return out, nil
}

func samePermissionSet(left, right []permission.Spec) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Key() != right[i].Key() {
			return false
		}
	}
	return true
}

func validSHA256Fingerprint(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return false
	}
	_, err := hex.DecodeString(hexPart)
	return err == nil
}

func fingerprint(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func main() {}
