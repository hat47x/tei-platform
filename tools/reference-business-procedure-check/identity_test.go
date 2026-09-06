package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestExperimentLocalProcedureRenameDoesNotChangeCrossViewResolution(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	before := evaluate(loaded.World, f)
	f.Procedure.ID = "order-entry.order-processing-local-rename"
	after := evaluate(loaded.World, f)

	if !reflect.DeepEqual(before.BusinessActionFunctionRows, after.BusinessActionFunctionRows) {
		t.Fatalf("experiment-local Procedure renameでFunction cross-viewが変わりました\nbefore=%+v\nafter=%+v", before.BusinessActionFunctionRows, after.BusinessActionFunctionRows)
	}
	if !reflect.DeepEqual(before.ExplicitBusinessMeaningRows, after.ExplicitBusinessMeaningRows) {
		t.Fatalf("experiment-local Procedure renameでexplicit semantic referenceが変わりました\nbefore=%+v\nafter=%+v", before.ExplicitBusinessMeaningRows, after.ExplicitBusinessMeaningRows)
	}
	if !reflect.DeepEqual(before.Diagnostics, after.Diagnostics) {
		t.Fatalf("experiment-local Procedure renameでdiagnosticが変わりました\nbefore=%+v\nafter=%+v", before.Diagnostics, after.Diagnostics)
	}
	if before.Procedure == after.Procedure {
		t.Fatal("test setupがProcedure local IDを変更できていません")
	}
}

func TestExplicitReferenceDoesNotGuessFromSimilarName(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	for i := range f.Steps {
		if f.Steps[i].ID == "BP-02" {
			f.Steps[i].Relations = []string{"sales.order.Customer"}
		}
	}
	result := evaluate(loaded.World, f)

	found := false
	for _, item := range result.Diagnostics {
		if item.Step == "BP-02" && item.Code == "BUSINESS_RELATION_REF_NOT_FOUND" {
			found = true
		}
	}
	if !found {
		t.Fatalf("similar nameを推測せずmissingとするdiagnosticがありません: %+v", result.Diagnostics)
	}
}
