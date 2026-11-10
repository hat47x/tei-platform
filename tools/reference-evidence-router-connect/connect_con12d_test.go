package evidencerouterconnect

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
	"tei/internal/environment"
)

// CON-12d: two records attempted, one collected and one refused. The report must
// say so rather than leave the reader to compare counts.
func TestAPartialRunIsNotReportedAsComplete(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth, answer: durable.EvidencePass,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records: []durable.ActionRecord{
			record("gate-1", deployment.EvidenceRolloutHealth),
			// The second record is not an evidence gate, so it is refused.
			{Action: durable.Action{ID: "db-2", Kind: durable.ActionDatabaseRelease, Mode: durable.ModeCommand}},
		},
	})

	if report.Attempted != 2 {
		t.Fatalf("attempted = %d, want 2", report.Attempted)
	}
	if report.Collected != 1 || report.Refused != 1 {
		t.Fatalf("collected=%d refused=%d, want 1 and 1", report.Collected, report.Refused)
	}
	if report.Complete {
		t.Fatal("a partial run was reported as complete")
	}
}

// A run where every attempt produced evidence is complete.
func TestAFullyCollectedRunIsComplete(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth, answer: durable.EvidencePass,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records: []durable.ActionRecord{
			record("gate-1", deployment.EvidenceRolloutHealth),
			record("gate-2", deployment.EvidenceRolloutHealth),
		},
	})
	if report.Attempted != 2 || report.Collected != 2 {
		t.Fatalf("attempted=%d collected=%d", report.Attempted, report.Collected)
	}
	if !report.Complete {
		t.Fatalf("a fully collected run was not complete: %+v", report)
	}
}

// Nothing attempted is not complete.
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

// A pass outcome in a complete run is still only the provider's claim.
func TestCompleteDoesNotMeanTheEvidenceIsTrue(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth, answer: durable.EvidencePass,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})
	if !report.Complete {
		t.Fatalf("expected a complete run: %+v", report)
	}
	found := false
	for _, note := range report.Notes {
		if note == "complete means every attempted record was collected; it is not proof that the evidence is true" {
			found = true
		}
	}
	if !found {
		t.Fatal("the notes do not bound what complete means")
	}
}
