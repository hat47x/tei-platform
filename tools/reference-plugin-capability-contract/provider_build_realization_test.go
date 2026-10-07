package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func lockedRuntimeBuild(
	pluginID string,
	version string,
	target string,
	profile string,
	planMarker string,
	artifactMarker string,
) plugin.LockedBuild {
	return plugin.LockedBuild{
		Plugin: plugin.Identity{
			ID:      core.MustCanonicalID(pluginID),
			Version: version,
		},
		OutputKind:      plugin.BuildOutputRuntimeArtifact,
		Target:          target,
		Profile:         profile,
		PlanFingerprint: strings.Repeat(planMarker, 64),
		ArtifactDigest:  "sha256:" + strings.Repeat(artifactMarker, 64),
	}
}

func TestLegacyBuiltinProviderCanAcquireExactArtifactFromBuildArtifactLock(t *testing.T) {
	contract := legacyContractQueryContract()
	logical := projectReferenceContractQueryProvider(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
	)
	if logical.ArtifactDigest != "" {
		t.Fatalf("legacy builtin unexpectedly has artifact %q", logical.ArtifactDigest)
	}
	build := lockedRuntimeBuild(
		logical.PluginID,
		logical.PluginVersion,
		"server-linux-amd64",
		"shared-runtime",
		"a",
		"1",
	)
	buildLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{build},
	}
	binding, err := bindProviderToBuildArtifact(
		logical,
		buildLock,
		build.Target,
		build.Profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	realized, err := realizeProviderFromBuild(logical, binding)
	if err != nil {
		t.Fatal(err)
	}
	if realized.ArtifactDigest != build.ArtifactDigest ||
		realized.RealizationTarget != build.Target ||
		realized.SourceDeclarationFingerprint != logical.SourceDeclarationFingerprint {
		t.Fatalf("realized provider=%+v", realized)
	}

	resolution, err := resolveCapability(ResolveInput{
		Scope:             ScopeApplicationRuntime,
		RealizationTarget: build.Target,
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
	if resolution.Provider.ArtifactDigest != build.ArtifactDigest {
		t.Fatalf("resolved provider=%+v", resolution.Provider)
	}
}

func TestProviderBuildRealizationDoesNotGrantConformance(t *testing.T) {
	contract := legacyContractQueryContract()
	logical := projectReferenceContractQueryProvider(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
	)
	build := lockedRuntimeBuild(
		logical.PluginID,
		logical.PluginVersion,
		"server-linux-amd64",
		"shared-runtime",
		"a",
		"1",
	)
	binding, err := bindProviderToBuildArtifact(
		logical,
		plugin.BuildArtifactLock{
			FormatVersion: plugin.BuildArtifactLockFormatVersion,
			Builds:        []plugin.LockedBuild{build},
		},
		build.Target,
		build.Profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	realized, err := realizeProviderFromBuild(logical, binding)
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolveCapability(ResolveInput{
		Scope:             ScopeApplicationRuntime,
		RealizationTarget: build.Target,
		Requirement:       Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:         []Contract{contract},
		Providers:         []Provider{realized},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("build provenance was treated as Conformance Evidence: %v", err)
	}
}

func TestBuildArtifactLockCanCoalesceDifferentLogicalProviders(t *testing.T) {
	contract := legacyContractQueryContract()
	oracleLogical := projectReferenceContractQueryProvider(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
	)
	postgresLogical := projectReferenceContractQueryProvider(
		t,
		"tei.database.postgres",
		core.CanonicalID("database.query.postgres"),
	)
	sharedDigestMarker := "7"
	oracleBuild := lockedRuntimeBuild(
		oracleLogical.PluginID,
		oracleLogical.PluginVersion,
		"web-browser",
		"coalesced-standard-runtime",
		"a",
		sharedDigestMarker,
	)
	postgresBuild := lockedRuntimeBuild(
		postgresLogical.PluginID,
		postgresLogical.PluginVersion,
		"web-browser",
		"coalesced-standard-runtime",
		"b",
		sharedDigestMarker,
	)
	buildLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{oracleBuild, postgresBuild},
	}
	if err := buildLock.Validate(); err != nil {
		t.Fatal(err)
	}

	oracleBinding, err := bindProviderToBuildArtifact(
		oracleLogical, buildLock, oracleBuild.Target, oracleBuild.Profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	postgresBinding, err := bindProviderToBuildArtifact(
		postgresLogical, buildLock, postgresBuild.Target, postgresBuild.Profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	oracle, err := realizeProviderFromBuild(oracleLogical, oracleBinding)
	if err != nil {
		t.Fatal(err)
	}
	postgres, err := realizeProviderFromBuild(postgresLogical, postgresBinding)
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
	left := Provider{
		PluginID:            "tei.reference.split",
		PluginVersion:       "1.0.0",
		ImplementationID:    "split.left",
		Scope:               ScopeApplicationRuntime,
		ContractID:          contract.ID,
		ContractVersion:     contract.Version,
		ContractFingerprint: fingerprint(contract),
		ConformanceProfile:  contract.ConformanceProfile,
	}
	right := left
	right.ImplementationID = "split.right"
	leftBuild := lockedRuntimeBuild(
		left.PluginID, left.PluginVersion, "web-browser", "split-left", "c", "3",
	)
	rightBuild := lockedRuntimeBuild(
		right.PluginID, right.PluginVersion, "web-browser", "split-right", "d", "4",
	)
	buildLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{leftBuild, rightBuild},
	}
	if err := buildLock.Validate(); err != nil {
		t.Fatal(err)
	}
	leftBinding, err := bindProviderToBuildArtifact(
		left, buildLock, leftBuild.Target, leftBuild.Profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	rightBinding, err := bindProviderToBuildArtifact(
		right, buildLock, rightBuild.Target, rightBuild.Profile,
	)
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
	definition := plugin.Definition{
		ID:      core.MustCanonicalID(left.PluginID),
		Version: left.PluginVersion,
		Artifact: plugin.Artifact{Digest: leftRealized.ArtifactDigest},
	}
	runtimeLock := plugin.Lock{
		FormatVersion: plugin.LockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: core.MustCanonicalID("reference"),
			Plugins: []plugin.LockedPlugin{{
				ID:             definition.ID,
				Version:        definition.Version,
				Fingerprint:    plugin.DefinitionFingerprint(definition),
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
	logical := Provider{
		PluginID:         "tei.reference.rebuild",
		PluginVersion:    "1.0.0",
		ImplementationID: "rebuild.runtime",
		Scope:            ScopeApplicationRuntime,
	}
	firstBuild := lockedRuntimeBuild(
		logical.PluginID, logical.PluginVersion, "desktop", "runtime", "a", "6",
	)
	secondBuild := firstBuild
	secondBuild.PlanFingerprint = strings.Repeat("b", 64)
	firstLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{firstBuild},
	}
	secondLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{secondBuild},
	}
	firstBinding, err := bindProviderToBuildArtifact(
		logical, firstLock, firstBuild.Target, firstBuild.Profile,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondBinding, err := bindProviderToBuildArtifact(
		logical, secondLock, secondBuild.Target, secondBuild.Profile,
	)
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
