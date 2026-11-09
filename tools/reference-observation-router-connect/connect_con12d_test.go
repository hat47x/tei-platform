package observationrouterconnect

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/observation"
	"tei/internal/environment"
)

// CON-12d: two kinds attempted, one observed and one refused. The report must
// say so rather than leave the reader to compare counts.
func TestAPartialRunIsNotReportedAsComplete(t *testing.T) {
	env := envWith(
		pair(deployment.ObservationDatabaseState, "db.observer"),
		pair(deployment.ObservationApplicationRuntime, "runtime.observer"),
	)
	// Only the database provider is registered; the runtime one is not.
	dbOnly := &recordingProvider{
		id: "db.observer", kind: deployment.ObservationDatabaseState,
	}
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{"db.observer": dbOnly},
	})

	if report.Attempted != 2 {
		t.Fatalf("attempted = %d, want 2", report.Attempted)
	}
	if report.Observed != 1 || report.Refused != 1 {
		t.Fatalf("observed=%d refused=%d, want 1 and 1", report.Observed, report.Refused)
	}
	if report.Complete {
		t.Fatal("a partial run was reported as complete")
	}
}

// A run where every attempt succeeded is complete. This is the other side of
// the same verdict, so "complete" is not simply always false.
func TestAFullyObservedRunIsComplete(t *testing.T) {
	env := envWith(
		pair(deployment.ObservationDatabaseState, "db.observer"),
		pair(deployment.ObservationApplicationRuntime, "runtime.observer"),
	)
	report := Run(context.Background(), Input{
		Environment: env,
		Providers: map[core.CanonicalID]observation.Provider{
			"db.observer":      &recordingProvider{id: "db.observer", kind: deployment.ObservationDatabaseState},
			"runtime.observer": &recordingProvider{id: "runtime.observer", kind: deployment.ObservationApplicationRuntime},
		},
	})
	if report.Attempted != 2 || report.Observed != 2 {
		t.Fatalf("attempted=%d observed=%d", report.Attempted, report.Observed)
	}
	if !report.Complete {
		t.Fatalf("a fully observed run was not complete: %+v", report)
	}
}

// Nothing attempted is not complete. An empty run must not read as success.
func TestAnEmptyRunIsNotComplete(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: environment.Environment{ID: "development"},
	})
	if report.Attempted != 0 {
		t.Fatalf("attempted = %d", report.Attempted)
	}
	if report.Complete {
		t.Fatal("a run with nothing attempted was reported as complete")
	}
}

// The verdict is not a claim about the world: a complete run still has to say
// that it is not proof.
func TestCompleteDoesNotMeanTheWorldMatches(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: envWith(pair(deployment.ObservationDatabaseState, "db.observer")),
		Providers: map[core.CanonicalID]observation.Provider{
			"db.observer": &recordingProvider{id: "db.observer", kind: deployment.ObservationDatabaseState},
		},
	})
	if !report.Complete {
		t.Fatalf("expected a complete run: %+v", report)
	}
	found := false
	for _, note := range report.Notes {
		if note == "complete means every attempted kind was observed; it is not proof that the world matches" {
			found = true
		}
	}
	if !found {
		t.Fatal("the notes do not bound what complete means")
	}
}
