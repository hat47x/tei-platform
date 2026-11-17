package runtime_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/evidence"
	"tei/internal/plugin"
	"tei/internal/plugin/runtime"
)

// evidencePlan builds a plan carrying one evidence provider capability.
func evidencePlan(id core.CanonicalID, kinds ...deployment.EvidenceKind) runtime.EnvironmentPlan {
	return runtime.EnvironmentPlan{
		Environment: core.MustCanonicalID("development"),
		Capabilities: []runtime.CapabilityRuntime{{
			Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.evidence"), Version: "1.0.0"},
			Capability: plugin.Capability{
				Kind:     plugin.CapabilityEvidenceProvider,
				ID:       id,
				Evidence: kinds,
			},
			Boundary: runtime.BoundaryInProcess,
		}},
	}
}

// Supports is the gate a caller consults before collecting, and it must answer
// from the plan. A plan that does not carry the capability collects nothing,
// because the capability is the grant and the adapter holds only an ID.
func TestEvidenceProviderSupportsIsDecidedByThePlan(t *testing.T) {
	id := core.MustCanonicalID("evidence.rollout")
	adapter := runtime.EvidenceProvider{
		Plan:         evidencePlan(id, deployment.EvidenceRolloutHealth),
		CapabilityID: id,
	}
	if !adapter.Supports(deployment.EvidenceRolloutHealth) {
		t.Fatal("the declared evidence kind must be supported")
	}
	if adapter.Supports(deployment.EvidenceKind("deploy-plan")) {
		t.Fatal("an undeclared evidence kind must not be supported")
	}

	empty := runtime.EvidenceProvider{
		Plan:         runtime.EnvironmentPlan{Environment: core.MustCanonicalID("development")},
		CapabilityID: id,
	}
	if empty.Supports(deployment.EvidenceRolloutHealth) {
		t.Fatal("a plan that carries no capability must support nothing")
	}
}

// Collect refuses an unsupported kind. The refusal is also produced one layer
// down, by Registry.Invoke, so this pins the observable contract (an error, not
// a collection) rather than which check produced it.
func TestEvidenceProviderRefusesAnUnsupportedKind(t *testing.T) {
	id := core.MustCanonicalID("evidence.rollout")
	adapter := runtime.EvidenceProvider{
		Plan:         evidencePlan(id, deployment.EvidenceRolloutHealth),
		CapabilityID: id,
	}
	if _, err := adapter.Collect(context.Background(), evidence.Request{Kind: deployment.EvidenceKind("other")}); err == nil {
		t.Fatal("an unsupported evidence kind must be refused")
	}
}

// The capability kind is part of the identity: a plan whose entry is a
// different kind must not answer for evidence. Reading by ID alone would let an
// adapter for one capability answer for another.
func TestEvidenceProviderDoesNotReadAnotherCapabilityKind(t *testing.T) {
	id := core.MustCanonicalID("shared.id")
	plan := runtime.EnvironmentPlan{
		Environment: core.MustCanonicalID("development"),
		Capabilities: []runtime.CapabilityRuntime{{
			Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"},
			// The same ID under a different kind.
			Capability: plugin.Capability{
				Kind:     plugin.CapabilityObservationProvider,
				ID:       id,
				Evidence: []deployment.EvidenceKind{deployment.EvidenceRolloutHealth},
			},
			Boundary: runtime.BoundaryInProcess,
		}},
	}
	adapter := runtime.EvidenceProvider{Plan: plan, CapabilityID: id}
	if adapter.Supports(deployment.EvidenceRolloutHealth) {
		t.Fatal("an observation capability must not answer for an evidence request")
	}
}

// Collect must not report a collection when the plan carries no registry to
// invoke: a zero-valued registry cannot produce a payload, so an error is the
// only honest answer.
func TestCollectWithoutARegistryDoesNotReportACollection(t *testing.T) {
	id := core.MustCanonicalID("evidence.rollout")
	adapter := runtime.EvidenceProvider{
		Plan:         evidencePlan(id, deployment.EvidenceRolloutHealth),
		CapabilityID: id,
	}
	if _, err := adapter.Collect(context.Background(), evidence.Request{
		Kind:     deployment.EvidenceRolloutHealth,
		ActionID: "action-1",
	}); err == nil {
		t.Fatal("a plan with no registry must not report a collection")
	}
}
