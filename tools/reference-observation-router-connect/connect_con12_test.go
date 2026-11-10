package observationrouterconnect

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/observation"
)

// CON-12b: the declaration selects A, but the Host registry also holds B. B must
// not be entered, even though it supports the kind and would answer successfully.
func TestOnlyTheDeclaredProviderIsEntered(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer.a"))
	selected := &recordingProvider{id: "db.observer.a", kind: deployment.ObservationDatabaseState}
	other := &recordingProvider{id: "db.observer.b", kind: deployment.ObservationDatabaseState}

	report := Run(context.Background(), Input{
		Environment: env,
		Providers: map[core.CanonicalID]observation.Provider{
			"db.observer.a": selected,
			"db.observer.b": other,
		},
	})

	if selected.calls != 1 {
		t.Fatalf("the declared provider was entered %d times", selected.calls)
	}
	if other.calls != 0 {
		t.Fatalf("a provider the declaration did not select was entered %d times", other.calls)
	}
	if report.Outcomes[0].ProviderID != "db.observer.a" {
		t.Fatalf("the outcome names %q", report.Outcomes[0].ProviderID)
	}
	// Same output does not make the two interchangeable: the report must name
	// which provider actually ran.
	if report.Invoked != 1 {
		t.Fatalf("invoked = %d", report.Invoked)
	}
}

// CON-12b, the refusing half: a provider whose ID disagrees with the binding is
// refused by the existing Router before its Observe is entered.
func TestAProviderWithAnotherIDIsNotEntered(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer.a"))
	// The registry is keyed by the declared id, but the value claims another id.
	imposter := &recordingProvider{id: "db.observer.b", kind: deployment.ObservationDatabaseState}

	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{"db.observer.a": imposter},
	})

	if report.Outcomes[0].Status != StatusNotConforming {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
	if report.Invoked != 0 {
		t.Fatalf("a non-conforming provider was entered: %+v", report)
	}
	if imposter.calls != 0 {
		t.Fatalf("Observe ran %d times for a provider that does not conform", imposter.calls)
	}
}
