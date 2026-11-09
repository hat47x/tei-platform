package main

import (
	"fmt"

	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// transformPackageBuildPlan binds the exact transform Runtime Engine identity
// into the TEI-native package build. The same path is exercised by the
// source-closure propagation regression tests.
func transformPackageBuildPlan(
	base plugin.BuildPlan,
	builder plugin.BuilderIdentity,
	engine pruntime.RuntimeEngineIdentity,
	engineFingerprint string,
) (plugin.BuildPlan, string, error) {
	if err := engine.Validate(); err != nil {
		return plugin.BuildPlan{}, "", fmt.Errorf("transform Runtime Engine: %w", err)
	}
	actual, err := pruntime.RuntimeEngineIdentityFingerprint(engine)
	if err != nil {
		return plugin.BuildPlan{}, "", err
	}
	if actual != engineFingerprint {
		return plugin.BuildPlan{}, "", fmt.Errorf("transform Runtime Engine fingerprint does not match engine identity")
	}

	plan := base
	plan.Builder = builder
	plan.Dependencies = append(
		append([]plugin.BuildDependency(nil), base.Dependencies...),
		plugin.BuildDependency{
			Kind:           "runtime-engine",
			ID:             string(engine.ID),
			Version:        engine.Version,
			ArtifactDigest: engine.ImplementationDigest,
		},
	)
	plan.Parameters = append(
		append([]plugin.BuildParameter(nil), base.Parameters...),
		plugin.BuildParameter{Name: "runtime-engine-contract", Value: engine.ContractVersion},
		plugin.BuildParameter{Name: "runtime-engine-fingerprint", Value: engineFingerprint},
		plugin.BuildParameter{Name: "runtime-engine-target", Value: engine.Target},
	)
	fingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return plugin.BuildPlan{}, "", fmt.Errorf("transform package Build Plan: %w", err)
	}
	return plan, fingerprint, nil
}
