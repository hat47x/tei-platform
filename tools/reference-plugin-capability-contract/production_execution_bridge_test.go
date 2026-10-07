package main

import (
	"context"
	"strings"
	"testing"

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
