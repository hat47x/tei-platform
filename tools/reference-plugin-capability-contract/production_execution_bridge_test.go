package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/environment"
	"tei/internal/plugin"
	pluginruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

type productionExecutionFixture struct {
	Definition     plugin.Definition
	Mapping        LegacyCapabilityMapping
	CapabilityLock CapabilityResolutionLock
	RuntimeBinding RuntimeCapabilityBinding
	Profile        CapabilityExecutionProfile
	Authority      CapabilityExecutionAuthorityBinding
	Plan           pluginruntime.EnvironmentPlan
}

func newProductionExecutionFixture(
	t *testing.T,
	runtimeKind plugin.RuntimeKind,
	sandbox plugin.SandboxPolicy,
	isolation CapabilityIsolationPolicy,
	permissions []permission.Spec,
) productionExecutionFixture {
	t.Helper()
	contract := legacyContractQueryContract()
	capability := plugin.Capability{
		Kind:        plugin.CapabilityContractQueryExecutor,
		ID:          core.MustCanonicalID("database.query.reference"),
		Permissions: append([]permission.Spec(nil), permissions...),
	}
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.database.reference"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{
			Kind:    runtimeKind,
			Sandbox: sandbox,
		},
		Artifact: plugin.Artifact{
			Digest: "sha256:" + strings.Repeat("a", 64),
		},
		Provides: []plugin.Capability{capability},
	}
	if runtimeKind == plugin.RuntimeProcess {
		definition.Runtime.Command = "reference-plugin"
	}

	mapping := LegacyCapabilityMapping{
		Kind:         capability.Kind,
		CapabilityID: capability.ID,
		Scope:        ScopeApplicationRuntime,
		Contract:     contract,
	}
	provider, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	provider.RealizationTarget = "reference-target"
	profile, err := newCapabilityExecutionProfile(
		ScopeApplicationRuntime,
		provider.RealizationTarget,
		isolation,
		permissions,
	)
	if err != nil {
		t.Fatal(err)
	}
	provider, err = bindProviderExecutionProfile(provider, profile)
	if err != nil {
		t.Fatal(err)
	}
	input := ResolveInput{
		Scope:             ScopeApplicationRuntime,
		RealizationTarget: provider.RealizationTarget,
		Requirement:       Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:         []Contract{contract},
		Providers:         []Provider{provider},
		Evidence: []ConformanceEvidence{
			evidenceFor(provider, contract, "production-execution-conformance"),
		},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	capabilityLock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := verifyCapabilityExecutionAuthority(
		capabilityLock,
		profile,
		CapabilityExecutionGrant{
			CapabilityLockFingerprint:    capabilityLock.LockFingerprint,
			ExecutionProfileFingerprint:  profile.ProfileFingerprint,
			Permissions:                  append([]permission.Spec(nil), profile.Permissions...),
			AuthorityEvidenceFingerprint: fingerprint("production-execution-authority"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	environmentID := core.MustCanonicalID("reference")
	runtimeLock := plugin.Lock{
		FormatVersion: plugin.LockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: environmentID,
			Plugins: []plugin.LockedPlugin{{
				ID:             definition.ID,
				Version:        definition.Version,
				Fingerprint:    plugin.DefinitionFingerprint(definition),
				ArtifactDigest: definition.Artifact.Digest,
			}},
		}},
	}
	runtimeBinding, err := buildRuntimeCapabilityBinding(
		capabilityLock,
		runtimeLock,
		environmentID,
	)
	if err != nil {
		t.Fatal(err)
	}

	boundary := pluginruntime.BoundaryInProcess
	trust := pluginruntime.TrustHostTrusted
	if runtimeKind == plugin.RuntimeProcess {
		boundary = pluginruntime.BoundaryProcess
		trust = pluginruntime.TrustExternalProcess
	}
	plan := pluginruntime.EnvironmentPlan{
		Environment: environmentID,
		Plugins: []pluginruntime.PluginRuntime{{
			Plugin:                plugin.Identity{ID: definition.ID, Version: definition.Version},
			DescriptorFingerprint: plugin.DefinitionFingerprint(definition),
			Boundary:              boundary,
			Trust:                 trust,
			Command:               definition.Runtime.Command,
			Args:                  append([]string(nil), definition.Runtime.Args...),
			SandboxPolicy:         sandbox,
			Artifact:              definition.Artifact,
		}},
		Capabilities: []pluginruntime.CapabilityRuntime{{
			Plugin:      plugin.Identity{ID: definition.ID, Version: definition.Version},
			Capability:  capability,
			Boundary:    boundary,
			Permissions: permission.Normalize(permissions),
		}},
	}

	return productionExecutionFixture{
		Definition:      definition,
		Mapping:         mapping,
		CapabilityLock: capabilityLock,
		RuntimeBinding: runtimeBinding,
		Profile:         profile,
		Authority:       authority,
		Plan:            plan,
	}
}

func TestProductionExecutionBridgeMatchesExistingHostTrustedAuthority(t *testing.T) {
	permissions := []permission.Spec{{
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}}
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeBuiltin,
		plugin.SandboxNone,
		CapabilityIsolationHostTrusted,
		permissions,
	)
	binding, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Authority != productionExecutionBridgeAuthority ||
		binding.Environment != fx.Plan.Environment ||
		binding.PluginID != string(fx.Definition.ID) ||
		!validSHA256Fingerprint(binding.BindingFingerprint) {
		t.Fatalf("production execution bridge=%+v", binding)
	}
	if err := verifyLegacyProductionExecutionBridge(
		binding,
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	); err != nil {
		t.Fatal(err)
	}
}

func TestProductionExecutionBridgeMatchesRequiredProcessSandbox(t *testing.T) {
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeProcess,
		plugin.SandboxRequired,
		CapabilityIsolationHostEnforcedSandbox,
		nil,
	)
	if _, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	); err != nil {
		t.Fatal(err)
	}
}

func TestProductionExecutionBridgeRejectsSandboxPreferredForEnforcedProfile(t *testing.T) {
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeProcess,
		plugin.SandboxRequired,
		CapabilityIsolationHostEnforcedSandbox,
		nil,
	)
	fx.Plan.Plugins[0].SandboxPolicy = plugin.SandboxPreferred
	if _, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	); err == nil || !strings.Contains(err.Error(), "sandbox") {
		t.Fatalf("preferred sandbox satisfied enforced profile: %v", err)
	}
}

func TestProductionExecutionBridgeRejectsAmbientExtraPermission(t *testing.T) {
	permissions := []permission.Spec{{
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}}
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeBuiltin,
		plugin.SandboxNone,
		CapabilityIsolationHostTrusted,
		permissions,
	)
	fx.Plan.Capabilities[0].Permissions = append(
		fx.Plan.Capabilities[0].Permissions,
		permission.Spec{
			Kind:   permission.KindNetwork,
			Access: permission.AccessConnect,
			Scope:  "unexpected-network",
		},
	)
	if _, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	); err == nil || !strings.Contains(err.Error(), "Permission grant") {
		t.Fatalf("ambient production Permission was hidden: %v", err)
	}
}

func TestProductionExecutionBridgeRejectsLegacyCapabilityDeclarationDrift(t *testing.T) {
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeBuiltin,
		plugin.SandboxNone,
		CapabilityIsolationHostTrusted,
		nil,
	)
	fx.Plan.Capabilities[0].Capability.Features = []environment.Feature{"drifted"}
	if _, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	); err == nil || !strings.Contains(err.Error(), "source declaration") {
		t.Fatalf("legacy Capability declaration drift was accepted: %v", err)
	}
}

func TestProductionExecutionBridgeIgnoresUnrelatedPluginRuntime(t *testing.T) {
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeBuiltin,
		plugin.SandboxNone,
		CapabilityIsolationHostTrusted,
		nil,
	)
	base, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	unrelated := pluginruntime.PluginRuntime{
		Plugin: plugin.Identity{
			ID:      core.MustCanonicalID("tei.unrelated"),
			Version: "1.0.0",
		},
		DescriptorFingerprint: strings.Repeat("b", 64),
		Boundary:              pluginruntime.BoundaryInProcess,
		Trust:                 pluginruntime.TrustHostTrusted,
	}
	fx.Plan.Plugins = append(fx.Plan.Plugins, unrelated)
	after, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	if base.BindingFingerprint != after.BindingFingerprint {
		t.Fatal("unrelated Plugin runtime made production execution bridge stale")
	}
}

func TestProductionExecutionBridgeChangesWhenSelectedResourcePolicyChanges(t *testing.T) {
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeBuiltin,
		plugin.SandboxNone,
		CapabilityIsolationHostTrusted,
		nil,
	)
	base, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	fx.Plan.Capabilities[0].LeasePolicy = structToLeasePolicyForTest()
	after, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	if base.BindingFingerprint == after.BindingFingerprint {
		t.Fatal("selected production resource/lease policy change was invisible")
	}
	if err := verifyLegacyProductionExecutionBridge(
		base,
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	); err == nil || !strings.Contains(err.Error(), "current production runtime authority") {
		t.Fatalf("stale production execution bridge survived resource/lease drift: %v", err)
	}
}

func structToLeasePolicyForTest() resource.LeasePolicy {
	return resource.LeasePolicy{
		MaxLifetime: "10m",
		RenewBefore: "2m",
		RevokeTimeout: "5s",
		RenewalFailure: resource.RenewalFailClosed,
	}
}


type referenceArtifactVerifier struct{}

func (referenceArtifactVerifier) Verify(
	ctx context.Context,
	identity plugin.Identity,
	artifact plugin.Artifact,
) (pluginruntime.ArtifactVerification, error) {
	if err := ctx.Err(); err != nil {
		return pluginruntime.ArtifactVerification{}, err
	}
	return pluginruntime.ArtifactVerification{
		VerifierID:        "reference-artifact-verifier",
		DigestVerified:    artifact.Digest != "",
		SignatureVerified: artifact.SignatureRef == "",
		ProvenanceChecked: artifact.Provenance == "",
	}, nil
}

type recordingNetworkEnforcer struct {
	Calls  int
	Policy pluginruntime.NetworkPolicy
}

func (e *recordingNetworkEnforcer) ApplyNetworkPolicy(
	ctx context.Context,
	policy pluginruntime.NetworkPolicy,
) (pluginruntime.Cleanup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.Calls++
	e.Policy = policy
	return func(context.Context) error { return nil }, nil
}

type recordingFilesystemEnforcer struct {
	Calls  int
	Policy pluginruntime.FilesystemPolicy
}

func (e *recordingFilesystemEnforcer) ApplyFilesystemPolicy(
	ctx context.Context,
	policy pluginruntime.FilesystemPolicy,
) (pluginruntime.Cleanup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.Calls++
	e.Policy = policy
	return func(context.Context) error { return nil }, nil
}

type recordingSandboxedRunner struct {
	Calls int
}

func (r *recordingSandboxedRunner) InvokeSandboxed(
	ctx context.Context,
	command pluginruntime.ProcessCommand,
	req pluginruntime.Request,
) (pluginruntime.Response, error) {
	if err := ctx.Err(); err != nil {
		return pluginruntime.Response{}, err
	}
	r.Calls++
	if command.Executable == "" {
		return pluginruntime.Response{}, fmt.Errorf("sandbox runner received empty executable")
	}
	return pluginruntime.Response{
		Status: pluginruntime.ResponseOK,
	}, nil
}

func TestGenericCapabilityBridgeInvokesExistingHostSandboxPath(t *testing.T) {
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeProcess,
		plugin.SandboxRequired,
		CapabilityIsolationHostEnforcedSandbox,
		nil,
	)
	bridge, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}

	network := &recordingNetworkEnforcer{}
	filesystem := &recordingFilesystemEnforcer{}
	runner := &recordingSandboxedRunner{}
	registry := pluginruntime.Registry{
		Verifier: referenceArtifactVerifier{},
		Sandboxes: pluginruntime.HostSandboxLauncher{
			Network:    network,
			Filesystem: filesystem,
			Runner:     runner,
		},
	}
	invocation, err := invokeLegacyCapabilityWithProductionBridge(
		context.Background(),
		bridge,
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		registry,
		fx.Plan,
		fx.Mapping,
		"query",
		"request-1",
		map[string]any{"id": "customer-1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.Response.Status != pluginruntime.ResponseOK ||
		!invocation.Provenance.Sandboxed ||
		invocation.Provenance.Verification == nil ||
		!invocation.Provenance.Verification.DigestVerified {
		t.Fatalf("production invocation=%+v", invocation)
	}
	if network.Calls != 1 || !network.Policy.DenyByDefault {
		t.Fatalf("network policy was not enforced: %+v", network)
	}
	if filesystem.Calls != 1 || !filesystem.Policy.DenyByDefault {
		t.Fatalf("filesystem policy was not enforced: %+v", filesystem)
	}
	if runner.Calls != 1 {
		t.Fatalf("sandboxed runner calls=%d want=1", runner.Calls)
	}
}

func TestGenericCapabilityBridgeDoesNotBypassMissingHostSandboxEnforcement(t *testing.T) {
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeProcess,
		plugin.SandboxRequired,
		CapabilityIsolationHostEnforcedSandbox,
		nil,
	)
	bridge, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}

	registry := pluginruntime.Registry{
		Verifier: referenceArtifactVerifier{},
		Sandboxes: pluginruntime.HostSandboxLauncher{
			Runner: &recordingSandboxedRunner{},
		},
	}
	if _, err := invokeLegacyCapabilityWithProductionBridge(
		context.Background(),
		bridge,
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		registry,
		fx.Plan,
		fx.Mapping,
		"query",
		"request-2",
		nil,
	); err == nil || !strings.Contains(err.Error(), "deny-by-default network policy") {
		t.Fatalf("generic bridge bypassed missing host enforcement: %v", err)
	}
}


type recordingResourceBroker struct {
	AcquireCalls int
	ReleaseCalls int
	LastAcquire  resource.AcquireRequest
}

func (b *recordingResourceBroker) Acquire(
	ctx context.Context,
	req resource.AcquireRequest,
) (resource.Lease, error) {
	if err := ctx.Err(); err != nil {
		return resource.Lease{}, err
	}
	b.AcquireCalls++
	b.LastAcquire = req
	if len(req.Permissions) != 1 || len(req.Bindings) != 1 {
		return resource.Lease{}, fmt.Errorf(
			"reference broker expects one permission/binding",
		)
	}
	permissionSpec := req.Permissions[0]
	binding := req.Bindings[0]
	now := time.Now().UTC()
	return resource.Lease{
		ID:                       "reference-lease",
		Revision:                 1,
		IssuedAt:                 now.Format(time.RFC3339Nano),
		ExpiresAt:                now.Add(20 * time.Minute).Format(time.RFC3339Nano),
		RuntimePolicyFingerprint: req.RuntimePolicyFingerprint,
		Resources: []resource.Resolved{{
			Kind:              permissionSpec.Kind,
			Access:            permissionSpec.Access,
			Scope:             permissionSpec.Scope,
			ResourceID:        binding.ResourceID,
			MountSourceHandle: "workspace-source-handle",
			MountPath:         "/workspace/source",
		}},
	}, nil
}

func (b *recordingResourceBroker) Release(
	ctx context.Context,
	lease resource.Lease,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease.ID != "reference-lease" {
		return fmt.Errorf("unexpected lease %q", lease.ID)
	}
	b.ReleaseCalls++
	return nil
}

func TestGenericCapabilityBridgeUsesExistingResourceBrokerAndSandboxMount(t *testing.T) {
	permissions := []permission.Spec{{
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}}
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeProcess,
		plugin.SandboxRequired,
		CapabilityIsolationHostEnforcedSandbox,
		permissions,
	)
	fx.Plan.Capabilities[0].ResourceBindings = []resource.Binding{{
		Kind:       permission.KindFilesystem,
		Scope:      "workspace-source",
		ResourceID: core.MustCanonicalID("resource.workspace-source"),
	}}
	fx.Plan.Capabilities[0].LeasePolicy = resource.DefaultLeasePolicy()
	fx.Plan.RuntimePolicyFingerprint = pluginruntime.PolicyFingerprint(fx.Plan)

	bridge, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}

	network := &recordingNetworkEnforcer{}
	filesystem := &recordingFilesystemEnforcer{}
	runner := &recordingSandboxedRunner{}
	broker := &recordingResourceBroker{}
	registry := pluginruntime.Registry{
		Verifier:  referenceArtifactVerifier{},
		Resources: broker,
		Sandboxes: pluginruntime.HostSandboxLauncher{
			Network:    network,
			Filesystem: filesystem,
			Runner:     runner,
		},
	}
	invocation, err := invokeLegacyCapabilityWithProductionBridge(
		context.Background(),
		bridge,
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		registry,
		fx.Plan,
		fx.Mapping,
		"query",
		"request-resource-1",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if broker.AcquireCalls != 1 || broker.ReleaseCalls != 1 {
		t.Fatalf(
			"resource broker calls acquire=%d release=%d",
			broker.AcquireCalls,
			broker.ReleaseCalls,
		)
	}
	if broker.LastAcquire.CapabilityKind != string(fx.Mapping.Kind) ||
		broker.LastAcquire.CapabilityID != fx.Mapping.CapabilityID ||
		broker.LastAcquire.RuntimePolicyFingerprint !=
			fx.Plan.RuntimePolicyFingerprint {
		t.Fatalf("resource acquire request=%+v", broker.LastAcquire)
	}
	if !invocation.Provenance.ResourceLeaseReleased ||
		invocation.Provenance.ResourceLeaseID != "reference-lease" ||
		len(invocation.Provenance.Resources) != 1 {
		t.Fatalf("resource invocation provenance=%+v", invocation.Provenance)
	}
	if filesystem.Calls != 1 ||
		len(filesystem.Policy.Mounts) != 1 ||
		filesystem.Policy.Mounts[0].Target != "/workspace/source" ||
		filesystem.Policy.Mounts[0].Access != permission.AccessRead {
		t.Fatalf("filesystem sandbox policy=%+v", filesystem.Policy)
	}
	if len(network.Policy.Allowlist) != 0 ||
		!network.Policy.DenyByDefault {
		t.Fatalf("network sandbox policy=%+v", network.Policy)
	}
	if runner.Calls != 1 {
		t.Fatalf("sandboxed runner calls=%d want=1", runner.Calls)
	}
}

func TestGenericCapabilityBridgeDoesNotBypassMissingResourceBroker(t *testing.T) {
	permissions := []permission.Spec{{
		Kind:   permission.KindFilesystem,
		Access: permission.AccessRead,
		Scope:  "workspace-source",
	}}
	fx := newProductionExecutionFixture(
		t,
		plugin.RuntimeProcess,
		plugin.SandboxRequired,
		CapabilityIsolationHostEnforcedSandbox,
		permissions,
	)
	fx.Plan.Capabilities[0].ResourceBindings = []resource.Binding{{
		Kind:       permission.KindFilesystem,
		Scope:      "workspace-source",
		ResourceID: core.MustCanonicalID("resource.workspace-source"),
	}}
	fx.Plan.Capabilities[0].LeasePolicy = resource.DefaultLeasePolicy()
	fx.Plan.RuntimePolicyFingerprint = pluginruntime.PolicyFingerprint(fx.Plan)

	bridge, err := buildLegacyProductionExecutionBridge(
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		fx.Plan,
		fx.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	registry := pluginruntime.Registry{
		Verifier: referenceArtifactVerifier{},
		Sandboxes: pluginruntime.HostSandboxLauncher{
			Network:    &recordingNetworkEnforcer{},
			Filesystem: &recordingFilesystemEnforcer{},
			Runner:     &recordingSandboxedRunner{},
		},
	}
	if _, err := invokeLegacyCapabilityWithProductionBridge(
		context.Background(),
		bridge,
		fx.CapabilityLock,
		fx.RuntimeBinding,
		fx.Profile,
		fx.Authority,
		registry,
		fx.Plan,
		fx.Mapping,
		"query",
		"request-resource-2",
		nil,
	); err == nil || !strings.Contains(err.Error(), "host has no resource broker") {
		t.Fatalf("generic bridge bypassed missing Resource Broker: %v", err)
	}
}
