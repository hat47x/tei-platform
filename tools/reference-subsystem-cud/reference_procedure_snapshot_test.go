package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type referenceProcedure struct {
	SchemaVersion string `json:"schemaVersion"`
	ArtifactRole  string `json:"artifactRole"`
	Canonical     bool   `json:"canonical"`
	Steps         []struct {
		ID                  string   `json:"id"`
		Realization         string   `json:"realization"`
		ApplicationPatterns []string `json:"applicationPatterns"`
	} `json:"steps"`
}

func actualProcedureCandidate(t *testing.T) ([]byte, []byte, Facts) {
	t.Helper()
	worldBytes, facts := referenceWorldCandidate(t)
	sourcePath := filepath.Join("..", "..", "reference", "business-system-design", "order-entry-business-procedure.json")
	procedureBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	var source referenceProcedure
	if err := json.Unmarshal(procedureBytes, &source); err != nil {
		t.Fatal(err)
	}
	if source.SchemaVersion != "business-procedure-reference-v0" ||
		source.ArtifactRole != "reference-authoring-fixture" || source.Canonical {
		t.Fatalf("Business fixture must not be treated as accepted canonical source: %+v", source)
	}
	// Reproduce only the two semantics this E0 probe actually supports:
	// a declared system Pattern and an unresolved manual step. Other
	// Business realizations are deliberately not forced into these types.
	foundManual, foundSystem := false, false
	for _, step := range source.Steps {
		switch step.ID {
		case "BP-04":
			if step.Realization != "manual-unresolved" || len(step.ApplicationPatterns) != 0 {
				t.Fatalf("manual step must not invent a Function: %+v", step)
			}
			facts.ProcedureSteps = append(facts.ProcedureSteps, ProcedureStep{
				ID: step.ID, Subsystem: "sales", Realization: "manual", Source: sourcePath + "#BP-04",
			})
			foundManual = true
		case "BP-07":
			if step.Realization != "application-use" ||
				len(step.ApplicationPatterns) != 1 || step.ApplicationPatterns[0] != "order_entry.submit_order" {
				t.Fatalf("system step lost its explicit Pattern identity: %+v", step)
			}
			facts.ProcedureSteps = append(facts.ProcedureSteps, ProcedureStep{
				ID: step.ID, Subsystem: "sales", Realization: "system",
				Pattern: step.ApplicationPatterns[0], Source: sourcePath + "#BP-07",
			})
			foundSystem = true
		}
	}
	if !foundManual || !foundSystem {
		t.Fatal("Reference Business Procedure no longer contains both expected steps")
	}
	// The default candidate also contained a synthetic manual and system
	// step. Retain only source-located declarations for this test.
	facts.ProcedureSteps = facts.ProcedureSteps[len(facts.ProcedureSteps)-2:]
	facts.ProcedureJSONFingerprint = snapshotDigest(procedureBytes)
	return worldBytes, procedureBytes, facts
}

func TestReferenceProcedureSourceBindingAndManualNonInference(t *testing.T) {
	worldBytes, procedureBytes, facts := actualProcedureCandidate(t)
	report, err := InspectBoundProcedureSnapshot(worldBytes, procedureBytes, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "no-findings-in-declared-scope" || len(report.Diagnostics) != 0 {
		t.Fatalf("candidate Procedure/World cross-reference: %+v", report)
	}
	if report.Coverage.ProcedureSteps != 2 ||
		report.InputFingerprints == nil ||
		report.InputFingerprints.ProcedureJSON != facts.ProcedureJSONFingerprint ||
		report.AuthorityLimit != authorityLimit {
		t.Fatalf("Procedure input cannot be promoted to accepted meaning: %+v", report)
	}
}

func TestReferenceProcedureSourceChangedWithoutWorldChangeFailsClosed(t *testing.T) {
	worldBytes, procedureBytes, facts := actualProcedureCandidate(t)
	// The compiled World remains unchanged while the Procedure artifact
	// acquires new bytes. This was invisible to a World-only fingerprint.
	changed := append(append([]byte(nil), procedureBytes...), byte(' '))
	report, err := InspectBoundProcedureSnapshot(worldBytes, changed, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" ||
		codeCount(report, "PROCEDURE_SNAPSHOT_STALE") != 1 ||
		codeCount(report, "WORLD_SNAPSHOT_STALE") != 0 {
		t.Fatalf("stale Procedure facts must not pass: %+v", report)
	}
}

func TestReferenceProcedurePinCannotPassWithoutSource(t *testing.T) {
	worldBytes, procedureBytes, facts := actualProcedureCandidate(t)
	unavailable, err := InspectSnapshot(worldBytes, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if unavailable.Status != "unresolved" || codeCount(unavailable, "PROCEDURE_SNAPSHOT_UNAVAILABLE") != 1 {
		t.Fatalf("candidate pin with absent Procedure bytes must be unresolved: %+v", unavailable)
	}
	facts.ProcedureJSONFingerprint = ""
	unbound, err := InspectBoundProcedureSnapshot(worldBytes, procedureBytes, mustJSON(t, facts))
	if err != nil {
		t.Fatal(err)
	}
	if unbound.Status != "unresolved" || codeCount(unbound, "PROCEDURE_SNAPSHOT_UNBOUND") != 1 {
		t.Fatalf("Procedure bytes without an exact pin must be unresolved: %+v", unbound)
	}
}

func TestBoundProcedureRejectsInvalidJSONBeforeInspecting(t *testing.T) {
	worldBytes, _, facts := actualProcedureCandidate(t)
	for _, invalid := range [][]byte{[]byte(""), []byte("{"), []byte("null"), []byte("[]")} {
		if _, err := InspectBoundProcedureSnapshot(worldBytes, invalid, mustJSON(t, facts)); err == nil {
			t.Fatalf("invalid Procedure JSON accepted: %q", invalid)
		}
	}
}
