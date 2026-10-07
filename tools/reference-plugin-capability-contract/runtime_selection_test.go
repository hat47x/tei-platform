package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/realization"
)

func exactLegacyRuntimeDefinition(
	pluginID, capabilityID string,
	provider environment.DatabaseProvider,
	digestMarker string,
) plugin.Definition {
	return plugin.Definition{
		ID:      core.MustCanonicalID(pluginID),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Artifact: plugin.Artifact{
			Digest: "sha256:" + strings.Repeat(digestMarker, 64),
		},
		Provides: []plugin.Capability{{
			Kind:              plugin.CapabilityContractQueryExecutor,
			ID:                core.MustCanonicalID(capabilityID),
			DatabaseProviders: []environment.DatabaseProvider{provider},
		}},
	}
}

func legacyRuntimeMapping(
	capabilityID string,
) LegacyCapabilityMapping {
	return LegacyCapabilityMapping{
		Kind:         plugin.CapabilityContractQueryExecutor,
		CapabilityID: core.MustCanonicalID(capabilityID),
		Scope:        ScopeApplicationRuntime,
		Contract:     legacyContractQueryContract(),
	}
}

func selectedContractQueryExecutor(
	environmentID, instanceID, databaseID, contractID, queryID string,
	provider environment.DatabaseProvider,
	definition plugin.Definition,
) realization.ResolvedContractQueryExecutor {
	capability := definition.Provides[0]
	return realization.ResolvedContractQueryExecutor{
		Binding: realization.ContractQueryExecutorBinding{
			Environment: core.MustCanonicalID(environmentID),
			Instance:    core.MustCanonicalID(instanceID),
			Target: realization.ContractQueryTarget{
				Database: core.MustCanonicalID(databaseID),
				Contract: core.MustCanonicalID(contractID),
				Query:    core.MustLocalName(queryID),
			},
			Executor: capability.ID,
		},
		Instance: environment.DatabaseInstance{
			ID:         core.MustCanonicalID(instanceID),
			Definition: core.MustCanonicalID(databaseID),
			Provider:   provider,
		},
		Capability: plugin.ResolvedCapability{
			Plugin: plugin.Identity{
				ID:      definition.ID,
				Version: definition.Version,
			},
			Capability: capability,
		},
	}
}

func evidenceForLegacyRuntimeSelection(
	t *testing.T,
	definition plugin.Definition,
	mapping LegacyCapabilityMapping,
	marker string,
) []ConformanceEvidence {
	t.Helper()
	provider, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	return []ConformanceEvidence{
		evidenceFor(provider, mapping.Contract, marker),
	}
}

func TestExistingTargetSpecificBindingOwnsRuntimeProviderSelection(t *testing.T) {
	oracleDefinition := exactLegacyRuntimeDefinition(
		"tei.database.oracle",
		"database.query.oracle",
		environment.ProviderOracle,
		"1",
	)
	postgresDefinition := exactLegacyRuntimeDefinition(
		"tei.database.postgres",
		"database.query.postgres",
		environment.ProviderPostgres,
		"2",
	)
	oracleMapping := legacyRuntimeMapping("database.query.oracle")
	postgresMapping := legacyRuntimeMapping("database.query.postgres")

	oracleSelected := selectedContractQueryExecutor(
		"development",
		"sales.oracle",
		"sales_oracle",
		"sales.oracle.contract",
		"find",
		environment.ProviderOracle,
		oracleDefinition,
	)
	postgresSelected := selectedContractQueryExecutor(
		"development",
		"analytics.postgres",
		"analytics_postgres",
		"analytics.postgres.contract",
		"find",
		environment.ProviderPostgres,
		postgresDefinition,
	)

	oracleLock, err := resolveLegacyContractQueryCapabilityForSelection(
		oracleSelected,
		oracleDefinition,
		oracleMapping,
		evidenceForLegacyRuntimeSelection(
			t,
			oracleDefinition,
			oracleMapping,
			"oracle-selection-conformance",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	postgresLock, err := resolveLegacyContractQueryCapabilityForSelection(
		postgresSelected,
		postgresDefinition,
		postgresMapping,
		evidenceForLegacyRuntimeSelection(
			t,
			postgresDefinition,
			postgresMapping,
			"postgres-selection-conformance",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	if oracleLock.Wiring != nil || postgresLock.Wiring != nil {
		t.Fatalf(
			"target-selected runtime capability added generic wiring: oracle=%+v postgres=%+v",
			oracleLock.Wiring,
			postgresLock.Wiring,
		)
	}
	if oracleLock.Requirement != postgresLock.Requirement {
		t.Fatalf(
			"same generic runtime contract produced different requirements: oracle=%+v postgres=%+v",
			oracleLock.Requirement,
			postgresLock.Requirement,
		)
	}
	if oracleLock.Provider.PluginID == postgresLock.Provider.PluginID {
		t.Fatal("independent target selections collapsed to one runtime provider")
	}

	environmentID := core.MustCanonicalID("development")
	runtimeLock, err := plugin.BuildLock(
		plugin.Catalog{Plugins: []plugin.Definition{
			oracleDefinition,
			postgresDefinition,
		}},
		[]environment.Environment{{
			ID: environmentID,
			Plugins: []environment.ExtensionRequirement{
				{ID: oracleDefinition.ID, CompatibleMajor: 1},
				{ID: postgresDefinition.ID, CompatibleMajor: 1},
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	oracleRuntimeBinding, err := buildLegacyRuntimeCapabilityBinding(
		oracleLock,
		runtimeLock,
		environmentID,
		oracleDefinition,
		oracleMapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	postgresRuntimeBinding, err := buildLegacyRuntimeCapabilityBinding(
		postgresLock,
		runtimeLock,
		environmentID,
		postgresDefinition,
		postgresMapping,
	)
	if err != nil {
		t.Fatal(err)
	}

	oracleSelectionBinding, err := buildRuntimeCapabilitySelectionBinding(
		oracleRuntimeBinding,
		oracleLock,
		oracleSelected,
	)
	if err != nil {
		t.Fatal(err)
	}
	postgresSelectionBinding, err := buildRuntimeCapabilitySelectionBinding(
		postgresRuntimeBinding,
		postgresLock,
		postgresSelected,
	)
	if err != nil {
		t.Fatal(err)
	}
	if oracleSelectionBinding.SelectionFingerprint ==
		postgresSelectionBinding.SelectionFingerprint {
		t.Fatal("different target-specific selections produced one selection fingerprint")
	}
	if err := verifyRuntimeCapabilitySelectionBinding(
		oracleSelectionBinding,
		oracleRuntimeBinding,
		oracleLock,
		oracleSelected,
	); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeCapabilitySelectionBinding(
		postgresSelectionBinding,
		postgresRuntimeBinding,
		postgresLock,
		postgresSelected,
	); err != nil {
		t.Fatal(err)
	}
}

func TestTargetSpecificRuntimeSelectionRejectsGenericWiringAsSecondSelector(t *testing.T) {
	definition := exactLegacyRuntimeDefinition(
		"tei.database.oracle",
		"database.query.oracle",
		environment.ProviderOracle,
		"3",
	)
	mapping := legacyRuntimeMapping("database.query.oracle")
	selected := selectedContractQueryExecutor(
		"development",
		"sales.oracle",
		"sales_oracle",
		"sales.oracle.contract",
		"find",
		environment.ProviderOracle,
		definition,
	)
	provider, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	input := ResolveInput{
		Scope: ScopeApplicationRuntime,
		Requirement: Requirement{
			ContractID:      mapping.Contract.ID,
			CompatibleMajor: 1,
		},
		Contracts: []Contract{mapping.Contract},
		Providers: []Provider{provider},
		Evidence:  []ConformanceEvidence{evidenceFor(provider, mapping.Contract, "wired")},
		Wirings: []Wiring{{
			ContractID:       mapping.Contract.ID,
			PluginID:         provider.PluginID,
			PluginVersion:    provider.PluginVersion,
			ImplementationID: provider.ImplementationID,
		}},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}

	envID := core.MustCanonicalID("development")
	runtimeLock, err := plugin.BuildLock(
		plugin.Catalog{Plugins: []plugin.Definition{definition}},
		[]environment.Environment{{
			ID: envID,
			Plugins: []environment.ExtensionRequirement{{
				ID: definition.ID, CompatibleMajor: 1,
			}},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	runtimeBinding, err := buildLegacyRuntimeCapabilityBinding(
		lock,
		runtimeLock,
		envID,
		definition,
		mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = buildRuntimeCapabilitySelectionBinding(
		runtimeBinding,
		lock,
		selected,
	)
	if err == nil || !strings.Contains(err.Error(), "cannot also use generic capability wiring") {
		t.Fatalf("second runtime selector was accepted: %v", err)
	}
}

func TestSelectedRuntimeCapabilityRejectsForgedInstanceProvider(t *testing.T) {
	definition := exactLegacyRuntimeDefinition(
		"tei.database.oracle",
		"database.query.oracle",
		environment.ProviderOracle,
		"4",
	)
	mapping := legacyRuntimeMapping("database.query.oracle")
	selected := selectedContractQueryExecutor(
		"development",
		"sales.oracle",
		"sales_oracle",
		"sales.oracle.contract",
		"find",
		environment.ProviderOracle,
		definition,
	)
	selected.Instance.Provider = environment.ProviderPostgres

	_, err := resolveLegacyContractQueryCapabilityForSelection(
		selected,
		definition,
		mapping,
		evidenceForLegacyRuntimeSelection(t, definition, mapping, "forged-provider"),
	)
	if err == nil || !strings.Contains(err.Error(), "does not support instance provider") {
		t.Fatalf("forged instance provider was accepted: %v", err)
	}
}

func TestSelectedRuntimeCapabilityRejectsTargetDatabaseMismatch(t *testing.T) {
	definition := exactLegacyRuntimeDefinition(
		"tei.database.oracle",
		"database.query.oracle",
		environment.ProviderOracle,
		"5",
	)
	mapping := legacyRuntimeMapping("database.query.oracle")
	selected := selectedContractQueryExecutor(
		"development",
		"sales.oracle",
		"sales_oracle",
		"sales.oracle.contract",
		"find",
		environment.ProviderOracle,
		definition,
	)
	selected.Binding.Target.Database = core.MustCanonicalID("different_database")

	_, err := resolveLegacyContractQueryCapabilityForSelection(
		selected,
		definition,
		mapping,
		evidenceForLegacyRuntimeSelection(t, definition, mapping, "wrong-database"),
	)
	if err == nil || !strings.Contains(err.Error(), "target belongs to") {
		t.Fatalf("target Database mismatch was accepted: %v", err)
	}
}


func TestRuntimeCapabilitySelectionReplaysCurrentProductionBindingAuthority(t *testing.T) {
	definition := exactLegacyRuntimeDefinition(
		"tei.database.oracle",
		"database.query.oracle",
		environment.ProviderOracle,
		"6",
	)
	mapping := legacyRuntimeMapping("database.query.oracle")
	envID := core.MustCanonicalID("development")
	instanceID := core.MustCanonicalID("sales.oracle")
	target := realization.ContractQueryTarget{
		Database: core.MustCanonicalID("sales_oracle"),
		Contract: core.MustCanonicalID("sales.oracle.contract"),
		Query:    core.MustLocalName("find"),
	}
	env := environment.Environment{
		ID: envID,
		Plugins: []environment.ExtensionRequirement{{
			ID: definition.ID, CompatibleMajor: 1,
		}},
		Instances: []environment.DatabaseInstance{{
			ID:         instanceID,
			Definition: target.Database,
			Provider:   environment.ProviderOracle,
		}},
	}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{definition}}
	resolvedPlugins, err := catalog.ResolveEnvironment(env)
	if err != nil {
		t.Fatal(err)
	}
	bindings := []realization.ContractQueryExecutorBinding{{
		Environment: envID,
		Instance:    instanceID,
		Target:      target,
		Executor:    definition.Provides[0].ID,
	}}
	selected, err := realization.ResolveContractQueryExecutor(
		env,
		resolvedPlugins,
		bindings,
		instanceID,
		target,
	)
	if err != nil {
		t.Fatal(err)
	}

	capabilityLock, err := resolveLegacyContractQueryCapabilityForSelection(
		selected,
		definition,
		mapping,
		evidenceForLegacyRuntimeSelection(
			t,
			definition,
			mapping,
			"production-binding-conformance",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	runtimeLock, err := plugin.BuildLock(
		catalog,
		[]environment.Environment{env},
	)
	if err != nil {
		t.Fatal(err)
	}
	runtimeBinding, err := buildLegacyRuntimeCapabilityBinding(
		capabilityLock,
		runtimeLock,
		envID,
		definition,
		mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectionBinding, err := buildRuntimeCapabilitySelectionBinding(
		runtimeBinding,
		capabilityLock,
		selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		env,
		resolvedPlugins,
		bindings,
	); err != nil {
		t.Fatal(err)
	}

	if err := verifyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		env,
		resolvedPlugins,
		nil,
	); err == nil || !strings.Contains(err.Error(), "Bindingがありません") {
		t.Fatalf("removed production Binding remained current: %v", err)
	}

	duplicate := append(
		append([]realization.ContractQueryExecutorBinding(nil), bindings...),
		bindings[0],
	)
	if err := verifyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		env,
		resolvedPlugins,
		duplicate,
	); err == nil || !strings.Contains(err.Error(), "Bindingが複数あります") {
		t.Fatalf("duplicate production Binding remained current: %v", err)
	}
}

func TestRuntimeCapabilitySelectionRejectsCurrentBindingToDifferentExecutor(t *testing.T) {
	oracleDefinition := exactLegacyRuntimeDefinition(
		"tei.database.oracle",
		"database.query.oracle",
		environment.ProviderOracle,
		"7",
	)
	alternateDefinition := exactLegacyRuntimeDefinition(
		"tei.database.oracle.alternate",
		"database.query.oracle_alternate",
		environment.ProviderOracle,
		"8",
	)
	mapping := legacyRuntimeMapping("database.query.oracle")
	envID := core.MustCanonicalID("development")
	instanceID := core.MustCanonicalID("sales.oracle")
	target := realization.ContractQueryTarget{
		Database: core.MustCanonicalID("sales_oracle"),
		Contract: core.MustCanonicalID("sales.oracle.contract"),
		Query:    core.MustLocalName("find"),
	}
	env := environment.Environment{
		ID: envID,
		Plugins: []environment.ExtensionRequirement{
			{ID: oracleDefinition.ID, CompatibleMajor: 1},
			{ID: alternateDefinition.ID, CompatibleMajor: 1},
		},
		Instances: []environment.DatabaseInstance{{
			ID:         instanceID,
			Definition: target.Database,
			Provider:   environment.ProviderOracle,
		}},
	}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{
		oracleDefinition,
		alternateDefinition,
	}}
	resolvedPlugins, err := catalog.ResolveEnvironment(env)
	if err != nil {
		t.Fatal(err)
	}
	originalBindings := []realization.ContractQueryExecutorBinding{{
		Environment: envID,
		Instance:    instanceID,
		Target:      target,
		Executor:    oracleDefinition.Provides[0].ID,
	}}
	selected, err := realization.ResolveContractQueryExecutor(
		env,
		resolvedPlugins,
		originalBindings,
		instanceID,
		target,
	)
	if err != nil {
		t.Fatal(err)
	}
	capabilityLock, err := resolveLegacyContractQueryCapabilityForSelection(
		selected,
		oracleDefinition,
		mapping,
		evidenceForLegacyRuntimeSelection(
			t,
			oracleDefinition,
			mapping,
			"original-selection",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	runtimeLock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	runtimeBinding, err := buildLegacyRuntimeCapabilityBinding(
		capabilityLock,
		runtimeLock,
		envID,
		oracleDefinition,
		mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectionBinding, err := buildRuntimeCapabilitySelectionBinding(
		runtimeBinding,
		capabilityLock,
		selected,
	)
	if err != nil {
		t.Fatal(err)
	}

	changedBindings := []realization.ContractQueryExecutorBinding{{
		Environment: envID,
		Instance:    instanceID,
		Target:      target,
		Executor:    alternateDefinition.Provides[0].ID,
	}}
	err = verifyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		env,
		resolvedPlugins,
		changedBindings,
	)
	if err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("changed authoritative executor remained current: %v", err)
	}
}


func TestLegacyRuntimeSelectionReplayRequiresCurrentPluginAuthority(t *testing.T) {
	definition := exactLegacyRuntimeDefinition(
		"tei.database.oracle",
		"database.query.oracle",
		environment.ProviderOracle,
		"9",
	)
	mapping := legacyRuntimeMapping("database.query.oracle")
	envID := core.MustCanonicalID("development")
	instanceID := core.MustCanonicalID("sales.oracle")
	target := realization.ContractQueryTarget{
		Database: core.MustCanonicalID("sales_oracle"),
		Contract: core.MustCanonicalID("sales.oracle.contract"),
		Query:    core.MustLocalName("find"),
	}
	env := environment.Environment{
		ID: envID,
		Plugins: []environment.ExtensionRequirement{{
			ID: definition.ID, CompatibleMajor: 1,
		}},
		Instances: []environment.DatabaseInstance{{
			ID:         instanceID,
			Definition: target.Database,
			Provider:   environment.ProviderOracle,
		}},
	}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{definition}}
	resolvedPlugins, err := catalog.ResolveEnvironment(env)
	if err != nil {
		t.Fatal(err)
	}
	bindings := []realization.ContractQueryExecutorBinding{{
		Environment: envID,
		Instance:    instanceID,
		Target:      target,
		Executor:    definition.Provides[0].ID,
	}}
	selected, err := realization.ResolveContractQueryExecutor(
		env,
		resolvedPlugins,
		bindings,
		instanceID,
		target,
	)
	if err != nil {
		t.Fatal(err)
	}
	capabilityLock, err := resolveLegacyContractQueryCapabilityForSelection(
		selected,
		definition,
		mapping,
		evidenceForLegacyRuntimeSelection(
			t,
			definition,
			mapping,
			"current-plugin-authority",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	runtimeLock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	runtimeBinding, err := buildLegacyRuntimeCapabilityBinding(
		capabilityLock,
		runtimeLock,
		envID,
		definition,
		mapping,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectionBinding, err := buildRuntimeCapabilitySelectionBinding(
		runtimeBinding,
		capabilityLock,
		selected,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifyLegacyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		runtimeLock,
		env,
		catalog,
		bindings,
		definition,
		mapping,
	); err != nil {
		t.Fatal(err)
	}

	catalogDriftDefinition := definition
	catalogDriftDefinition.Provides = append(
		[]plugin.Capability(nil),
		definition.Provides...,
	)
	catalogDriftDefinition.Provides[0] = cloneLegacyCapabilityForTest(
		t,
		catalogDriftDefinition.Provides[0],
	)
	catalogDriftDefinition.Provides[0].DatabaseProviders = append(
		catalogDriftDefinition.Provides[0].DatabaseProviders,
		environment.ProviderPostgres,
	)
	err = verifyLegacyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		runtimeLock,
		env,
		plugin.Catalog{Plugins: []plugin.Definition{catalogDriftDefinition}},
		bindings,
		definition,
		mapping,
	)
	if err == nil || !strings.Contains(err.Error(), "fingerprint diverged") {
		t.Fatalf("catalog definition drift remained current: %v", err)
	}

	legacyDefinitionDrift := catalogDriftDefinition
	err = verifyLegacyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		runtimeLock,
		env,
		catalog,
		bindings,
		legacyDefinitionDrift,
		mapping,
	)
	if err == nil || !strings.Contains(err.Error(), "authoritative legacy Plugin definition") {
		t.Fatalf("legacy authority drift remained current: %v", err)
	}

	runtimeLockDrift := runtimeLock
	runtimeLockDrift.Environments = append(
		[]plugin.EnvironmentLock(nil),
		runtimeLock.Environments...,
	)
	runtimeLockDrift.Environments[0].Plugins = append(
		[]plugin.LockedPlugin(nil),
		runtimeLock.Environments[0].Plugins...,
	)
	runtimeLockDrift.Environments[0].Plugins[0].ArtifactDigest =
		"sha256:" + strings.Repeat("a", 64)
	err = verifyLegacyRuntimeCapabilitySelectionAgainstCurrentAuthority(
		selectionBinding,
		runtimeBinding,
		capabilityLock,
		runtimeLockDrift,
		env,
		catalog,
		bindings,
		definition,
		mapping,
	)
	if err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("runtime Plugin Lock artifact drift remained current: %v", err)
	}
}
