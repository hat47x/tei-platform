package main

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestCustomerEditConflictProbeDoesNotBecomeAcceptedCRUDUpdate(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	probe, err := readProbe(filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json"))
	if err != nil {
		t.Fatal(err)
	}
	if probe.Step != "BP-02" || probe.ApplicationPattern != "order_entry.customer_reference" {
		t.Fatalf("customer-edit conflict probeの対象が変化しました: %+v", probe)
	}

	result := evaluate(loaded.World, f)
	applyAccessProbe(&result, loaded.World, f, probe)
	if err := checkExpectedDiagnostics(result.Diagnostics, "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT"); err != nil {
		t.Fatalf("intentional customer-edit conflictがblocking diagnosticとして残っていません: %v\n%+v", err, result.Diagnostics)
	}

	rows, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	row := businessActionCRUDRowsByStep(rows)["BP-02"]
	if len(row.Cells) != 1 || row.Cells[0].Data != "customer.customer" || row.Cells[0].Operation != crudReadProbe {
		t.Fatalf("intentional edit probeがaccepted BP-02 CRUD meaningへ逆流しました: %+v", row)
	}
}

func TestCurrentPrimaryReferenceHasNoAcceptedCUDCells(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		for _, cell := range row.Cells {
			if cell.Operation == crudCreateProbe || cell.Operation == crudUpdateProbe || cell.Operation == crudDeleteProbe {
				t.Fatalf("current TEI-original primary Referenceにaccepted sourceのないC/U/D cellが現れました: step=%s cell=%+v", row.StepID, cell)
			}
		}
	}
}
