package main

// This read-only projection consumes the existing Contract Query executor
// resolver. It is a development handoff hint, not Provider Conformance,
// current Host Acceptance, an executable Plan, or permission to invoke a plugin.

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/realization"
)

type QueryExecutorHandoff struct {
	Status          string `json:"status"`
	Reason          string `json:"reason"`
	Environment     string `json:"environment,omitempty"`
	Instance        string `json:"instance,omitempty"`
	Target          string `json:"target,omitempty"`
	Executor        string `json:"executor,omitempty"`
	Plugin          string `json:"plugin,omitempty"`
	PluginVersion   string `json:"pluginVersion,omitempty"`
}

// ObserveQueryExecutorHandoff does not resolve a Provider by its name or
// technology. The actual executor, its Plugin and compatible DB provider are
// checked by the existing realization.ResolveContractQueryExecutor function.
// No Host-granted runtime authority is inferred from a static resolution.
func ObserveQueryExecutorHandoff(world ir.World, use *DeclaredContractQuery, binding *EnvironmentBindingObservation) QueryExecutorHandoff {
	out := QueryExecutorHandoff{Status: "unresolved", Reason: "contract-and-environment-not-qualified"}
	if use == nil || use.Status != "declared-requirement" || binding == nil ||
		binding.Status != "declared-environment-binding" || binding.Environment == "" ||
		binding.Instance == "" || use.TargetDatabase != binding.Definition ||
		use.Contract == "" || use.Query == "" {
		return out
	}
	out.Environment = binding.Environment
	out.Instance = binding.Instance
	target := realization.ContractQueryTarget{
		Database: core.CanonicalID(use.TargetDatabase),
		Contract: core.CanonicalID(use.Contract),
		Query:    core.LocalName(use.Query),
	}
	out.Target = target.Identity()

	count := 0
	var environmentIndex int
	for i := range world.Environments {
		if string(world.Environments[i].ID) == binding.Environment {
			count++
			environmentIndex = i
		}
	}
	if count != 1 {
		out.Reason = "environment-not-unique-or-missing"
		return out
	}
	env := world.Environments[environmentIndex]

	bindingCount := 0
	for _, candidate := range world.ContractQueryExecutorBindings {
		if candidate.Environment == env.ID &&
			string(candidate.Instance) == binding.Instance && candidate.Target == target {
			bindingCount++
		}
	}
	if bindingCount == 0 {
		// Absence in this Workspace does not prove that no external execution
		// path exists; avoid making the binding a mandatory authoring field.
		out.Status, out.Reason = "not-declared", "no-explicit-executor-binding-in-workspace"
		return out
	}
	if bindingCount > 1 {
		out.Reason = "ambiguous-contract-query-executor-binding"
		return out
	}

	// These are the existing Plugin/Contract Query executor resolvers, not
	// copies with weaker acceptance or ambiguity rules.
	catalog := plugin.Catalog{Plugins: world.Plugins, Bundles: world.Bundles}
	selected, err := catalog.ResolveEnvironment(env)
	if err != nil {
		out.Reason = "environment-plugin-set-unresolved"
		return out
	}
	resolved, err := realization.ResolveContractQueryExecutor(
		env, selected, world.ContractQueryExecutorBindings,
		core.CanonicalID(binding.Instance), target,
	)
	if err != nil {
		out.Reason = "existing-executor-resolver-rejected-declaration"
		return out
	}
	out.Executor = string(resolved.Binding.Executor)
	out.Plugin = string(resolved.Capability.Plugin.ID)
	out.PluginVersion = resolved.Capability.Plugin.Version
	out.Status = "resolved-declaration"
	out.Reason = "static-executor-binding-resolved-host-and-conformance-not-verified"
	return out
}
