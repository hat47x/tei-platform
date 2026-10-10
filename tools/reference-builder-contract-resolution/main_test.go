package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func referenceIntent(contract BuilderContract) BuilderIntent {
	return BuilderIntent{
		ContractID:      contract.ID,
		ContractVersion: contract.Version,
		RecipeKind:      "deterministic-copy",
		Target:          "host/reference",
		Profile:         "reference",
	}
}

func buildPlanFromResolution(resolution BuilderResolution) plugin.BuildPlan {
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID("tei.reference.resolved_builder_plugin"), Version: "1.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources: []plugin.BuildSource{{
			Kind:    "reference-source",
			ID:      "plugin-source",
			Locator: "reference://plugin-source",
			Digest:  "sha256:" + strings.Repeat("b", 64),
		}},
		Builder: resolution.Builder,
		Recipe: plugin.BuildRecipe{
			Kind:       "deterministic-copy",
			Entrypoint: "plugin-source",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host/reference",
		Profile:    "reference",
	}
}

func TestExplicitWiringSelectsExactBuilderProvider(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	resolution, err := resolveBuilder(intent, BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: providers,
		Evidence:  evidence,
	}, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Provider.ProviderID != providers[0].ProviderID ||
		resolution.Provider.ImplementationID != providers[0].ImplementationID ||
		!digestEqual(resolution.Provider.ArtifactDigest, providers[0].ArtifactDigest) {
		t.Fatalf("resolved provider: %#v", resolution.Provider)
	}
	if resolution.Builder.ID != providers[0].ProviderID ||
		resolution.Builder.Version != providers[0].ProviderVersion ||
		!digestEqual(resolution.Builder.ArtifactDigest, providers[0].ArtifactDigest) {
		t.Fatalf("exact BuilderIdentity: %#v", resolution.Builder)
	}
	if resolution.Builder.ContractVersion != contract.ID+"@"+contract.Version {
		t.Fatalf("builder contract version: %s", resolution.Builder.ContractVersion)
	}
}

func TestFreshResolutionRejectsAmbiguousCompatibleProviders(t *testing.T) {
	contract, providers, evidence, _ := referenceFixture()
	_, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: providers,
		Evidence:  evidence,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous resolution, got %v", err)
	}
}

func TestWiredProviderWithWrongAuthorityScopeIsRejected(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	providers[0].Scope = "application-runtime"
	_, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: providers,
		Evidence:  evidence,
	}, &wiring)
	if err == nil || !strings.Contains(err.Error(), "not build-tool") {
		t.Fatalf("wrong-scope provider error: %v", err)
	}
}

func TestContractFingerprintDriftIsRejected(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	providers[0].ContractFingerprint = strings.Repeat("f", 64)
	_, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: providers,
		Evidence:  evidence,
	}, &wiring)
	if err == nil || !strings.Contains(err.Error(), "fingerprint drifted") {
		t.Fatalf("contract drift error: %v", err)
	}
}

func TestProviderWithoutConformanceEvidenceIsRejected(t *testing.T) {
	contract, providers, _, wiring := referenceFixture()
	_, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: providers,
	}, &wiring)
	if err == nil || !strings.Contains(err.Error(), "Conformance Evidence") {
		t.Fatalf("missing conformance error: %v", err)
	}
}

func TestUnsupportedRecipeTargetAndProfileAreRejected(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	for _, mutate := range []func(*BuilderIntent){
		func(intent *BuilderIntent) { intent.RecipeKind = "unsupported-recipe" },
		func(intent *BuilderIntent) { intent.Target = "host/other" },
		func(intent *BuilderIntent) { intent.Profile = "other" },
	} {
		intent := referenceIntent(contract)
		mutate(&intent)
		if _, err := resolveBuilder(intent, BuilderCatalog{
			Contracts: []BuilderContract{contract},
			Providers: providers,
			Evidence:  evidence,
		}, &wiring); err == nil {
			t.Fatalf("unsupported intent was accepted: %#v", intent)
		}
	}
}

func TestLockedReplaySurvivesNewCompatibleProviderWhileFreshResolutionBecomesAmbiguous(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	initial := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	resolution, err := resolveBuilder(intent, initial, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockBuilderResolution(resolution)
	if err != nil {
		t.Fatal(err)
	}

	expanded := initial
	expanded.Providers = append(append([]BuilderProvider(nil), initial.Providers...), providers[1])
	expanded.Evidence = append(append([]BuilderConformanceEvidence(nil), initial.Evidence...), evidence[1])

	if _, err := resolveBuilder(intent, expanded, nil); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("fresh resolution did not become ambiguous: %v", err)
	}
	replayed, err := replayBuilderResolution(intent, expanded, lock)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Provider.ProviderID != resolution.Provider.ProviderID ||
		replayed.Provider.ImplementationID != resolution.Provider.ImplementationID ||
		!digestEqual(replayed.Provider.ArtifactDigest, resolution.Provider.ArtifactDigest) {
		t.Fatalf("locked replay drifted provider: %#v != %#v", replayed.Provider, resolution.Provider)
	}
	if canonicalFingerprint(replayed.ResolutionFingerprint) != canonicalFingerprint(resolution.ResolutionFingerprint) {
		t.Fatalf("locked replay changed resolution fingerprint: %s != %s", replayed.ResolutionFingerprint, resolution.ResolutionFingerprint)
	}
}

func TestLockedReplayRejectsProviderArtifactDrift(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	catalog := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	resolution, err := resolveBuilder(intent, catalog, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockBuilderResolution(resolution)
	if err != nil {
		t.Fatal(err)
	}

	catalog.Providers[0].ArtifactDigest = "sha256:" + strings.Repeat("e", 64)
	catalog.Evidence[0].ArtifactDigest = catalog.Providers[0].ArtifactDigest
	if _, err := replayBuilderResolution(intent, catalog, lock); err == nil {
		t.Fatal("locked replay accepted a different physical builder artifact")
	}
}

func TestLockedReplayRejectsConformanceEvidenceDrift(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	catalog := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	resolution, err := resolveBuilder(intent, catalog, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockBuilderResolution(resolution)
	if err != nil {
		t.Fatal(err)
	}

	catalog.Evidence[0].EvidenceFingerprint = strings.Repeat("e", 64)
	if _, err := replayBuilderResolution(intent, catalog, lock); err == nil ||
		!strings.Contains(err.Error(), "Evidence fingerprint") {
		t.Fatalf("locked replay Evidence drift error: %v", err)
	}
}

func TestWiringWithoutArtifactDigestRejectsPhysicalArtifactAmbiguity(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	clone := providers[0]
	clone.ArtifactDigest = "sha256:" + strings.Repeat("e", 64)
	e2 := evidence[0]
	e2.ArtifactDigest = clone.ArtifactDigest
	e2.EvidenceFingerprint = strings.Repeat("f", 64)

	wiring.ArtifactDigest = ""
	_, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0], clone},
		Evidence:  []BuilderConformanceEvidence{evidence[0], e2},
	}, &wiring)
	if err == nil || !strings.Contains(err.Error(), "ambiguous across physical artifacts") {
		t.Fatalf("physical artifact ambiguity error: %v", err)
	}
}

func TestExactArtifactWiringDisambiguatesPhysicalRealizations(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	clone := providers[0]
	clone.ArtifactDigest = "sha256:" + strings.Repeat("e", 64)
	e2 := evidence[0]
	e2.ArtifactDigest = clone.ArtifactDigest
	e2.EvidenceFingerprint = strings.Repeat("f", 64)

	resolution, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0], clone},
		Evidence:  []BuilderConformanceEvidence{evidence[0], e2},
	}, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	if !digestEqual(resolution.Provider.ArtifactDigest, providers[0].ArtifactDigest) {
		t.Fatalf("explicit artifact wiring selected %s", resolution.Provider.ArtifactDigest)
	}
}

func TestProviderSupportListOrderDoesNotChangeResolutionIdentity(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	base := providers[0]
	base.RecipeKinds = []string{"other-recipe", "deterministic-copy"}
	base.Targets = []string{"host/other", "host/reference"}
	base.Profiles = []string{"other", "reference"}
	catalog := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{base},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	left, err := resolveBuilder(referenceIntent(contract), catalog, &wiring)
	if err != nil {
		t.Fatal(err)
	}

	reversed := base
	reversed.RecipeKinds = []string{"deterministic-copy", "other-recipe"}
	reversed.Targets = []string{"host/reference", "host/other"}
	reversed.Profiles = []string{"reference", "other"}
	catalog.Providers[0] = reversed
	right, err := resolveBuilder(referenceIntent(contract), catalog, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatal("provider support-list declaration order changed resolution identity")
	}
}

func TestDigestHexCaseDoesNotChangeBuilderResolutionIdentity(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	lower, err := resolveBuilder(intent, BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}, &wiring)
	if err != nil {
		t.Fatal(err)
	}

	upperProvider := providers[0]
	upperProvider.ArtifactDigest = "sha256:" + strings.ToUpper(strings.TrimPrefix(upperProvider.ArtifactDigest, "sha256:"))
	upperEvidence := evidence[0]
	upperEvidence.ArtifactDigest = upperProvider.ArtifactDigest
	upperWiring := wiring
	upperWiring.ArtifactDigest = upperProvider.ArtifactDigest
	upper, err := resolveBuilder(intent, BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{upperProvider},
		Evidence:  []BuilderConformanceEvidence{upperEvidence},
	}, &upperWiring)
	if err != nil {
		t.Fatal(err)
	}
	if lower.ResolutionFingerprint != upper.ResolutionFingerprint {
		t.Fatal("valid digest hex case changed builder resolution identity")
	}
}

func TestBuilderResolutionDoesNotPromoteAuthority(t *testing.T) {
	for _, want := range []string{
		"builder-resolution-is-not-build-execution",
		"builder-resolution-is-not-build-acceptance",
		"builder-resolution-is-not-runtime-authority",
		"builder-resolution-is-not-provider-trust",
		"locked-replay-is-not-fresh-resolution",
	} {
		found := false
		for _, got := range authorityLimits {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("authority limit %q missing: %#v", want, authorityLimits)
		}
	}
}

func TestBuilderResolutionUsesCanonicalProviderID(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	providers[0].ProviderID = core.MustCanonicalID("tei.builder.copy_explicit")
	evidence[0].ProviderID = providers[0].ProviderID
	wiring.ProviderID = providers[0].ProviderID
	resolution, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Builder.ID != providers[0].ProviderID {
		t.Fatalf("BuilderIdentity ID=%s want provider=%s", resolution.Builder.ID, providers[0].ProviderID)
	}
}


func TestMalformedExplicitWiringIsRejectedBeforeProviderLookup(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	wiring.ProviderVersion = "not-a-version"
	_, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: providers,
		Evidence:  evidence,
	}, &wiring)
	if err == nil || !strings.Contains(err.Error(), "builder wiring provider version") {
		t.Fatalf("malformed wiring error: %v", err)
	}

	wiring = BuilderWiring{
		ContractID:       contract.ID,
		ProviderID:       providers[0].ProviderID,
		ProviderVersion:  providers[0].ProviderVersion,
		ImplementationID: providers[0].ImplementationID,
		ArtifactDigest:   "sha256:not-hex",
	}
	if _, err := resolveBuilder(referenceIntent(contract), BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: providers,
		Evidence:  evidence,
	}, &wiring); err == nil || !strings.Contains(err.Error(), "builder wiring artifact digest") {
		t.Fatalf("malformed wiring digest error: %v", err)
	}
}

func TestLockedReplayIgnoresUnrelatedSupportExpansion(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	catalog := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	resolution, err := resolveBuilder(intent, catalog, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockBuilderResolution(resolution)
	if err != nil {
		t.Fatal(err)
	}

	expandedProvider := providers[0]
	expandedProvider.RecipeKinds = append(append([]string(nil), expandedProvider.RecipeKinds...), "additional-recipe")
	expandedProvider.Targets = append(append([]string(nil), expandedProvider.Targets...), "host/additional")
	expandedProvider.Profiles = append(append([]string(nil), expandedProvider.Profiles...), "additional-profile")
	catalog.Providers[0] = expandedProvider

	replayed, err := replayBuilderResolution(intent, catalog, lock)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ResolutionFingerprint != resolution.ResolutionFingerprint {
		t.Fatalf("unrelated support expansion changed locked resolution identity: %s != %s", replayed.ResolutionFingerprint, resolution.ResolutionFingerprint)
	}
}


func TestLockRejectsTamperedResolutionCrossFields(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	resolution, err := resolveBuilder(intent, BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}, &wiring)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		edit func(*BuilderResolution)
	}{
		{
			name: "contract semantic drift",
			edit: func(value *BuilderResolution) {
				value.Contract.OutputSemantics = "tampered-output"
			},
		},
		{
			name: "provider contract drift",
			edit: func(value *BuilderResolution) {
				value.Provider.ContractVersion = "9.9.9"
			},
		},
		{
			name: "builder identity drift",
			edit: func(value *BuilderResolution) {
				value.Builder.ID = core.MustCanonicalID("tei.builder.other")
			},
		},
		{
			name: "resolution fingerprint drift",
			edit: func(value *BuilderResolution) {
				value.ResolutionFingerprint = strings.Repeat("f", 64)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tampered := resolution
			tt.edit(&tampered)
			if _, err := lockBuilderResolution(tampered); err == nil {
				t.Fatal("tampered Resolution was accepted for locking")
			}
		})
	}
}

func TestReplayRejectsLockWhoseBuilderDoesNotMatchLockedProvider(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	catalog := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	resolution, err := resolveBuilder(intent, catalog, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockBuilderResolution(resolution)
	if err != nil {
		t.Fatal(err)
	}
	lock.Builder.ID = core.MustCanonicalID("tei.builder.other")
	if _, err := replayBuilderResolution(intent, catalog, lock); err == nil ||
		!strings.Contains(err.Error(), "BuilderIdentity does not match locked Provider") {
		t.Fatalf("mismatched lock BuilderIdentity error: %v", err)
	}
}


func TestResolvedBuildPlanDoesNotImplicitlyChangeAfterCatalogExpansion(t *testing.T) {
	contract, providers, evidence, wiring := referenceFixture()
	intent := referenceIntent(contract)
	initial := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	resolution, err := resolveBuilder(intent, initial, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockBuilderResolution(resolution)
	if err != nil {
		t.Fatal(err)
	}
	plan := buildPlanFromResolution(resolution)
	before, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}

	expanded := initial
	expanded.Providers = append(append([]BuilderProvider(nil), initial.Providers...), providers[1])
	expanded.Evidence = append(append([]BuilderConformanceEvidence(nil), initial.Evidence...), evidence[1])
	if _, err := resolveBuilder(intent, expanded, nil); err == nil {
		t.Fatal("fresh resolution unexpectedly remained unambiguous")
	}

	replayed, err := replayBuilderResolution(intent, expanded, lock)
	if err != nil {
		t.Fatal(err)
	}
	replayedPlan := buildPlanFromResolution(replayed)
	after, err := plugin.BuildPlanFingerprint(replayedPlan)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("catalog expansion changed exact resolved BuildPlan fingerprint: %s != %s", before, after)
	}
	if plan.Builder != replayedPlan.Builder {
		t.Fatalf("catalog expansion changed exact BuildPlan Builder: %#v != %#v", plan.Builder, replayedPlan.Builder)
	}
}
