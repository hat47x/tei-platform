package main

import (
	"fmt"
	"strings"

	"tei/internal/plugin"
)

const bootstrapSeedFormatVersion = 1
const bootstrapStage0 = "stage0"

// autoSeedSources are labels that would hide an automatic selection behind the
// explicit seed API. The resolver never scans providers, PATH, install order or
// the catalog for a seed; these labels are rejected so a caller cannot present
// an automatic choice as an explicit stage0 seed.
var autoSeedSources = map[string]struct{}{
	"auto":          {},
	"automatic":     {},
	"default":       {},
	"detected":      {},
	"path":          {},
	"catalog":       {},
	"install-order": {},
	"provider-scan": {},
}

// BootstrapSeed is an explicit, caller-supplied stage0 BuilderIdentity. It is
// not discovered from the Builder catalog or any provider resolution. The
// caller must name the exact identity and its provenance.
type BootstrapSeed struct {
	Stage   string                 `json:"stage"`
	Source  string                 `json:"source"`
	Builder plugin.BuilderIdentity `json:"builder"`
}

// BootstrapStage0Binding records the accepted explicit seed. It grants no
// provider conformance, Build Acceptance or Runtime Authority.
type BootstrapStage0Binding struct {
	FormatVersion      int                    `json:"formatVersion"`
	Stage              string                 `json:"stage"`
	Source             string                 `json:"source"`
	Builder            plugin.BuilderIdentity `json:"builder"`
	BindingFingerprint string                 `json:"bindingFingerprint"`
}

func validateBootstrapSeed(seed BootstrapSeed) error {
	if seed.Stage != bootstrapStage0 {
		return fmt.Errorf("stage0 seed must declare stage %q", bootstrapStage0)
	}
	source := strings.TrimSpace(seed.Source)
	if source == "" {
		return fmt.Errorf("stage0 seed provenance is required")
	}
	if _, automatic := autoSeedSources[strings.ToLower(source)]; automatic {
		return fmt.Errorf("stage0 seed provenance %q looks auto-detected; the seed must be explicit", seed.Source)
	}
	if strings.TrimSpace(string(seed.Builder.ID)) == "" ||
		strings.TrimSpace(seed.Builder.Version) == "" ||
		strings.TrimSpace(seed.Builder.ContractVersion) == "" {
		return fmt.Errorf("stage0 seed BuilderIdentity is incomplete")
	}
	if _, err := canonicalDigestChecked(seed.Builder.ArtifactDigest); err != nil {
		return fmt.Errorf("stage0 seed artifact: %w", err)
	}
	return nil
}

// Validate checks an already bound stage0 seed and re-derives its fingerprint
// from the stored fields, so a tampered provenance or identity is rejected.
func (b BootstrapStage0Binding) Validate() error {
	if b.FormatVersion != bootstrapSeedFormatVersion {
		return fmt.Errorf("unsupported stage0 seed binding format %d", b.FormatVersion)
	}
	if err := validateBootstrapSeed(BootstrapSeed{Stage: b.Stage, Source: b.Source, Builder: b.Builder}); err != nil {
		return err
	}
	if err := validateFingerprint("stage0 seed binding", b.BindingFingerprint); err != nil {
		return err
	}
	expected, err := bootstrapSeedFingerprint(b)
	if err != nil {
		return err
	}
	if canonicalFingerprint(b.BindingFingerprint) != expected {
		return fmt.Errorf("stage0 seed binding fingerprint does not match its fields")
	}
	return nil
}

// bindBootstrapSeed accepts only an explicit seed. It has no catalog argument,
// so it cannot auto-select a provider.
func bindBootstrapSeed(seed BootstrapSeed) (BootstrapStage0Binding, error) {
	if err := validateBootstrapSeed(seed); err != nil {
		return BootstrapStage0Binding{}, err
	}
	binding := BootstrapStage0Binding{
		FormatVersion: bootstrapSeedFormatVersion,
		Stage:         bootstrapStage0,
		Source:        strings.TrimSpace(seed.Source),
		Builder:       canonicalBuilderIdentity(seed.Builder),
	}
	binding.Builder.ArtifactDigest = canonicalDigest(binding.Builder.ArtifactDigest)
	fingerprint, err := bootstrapSeedFingerprint(binding)
	if err != nil {
		return BootstrapStage0Binding{}, err
	}
	binding.BindingFingerprint = fingerprint
	if err := binding.Validate(); err != nil {
		return BootstrapStage0Binding{}, err
	}
	return binding, nil
}

func bootstrapSeedFingerprint(binding BootstrapStage0Binding) (string, error) {
	seed := BootstrapSeed{Stage: binding.Stage, Source: binding.Source, Builder: binding.Builder}
	if err := validateBootstrapSeed(seed); err != nil {
		return "", err
	}
	normalized := binding
	normalized.Source = strings.TrimSpace(binding.Source)
	normalized.Builder = canonicalBuilderIdentity(binding.Builder)
	normalized.Builder.ArtifactDigest = canonicalDigest(normalized.Builder.ArtifactDigest)
	return fingerprint(struct {
		FormatVersion int                    `json:"formatVersion"`
		Stage         string                 `json:"stage"`
		Source        string                 `json:"source"`
		Builder       plugin.BuilderIdentity `json:"builder"`
	}{
		FormatVersion: normalized.FormatVersion,
		Stage:         normalized.Stage,
		Source:        normalized.Source,
		Builder:       normalized.Builder,
	})
}

// resolveStage1AfterExplicitSeed admits a validated stage0 seed, then requires
// a normal provider resolution for stage1+. The seed BuilderIdentity is never
// passed to resolveBuilder and is never used as a fallback when no provider is
// eligible.
func resolveStage1AfterExplicitSeed(
	seed BootstrapStage0Binding,
	intent BuilderIntent,
	catalog BuilderCatalog,
	wiring *BuilderWiring,
) (BuilderResolution, error) {
	if err := seed.Validate(); err != nil {
		return BuilderResolution{}, fmt.Errorf("stage0 seed: %w", err)
	}
	resolution, err := resolveBuilder(intent, catalog, wiring)
	if err != nil {
		return BuilderResolution{}, fmt.Errorf(
			"stage1+ builder provider resolution is required; the stage0 seed is not an automatic default: %w", err)
	}
	return resolution, nil
}
