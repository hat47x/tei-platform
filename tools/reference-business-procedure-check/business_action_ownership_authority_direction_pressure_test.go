package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestBusinessProcedureSourceDoesNotOwnDatabaseOwnershipBoundary(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(fixture{}),
		reflect.TypeOf(procedure{}),
		reflect.TypeOf(step{}),
	} {
		for _, forbidden := range []string{
			"Database",
			"Databases",
			"DatabaseOwnership",
			"Ownership",
			"Access",
			"Subsystem",
			"SubjectArea",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("Business Procedure source type %sがderived ownership authority %qを所有しています", typ.Name(), forbidden)
			}
		}
	}
}

func TestBusinessActionOwnershipBoundaryProjectionDoesNotInferBoundaryFromBusinessWording(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	baseline, err := deriveBusinessActionOwnershipBoundaryProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Procedure = f.Procedure
	changed.Procedure.Purpose = "work across an external subsystem and update externally owned records"
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-02" {
			changed.Steps[i].BusinessAction = "maintain customer data in an external subsystem"
		}
	}

	projected, err := deriveBusinessActionOwnershipBoundaryProjectionProbe(loaded.World, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projected, baseline) {
		t.Fatalf("Business wordingだけからownership/subsystem boundaryを推測しました: baseline=%+v changed=%+v", baseline, projected)
	}
}
