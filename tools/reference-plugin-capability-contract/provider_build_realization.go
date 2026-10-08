package main

import (
	"fmt"
	"strings"

	"tei/internal/plugin"
)

const providerBuildRealizationSchema = "tei.reference.provider-build-realization/v0"

// ProviderBuildRealization is a derived bridge from one logical Capability
// Provider to an exact BuildPlan and locked build output. It does not grant
// Conformance or Runtime Authority.
type ProviderBuildRealization struct {
	Schema                           string `json:"schema"`
	PluginID                         string `json:"pluginId"`
	PluginVersion                    string `json:"pluginVersion"`
	ImplementationID                 string `json:"implementationId"`
	SourceDeclarationFingerprint     string `json:"sourceDeclarationFingerprint,omitempty"`
	PluginBuildDefinitionFingerprint string `json:"pluginBuildDefinitionFingerprint"`
	Target                           string `json:"target"`
	Profile                          string `json:"profile,omitempty"`
	BuildPlanFingerprint             string `json:"buildPlanFingerprint"`
	ArtifactDigest                   string `json:"artifactDigest"`
	BindingFingerprint               string `json:"bindingFingerprint"`
}

func bindProviderToBuildArtifact(
	logical Provider,
	plan plugin.BuildPlan,
	buildLock plugin.BuildArtifactLock,
) (ProviderBuildRealization, error) {
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
	if err := plan.Validate(); err != nil {
		return ProviderBuildRealization{}, err
	}
	if string(plan.Plugin.ID) != logical.PluginID ||
		plan.Plugin.Version != logical.PluginVersion {
		return ProviderBuildRealization{}, fmt.Errorf(
			"BuildPlan Plugin %s@%s differs from logical provider %s@%s",
			plan.Plugin.ID,
			plan.Plugin.Version,
			logical.PluginID,
			logical.PluginVersion,
		)
	}
	if plan.OutputKind != plugin.BuildOutputRuntimeArtifact {
		return ProviderBuildRealization{}, fmt.Errorf(
			"provider realization requires %s BuildPlan output, got %s",
			plugin.BuildOutputRuntimeArtifact,
			plan.OutputKind,
		)
	}
	if strings.TrimSpace(plan.Target) == "" ||
		plan.Target != strings.TrimSpace(plan.Target) {
		return ProviderBuildRealization{}, fmt.Errorf(
			"provider BuildPlan target is not canonical",
		)
	}
	if plan.Profile != strings.TrimSpace(plan.Profile) {
		return ProviderBuildRealization{}, fmt.Errorf(
			"provider BuildPlan profile has surrounding whitespace",
		)
	}
	if logical.RealizationTarget != "" &&
		logical.RealizationTarget != plan.Target {
		return ProviderBuildRealization{}, fmt.Errorf(
			"logical provider target %q differs from BuildPlan target %q",
			logical.RealizationTarget,
			plan.Target,
		)
	}
	if err := buildLock.Validate(); err != nil {
		return ProviderBuildRealization{}, err
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return ProviderBuildRealization{}, err
	}

	var matches []plugin.LockedBuild
	for _, build := range buildLock.Builds {
		if build.Plugin != plan.Plugin ||
			build.OutputKind != plan.OutputKind ||
			build.Target != plan.Target ||
			build.Profile != plan.Profile ||
			build.PlanFingerprint != planFingerprint {
			continue
		}
		matches = append(matches, build)
	}
	if len(matches) != 1 {
		return ProviderBuildRealization{}, fmt.Errorf(
			"provider %s@%s/%s requires exactly one locked output for BuildPlan %s; found %d",
			logical.PluginID,
			logical.PluginVersion,
			logical.ImplementationID,
			planFingerprint,
			len(matches),
		)
	}

	build := matches[0]
	binding := ProviderBuildRealization{
		Schema:                           providerBuildRealizationSchema,
		PluginID:                         logical.PluginID,
		PluginVersion:                    logical.PluginVersion,
		ImplementationID:                 logical.ImplementationID,
		SourceDeclarationFingerprint:     logical.SourceDeclarationFingerprint,
		PluginBuildDefinitionFingerprint: plan.PluginDefinitionFingerprint,
		Target:                           plan.Target,
		Profile:                          plan.Profile,
		BuildPlanFingerprint:             planFingerprint,
		ArtifactDigest:                   build.ArtifactDigest,
	}
	binding.BindingFingerprint = providerBuildRealizationFingerprint(binding)
	if err := validateProviderBuildRealization(binding); err != nil {
		return ProviderBuildRealization{}, err
	}
	return binding, nil
}

func bindLegacyProviderToBuildArtifact(
	definition plugin.Definition,
	mapping LegacyCapabilityMapping,
	plan plugin.BuildPlan,
	buildLock plugin.BuildArtifactLock,
) (ProviderBuildRealization, Provider, error) {
	logical, err := projectLegacyCapabilityProvider(definition, mapping)
	if err != nil {
		return ProviderBuildRealization{}, Provider{}, err
	}
	if plan.PluginDefinitionFingerprint != plugin.BuildDefinitionFingerprint(definition) {
		return ProviderBuildRealization{}, Provider{}, fmt.Errorf(
			"BuildPlan definition fingerprint does not match authoritative legacy Plugin definition",
		)
	}
	binding, err := bindProviderToBuildArtifact(logical, plan, buildLock)
	if err != nil {
		return ProviderBuildRealization{}, Provider{}, err
	}
	realized, err := realizeProviderFromBuild(logical, binding)
	if err != nil {
		return ProviderBuildRealization{}, Provider{}, err
	}
	return binding, realized, nil
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
	plan plugin.BuildPlan,
	buildLock plugin.BuildArtifactLock,
) error {
	expected, err := bindProviderToBuildArtifact(
		logical,
		plan,
		buildLock,
	)
	if err != nil {
		return err
	}
	if binding != expected {
		return fmt.Errorf(
			"provider build realization differs from current BuildPlan / BuildArtifactLock",
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
	if !validHex64(binding.PluginBuildDefinitionFingerprint) {
		return fmt.Errorf(
			"provider build realization Plugin definition fingerprint is invalid",
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
