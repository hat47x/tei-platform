package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func providerRuntimeBuildPlan(
	pluginID string,
	version string,
	definitionFingerprint string,
	target string,
	profile string,
	entrypoint string,
	sourceMarker string,
) plugin.BuildPlan {
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID(pluginID), Version: version},
		PluginDefinitionFingerprint: definitionFingerprint,
		Sources: []plugin.BuildSource{{
			Kind:    "implementation-source",
			ID:      "source",
			Locator: entrypoint,
			Digest:  "sha256:" + strings.Repeat(sourceMarker, 64),
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.reference"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("f", 64),
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "reference-runtime",
			Entrypoint: entrypoint,
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     target,
		Profile:    profile,
	}
}

func buildLockForPlan(
	t *testing.T,
	plan plugin.BuildPlan,
	artifactMarker string,
) (plugin.BuildArtifactLock, plugin.LockedBuild) {
	t.Helper()
	locked, err := plugin.LockBuildOutput(
		plan,
		"sha256:"+strings.Repeat(artifactMarker, 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	lock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{locked},
	}
	if err := lock.Validate(); err != nil {
		t.Fatal(err)
	}
	return lock, locked
}

func legacyContractQueryMapping(capabilityID core.CanonicalID) LegacyCapabilityMapping {
	return LegacyCapabilityMapping{
		Kind:         plugin.CapabilityContractQueryExecutor,
		CapabilityID: capabilityID,
		Scope:        ScopeApplicationRuntime,
		Contract:     legacyContractQueryContract(),
	}
}

func TestLegacyBuiltinProviderCanAcquireExactArtifactFromBuildArtifactLock(t *testing.T) {
	definition := referencePluginDefinition(t, "tei.database.oracle")
	mapping := legacyContractQueryMapping(core.CanonicalID("database.query.oracle"))
	logical, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	if logical.ArtifactDigest != "" {
		t.Fatalf("legacy builtin unexpectedly has artifact %q", logical.ArtifactDigest)
	}
	plan := providerRuntimeBuildPlan(
		logical.PluginID,
		logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		"server-linux-amd64",
		"shared-runtime",
		"cmd/tei-runtime",
		"a",
	)
	buildLock, locked := buildLockForPlan(t, plan, "1")
	binding, realized, err := bindLegacyProviderToBuildArtifact(
		definition,
		mapping,
		plan,
		buildLock,
	)
	if err != nil {
		t.Fatal(err)
	}
	if realized.ArtifactDigest != locked.ArtifactDigest ||
		realized.RealizationTarget != plan.Target ||
		realized.SourceDeclarationFingerprint != logical.SourceDeclarationFingerprint ||
		binding.PluginBuildDefinitionFingerprint != plan.PluginDefinitionFingerprint {
		t.Fatalf("realized provider=%+v binding=%+v", realized, binding)
	}

	contract := legacyContractQueryContract()
	resolution, err := resolveCapability(ResolveInput{
		Scope:             ScopeApplicationRuntime,
		RealizationTarget: plan.Target,
		Requirement:       Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:         []Contract{contract},
		Providers:         []Provider{realized},
		Evidence: []ConformanceEvidence{
			evidenceFor(realized, contract, "build-backed-legacy-conformance"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Provider.ArtifactDigest != locked.ArtifactDigest {
		t.Fatalf("resolved provider=%+v", resolution.Provider)
	}
}

func TestLegacyProviderBuildRejectsBuildPlanForDifferentAuthoritativeDefinition(t *testing.T) {
	definition := referencePluginDefinition(t, "tei.database.oracle")
	mapping := legacyContractQueryMapping(core.CanonicalID("database.query.oracle"))
	logical, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	plan := providerRuntimeBuildPlan(
		logical.PluginID,
		logical.PluginVersion,
		strings.Repeat("a", 64),
		"server-linux-amd64",
		"runtime",
		"cmd/tei-runtime",
		"a",
	)
	buildLock, _ := buildLockForPlan(t, plan, "1")
	if _, _, err := bindLegacyProviderToBuildArtifact(
		definition, mapping, plan, buildLock,
	); err == nil || !strings.Contains(err.Error(), "authoritative legacy Plugin definition") {
		t.Fatalf("unrelated BuildPlan was accepted for legacy Provider: %v", err)
	}
}

func TestProviderBuildRealizationDoesNotGrantConformance(t *testing.T) {
	definition := referencePluginDefinition(t, "tei.database.oracle")
	mapping := legacyContractQueryMapping(core.CanonicalID("database.query.oracle"))
	logical, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	plan := providerRuntimeBuildPlan(
		logical.PluginID,
		logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		"server-linux-amd64",
		"runtime",
		"cmd/tei-runtime",
		"a",
	)
	buildLock, _ := buildLockForPlan(t, plan, "1")
	_, realized, err := bindLegacyProviderToBuildArtifact(
		definition, mapping, plan, buildLock,
	)
	if err != nil {
		t.Fatal(err)
	}
	contract := legacyContractQueryContract()
	_, err = resolveCapability(ResolveInput{
		Scope:             ScopeApplicationRuntime,
		RealizationTarget: plan.Target,
		Requirement:       Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:         []Contract{contract},
		Providers:         []Provider{realized},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("build provenance was treated as Conformance Evidence: %v", err)
	}
}

func TestBuildArtifactLockCanCoalesceDifferentLogicalProviders(t *testing.T) {
	oracleDefinition := referencePluginDefinition(t, "tei.database.oracle")
	postgresDefinition := referencePluginDefinition(t, "tei.database.postgres")
	oracleMapping := legacyContractQueryMapping(core.CanonicalID("database.query.oracle"))
	postgresMapping := legacyContractQueryMapping(core.CanonicalID("database.query.postgres"))
	oracleLogical, err := projectLegacyCapabilityProvider(oracleDefinition, oracleMapping)
	if err != nil {
		t.Fatal(err)
	}
	postgresLogical, err := projectLegacyCapabilityProvider(postgresDefinition, postgresMapping)
	if err != nil {
		t.Fatal(err)
	}
	oraclePlan := providerRuntimeBuildPlan(
		oracleLogical.PluginID,
		oracleLogical.PluginVersion,
		plugin.BuildDefinitionFingerprint(oracleDefinition),
		"web-browser",
		"coalesced-standard-runtime",
		"cmd/tei-web-runtime",
		"a",
	)
	postgresPlan := providerRuntimeBuildPlan(
		postgresLogical.PluginID,
		postgresLogical.PluginVersion,
		plugin.BuildDefinitionFingerprint(postgresDefinition),
		"web-browser",
		"coalesced-standard-runtime",
		"cmd/tei-web-runtime",
		"b",
	)
	oracleLocked, err := plugin.LockBuildOutput(
		oraclePlan, "sha256:"+strings.Repeat("7", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	postgresLocked, err := plugin.LockBuildOutput(
		postgresPlan, "sha256:"+strings.Repeat("7", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	buildLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{oracleLocked, postgresLocked},
	}
	if err := buildLock.Validate(); err != nil {
		t.Fatal(err)
	}
	_, oracle, err := bindLegacyProviderToBuildArtifact(
		oracleDefinition, oracleMapping, oraclePlan, buildLock,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, postgres, err := bindLegacyProviderToBuildArtifact(
		postgresDefinition, postgresMapping, postgresPlan, buildLock,
	)
	if err != nil {
		t.Fatal(err)
	}
	if oracle.ArtifactDigest != postgres.ArtifactDigest {
		t.Fatal("coalesced providers did not share physical artifact")
	}
	if oracle.PluginID == postgres.PluginID ||
		oracle.ImplementationID == postgres.ImplementationID {
		t.Fatal("physical coalescing collapsed logical provider identity")
	}

	contract := legacyContractQueryContract()
	input := ResolveInput{
		Scope:             ScopeApplicationRuntime,
		RealizationTarget: "web-browser",
		Requirement:       Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:         []Contract{contract},
		Providers:         []Provider{oracle, postgres},
		Evidence: []ConformanceEvidence{
			evidenceFor(oracle, contract, "oracle-coalesced-conformance"),
			evidenceFor(postgres, contract, "postgres-coalesced-conformance"),
		},
	}
	if _, err := resolveCapability(input); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("physical coalescing became provider-selection authority: %v", err)
	}
}

func TestBuildArtifactLockCanDescribeSplitProviderArtifactsButRuntimeLockStillCannot(t *testing.T) {
	contract := portableRuntimeContract()
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.split"),
		Version: "1.0.0",
	}
	definitionFingerprint := plugin.BuildDefinitionFingerprint(definition)
	left := Provider{
		PluginID:            string(definition.ID),
		PluginVersion:       definition.Version,
		ImplementationID:    "split.left",
		Scope:               ScopeApplicationRuntime,
		ContractID:          contract.ID,
		ContractVersion:     contract.Version,
		ContractFingerprint: fingerprint(contract),
		ConformanceProfile:  contract.ConformanceProfile,
	}
	right := left
	right.ImplementationID = "split.right"
	leftPlan := providerRuntimeBuildPlan(
		left.PluginID, left.PluginVersion, definitionFingerprint,
		"web-browser", "split-left", "cmd/split-left", "c",
	)
	rightPlan := providerRuntimeBuildPlan(
		right.PluginID, right.PluginVersion, definitionFingerprint,
		"web-browser", "split-right", "cmd/split-right", "d",
	)
	leftLocked, err := plugin.LockBuildOutput(
		leftPlan, "sha256:"+strings.Repeat("3", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	rightLocked, err := plugin.LockBuildOutput(
		rightPlan, "sha256:"+strings.Repeat("4", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	buildLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{leftLocked, rightLocked},
	}
	leftBinding, err := bindProviderToBuildArtifact(left, leftPlan, buildLock)
	if err != nil {
		t.Fatal(err)
	}
	rightBinding, err := bindProviderToBuildArtifact(right, rightPlan, buildLock)
	if err != nil {
		t.Fatal(err)
	}
	leftRealized, err := realizeProviderFromBuild(left, leftBinding)
	if err != nil {
		t.Fatal(err)
	}
	rightRealized, err := realizeProviderFromBuild(right, rightBinding)
	if err != nil {
		t.Fatal(err)
	}
	if leftRealized.ArtifactDigest == rightRealized.ArtifactDigest {
		t.Fatal("split provider artifacts unexpectedly collapsed")
	}

	leftInput := targetResolveInput(contract, "web-browser", leftRealized)
	leftResolution, err := resolveCapability(leftInput)
	if err != nil {
		t.Fatal(err)
	}
	leftLock, err := buildCapabilityResolutionLock(leftInput, leftResolution)
	if err != nil {
		t.Fatal(err)
	}
	definitionWithArtifact := definition
	definitionWithArtifact.Artifact = plugin.Artifact{Digest: leftRealized.ArtifactDigest}
	runtimeLock := plugin.Lock{
		FormatVersion: plugin.LockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: core.MustCanonicalID("reference"),
			Plugins: []plugin.LockedPlugin{{
				ID:             definitionWithArtifact.ID,
				Version:        definitionWithArtifact.Version,
				Fingerprint:    plugin.DefinitionFingerprint(definitionWithArtifact),
				ArtifactDigest: leftRealized.ArtifactDigest,
			}},
		}},
	}
	if _, err := buildRuntimeCapabilityBinding(
		leftLock, runtimeLock, core.MustCanonicalID("reference"),
	); err != nil {
		t.Fatalf("left split provider could not bind to Runtime Plugin Lock: %v", err)
	}

	rightInput := targetResolveInput(contract, "web-browser", rightRealized)
	rightResolution, err := resolveCapability(rightInput)
	if err != nil {
		t.Fatal(err)
	}
	rightLock, err := buildCapabilityResolutionLock(rightInput, rightResolution)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildRuntimeCapabilityBinding(
		rightLock, runtimeLock, core.MustCanonicalID("reference"),
	); err == nil || !strings.Contains(err.Error(), "artifact") {
		t.Fatalf("Plugin-level Runtime Lock incorrectly authorized second split artifact: %v", err)
	}
}

func TestProviderBuildRealizationSeparatesBuildProvenanceFromArtifactIdentity(t *testing.T) {
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.rebuild"),
		Version: "1.0.0",
	}
	logical := Provider{
		PluginID:         string(definition.ID),
		PluginVersion:    definition.Version,
		ImplementationID: "rebuild.runtime",
		Scope:            ScopeApplicationRuntime,
	}
	firstPlan := providerRuntimeBuildPlan(
		logical.PluginID, logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		"desktop", "runtime", "cmd/rebuild", "a",
	)
	secondPlan := firstPlan
	secondPlan.Parameters = []plugin.BuildParameter{{Name: "rebuild-marker", Value: "2"}}
	firstLock, _ := buildLockForPlan(t, firstPlan, "6")
	secondLock, _ := buildLockForPlan(t, secondPlan, "6")
	firstBinding, err := bindProviderToBuildArtifact(logical, firstPlan, firstLock)
	if err != nil {
		t.Fatal(err)
	}
	secondBinding, err := bindProviderToBuildArtifact(logical, secondPlan, secondLock)
	if err != nil {
		t.Fatal(err)
	}
	if firstBinding.BindingFingerprint == secondBinding.BindingFingerprint {
		t.Fatal("BuildPlan provenance change did not change realization binding")
	}
	firstProvider, err := realizeProviderFromBuild(logical, firstBinding)
	if err != nil {
		t.Fatal(err)
	}
	secondProvider, err := realizeProviderFromBuild(logical, secondBinding)
	if err != nil {
		t.Fatal(err)
	}
	if firstProvider != secondProvider {
		t.Fatalf("same artifact bytes changed Provider identity: first=%+v second=%+v", firstProvider, secondProvider)
	}
}

func TestProviderBuildRealizationReplayDetectsBuildPlanDrift(t *testing.T) {
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.replay-build"),
		Version: "1.0.0",
	}
	logical := Provider{
		PluginID:         string(definition.ID),
		PluginVersion:    definition.Version,
		ImplementationID: "replay.runtime",
		Scope:            ScopeApplicationRuntime,
	}
	firstPlan := providerRuntimeBuildPlan(
		logical.PluginID, logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		"desktop", "runtime", "cmd/replay", "a",
	)
	firstLock, _ := buildLockForPlan(t, firstPlan, "8")
	binding, err := bindProviderToBuildArtifact(logical, firstPlan, firstLock)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyProviderBuildRealization(
		binding, logical, firstPlan, firstLock,
	); err != nil {
		t.Fatal(err)
	}

	secondPlan := firstPlan
	secondPlan.Parameters = []plugin.BuildParameter{{Name: "changed", Value: "true"}}
	secondLock, _ := buildLockForPlan(t, secondPlan, "8")
	if err := verifyProviderBuildRealization(
		binding, logical, secondPlan, secondLock,
	); err == nil || !strings.Contains(err.Error(), "differs from current BuildPlan / BuildArtifactLock") {
		t.Fatalf("BuildPlan drift did not stale provider realization: %v", err)
	}
}

func TestProviderBuildRealizationRejectsSourceDeclarationDrift(t *testing.T) {
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.source-bound"),
		Version: "1.0.0",
	}
	logical := Provider{
		PluginID:                     string(definition.ID),
		PluginVersion:                definition.Version,
		ImplementationID:             "source-bound.runtime",
		SourceDeclarationFingerprint: "sha256:" + strings.Repeat("1", 64),
		Scope:                        ScopeApplicationRuntime,
	}
	plan := providerRuntimeBuildPlan(
		logical.PluginID, logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		"desktop", "runtime", "cmd/source-bound", "c",
	)
	lock, _ := buildLockForPlan(t, plan, "9")
	binding, err := bindProviderToBuildArtifact(logical, plan, lock)
	if err != nil {
		t.Fatal(err)
	}

	drifted := logical
	drifted.SourceDeclarationFingerprint = "sha256:" + strings.Repeat("2", 64)
	if _, err := realizeProviderFromBuild(drifted, binding); err == nil ||
		!strings.Contains(err.Error(), "does not belong to logical provider") {
		t.Fatalf("source declaration drift reused old build realization: %v", err)
	}
}
