package main

import (
	"fmt"
	"sort"
	"strings"
)

const targetCapabilityPackagingPlanSchema = "tei.reference.target-capability-packaging-plan/v0"

type TargetPackagingProviderRef struct {
	PluginID                         string `json:"pluginId"`
	PluginVersion                    string `json:"pluginVersion"`
	ImplementationID                 string `json:"implementationId"`
	SourceDeclarationFingerprint     string `json:"sourceDeclarationFingerprint,omitempty"`
	ContractID                       string `json:"contractId"`
	ContractVersion                  string `json:"contractVersion"`
	ContractFingerprint              string `json:"contractFingerprint"`
	CapabilityLockFingerprint        string `json:"capabilityLockFingerprint"`
	ProviderBuildRealizationFingerprint string `json:"providerBuildRealizationFingerprint"`
}

type TargetPackagingArtifact struct {
	ArtifactDigest string                       `json:"artifactDigest"`
	Providers      []TargetPackagingProviderRef `json:"providers"`
}

type TargetCapabilityPackagingPlan struct {
	Schema                 string                    `json:"schema"`
	Scope                  AuthorityScope            `json:"scope"`
	Target                 string                    `json:"target"`
	LogicalProviderCount   int                       `json:"logicalProviderCount"`
	PhysicalArtifactCount  int                       `json:"physicalArtifactCount"`
	CoalescedArtifactCount int                       `json:"coalescedArtifactCount"`
	SplitPluginCount       int                       `json:"splitPluginCount"`
	Artifacts              []TargetPackagingArtifact `json:"artifacts"`
	PlanFingerprint        string                    `json:"planFingerprint"`
}

type TargetPackagingInput struct {
	CapabilityLock   CapabilityResolutionLock
	BuildRealization ProviderBuildRealization
}

func buildTargetCapabilityPackagingPlan(
	inputs []TargetPackagingInput,
) (TargetCapabilityPackagingPlan, error) {
	if len(inputs) == 0 {
		return TargetCapabilityPackagingPlan{}, fmt.Errorf(
			"target packaging plan requires at least one Provider",
		)
	}

	first := inputs[0]
	if err := validateCapabilityResolutionLock(first.CapabilityLock); err != nil {
		return TargetCapabilityPackagingPlan{}, err
	}
	if err := validateProviderBuildRealization(first.BuildRealization); err != nil {
		return TargetCapabilityPackagingPlan{}, err
	}
	if first.CapabilityLock.RealizationTarget == "" {
		return TargetCapabilityPackagingPlan{}, fmt.Errorf(
			"target packaging requires Target-specific Capability Lock",
		)
	}
	scope := first.CapabilityLock.Scope
	target := first.CapabilityLock.RealizationTarget

	artifactProviders := map[string][]TargetPackagingProviderRef{}
	providerKeys := map[string]string{}
	pluginArtifacts := map[string]map[string]struct{}{}

	for i, input := range inputs {
		lock := input.CapabilityLock
		binding := input.BuildRealization
		if err := validateCapabilityResolutionLock(lock); err != nil {
			return TargetCapabilityPackagingPlan{}, fmt.Errorf(
				"input[%d] Capability Lock: %w", i, err,
			)
		}
		if err := validateProviderBuildRealization(binding); err != nil {
			return TargetCapabilityPackagingPlan{}, fmt.Errorf(
				"input[%d] build realization: %w", i, err,
			)
		}
		if lock.Scope != scope {
			return TargetCapabilityPackagingPlan{}, fmt.Errorf(
				"target packaging cannot mix Authority scopes %q and %q",
				scope, lock.Scope,
			)
		}
		if lock.RealizationTarget != target || binding.Target != target {
			return TargetCapabilityPackagingPlan{}, fmt.Errorf(
				"target packaging cannot mix realization Targets",
			)
		}
		if lock.Provider.PluginID != binding.PluginID ||
			lock.Provider.PluginVersion != binding.PluginVersion ||
			lock.Provider.ImplementationID != binding.ImplementationID ||
			lock.Provider.SourceDeclarationFingerprint != binding.SourceDeclarationFingerprint ||
			lock.Provider.ArtifactDigest != binding.ArtifactDigest ||
			lock.Provider.RealizationTarget != binding.Target {
			return TargetCapabilityPackagingPlan{}, fmt.Errorf(
				"input[%d] Build realization does not describe locked Provider", i,
			)
		}

		providerKey := lock.Provider.PluginID + "\x00" +
			lock.Provider.PluginVersion + "\x00" +
			lock.Provider.ImplementationID
		if previousArtifact, duplicate := providerKeys[providerKey]; duplicate {
			if previousArtifact != binding.ArtifactDigest {
				return TargetCapabilityPackagingPlan{}, fmt.Errorf(
					"logical Provider %s is assigned to multiple artifacts",
					providerKey,
				)
			}
			return TargetCapabilityPackagingPlan{}, fmt.Errorf(
				"logical Provider %s is declared more than once",
				providerKey,
			)
		}
		providerKeys[providerKey] = binding.ArtifactDigest

		ref := TargetPackagingProviderRef{
			PluginID:                         lock.Provider.PluginID,
			PluginVersion:                    lock.Provider.PluginVersion,
			ImplementationID:                 lock.Provider.ImplementationID,
			SourceDeclarationFingerprint:     lock.Provider.SourceDeclarationFingerprint,
			ContractID:                       lock.Contract.ID,
			ContractVersion:                  lock.Contract.Version,
			ContractFingerprint:              lock.ContractFingerprint,
			CapabilityLockFingerprint:        lock.LockFingerprint,
			ProviderBuildRealizationFingerprint: binding.BindingFingerprint,
		}
		artifactProviders[binding.ArtifactDigest] = append(
			artifactProviders[binding.ArtifactDigest],
			ref,
		)

		pluginKey := lock.Provider.PluginID + "@" + lock.Provider.PluginVersion
		if pluginArtifacts[pluginKey] == nil {
			pluginArtifacts[pluginKey] = map[string]struct{}{}
		}
		pluginArtifacts[pluginKey][binding.ArtifactDigest] = struct{}{}
	}

	var digests []string
	for digest := range artifactProviders {
		digests = append(digests, digest)
	}
	sort.Strings(digests)

	plan := TargetCapabilityPackagingPlan{
		Schema:                targetCapabilityPackagingPlanSchema,
		Scope:                 scope,
		Target:                target,
		LogicalProviderCount:  len(providerKeys),
		PhysicalArtifactCount: len(digests),
	}
	for _, digest := range digests {
		providers := artifactProviders[digest]
		sort.Slice(providers, func(i, j int) bool {
			return targetPackagingProviderKey(providers[i]) <
				targetPackagingProviderKey(providers[j])
		})
		if len(providers) > 1 {
			plan.CoalescedArtifactCount++
		}
		plan.Artifacts = append(plan.Artifacts, TargetPackagingArtifact{
			ArtifactDigest: digest,
			Providers:      providers,
		})
	}
	for _, artifacts := range pluginArtifacts {
		if len(artifacts) > 1 {
			plan.SplitPluginCount++
		}
	}
	plan.PlanFingerprint = targetCapabilityPackagingPlanFingerprint(plan)
	if err := validateTargetCapabilityPackagingPlan(plan); err != nil {
		return TargetCapabilityPackagingPlan{}, err
	}
	return plan, nil
}

func validateTargetCapabilityPackagingPlan(
	plan TargetCapabilityPackagingPlan,
) error {
	if plan.Schema != targetCapabilityPackagingPlanSchema {
		return fmt.Errorf(
			"target packaging schema %q is not supported", plan.Schema,
		)
	}
	if !plan.Scope.Valid() ||
		strings.TrimSpace(plan.Target) == "" ||
		plan.Target != strings.TrimSpace(plan.Target) {
		return fmt.Errorf("target packaging scope / Target is invalid")
	}
	if len(plan.Artifacts) == 0 {
		return fmt.Errorf("target packaging requires at least one artifact")
	}

	providers := map[string]string{}
	pluginArtifacts := map[string]map[string]struct{}{}
	coalesced := 0
	for i, artifact := range plan.Artifacts {
		if !validSHA256Fingerprint(artifact.ArtifactDigest) {
			return fmt.Errorf("target packaging artifact[%d] digest is invalid", i)
		}
		if len(artifact.Providers) == 0 {
			return fmt.Errorf("target packaging artifact[%d] has no Providers", i)
		}
		if len(artifact.Providers) > 1 {
			coalesced++
		}
		for j, ref := range artifact.Providers {
			if ref.PluginID == "" ||
				ref.PluginVersion == "" ||
				ref.ImplementationID == "" ||
				ref.ContractID == "" ||
				ref.ContractVersion == "" ||
				!validSHA256Fingerprint(ref.ContractFingerprint) ||
				!validSHA256Fingerprint(ref.CapabilityLockFingerprint) ||
				!validSHA256Fingerprint(ref.ProviderBuildRealizationFingerprint) {
				return fmt.Errorf(
					"target packaging artifact[%d] provider[%d] is incomplete or invalid",
					i,
					j,
				)
			}
			if _, err := parseVersion(ref.PluginVersion); err != nil {
				return fmt.Errorf(
					"target packaging provider Plugin version: %w",
					err,
				)
			}
			if _, err := parseVersion(ref.ContractVersion); err != nil {
				return fmt.Errorf(
					"target packaging provider Contract version: %w",
					err,
				)
			}
			if ref.SourceDeclarationFingerprint != "" &&
				!validSHA256Fingerprint(ref.SourceDeclarationFingerprint) {
				return fmt.Errorf(
					"target packaging provider source declaration fingerprint is invalid",
				)
			}
			key := ref.PluginID + "\x00" + ref.PluginVersion + "\x00" +
				ref.ImplementationID
			if previous, duplicate := providers[key]; duplicate {
				if previous != artifact.ArtifactDigest {
					return fmt.Errorf(
						"logical Provider %s is assigned to multiple artifacts",
						key,
					)
				}
				return fmt.Errorf(
					"logical Provider %s is declared more than once",
					key,
				)
			}
			providers[key] = artifact.ArtifactDigest

			pluginKey := ref.PluginID + "@" + ref.PluginVersion
			if pluginArtifacts[pluginKey] == nil {
				pluginArtifacts[pluginKey] = map[string]struct{}{}
			}
			pluginArtifacts[pluginKey][artifact.ArtifactDigest] = struct{}{}
		}
	}
	splitPlugins := 0
	for _, artifacts := range pluginArtifacts {
		if len(artifacts) > 1 {
			splitPlugins++
		}
	}
	if plan.LogicalProviderCount != len(providers) ||
		plan.PhysicalArtifactCount != len(plan.Artifacts) ||
		plan.CoalescedArtifactCount != coalesced ||
		plan.SplitPluginCount != splitPlugins {
		return fmt.Errorf("target packaging counts are stale")
	}
	if !validSHA256Fingerprint(plan.PlanFingerprint) {
		return fmt.Errorf("target packaging fingerprint is invalid")
	}
	if targetCapabilityPackagingPlanFingerprint(plan) != plan.PlanFingerprint {
		return fmt.Errorf("target packaging fingerprint is stale")
	}
	return nil
}

func normalizeTargetCapabilityPackagingPlan(
	plan TargetCapabilityPackagingPlan,
) TargetCapabilityPackagingPlan {
	out := plan
	out.Artifacts = make([]TargetPackagingArtifact, len(plan.Artifacts))
	for i, artifact := range plan.Artifacts {
		out.Artifacts[i] = artifact
		out.Artifacts[i].Providers = append(
			[]TargetPackagingProviderRef(nil),
			artifact.Providers...,
		)
		sort.Slice(out.Artifacts[i].Providers, func(a, b int) bool {
			return targetPackagingProviderKey(out.Artifacts[i].Providers[a]) <
				targetPackagingProviderKey(out.Artifacts[i].Providers[b])
		})
	}
	sort.Slice(out.Artifacts, func(i, j int) bool {
		return out.Artifacts[i].ArtifactDigest < out.Artifacts[j].ArtifactDigest
	})
	return out
}

func targetPackagingProviderKey(ref TargetPackagingProviderRef) string {
	return ref.PluginID + "\x00" + ref.PluginVersion + "\x00" +
		ref.ImplementationID + "\x00" + ref.ContractID + "\x00" +
		ref.ContractVersion
}

func targetCapabilityPackagingPlanFingerprint(
	plan TargetCapabilityPackagingPlan,
) string {
	plan.PlanFingerprint = ""
	return fingerprint(normalizeTargetCapabilityPackagingPlan(plan))
}
