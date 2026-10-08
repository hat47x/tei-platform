package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

type runtimeCapabilityFixture struct {
	Definition     plugin.Definition
	Mapping        LegacyCapabilityMapping
	Input          ResolveInput
	CapabilityLock CapabilityResolutionLock
	RuntimeLock    plugin.Lock
	Environment    core.CanonicalID
}

func newRuntimeCapabilityFixture(t *testing.T) runtimeCapabilityFixture {
	t.Helper()
	contract := legacyContractQueryContract()
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.database.oracle"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Artifact: plugin.Artifact{
			Digest: "sha256:" + strings.Repeat("a", 64),
		},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityContractQueryExecutor,
			ID:   core.MustCanonicalID("database.query.oracle"),
		}},
	}
	mapping := LegacyCapabilityMapping{
		Kind:         plugin.CapabilityContractQueryExecutor,
		CapabilityID: core.MustCanonicalID("database.query.oracle"),
		Scope:        ScopeApplicationRuntime,
		Contract:     contract,
	}
	provider, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "runtime-capability")},
		Wirings: []Wiring{{
			ContractID:       contract.ID,
			PluginID:         provider.PluginID,
			PluginVersion:    provider.PluginVersion,
			ImplementationID: provider.ImplementationID,
		}},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	capabilityLock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}

	environmentID := core.MustCanonicalID("development")
	env := environment.Environment{
		ID: environmentID,
		Plugins: []environment.ExtensionRequirement{{
			ID:              definition.ID,
			CompatibleMajor: 1,
		}},
	}
	runtimeLock, err := plugin.BuildLock(
		plugin.Catalog{Plugins: []plugin.Definition{definition}},
		[]environment.Environment{env},
	)
	if err != nil {
		t.Fatal(err)
	}
	return runtimeCapabilityFixture{
		Definition:     definition,
		Mapping:        mapping,
		Input:          input,
		CapabilityLock: capabilityLock,
		RuntimeLock:    runtimeLock,
		Environment:    environmentID,
	}
}

func TestRuntimeCapabilityBindingUsesExistingPluginLockAuthority(t *testing.T) {
	fixture := newRuntimeCapabilityFixture(t)
	binding, err := buildLegacyRuntimeCapabilityBinding(
		fixture.CapabilityLock,
		fixture.RuntimeLock,
		fixture.Environment,
		fixture.Definition,
		fixture.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Environment != fixture.Environment ||
		binding.CapabilityLockFingerprint != fixture.CapabilityLock.LockFingerprint ||
		binding.PluginID != string(fixture.Definition.ID) ||
		binding.PluginVersion != fixture.Definition.Version ||
		binding.PluginDefinitionFingerprint != plugin.DefinitionFingerprint(fixture.Definition) ||
		binding.ArtifactDigest != fixture.Definition.Artifact.Digest {
		t.Fatalf("runtime capability binding=%+v", binding)
	}
	if err := verifyRuntimeCapabilityBinding(
		binding,
		fixture.CapabilityLock,
		fixture.RuntimeLock,
	); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeCapabilityBindingIgnoresUnrelatedPluginLockEntries(t *testing.T) {
	fixture := newRuntimeCapabilityFixture(t)
	binding, err := buildLegacyRuntimeCapabilityBinding(
		fixture.CapabilityLock,
		fixture.RuntimeLock,
		fixture.Environment,
		fixture.Definition,
		fixture.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}

	changed := fixture.RuntimeLock
	changed.Environments = append([]plugin.EnvironmentLock(nil), fixture.RuntimeLock.Environments...)
	changed.Environments[0].Plugins = append(
		append([]plugin.LockedPlugin(nil), changed.Environments[0].Plugins...),
		plugin.LockedPlugin{
			ID:             core.MustCanonicalID("tei.unrelated"),
			Version:        "1.0.0",
			Fingerprint:    strings.Repeat("b", 64),
			ArtifactDigest: "sha256:" + strings.Repeat("c", 64),
		},
	)
	if err := verifyRuntimeCapabilityBinding(
		binding,
		fixture.CapabilityLock,
		changed,
	); err != nil {
		t.Fatalf("unrelated Plugin Lock change invalidated capability binding: %v", err)
	}
}

func TestRuntimeCapabilityBindingRejectsProviderRemovalAndDrift(t *testing.T) {
	fixture := newRuntimeCapabilityFixture(t)
	binding, err := buildLegacyRuntimeCapabilityBinding(
		fixture.CapabilityLock,
		fixture.RuntimeLock,
		fixture.Environment,
		fixture.Definition,
		fixture.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}

	removed := fixture.RuntimeLock
	removed.Environments = append([]plugin.EnvironmentLock(nil), fixture.RuntimeLock.Environments...)
	removed.Environments[0].Plugins = nil
	if err := verifyRuntimeCapabilityBinding(
		binding,
		fixture.CapabilityLock,
		removed,
	); err == nil {
		t.Fatal("removed runtime provider remained bound")
	}

	definitionDrift := fixture.RuntimeLock
	definitionDrift.Environments = append([]plugin.EnvironmentLock(nil), fixture.RuntimeLock.Environments...)
	definitionDrift.Environments[0].Plugins = append(
		[]plugin.LockedPlugin(nil),
		fixture.RuntimeLock.Environments[0].Plugins...,
	)
	definitionDrift.Environments[0].Plugins[0].Fingerprint = strings.Repeat("d", 64)
	if err := verifyRuntimeCapabilityBinding(
		binding,
		fixture.CapabilityLock,
		definitionDrift,
	); err == nil || !strings.Contains(err.Error(), "definition fingerprint") {
		t.Fatalf("Plugin definition drift was not rejected: %v", err)
	}

	artifactDrift := fixture.RuntimeLock
	artifactDrift.Environments = append([]plugin.EnvironmentLock(nil), fixture.RuntimeLock.Environments...)
	artifactDrift.Environments[0].Plugins = append(
		[]plugin.LockedPlugin(nil),
		fixture.RuntimeLock.Environments[0].Plugins...,
	)
	artifactDrift.Environments[0].Plugins[0].ArtifactDigest =
		"sha256:" + strings.Repeat("e", 64)
	if err := verifyRuntimeCapabilityBinding(
		binding,
		fixture.CapabilityLock,
		artifactDrift,
	); err == nil || !strings.Contains(err.Error(), "artifact digest") {
		t.Fatalf("Plugin artifact drift was not rejected: %v", err)
	}
}

func TestRuntimeCapabilityBindingRejectsWrongScope(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "authoring-capability")},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	capabilityLock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	_, err = buildRuntimeCapabilityBinding(
		capabilityLock,
		plugin.Lock{FormatVersion: plugin.LockFormatVersion},
		core.MustCanonicalID("development"),
	)
	if err == nil || !strings.Contains(err.Error(), "application-runtime") {
		t.Fatalf("authoring capability was bound to Runtime Plugin Lock: %v", err)
	}
}

func TestLegacyRuntimeBindingRejectsProjectionNotDerivedFromLockedDefinition(t *testing.T) {
	fixture := newRuntimeCapabilityFixture(t)

	changedDefinition := fixture.Definition
	changedDefinition.Provides = append(
		[]plugin.Capability(nil),
		fixture.Definition.Provides...,
	)
	changedDefinition.Provides[0].ID = core.MustCanonicalID("database.query.changed")

	_, err := buildLegacyRuntimeCapabilityBinding(
		fixture.CapabilityLock,
		fixture.RuntimeLock,
		fixture.Environment,
		changedDefinition,
		LegacyCapabilityMapping{
			Kind:         plugin.CapabilityContractQueryExecutor,
			CapabilityID: core.MustCanonicalID("database.query.changed"),
			Scope:        ScopeApplicationRuntime,
			Contract:     legacyContractQueryContract(),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "was not derived") {
		t.Fatalf("non-authoritative projection was accepted: %v", err)
	}
}

func TestRuntimeCapabilityBindingRejectsDuplicateEnvironmentLock(t *testing.T) {
	fixture := newRuntimeCapabilityFixture(t)
	binding, err := buildLegacyRuntimeCapabilityBinding(
		fixture.CapabilityLock,
		fixture.RuntimeLock,
		fixture.Environment,
		fixture.Definition,
		fixture.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := fixture.RuntimeLock
	duplicate.Environments = append(
		append([]plugin.EnvironmentLock(nil), fixture.RuntimeLock.Environments...),
		fixture.RuntimeLock.Environments[0],
	)
	if err := verifyRuntimeCapabilityBinding(
		binding,
		fixture.CapabilityLock,
		duplicate,
	); err == nil || !strings.Contains(err.Error(), "exactly one environment") {
		t.Fatalf("duplicate Environment Lock was accepted: %v", err)
	}
}

func TestRuntimeCapabilityBindingSeparatesBuildPlanProvenanceFromRuntimeArtifactIdentity(t *testing.T) {
	fixture := newRuntimeCapabilityFixture(t)
	binding, err := buildLegacyRuntimeCapabilityBinding(
		fixture.CapabilityLock,
		fixture.RuntimeLock,
		fixture.Environment,
		fixture.Definition,
		fixture.Mapping,
	)
	if err != nil {
		t.Fatal(err)
	}

	traced := fixture.RuntimeLock
	traced.FormatVersion = plugin.BuildTraceLockFormatVersion
	traced.Environments = append([]plugin.EnvironmentLock(nil), fixture.RuntimeLock.Environments...)
	traced.Environments[0].Plugins = append(
		[]plugin.LockedPlugin(nil),
		fixture.RuntimeLock.Environments[0].Plugins...,
	)
	traced.Environments[0].Plugins[0].BuildPlanFingerprint = strings.Repeat("f", 64)

	if err := verifyRuntimeCapabilityBinding(
		binding,
		fixture.CapabilityLock,
		traced,
	); err != nil {
		t.Fatalf("build provenance refresh changed runtime capability identity: %v", err)
	}
}

func TestRuntimeCapabilityBindingRejectsSplitProviderArtifactUntilRuntimeAuthorityCanRepresentIt(t *testing.T) {
	fixture := newRuntimeCapabilityFixture(t)
	contract := legacyContractQueryContract()

	splitProvider := fixture.Input.Providers[0]
	splitProvider.ArtifactDigest = "sha256:" + strings.Repeat("9", 64)
	splitInput := fixture.Input
	splitInput.Providers = []Provider{splitProvider}
	splitInput.Evidence = []ConformanceEvidence{
		evidenceFor(splitProvider, contract, "split-provider-conformance"),
	}
	splitInput.Wirings = []Wiring{{
		ContractID:       contract.ID,
		PluginID:         splitProvider.PluginID,
		PluginVersion:    splitProvider.PluginVersion,
		ImplementationID: splitProvider.ImplementationID,
	}}
	resolution, err := resolveCapability(splitInput)
	if err != nil {
		t.Fatal(err)
	}
	splitCapabilityLock, err := buildCapabilityResolutionLock(splitInput, resolution)
	if err != nil {
		t.Fatal(err)
	}

	_, err = buildRuntimeCapabilityBinding(
		splitCapabilityLock,
		fixture.RuntimeLock,
		fixture.Environment,
	)
	if err == nil || !strings.Contains(err.Error(), "does not match capability provider artifact") {
		t.Fatalf(
			"split capability artifact was falsely authorized by plugin-level Runtime Lock: %v",
			err,
		)
	}
}
