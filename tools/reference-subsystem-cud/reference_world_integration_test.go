package main

import (
	"os"
	"path/filepath"
	"testing"
)

// referenceWorldCandidate uses actual repository World bytes, but all owner,
// operation and step facts below are synthetic test declarations. A passing
// result must not be interpreted as accepted Business or observed CUD effects.
func referenceWorldCandidate(t *testing.T) ([]byte, Facts) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "reference", "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw, Facts{
		SchemaVersion:        factsSchema,
		WorldJSONFingerprint: snapshotDigest(raw),
		Owners: []Owner{
			{Data: "sales.order", Subsystem: "sales", Source: "synthetic-test:sales-owner"},
			{Data: "customer.customer", Subsystem: "customer", Source: "synthetic-test:customer-owner"},
		},
		FunctionIO: []FunctionIO{
			{Pattern: "order_entry.order_record", Subsystem: "sales", Data: "sales.order", Operation: "U", Source: "synthetic-test:record-update"},
			{Pattern: "order_entry.recent_orders", Subsystem: "sales", Data: "customer.customer", Operation: "R", Source: "synthetic-test:foreign-read"},
		},
		ProcedureSteps: []ProcedureStep{
			{ID: "manual-check", Subsystem: "sales", Realization: "manual", Source: "synthetic-test:manual"},
			{ID: "submit-order", Subsystem: "sales", Realization: "system", Pattern: "order_entry.submit_order", Source: "synthetic-test:submit"},
		},
	}
}

func TestRepositoryWorldProjectsRealSemanticAndPatternIdentities(t *testing.T) {
	raw, facts := referenceWorldCandidate(t)
	report, err := InspectSnapshot(raw, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "no-findings-in-declared-scope" || len(report.Diagnostics) != 0 {
		t.Fatalf("actual World identity projection: %+v", report)
	}
	if report.InputFingerprints == nil || report.InputFingerprints.WorldJSON != facts.WorldJSONFingerprint {
		t.Fatalf("exact input snapshot not bound: %+v", report.InputFingerprints)
	}
	if report.Coverage.WorldData < 2 || report.Coverage.WorldPatterns < 3 {
		t.Fatalf("actual repository World lacks the reference IDs: %+v", report.Coverage)
	}
}

func TestRepositoryWorldDoesNotInventCUDFromCommandPattern(t *testing.T) {
	raw, facts := referenceWorldCandidate(t)
	// The system step links to a real command Pattern but has no I/O facts.
	// Presence of that Pattern alone must not imply C, U or D effects.
	facts.FunctionIO = nil
	report, err := InspectSnapshot(raw, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "no-findings-in-declared-scope" || report.Coverage.FunctionIOFacts != 0 {
		t.Fatalf("invented command effects: %+v", report)
	}
}

func TestRepositoryWorldForeignWriteAndUnknownWriter(t *testing.T) {
	raw, facts := referenceWorldCandidate(t)
	facts.FunctionIO[1].Operation = "U" // Explicit candidate, NOT an observed browse effect.
	report, err := InspectSnapshot(raw, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "review-needed" || codeCount(report, "CROSS_SUBSYSTEM_WRITE_INTENT") != 1 {
		t.Fatalf("foreign write must require review: %+v", report)
	}
	facts.FunctionIO[1].Pattern = "missing.application.pattern"
	report, err = InspectSnapshot(raw, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" || codeCount(report, "IO_PATTERN_UNRESOLVED") != 1 || codeCount(report, "CROSS_SUBSYSTEM_WRITE_INTENT") != 0 {
		t.Fatalf("unknown writer must not imply foreign write: %+v", report)
	}
}

func TestRepositoryWorldSnapshotChangeInvalidatesFacts(t *testing.T) {
	raw, facts := referenceWorldCandidate(t)
	// A whitespace-only World change still has distinct snapshot bytes.
	changed := append(append([]byte(nil), raw...), byte(' '))
	report, err := InspectSnapshot(changed, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" || codeCount(report, "WORLD_SNAPSHOT_STALE") != 1 {
		t.Fatalf("stale facts must not be accepted: %+v", report)
	}
}
