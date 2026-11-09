package evidencerouterconnect

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
)

// CON-12b: the declaration selects A, but the Host registry also holds B. B must
// not be entered, even though it supports the kind and would answer successfully.
func TestOnlyTheDeclaredProviderIsEntered(t *testing.T) {
	selected := &recordingProvider{
		id: "health.a", kind: deployment.EvidenceRolloutHealth, answer: durable.EvidencePass,
	}
	other := &recordingProvider{
		id: "health.b", kind: deployment.EvidenceRolloutHealth, answer: durable.EvidencePass,
	}

	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.a"),
		Providers: map[core.CanonicalID]evidence.Provider{
			"health.a": selected,
			"health.b": other,
		},
		Records: []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})

	if selected.calls != 1 {
		t.Fatalf("the declared provider was entered %d times", selected.calls)
	}
	if other.calls != 0 {
		t.Fatalf("a provider the declaration did not select was entered %d times", other.calls)
	}
	if report.Outcomes[0].ProviderID != "health.a" {
		t.Fatalf("the outcome names %q", report.Outcomes[0].ProviderID)
	}
	if report.Invoked != 1 {
		t.Fatalf("invoked = %d", report.Invoked)
	}
}

// CON-12b, the refusing half: a provider whose ID disagrees with the binding is
// refused by the existing Router before its Collect is entered.
func TestAProviderWithAnotherIDIsNotEntered(t *testing.T) {
	// The registry is keyed by the declared id, but the value claims another id.
	imposter := &recordingProvider{
		id: "health.b", kind: deployment.EvidenceRolloutHealth, answer: durable.EvidencePass,
	}

	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.a"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.a": imposter},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})

	if report.Outcomes[0].Status != StatusNotConforming {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
	if report.Invoked != 0 {
		t.Fatalf("a non-conforming provider was entered: %+v", report)
	}
	if imposter.calls != 0 {
		t.Fatalf("Collect ran %d times for a provider that does not conform", imposter.calls)
	}
}
