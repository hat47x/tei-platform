package observationrouterconnect

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/observation"
)

// CON-12d, the unknown half: a provider that fails AFTER it was entered must not
// be counted as a refusal, because the call already happened and whether the
// world was read is unknown.
func TestAPostEntryFailureIsUnknownNotRefused(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer"))
	provider := &recordingProvider{
		id: "db.observer", kind: deployment.ObservationDatabaseState, answer: "error",
	}
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{"db.observer": provider},
	})

	if report.Invoked != 1 {
		t.Fatalf("invoked = %d, want 1", report.Invoked)
	}
	if report.Unknown != 1 {
		t.Fatalf("unknown = %d, want 1", report.Unknown)
	}
	if report.Refused != 0 {
		t.Fatalf("a post-entry failure was counted as a refusal: refused=%d", report.Refused)
	}
	if report.Observed != 0 {
		t.Fatalf("a failed observation was counted as observed: %d", report.Observed)
	}
	if report.Complete {
		t.Fatal("a run with an unknown outcome was reported as complete")
	}
}

// A pre-entry refusal is still a refusal, not an unknown. The two must not
// collapse into one number.
func TestAPreEntryRefusalIsStillRefused(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer"))
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{},
	})
	if report.Refused != 1 || report.Unknown != 0 {
		t.Fatalf("refused=%d unknown=%d, want 1 and 0", report.Refused, report.Unknown)
	}
	if report.Invoked != 0 {
		t.Fatalf("invoked = %d", report.Invoked)
	}
}

// Unknown and refused are both present in one run: one provider fails after
// entry, the other is not registered. Neither may absorb the other.
func TestUnknownAndRefusedStaySeparateInOneRun(t *testing.T) {
	env := envWith(
		pair(deployment.ObservationDatabaseState, "db.observer"),
		pair(deployment.ObservationApplicationRuntime, "runtime.observer"),
	)
	report := Run(context.Background(), Input{
		Environment: env,
		Providers: map[core.CanonicalID]observation.Provider{
			// Entered, then fails.
			"db.observer": &recordingProvider{
				id: "db.observer", kind: deployment.ObservationDatabaseState, answer: "error",
			},
			// Not registered at all.
		},
	})
	if report.Attempted != 2 {
		t.Fatalf("attempted = %d", report.Attempted)
	}
	if report.Unknown != 1 || report.Refused != 1 {
		t.Fatalf("unknown=%d refused=%d, want 1 and 1", report.Unknown, report.Refused)
	}
	if report.Invoked != 1 {
		t.Fatalf("invoked = %d, want 1", report.Invoked)
	}
	if report.Complete {
		t.Fatal("a run with an unknown was reported as complete")
	}
}

// The report must explain that unknown is a subset of invoked, so a reader does
// not add the three numbers together.
func TestTheNotesExplainUnknownIsInsideInvoked(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: envWith(pair(deployment.ObservationDatabaseState, "db.observer")),
	})
	found := false
	for _, note := range report.Notes {
		if note == "unknownCount counts entries that failed after the provider was called; it is a subset of invokedCount" {
			found = true
		}
	}
	if !found {
		t.Fatal("the notes do not explain that unknown is a subset of invoked")
	}
}
