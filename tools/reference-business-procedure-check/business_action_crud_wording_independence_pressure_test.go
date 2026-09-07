package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestBusinessActionCRUDProjectionDoesNotInferUpdateFromBusinessWording(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	baseline, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	baselineRow := businessActionCRUDRowsByStep(baseline)["BP-01"]
	if !baselineRow.NoCRUDTraceFromCurrentRefs || len(baselineRow.Cells) != 0 {
		t.Fatalf("test setup: BP-01 baselineにCRUD traceがあります: %+v", baselineRow)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	found := false
	for i := range changed.Steps {
		if changed.Steps[i].ID != "BP-01" {
			continue
		}
		changed.Steps[i].BusinessAction = "edit, correct, and update the selected order before submission"
		found = true
	}
	if !found {
		t.Fatal("test setup: BP-01がありません")
	}

	projected, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, changed)
	if err != nil {
		t.Fatal(err)
	}
	changedRow := businessActionCRUDRowsByStep(projected)["BP-01"]
	if !reflect.DeepEqual(changedRow, baselineRow) {
		t.Fatalf("Business wordingだけからCRUD operationを推測しました: baseline=%+v changed=%+v", baselineRow, changedRow)
	}
}

func TestBusinessActionCRUDProjectionDoesNotInferCreateDeleteFromPurposeOrActionWords(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Procedure = f.Procedure
	changed.Procedure.Purpose = "create, maintain, and delete orders as requested by the surrounding business work"
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-01" {
			changed.Steps[i].BusinessAction = "create or delete the selected order record when needed"
		}
	}

	rows, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, changed)
	if err != nil {
		t.Fatal(err)
	}
	row := businessActionCRUDRowsByStep(rows)["BP-01"]
	if !row.NoCRUDTraceFromCurrentRefs || len(row.Cells) != 0 {
		t.Fatalf("Purpose/BusinessActionのC/D語からCRUD operationを推測しました: %+v", row)
	}
}
