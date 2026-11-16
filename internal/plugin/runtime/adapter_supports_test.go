package runtime_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/deployment/observation"
	"tei/internal/plugin"
	"tei/internal/plugin/runtime"
)

// planWith builds a plan that carries exactly one capability.
func planWith(kind plugin.CapabilityKind, id core.CanonicalID, capability plugin.Capability) runtime.EnvironmentPlan {
	return runtime.EnvironmentPlan{
		Environment: core.MustCanonicalID("development"),
		Capabilities: []runtime.CapabilityRuntime{{
			Plugin:     plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"},
			Capability: capability,
			Boundary:   runtime.BoundaryInProcess,
		}},
	}
}

// The adapter's Supports is the gate a caller consults before invoking. It must
// answer from the plan, not from the capability ID it was constructed with: a
// plan that does not carry the capability has to refuse, or the adapter would
// claim an authority the plan never granted.
func TestDeploymentExecutorSupportsIsDecidedByThePlan(t *testing.T) {
	id := core.MustCanonicalID("deploy.app")
	capability := plugin.Capability{
		Kind:    plugin.CapabilityDeploymentExecutor,
		ID:      id,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	adapter := runtime.DeploymentExecutor{
		Plan:         planWith(plugin.CapabilityDeploymentExecutor, id, capability),
		CapabilityID: id,
	}

	command := func(kind string) executor.Command { return executor.Command{Kind: kind} }
	if !adapter.Supports(command(string(deployment.ActionDatabaseRelease))) {
		t.Fatal("the declared action must be supported")
	}
	if adapter.Supports(command(string(deployment.ActionTrafficShift))) {
		t.Fatal("an undeclared action must not be supported")
	}

	// The same adapter pointed at a plan without the capability supports
	// nothing, because the plan is the authority and not the adapter's own field.
	empty := runtime.DeploymentExecutor{
		Plan:         runtime.EnvironmentPlan{Environment: core.MustCanonicalID("development")},
		CapabilityID: id,
	}
	if empty.Supports(command(string(deployment.ActionDatabaseRelease))) {
		t.Fatal("a plan that carries no capability must support nothing")
	}
}

// Execute refuses an unsupported action. The refusal is also produced one layer
// down — Registry.InvokeWithProvenance re-reads the plan and rejects a
// capability it cannot resolve — so removing Execute's own check still yields an
// error and this test does not distinguish the two. It pins the observable
// contract (a refusal, not an invocation) rather than the guard, which is
// recorded here so the coverage is not overstated.
func TestDeploymentExecutorRefusesAnUnsupportedAction(t *testing.T) {
	id := core.MustCanonicalID("deploy.app")
	capability := plugin.Capability{
		Kind:    plugin.CapabilityDeploymentExecutor,
		ID:      id,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	adapter := runtime.DeploymentExecutor{
		Plan:         planWith(plugin.CapabilityDeploymentExecutor, id, capability),
		CapabilityID: id,
	}
	// The registry is left zero-valued; the refusal comes back either from the
	// adapter's own check or from the registry's capability lookup.
	if _, err := adapter.Execute(context.Background(), executor.Command{Kind: string(deployment.ActionTrafficShift)}); err == nil {
		t.Fatal("an unsupported action must be refused")
	}
}

// The observation adapter has the same shape and the same obligation: the plan
// decides, and an unsupported kind is refused before the registry is asked.
func TestObservationProviderSupportsIsDecidedByThePlan(t *testing.T) {
	id := core.MustCanonicalID("observe.state")
	capability := plugin.Capability{
		Kind:         plugin.CapabilityObservationProvider,
		ID:           id,
		Observations: []deployment.ObservationKind{deployment.ObservationDatabaseState},
	}
	adapter := runtime.ObservationProvider{
		Plan:         planWith(plugin.CapabilityObservationProvider, id, capability),
		CapabilityID: id,
	}
	if !adapter.Supports(deployment.ObservationDatabaseState) {
		t.Fatal("the declared observation must be supported")
	}
	if adapter.Supports(deployment.ObservationApplicationRuntime) {
		t.Fatal("an undeclared observation must not be supported")
	}

	empty := runtime.ObservationProvider{
		Plan:         runtime.EnvironmentPlan{Environment: core.MustCanonicalID("development")},
		CapabilityID: id,
	}
	if empty.Supports(deployment.ObservationDatabaseState) {
		t.Fatal("a plan that carries no capability must support nothing")
	}

	// The kind is matched exactly, so a value outside the vocabulary is not a
	// match even when the capability declares observations.
	if adapter.Supports(deployment.ObservationKind("unknown")) {
		t.Fatal("an unknown observation kind must not match a declared one")
	}

	// And the refusal happens before the registry is asked.
	if _, err := adapter.Observe(context.Background(), observation.Request{Kind: deployment.ObservationApplicationRuntime}); err == nil {
		t.Fatal("an unsupported observation must be refused")
	}
}

// The adapter treats the plan as the authority. EnvironmentPlan.Capability
// returns the zero value both when the entry is missing and when it is
// ambiguous, so the `ok` guard in Supports is redundant today: a zero
// Capability declares no actions and no observations, and every predicate
// answers false for it. The guard is kept because it states the intent, and this
// test records that the outcome does not depend on it — a reader who removes it
// will not change behaviour, and one who makes Capability start returning a
// populated value on failure will.
func TestAnAbsentCapabilityDeclaresNothingSoEveryPredicateRefuses(t *testing.T) {
	missing, ok := (runtime.EnvironmentPlan{}).Capability(
		plugin.CapabilityObservationProvider, core.MustCanonicalID("observe.state"))
	if ok {
		t.Fatal("an empty plan has no capability")
	}
	if missing.Capability.Kind != "" || missing.Capability.ID != "" {
		t.Fatalf("a missing capability must be the zero value, or the ok guard would matter: %+v", missing.Capability)
	}

	// A duplicate pair is reported as absent the same way, with the same zero
	// value, so neither failure mode can be read as a grant.
	duplicated := runtime.EnvironmentPlan{
		Environment: core.MustCanonicalID("development"),
		Capabilities: []runtime.CapabilityRuntime{
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"},
				Capability: plugin.Capability{Kind: plugin.CapabilityObservationProvider, ID: core.MustCanonicalID("observe.state")}},
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.b"), Version: "1.0.0"},
				Capability: plugin.Capability{Kind: plugin.CapabilityObservationProvider, ID: core.MustCanonicalID("observe.state")}},
		},
	}
	ambiguous, ok := duplicated.Capability(plugin.CapabilityObservationProvider, core.MustCanonicalID("observe.state"))
	if ok {
		t.Fatal("an ambiguous capability must not resolve")
	}
	if ambiguous.Capability.ID != "" {
		t.Fatalf("an ambiguous capability must be the zero value: %+v", ambiguous.Capability)
	}

	// Both failure modes refuse through the adapter, so a caller cannot reach a
	// grant by presenting either one.
	for name, plan := range map[string]runtime.EnvironmentPlan{
		"missing":   {},
		"ambiguous": duplicated,
	} {
		adapter := runtime.ObservationProvider{
			Plan:         plan,
			CapabilityID: core.MustCanonicalID("observe.state"),
		}
		if adapter.Supports(deployment.ObservationDatabaseState) {
			t.Fatalf("%s capability was reported as supported", name)
		}
	}
}

// The capability ID on the plan is part of the identity: an adapter asking for a
// different ID gets nothing, so a plan cannot be read with the wrong key.
func TestTheCapabilityIDSelectsTheEntry(t *testing.T) {
	declared := core.MustCanonicalID("deploy.app")
	capability := plugin.Capability{
		Kind:    plugin.CapabilityDeploymentExecutor,
		ID:      declared,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := planWith(plugin.CapabilityDeploymentExecutor, declared, capability)

	other := runtime.DeploymentExecutor{
		Plan:         plan,
		CapabilityID: core.MustCanonicalID("deploy.other"),
	}
	if other.Supports(executor.Command{Kind: string(deployment.ActionDatabaseRelease)}) {
		t.Fatal("a different capability ID must not read another capability's entry")
	}
}
