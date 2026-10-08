package main

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func legacyPackagingInput(
	t *testing.T,
	pluginID string,
	capabilityID core.CanonicalID,
	target string,
	profile string,
	sourceMarker string,
	artifactMarker string,
) TargetPackagingInput {
	t.Helper()
	definition := referencePluginDefinition(t, pluginID)
	mapping := legacyContractQueryMapping(capabilityID)
	logical, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		t.Fatal(err)
	}
	plan := providerRuntimeBuildPlan(
		logical.PluginID,
		logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		target,
		profile,
		"cmd/"+pluginID,
		sourceMarker,
	)
	buildLock, _ := buildLockForPlan(t, plan, artifactMarker)
	binding, realized, err := bindLegacyProviderToBuildArtifact(
		definition,
		mapping,
		plan,
		buildLock,
	)
	if err != nil {
		t.Fatal(err)
	}
	contract := mapping.Contract
	input := ResolveInput{
		Scope:             ScopeApplicationRuntime,
		RealizationTarget: target,
		Requirement:       Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:         []Contract{contract},
		Providers:         []Provider{realized},
		Evidence: []ConformanceEvidence{
			evidenceFor(realized, contract, "packaging-"+pluginID),
		},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	return TargetPackagingInput{
		CapabilityLock:   lock,
		BuildRealization: binding,
	}
}

func syntheticPackagingInput(
	t *testing.T,
	definition plugin.Definition,
	implementationID string,
	contract Contract,
	target string,
	profile string,
	sourceMarker string,
	artifactMarker string,
) TargetPackagingInput {
	t.Helper()
	logical := Provider{
		PluginID:            string(definition.ID),
		PluginVersion:       definition.Version,
		ImplementationID:    implementationID,
		Scope:               ScopeApplicationRuntime,
		ContractID:          contract.ID,
		ContractVersion:     contract.Version,
		ContractFingerprint: fingerprint(contract),
		ConformanceProfile:  contract.ConformanceProfile,
	}
	plan := providerRuntimeBuildPlan(
		logical.PluginID,
		logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		target,
		profile,
		"cmd/"+implementationID,
		sourceMarker,
	)
	buildLock, _ := buildLockForPlan(t, plan, artifactMarker)
	binding, err := bindProviderToBuildArtifact(logical, plan, buildLock)
	if err != nil {
		t.Fatal(err)
	}
	realized, err := realizeProviderFromBuild(logical, binding)
	if err != nil {
		t.Fatal(err)
	}
	input := targetResolveInput(contract, target, realized)
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	return TargetPackagingInput{
		CapabilityLock:   lock,
		BuildRealization: binding,
	}
}

func TestTargetPackagingPlanRepresentsCoalescingWithoutCollapsingProviders(t *testing.T) {
	oracle := legacyPackagingInput(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
		"web-browser",
		"coalesced-runtime",
		"a",
		"7",
	)
	postgres := legacyPackagingInput(
		t,
		"tei.database.postgres",
		core.CanonicalID("database.query.postgres"),
		"web-browser",
		"coalesced-runtime",
		"b",
		"7",
	)
	plan, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{
		oracle, postgres,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.LogicalProviderCount != 2 ||
		plan.PhysicalArtifactCount != 1 ||
		plan.CoalescedArtifactCount != 1 ||
		plan.SplitPluginCount != 0 {
		t.Fatalf("coalesced packaging counts=%+v", plan)
	}
	if len(plan.Artifacts) != 1 || len(plan.Artifacts[0].Providers) != 2 {
		t.Fatalf("coalesced packaging artifacts=%+v", plan.Artifacts)
	}
	if plan.Artifacts[0].Providers[0].PluginID ==
		plan.Artifacts[0].Providers[1].PluginID {
		t.Fatal("coalesced packaging collapsed logical Plugin identity")
	}
}

func TestTargetPackagingPlanRepresentsSplitPluginArtifacts(t *testing.T) {
	contract := portableRuntimeContract()
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.split_packaging"),
		Version: "1.0.0",
	}
	left := syntheticPackagingInput(
		t, definition, "split.left", contract, "web-browser",
		"split-left", "c", "3",
	)
	right := syntheticPackagingInput(
		t, definition, "split.right", contract, "web-browser",
		"split-right", "d", "4",
	)
	plan, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{
		left, right,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.LogicalProviderCount != 2 ||
		plan.PhysicalArtifactCount != 2 ||
		plan.CoalescedArtifactCount != 0 ||
		plan.SplitPluginCount != 1 {
		t.Fatalf("split packaging counts=%+v", plan)
	}
}

func TestTargetPackagingPlanIsInputOrderIndependent(t *testing.T) {
	oracle := legacyPackagingInput(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
		"web-browser",
		"coalesced-runtime",
		"a",
		"7",
	)
	postgres := legacyPackagingInput(
		t,
		"tei.database.postgres",
		core.CanonicalID("database.query.postgres"),
		"web-browser",
		"coalesced-runtime",
		"b",
		"7",
	)
	left, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{oracle, postgres})
	if err != nil {
		t.Fatal(err)
	}
	right, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{postgres, oracle})
	if err != nil {
		t.Fatal(err)
	}
	if left.PlanFingerprint != right.PlanFingerprint {
		t.Fatalf("packaging fingerprint depends on input order: %s != %s", left.PlanFingerprint, right.PlanFingerprint)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("packaging plan depends on input order: left=%+v right=%+v", left, right)
	}
}

func TestTargetPackagingPlanRejectsMixedTargets(t *testing.T) {
	contract := portableRuntimeContract()
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.mixed_target"),
		Version: "1.0.0",
	}
	web := syntheticPackagingInput(
		t, definition, "mixed.web", contract, "web-browser",
		"web", "a", "1",
	)
	desktop := syntheticPackagingInput(
		t, definition, "mixed.desktop", contract, "desktop",
		"desktop", "b", "2",
	)
	if _, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{
		web, desktop,
	}); err == nil || !strings.Contains(err.Error(), "Targets") {
		t.Fatalf("mixed Target packaging was accepted: %v", err)
	}
}

func TestTargetPackagingPlanRejectsOneLogicalProviderOnMultipleArtifacts(t *testing.T) {
	contract := portableRuntimeContract()
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.duplicate_provider"),
		Version: "1.0.0",
	}
	first := syntheticPackagingInput(
		t, definition, "same.runtime", contract, "web-browser",
		"first", "a", "1",
	)
	second := syntheticPackagingInput(
		t, definition, "same.runtime", contract, "web-browser",
		"second", "b", "2",
	)
	if _, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{
		first, second,
	}); err == nil || !strings.Contains(err.Error(), "multiple artifacts") {
		t.Fatalf("duplicate logical Provider artifacts were accepted: %v", err)
	}
}

func TestTargetPackagingPlanValidationRejectsStaleCountsAndNormalizesOrder(t *testing.T) {
	oracle := legacyPackagingInput(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
		"web-browser",
		"coalesced-runtime",
		"a",
		"7",
	)
	postgres := legacyPackagingInput(
		t,
		"tei.database.postgres",
		core.CanonicalID("database.query.postgres"),
		"web-browser",
		"coalesced-runtime",
		"b",
		"7",
	)
	plan, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{oracle, postgres})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTargetCapabilityPackagingPlan(plan); err != nil {
		t.Fatal(err)
	}

	reordered := plan
	reordered.Artifacts = append([]TargetPackagingArtifact(nil), plan.Artifacts...)
	reordered.Artifacts[0].Providers = append(
		[]TargetPackagingProviderRef(nil),
		plan.Artifacts[0].Providers...,
	)
	reordered.Artifacts[0].Providers[0], reordered.Artifacts[0].Providers[1] =
		reordered.Artifacts[0].Providers[1], reordered.Artifacts[0].Providers[0]
	if targetCapabilityPackagingPlanFingerprint(reordered) != plan.PlanFingerprint {
		t.Fatal("provider declaration order changed normalized packaging identity")
	}

	stale := plan
	stale.LogicalProviderCount++
	stale.PlanFingerprint = targetCapabilityPackagingPlanFingerprint(stale)
	if err := validateTargetCapabilityPackagingPlan(stale); err == nil ||
		!strings.Contains(err.Error(), "counts") {
		t.Fatalf("stale packaging counts were accepted: %v", err)
	}
}


func TestTargetPackagingPlanAllowsOneImplementationToSatisfyMultipleContracts(t *testing.T) {
	firstContract := portableRuntimeContract()
	secondContract := firstContract
	secondContract.ID = "application-runtime.reference-portable-secondary"
	secondContract.ConformanceProfile = "tei.reference.portable-runtime-secondary/v0"

	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.multi_contract"),
		Version: "1.0.0",
	}
	logical := Provider{
		PluginID:            string(definition.ID),
		PluginVersion:       definition.Version,
		ImplementationID:    "multi-contract.runtime",
		Scope:               ScopeApplicationRuntime,
		ContractID:          firstContract.ID,
		ContractVersion:     firstContract.Version,
		ContractFingerprint: fingerprint(firstContract),
		ConformanceProfile:  firstContract.ConformanceProfile,
	}
	plan := providerRuntimeBuildPlan(
		logical.PluginID,
		logical.PluginVersion,
		plugin.BuildDefinitionFingerprint(definition),
		"web-browser",
		"runtime",
		"cmd/multi-contract",
		"a",
	)
	buildLock, _ := buildLockForPlan(t, plan, "5")
	binding, err := bindProviderToBuildArtifact(logical, plan, buildLock)
	if err != nil {
		t.Fatal(err)
	}
	firstProvider, err := realizeProviderFromBuild(logical, binding)
	if err != nil {
		t.Fatal(err)
	}

	secondLogical := logical
	secondLogical.ContractID = secondContract.ID
	secondLogical.ContractVersion = secondContract.Version
	secondLogical.ContractFingerprint = fingerprint(secondContract)
	secondLogical.ConformanceProfile = secondContract.ConformanceProfile
	secondProvider, err := realizeProviderFromBuild(secondLogical, binding)
	if err != nil {
		t.Fatal(err)
	}

	firstInput := targetResolveInput(firstContract, "web-browser", firstProvider)
	firstResolution, err := resolveCapability(firstInput)
	if err != nil {
		t.Fatal(err)
	}
	firstLock, err := buildCapabilityResolutionLock(firstInput, firstResolution)
	if err != nil {
		t.Fatal(err)
	}

	secondInput := targetResolveInput(secondContract, "web-browser", secondProvider)
	secondResolution, err := resolveCapability(secondInput)
	if err != nil {
		t.Fatal(err)
	}
	secondLock, err := buildCapabilityResolutionLock(secondInput, secondResolution)
	if err != nil {
		t.Fatal(err)
	}

	packaging, err := buildTargetCapabilityPackagingPlan([]TargetPackagingInput{
		{CapabilityLock: firstLock, BuildRealization: binding},
		{CapabilityLock: secondLock, BuildRealization: binding},
	})
	if err != nil {
		t.Fatal(err)
	}
	if packaging.LogicalProviderCount != 1 ||
		packaging.PhysicalArtifactCount != 1 ||
		packaging.CoalescedArtifactCount != 0 ||
		packaging.SplitPluginCount != 0 {
		t.Fatalf("multi-Contract implementation was miscounted: %+v", packaging)
	}
	if len(packaging.Artifacts) != 1 ||
		len(packaging.Artifacts[0].Providers) != 2 {
		t.Fatalf("multi-Contract refs were lost: %+v", packaging.Artifacts)
	}
}


func TestTargetPackagingPlanIsExplicitlyNonAuthoritative(t *testing.T) {
	oracle := legacyPackagingInput(
		t,
		"tei.database.oracle",
		core.CanonicalID("database.query.oracle"),
		"web-browser",
		"runtime",
		"a",
		"7",
	)
	plan, err := buildTargetCapabilityPackagingPlan(
		[]TargetPackagingInput{oracle},
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Authority != targetCapabilityPackagingPlanAuthority {
		t.Fatalf("packaging authority=%q", plan.Authority)
	}

	tampered := plan
	tampered.Authority = "runtime-authority"
	tampered.PlanFingerprint =
		targetCapabilityPackagingPlanFingerprint(tampered)
	if err := validateTargetCapabilityPackagingPlan(tampered); err == nil ||
		!strings.Contains(err.Error(), "authority") {
		t.Fatalf("authoritative packaging plan was accepted: %v", err)
	}
}


func TestTargetPackagingPlanReplayDetectsBuildProvenanceDrift(t *testing.T) {
	contract := portableRuntimeContract()
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.packaging_replay"),
		Version: "1.0.0",
	}
	first := syntheticPackagingInput(
		t, definition, "replay.runtime", contract, "web-browser",
		"runtime", "a", "5",
	)
	plan, err := buildTargetCapabilityPackagingPlan(
		[]TargetPackagingInput{first},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyTargetCapabilityPackagingPlan(
		plan,
		[]TargetPackagingInput{first},
	); err != nil {
		t.Fatal(err)
	}

	second := syntheticPackagingInput(
		t, definition, "replay.runtime", contract, "web-browser",
		"runtime", "b", "5",
	)
	if first.CapabilityLock.LockFingerprint !=
		second.CapabilityLock.LockFingerprint {
		t.Fatal("BuildPlan-only drift unexpectedly changed Capability Lock")
	}
	if first.BuildRealization.BindingFingerprint ==
		second.BuildRealization.BindingFingerprint {
		t.Fatal("BuildPlan drift did not change build realization")
	}
	if first.BuildRealization.ArtifactDigest !=
		second.BuildRealization.ArtifactDigest {
		t.Fatal("test requires unchanged artifact bytes")
	}
	if err := verifyTargetCapabilityPackagingPlan(
		plan,
		[]TargetPackagingInput{second},
	); err == nil || !strings.Contains(err.Error(), "current Capability / Build realizations") {
		t.Fatalf("stale packaging plan survived BuildPlan drift: %v", err)
	}
}
