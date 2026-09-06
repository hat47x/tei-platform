package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

type patternUsageEvidence struct {
	Artifact  string
	Procedure string
	Step      string
	Pattern   string
}

func TestMultipleProceduresCanReferenceSamePatternWithoutCanonicalProcedureIdentity(t *testing.T) {
	root := repoPath()
	workspaceResult, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	primaryPath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	followUpPath := filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json")
	primary, err := readFixture(primaryPath)
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(followUpPath)
	if err != nil {
		t.Fatal(err)
	}

	primaryResult := evaluate(workspaceResult.World, primary)
	followUpResult := evaluate(workspaceResult.World, followUp)
	if err := checkExpectedDiagnostics(primaryResult.Diagnostics, ""); err != nil {
		t.Fatalf("primary procedureにblocking diagnosticがあります: %v", err)
	}
	if err := checkExpectedDiagnostics(followUpResult.Diagnostics, ""); err != nil {
		t.Fatalf("follow-up procedureにblocking diagnosticがあります: %v", err)
	}

	patternID := "order_entry.order_record"
	usages := append(
		patternUsages(filepath.Base(primaryPath), primary, primaryResult, patternID),
		patternUsages(filepath.Base(followUpPath), followUp, followUpResult, patternID)...,
	)
	if len(usages) != 2 {
		t.Fatalf("同一Patternへの2つのProcedure usageをtraceできません: %+v", usages)
	}
	if usages[0].Procedure == usages[1].Procedure {
		t.Fatalf("異なるexperiment-local Procedureを区別できません: %+v", usages)
	}
	for _, usage := range usages {
		if usage.Pattern != patternID {
			t.Fatalf("unexpected Pattern usage: %+v", usage)
		}
	}
}

func TestArtifactAndStepEvidenceLocatorSurvivesExperimentLocalProcedureRename(t *testing.T) {
	root := repoPath()
	workspaceResult, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json")
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	beforeResult := evaluate(workspaceResult.World, f)
	before := patternUsages(filepath.Base(fixturePath), f, beforeResult, "order_entry.order_record")

	f.Procedure.ID = "order-entry.follow-up-local-rename"
	afterResult := evaluate(workspaceResult.World, f)
	after := patternUsages(filepath.Base(fixturePath), f, afterResult, "order_entry.order_record")

	if len(before) != 1 || len(after) != 1 {
		t.Fatalf("usage evidenceを一意に取得できません: before=%+v after=%+v", before, after)
	}
	if before[0].Procedure == after[0].Procedure {
		t.Fatal("test setupがProcedure local IDを変更できていません")
	}
	beforeLocator := []string{before[0].Artifact, before[0].Step, before[0].Pattern}
	afterLocator := []string{after[0].Artifact, after[0].Step, after[0].Pattern}
	if !reflect.DeepEqual(beforeLocator, afterLocator) {
		t.Fatalf("artifact + step + PatternのEvidence locatorがProcedure local renameで変わりました: before=%+v after=%+v", beforeLocator, afterLocator)
	}
	if !reflect.DeepEqual(beforeResult.BusinessActionPatternMeaningRows, afterResult.BusinessActionPatternMeaningRows) {
		t.Fatalf("Procedure local renameでPattern-derived meaningが変わりました")
	}
}

func TestArtifactRenameStillChangesNonCanonicalEvidenceLocator(t *testing.T) {
	f := fixture{Procedure: procedure{ID: "order-entry.follow-up"}}
	result := report{BusinessActionFunctionRows: []functionRow{{
		StepID:              "FU-01",
		ApplicationPatterns: []string{"order_entry.order_record"},
	}}}

	before := patternUsages("order-entry-follow-up-procedure.json", f, result, "order_entry.order_record")
	after := patternUsages("renamed-follow-up-procedure.json", f, result, "order_entry.order_record")
	if len(before) != 1 || len(after) != 1 {
		t.Fatalf("usage evidenceを一意に取得できません: before=%+v after=%+v", before, after)
	}
	if before[0].Artifact == after[0].Artifact {
		t.Fatal("test setupがartifact path identityを変更できていません")
	}
	if before[0].Step != after[0].Step || before[0].Pattern != after[0].Pattern {
		t.Fatalf("artifact rename以外のmeaning locatorまで変化しました: before=%+v after=%+v", before[0], after[0])
	}
}

func patternUsages(artifact string, f fixture, result report, patternID string) []patternUsageEvidence {
	var usages []patternUsageEvidence
	for _, row := range result.BusinessActionFunctionRows {
		for _, pattern := range row.ApplicationPatterns {
			if pattern != patternID {
				continue
			}
			usages = append(usages, patternUsageEvidence{
				Artifact:  artifact,
				Procedure: f.Procedure.ID,
				Step:      row.StepID,
				Pattern:   pattern,
			})
		}
	}
	return usages
}
