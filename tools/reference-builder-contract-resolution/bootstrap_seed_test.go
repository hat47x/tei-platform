package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func validBootstrapSeed() BootstrapSeed {
	return BootstrapSeed{
		Stage:  bootstrapStage0,
		Source: "explicit-caller",
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.stage0_seed"),
			Version:         "1.0.0",
			ContractVersion: "tei.reference.build/v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("e", 64),
		},
	}
}

func TestBootstrapSeedBindsExplicitStage0Identity(t *testing.T) {
	seed := validBootstrapSeed()
	binding, err := bindBootstrapSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	if err := binding.Validate(); err != nil {
		t.Fatal(err)
	}
	if binding.Stage != bootstrapStage0 || binding.Source != "explicit-caller" {
		t.Fatalf("stage0 provenance was not retained: %+v", binding)
	}
	if binding.Builder.ID != seed.Builder.ID ||
		binding.Builder.Version != seed.Builder.Version ||
		binding.Builder.ContractVersion != seed.Builder.ContractVersion ||
		binding.Builder.ArtifactDigest != seed.Builder.ArtifactDigest {
		t.Fatalf("stage0 BuilderIdentity changed: %+v", binding.Builder)
	}
	first, err := bootstrapSeedFingerprint(binding)
	if err != nil {
		t.Fatal(err)
	}
	second, err := bootstrapSeedFingerprint(binding)
	if err != nil || first != second {
		t.Fatalf("stage0 fingerprint is not deterministic: %q %q %v", first, second, err)
	}
	if first != binding.BindingFingerprint {
		t.Fatalf("binding fingerprint does not match its identity: %q %q", first, binding.BindingFingerprint)
	}
}

func TestBootstrapSeedRequiresExplicitIdentity(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*BootstrapSeed)
	}{
		{"missing stage", func(s *BootstrapSeed) { s.Stage = "" }},
		{"wrong stage", func(s *BootstrapSeed) { s.Stage = "stage1" }},
		{"missing source", func(s *BootstrapSeed) { s.Source = " " }},
		{"auto source", func(s *BootstrapSeed) { s.Source = "auto" }},
		{"default source", func(s *BootstrapSeed) { s.Source = "default" }},
		{"catalog source", func(s *BootstrapSeed) { s.Source = "catalog" }},
		{"missing id", func(s *BootstrapSeed) { s.Builder.ID = "" }},
		{"missing version", func(s *BootstrapSeed) { s.Builder.Version = "" }},
		{"missing contract version", func(s *BootstrapSeed) { s.Builder.ContractVersion = "" }},
		{"missing digest", func(s *BootstrapSeed) { s.Builder.ArtifactDigest = "" }},
		{"malformed digest", func(s *BootstrapSeed) { s.Builder.ArtifactDigest = "sha256:not-hex" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seed := validBootstrapSeed()
			tc.mutate(&seed)
			if _, err := bindBootstrapSeed(seed); err == nil {
				t.Fatalf("auto or incomplete seed was accepted: %+v", seed)
			}
		})
	}
}

func TestBootstrapSeedFingerprintFollowsIdentity(t *testing.T) {
	base := validBootstrapSeed()
	baseBinding, err := bindBootstrapSeed(base)
	if err != nil {
		t.Fatal(err)
	}
	baseFingerprint := baseBinding.BindingFingerprint

	changedSource := validBootstrapSeed()
	changedSource.Source = "explicit-host-config"
	sourceBinding, err := bindBootstrapSeed(changedSource)
	if err != nil {
		t.Fatal(err)
	}
	if sourceBinding.BindingFingerprint == baseFingerprint {
		t.Fatal("stage0 fingerprint ignored the provenance")
	}

	changedDigest := validBootstrapSeed()
	changedDigest.Builder.ArtifactDigest = "sha256:" + strings.ToUpper(strings.Repeat("f", 64))
	digestBinding, err := bindBootstrapSeed(changedDigest)
	if err != nil {
		t.Fatal(err)
	}
	if digestBinding.BindingFingerprint == baseFingerprint {
		t.Fatal("stage0 fingerprint ignored the artifact digest")
	}
	if !strings.HasSuffix(digestBinding.Builder.ArtifactDigest, strings.Repeat("f", 64)) ||
		strings.Contains(digestBinding.Builder.ArtifactDigest, "F") {
		t.Fatalf("stage0 digest was not canonicalized: %q", digestBinding.Builder.ArtifactDigest)
	}

	tampered := baseBinding
	tampered.BindingFingerprint = strings.Repeat("0", 64)
	if err := tampered.Validate(); err == nil {
		t.Fatal("tampered stage0 binding was accepted")
	}
}

func TestStage1ResolutionNeverFallsBackToStage0Seed(t *testing.T) {
	seed := validBootstrapSeed()
	binding, err := bindBootstrapSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	intent := BuilderIntent{
		ContractID:      "tei.reference.builder.copy",
		ContractVersion: "1.0.0",
		RecipeKind:      "deterministic-copy",
		Target:          "host/reference",
		Profile:         "reference",
	}
	if _, err := resolveStage1AfterExplicitSeed(binding, intent, BuilderCatalog{}, nil); err == nil ||
		!strings.Contains(err.Error(), "not an automatic default") {
		t.Fatalf("empty catalog fell back to the stage0 seed: %v", err)
	}

	contract, providers, evidence, wiring := referenceFixture()
	catalog := BuilderCatalog{Contracts: []BuilderContract{contract}, Providers: providers, Evidence: evidence}
	resolution, err := resolveStage1AfterExplicitSeed(binding, intent, catalog, &wiring)
	if err != nil {
		t.Fatal(err)
	}
	if digestEqual(resolution.Builder.ArtifactDigest, seed.Builder.ArtifactDigest) ||
		resolution.Builder.ID == seed.Builder.ID {
		t.Fatalf("stage1 resolution reused the stage0 seed as a provider: %+v", resolution.Builder)
	}
}

func TestStage1ResolutionRequiresValidatedSeed(t *testing.T) {
	intent := BuilderIntent{
		ContractID:      "tei.reference.builder.copy",
		ContractVersion: "1.0.0",
		RecipeKind:      "deterministic-copy",
		Target:          "host/reference",
		Profile:         "reference",
	}
	if _, err := resolveStage1AfterExplicitSeed(BootstrapStage0Binding{}, intent, BuilderCatalog{}, nil); err == nil {
		t.Fatal("invalid stage0 binding was accepted")
	}
}
