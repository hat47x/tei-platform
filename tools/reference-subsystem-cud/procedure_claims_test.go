package main

import (
	"encoding/json"
	"testing"
)

func rewriteProcedureForTest(t *testing.T, original []byte, change func(map[string]any)) []byte {
	t.Helper()
	var source map[string]any
	if err := json.Unmarshal(original, &source); err != nil {
		t.Fatal(err)
	}
	change(source)
	out, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPinnedProcedureRejectsFalseStepOrPatternClaim(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	tests := []struct {
		name   string
		change func(*Facts)
		code   string
	}{
		{
			name: "unknown step",
			change: func(f *Facts) {
				f.ProcedureSteps[1].ID = "BP-does-not-exist"
			},
			code: "PROCEDURE_STEP_NOT_FOUND",
		},
		{
			name: "known but different Pattern",
			change: func(f *Facts) {
				f.ProcedureSteps[1].Pattern = "order_entry.order_record"
			},
			code: "PROCEDURE_PATTERN_MISMATCH",
		},
		{
			name: "manual promoted to system",
			change: func(f *Facts) {
				f.ProcedureSteps[0].Realization = "system"
				f.ProcedureSteps[0].Pattern = "order_entry.order_record"
			},
			code: "PROCEDURE_REALIZATION_MISMATCH",
		},
		{
			name: "application step marked manual",
			change: func(f *Facts) {
				f.ProcedureSteps[1].Realization = "manual"
				f.ProcedureSteps[1].Pattern = ""
			},
			code: "PROCEDURE_REALIZATION_MISMATCH",
		},
		{
			name: "security step is not a system function",
			change: func(f *Facts) {
				f.ProcedureSteps[1].ID = "BP-06"
			},
			code: "PROCEDURE_REALIZATION_UNSUPPORTED",
		},
		{
			name: "empty Procedure candidate coverage",
			change: func(f *Facts) {
				f.ProcedureSteps = nil
			},
			code: "PROCEDURE_CANDIDATE_SCOPE_EMPTY",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := facts
			next.ProcedureSteps = append([]ProcedureStep(nil), facts.ProcedureSteps...)
			tc.change(&next)
			report, err := InspectBoundProcedureSnapshot(world, procedure, mustJSON(t, next))
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != "unresolved" || codeCount(report, tc.code) != 1 ||
				codeCount(report, "PROCEDURE_SNAPSHOT_STALE") != 0 {
				t.Fatalf("self-pinned facts must still be checked against source intent: %+v", report)
			}
		})
	}
}

func TestPinnedProcedureRejectsSourceAmbiguityWithMatchingNewDigest(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	tests := []struct {
		name   string
		change func(map[string]any)
		code   string
	}{
		{
			name: "unsupported schema",
			change: func(p map[string]any) {
				p["schemaVersion"] = "business-procedure-reference-v999"
			},
			code: "PROCEDURE_SOURCE_UNSUPPORTED",
		},
		{
			name: "unsupported authority role",
			change: func(p map[string]any) {
				p["canonical"] = true
			},
			code: "PROCEDURE_SOURCE_UNSUPPORTED",
		},
		{
			name: "duplicate source Step",
			change: func(p map[string]any) {
				steps := p["steps"].([]any)
				for _, item := range steps {
					if item.(map[string]any)["id"] == "BP-07" {
						p["steps"] = append(steps, item)
						return
					}
				}
				t.Fatal("BP-07 absent")
			},
			code: "PROCEDURE_SOURCE_STEP_AMBIGUOUS",
		},
		{
			name: "multiple explicit Patterns unsupported",
			change: func(p map[string]any) {
				for _, item := range p["steps"].([]any) {
					s := item.(map[string]any)
					if s["id"] == "BP-07" {
						s["applicationPatterns"] = []any{"order_entry.submit_order", "order_entry.order_record"}
						return
					}
				}
				t.Fatal("BP-07 absent")
			},
			code: "PROCEDURE_PATTERN_CARDINALITY_UNSUPPORTED",
		},
		{
			name: "changed declared Pattern with recomputed pin",
			change: func(p map[string]any) {
				for _, item := range p["steps"].([]any) {
					s := item.(map[string]any)
					if s["id"] == "BP-07" {
						s["applicationPatterns"] = []any{"order_entry.order_record"}
						return
					}
				}
				t.Fatal("BP-07 absent")
			},
			code: "PROCEDURE_PATTERN_MISMATCH",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			edited := rewriteProcedureForTest(t, procedure, tc.change)
			next := facts
			next.ProcedureJSONFingerprint = snapshotDigest(edited)
			report, err := InspectBoundProcedureSnapshot(world, edited, mustJSON(t, next))
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != "unresolved" || codeCount(report, tc.code) != 1 ||
				codeCount(report, "PROCEDURE_SNAPSHOT_STALE") != 0 ||
				report.InputFingerprints.ProcedureJSON != next.ProcedureJSONFingerprint {
				t.Fatalf("matched digest must not mask contradictory source declarations: %+v", report)
			}
		})
	}
}
