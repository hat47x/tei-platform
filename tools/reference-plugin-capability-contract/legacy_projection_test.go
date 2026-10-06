package main

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/workspace"
)

func legacyContractQueryContract() Contract {
	return Contract{
		ID:                 "application-runtime.contract-query-executor",
		Version:            "1.0.0",
		RequestSemantics:   "execute-explicit-database-contract-query-realization-with-bound-parameters",
		ResultSemantics:    "return-provider-result-with-explicit-zero-one-many-and-error-boundaries",
		FailureSemantics:   "execution-failure-is-not-empty-result",
		ConformanceProfile: "tei.reference.legacy-contract-query-executor/v0",
	}
}

func repositoryRootFromThisTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func referencePluginDefinition(t *testing.T, id string) plugin.Definition {
	t.Helper()
	root := repositoryRootFromThisTest(t)
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range loaded.World.Plugins {
		if string(definition.ID) == id {
			return definition
		}
	}
	t.Fatalf("reference Plugin %s was not found", id)
	return plugin.Definition{}
}

func projectReferenceContractQueryProvider(
	t *testing.T,
	pluginID string,
	capabilityID core.CanonicalID,
) Provider {
	t.Helper()
	provider, err := projectLegacyCapabilityProvider(
		referencePluginDefinition(t, pluginID),
		LegacyCapabilityMapping{
			Kind:         plugin.CapabilityContractQueryExecutor,
			CapabilityID: capabilityID,
			Scope:        ScopeApplicationRuntime,
			Contract:     legacyContractQueryContract(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestLegacyContractQueryCapabilityProjectsFromExistingPluginAuthority(t *testing.T) {
	contract := legacyContractQueryContract()
	provider := projectReferenceContractQueryProvider(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
	)
	if provider.PluginID != "tei.database.oracle" ||
		provider.PluginVersion != "1.0.0" ||
		provider.ImplementationID != "legacy-capability:contract-query-executor:database.query.oracle" ||
		provider.Scope != ScopeApplicationRuntime ||
		provider.ContractID != contract.ID {
		t.Fatalf("projected provider=%+v", provider)
	}
	if provider.ArtifactDigest != "" {
		t.Fatalf("legacy projection invented artifact identity: %q", provider.ArtifactDigest)
	}
}

func TestLegacyBuiltinProjectionExposesMissingImplementationIdentity(t *testing.T) {
	contract := legacyContractQueryContract()
	provider := projectReferenceContractQueryProvider(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
	)
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
	})
	if err == nil || !strings.Contains(err.Error(), "exact implementation artifact digest") {
		t.Fatalf("legacy builtin without exact implementation identity was accepted: %v", err)
	}
}

func TestLegacyRuntimeProvidersRemainAmbiguousAfterExactArtifactsAreSupplied(t *testing.T) {
	contract := legacyContractQueryContract()
	oracle := projectReferenceContractQueryProvider(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
	)
	postgres := projectReferenceContractQueryProvider(
		t,
		"tei.database.postgres",
		core.CanonicalID("database.query.postgres"),
	)

	// Current legacy builtin descriptors do not carry exact implementation
	// artifacts. These digests simulate the migration prerequisite after that
	// identity has been established by a separate trusted build/provenance path.
	oracle.ArtifactDigest = "sha256:" + strings.Repeat("1", 64)
	postgres.ArtifactDigest = "sha256:" + strings.Repeat("2", 64)

	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{postgres, oracle},
		Evidence: []ConformanceEvidence{
			evidenceFor(oracle, contract, "oracle-contract-query-conformance"),
			evidenceFor(postgres, contract, "postgres-contract-query-conformance"),
		},
	}
	if _, err := resolveCapability(input); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("runtime providers were silently selected: %v", err)
	}

	input.Wirings = []Wiring{{
		ContractID:       contract.ID,
		PluginID:         oracle.PluginID,
		PluginVersion:    oracle.PluginVersion,
		ImplementationID: oracle.ImplementationID,
	}}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Provider.PluginID != oracle.PluginID {
		t.Fatalf("wired provider=%+v", resolution.Provider)
	}
}

func TestProjectedRuntimeProviderCannotSatisfyAuthoringSourceAdapterScope(t *testing.T) {
	runtimeProvider := projectReferenceContractQueryProvider(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
	)
	runtimeProvider.ArtifactDigest = "sha256:" + strings.Repeat("3", 64)
	authoringContract := sourceAdapterContract()
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: authoringContract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{authoringContract},
		Providers:   []Provider{runtimeProvider},
		Evidence: []ConformanceEvidence{
			evidenceFor(runtimeProvider, legacyContractQueryContract(), "oracle-contract-query-conformance"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("runtime projection crossed authority scope: %v", err)
	}
}

func TestLegacyProjectionDisappearsWhenAuthoritativeCapabilityDisappears(t *testing.T) {
	definition := referencePluginDefinition(t, "tei.database.oracle")
	filtered := definition.Provides[:0]
	for _, capability := range definition.Provides {
		if capability.Kind == plugin.CapabilityContractQueryExecutor &&
			capability.ID == core.CanonicalID("database.query.oracle") {
			continue
		}
		filtered = append(filtered, capability)
	}
	definition.Provides = filtered

	_, err := projectLegacyCapabilityProvider(
		definition,
		LegacyCapabilityMapping{
			Kind:         plugin.CapabilityContractQueryExecutor,
			CapabilityID: core.CanonicalID("database.query.oracle"),
			Scope:        ScopeApplicationRuntime,
			Contract:     legacyContractQueryContract(),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "must provide exactly one") {
		t.Fatalf("projection survived removal from authoritative legacy definition: %v", err)
	}
}


func TestLegacyProjectionDoesNotMutateAuthoritativePluginDefinition(t *testing.T) {
	definition := referencePluginDefinition(t, "tei.database.oracle")
	before := definition
	before.Provides = append([]plugin.Capability(nil), definition.Provides...)

	_, err := projectLegacyCapabilityProvider(
		definition,
		LegacyCapabilityMapping{
			Kind:         plugin.CapabilityContractQueryExecutor,
			CapabilityID: core.CanonicalID("database.query.oracle"),
			Scope:        ScopeApplicationRuntime,
			Contract:     legacyContractQueryContract(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(definition, before) {
		t.Fatalf("legacy projection mutated authoritative Plugin definition")
	}
}
