package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/driver"
	"tei/internal/workspace"
)

func TestBusinessProcedureObservationDoesNotUseRuntimeReadOperationAsSemanticIdentity(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	steps := stepsByID(f)
	observation, ok := steps["BP-08"]
	if !ok {
		t.Fatal("BP-08 Observation Business Actionがありません")
	}
	if len(observation.ApplicationPatterns) != 0 || len(observation.Capabilities) != 0 {
		t.Fatalf("BP-08へ未検証のObservation Pattern / Capabilityを追加しました: %+v", observation)
	}

	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-08" {
			changed.Steps[i].Capabilities = []string{"internal.driver.ExecuteObservationRead"}
		}
	}
	result := evaluate(loaded.World, changed)
	if err := checkExpectedDiagnostics(result.Diagnostics, "BUSINESS_CAPABILITY_REF_NOT_FOUND"); err != nil {
		t.Fatalf("runtime Observation operationをSemantic Capabilityとして拒否できません: %v\n%+v", err, result.Diagnostics)
	}
}

func TestDriverObservationReadIdentityIsRuntimeEvidenceLocatorNotCanonicalID(t *testing.T) {
	typeOfExecution := reflect.TypeOf(driver.ObservationReadExecution{})
	field, ok := typeOfExecution.FieldByName("ReadIdentity")
	if !ok {
		t.Fatal("ObservationReadExecution.ReadIdentityがありません")
	}
	if field.Type.Kind() != reflect.String {
		t.Fatalf("current Observation runtime boundaryの再監査が必要です: ReadIdentity type=%s", field.Type)
	}

	// A plain runtime string is intentionally not interpreted here as a
	// core.CanonicalID. The authoritative reason is ADR-0023 and driver source:
	// Observation may use Query / Contract Query / Binding / Adapter or a
	// host/test-local read boundary, so Business Procedure must not promote this
	// execution locator merely to obtain a symmetric typed step reference.
}
