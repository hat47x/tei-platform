package main

import (
	"fmt"
	"strings"

	"tei/internal/plugin"
)

const providerBuildRealizationSchema = "tei.reference.provider-build-realization/v0"

// ProviderBuildRealization is a derived bridge from one logical Capability
// Provider to an exact build output. It does not grant Runtime Authority.
//
// BuildPlanFingerprint is retained for provenance, but it is deliberately not
// copied into Provider resolution identity. Equal output bytes may be rebuilt
// from a different accepted BuildPlan without changing the callable
// implementation artifact identity.
type ProviderBuildRealization struct {
	Schema                       string `json:"schema"`
	PluginID                     string `json:"pluginId"`
	PluginVersion                string `json:"pluginVersion"`
	ImplementationID             string `json:"implementationId"`
	SourceDeclarationFingerprint string `json:"sourceDeclarationFingerprint,omitempty"`
	Target                       string `json:"target"`
	Profile                      string `json:"profile,omitempty"`
	BuildPlanFingerprint         string `json:"buildPlanFingerprint"`
	ArtifactDigest               string `json:"artifactDigest"`
	BindingFingerprint           string `json:"bindingFingerprint"`
}

func bindProviderToBuildArtifact(
	logical Provider,
	buildLock plugin.BuildArtifactLock,
	target string,
	profile string,
) (ProviderBuildRealization, error) {
	if strings.TrimSpace(target) == "" || target != strings.TrimSpace(target) {
		return ProviderBuildRealization{}, fmt.Errorf(
			"provider build realization requires canonical non-empty target",
		)
	}
	if profile != strings.TrimSpace(profile) {
		return ProviderBuildRealization{}, fmt.Errorf(
			"provider build realization profile has surrounding whitespace",
		)
	}
	if logical.PluginID == "" ||
		logical.PluginVersion == "" ||
		logical.ImplementationID == "" {
		return ProviderBuildRealization{}, fmt.Errorf(
			"logical provider identity is incomplete",
		)
	}
	if _, err := parseVersion(logical.PluginVersion); err != nil {
		return ProviderBuildRealization{}, fmt.Errorf(
			"logical provider plugin version: %w",
			err,
		)
	}
	if logical.ArtifactDigest != "" {
		return ProviderBuildRealization{}, fmt.Errorf(
			"logical provider already binds physical artifact %s",
			logical.ArtifactDigest,
		)
	}
	if logical.RealizationTarget != "" &&
		logical.RealizationTarget != target {
		return ProviderBuildRealization{}, fmt.Errorf(
			"logical provider target %q differs from build target %q",
			logical.RealizationTarget,
			target,
		)
	}
	if err := buildLock.Validate(); err != nil {
		return ProviderBuildRealization{}, err
	}

	var matches []plugin.LockedBuild
	for _, build := range buildLock.Builds {
		if string(build.Plugin.ID) != logical.PluginID ||
			build.Plugin.Version != logical.PluginVersion ||
			build.OutputKind != plugin.BuildOutputRuntimeArtifact ||
			build.Target != target ||
			build.Profile != profile {
			continue
		}
		matches = append(matches, build)
	}
	if len(matches) != 1 {
		return ProviderBuildRealization{}, fmt.Errorf(
			"provider %s@%s/%s requires exactly one runtime build for target %s profile %q; found %d",
			logical.PluginID,
			logical.PluginVersion,
			logical.ImplementationID,
			target,
			profile,
			len(matches),
		)
	}

	build := matches[0]
	binding := ProviderBuildRealization{
		Schema:                       providerBuildRealizationSchema,
		PluginID:                     logical.PluginID,
		PluginVersion:                logical.PluginVersion,
		ImplementationID:             logical.ImplementationID,
		SourceDeclarationFingerprint: logical.SourceDeclarationFingerprint,
		Target:                       target,
		Profile:                      profile,
		BuildPlanFingerprint:         build.PlanFingerprint,
		ArtifactDigest:               build.ArtifactDigest,
	}
	binding.BindingFingerprint = providerBuildRealizationFingerprint(binding)
	if err := validateProviderBuildRealization(binding); err != nil {
		return ProviderBuildRealization{}, err
	}
	return binding, nil
}

func realizeProviderFromBuild(
	logical Provider,
	binding ProviderBuildRealization,
) (Provider, error) {
	if err := validateProviderBuildRealization(binding); err != nil {
		return Provider{}, err
	}
	if logical.PluginID != binding.PluginID ||
		logical.PluginVersion != binding.PluginVersion ||
		logical.ImplementationID != binding.ImplementationID ||
		logical.SourceDeclarationFingerprint != binding.SourceDeclarationFingerprint {
		return Provider{}, fmt.Errorf(
			"provider build realization does not belong to logical provider",
		)
	}
	if logical.ArtifactDigest != "" &&
		logical.ArtifactDigest != binding.ArtifactDigest {
		return Provider{}, fmt.Errorf(
			"logical provider artifact differs from build realization",
		)
	}
	if logical.RealizationTarget != "" &&
		logical.RealizationTarget != binding.Target {
		return Provider{}, fmt.Errorf(
			"logical provider target differs from build realization",
		)
	}

	out := logical
	out.RealizationTarget = binding.Target
	out.ArtifactDigest = binding.ArtifactDigest
	return out, nil
}

func verifyProviderBuildRealization(
	binding ProviderBuildRealization,
	logical Provider,
	buildLock plugin.BuildArtifactLock,
) error {
	expected, err := bindProviderToBuildArtifact(
		logical,
		buildLock,
		binding.Target,
		binding.Profile,
	)
	if err != nil {
		return err
	}
	if binding != expected {
		return fmt.Errorf(
			"provider build realization differs from current BuildArtifactLock",
		)
	}
	return nil
}

func validateProviderBuildRealization(
	binding ProviderBuildRealization,
) error {
	if binding.Schema != providerBuildRealizationSchema {
		return fmt.Errorf(
			"provider build realization schema %q is not supported",
			binding.Schema,
		)
	}
	if binding.PluginID == "" ||
		binding.PluginVersion == "" ||
		binding.ImplementationID == "" {
		return fmt.Errorf("provider build realization identity is incomplete")
	}
	if _, err := parseVersion(binding.PluginVersion); err != nil {
		return fmt.Errorf("provider build realization plugin version: %w", err)
	}
	if binding.SourceDeclarationFingerprint != "" &&
		!validSHA256Fingerprint(binding.SourceDeclarationFingerprint) {
		return fmt.Errorf(
			"provider build realization source declaration fingerprint is invalid",
		)
	}
	if strings.TrimSpace(binding.Target) == "" ||
		binding.Target != strings.TrimSpace(binding.Target) {
		return fmt.Errorf("provider build realization target is invalid")
	}
	if binding.Profile != strings.TrimSpace(binding.Profile) {
		return fmt.Errorf("provider build realization profile is not canonical")
	}
	if !validHex64(binding.BuildPlanFingerprint) {
		return fmt.Errorf(
			"provider build realization BuildPlan fingerprint is invalid",
		)
	}
	if !validSHA256Fingerprint(binding.ArtifactDigest) {
		return fmt.Errorf(
			"provider build realization artifact digest is invalid",
		)
	}
	if !validSHA256Fingerprint(binding.BindingFingerprint) {
		return fmt.Errorf(
			"provider build realization fingerprint is invalid",
		)
	}
	if providerBuildRealizationFingerprint(binding) != binding.BindingFingerprint {
		return fmt.Errorf("provider build realization fingerprint is stale")
	}
	return nil
}

func providerBuildRealizationFingerprint(
	binding ProviderBuildRealization,
) string {
	binding.BindingFingerprint = ""
	return fingerprint(binding)
}
