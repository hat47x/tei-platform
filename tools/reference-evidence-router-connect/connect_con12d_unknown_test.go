package evidencerouterconnect

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
)

// CON-12d, the unknown half: a provider that fails AFTER it was entered must not
// be counted as a refusal.
func TestAPostEntryFailureIsUnknownNotRefused(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth, fail: true,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
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
	if report.Collected != 0 {
		t.Fatalf("a failed collection was counted as collected: %d", report.Collected)
	}
	if report.Complete {
		t.Fatal("a run with an unknown outcome was reported as complete")
	}
}

// A pre-entry refusal is still a refusal, not an unknown.
func TestAPreEntryRefusalIsStillRefused(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})
	if report.Refused != 1 || report.Unknown != 0 {
		t.Fatalf("refused=%d unknown=%d, want 1 and 0", report.Refused, report.Unknown)
	}
	if report.Invoked != 0 {
		t.Fatalf("invoked = %d", report.Invoked)
	}
}

// Unknown and refused both present in one run.
func TestUnknownAndRefusedStaySeparateInOneRun(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth, fail: true,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records: []durable.ActionRecord{
			// Entered, then fails.
			record("gate-1", deployment.EvidenceRolloutHealth),
			// Not an evidence gate: refused before entry.
			{Action: durable.Action{ID: "db-2", Kind: durable.ActionDatabaseRelease, Mode: durable.ModeCommand}},
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

// The notes must explain that unknown is a subset of invoked.
func TestTheNotesExplainUnknownIsInsideInvoked(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
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
