package main

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func TestBaselineReferenceResolvesAndKeepsManualActions(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	result := evaluate(loaded.World, f)
	if err := checkExpectedDiagnostics(result.Diagnostics, ""); err != nil {
		t.Fatalf("baselineにblocking diagnosticがあります: %v\n%+v", err, result.Diagnostics)
	}

	manual := map[string]bool{}
	for _, row := range result.BusinessActionFunctionRows {
		if row.NoApplicationFunction {
			manual[row.StepID] = true
		}
	}
	for _, stepID := range []string{"BP-04", "BP-09"} {
		if !manual[stepID] {
			t.Fatalf("%sはApplication Functionなしで保持される必要があります: %+v", stepID, result.BusinessActionFunctionRows)
		}
	}

	if result.CanonicalProcedureIdentityUsed {
		t.Fatal("derived ReferenceがCanonical Procedure identityを使用しています")
	}
}

func TestCustomerEditConflictIsDetected(t *testing.T) {
	root := repoPath()
	result, err := run(
		filepath.Join(root, "reference", "workspace"),
		filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"),
		filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkExpectedDiagnostics(result.Diagnostics, "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT"); err != nil {
		t.Fatalf("intentional conflictが期待どおり再現されません: %v\n%+v", err, result.Diagnostics)
	}
}

func TestReadAccessBaselineDoesNotConflict(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	result := evaluate(loaded.World, f)
	for _, item := range result.Diagnostics {
		if item.Code == "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT" {
			t.Fatalf("read-only baselineでaccess conflictが発生しました: %+v", item)
		}
	}
}

func TestFixtureDecodeRejectsUnknownFields(t *testing.T) {
	_, err := decodeFixture(strings.NewReader(`{
		"schemaVersion":"business-procedure-reference-v0",
		"artifactRole":"derived-reference-fixture",
		"canonical":false,
		"application":"order_entry",
		"procedure":{"id":"p","identityStatus":"experiment-local","purpose":"p","trigger":{"status":"candidate"},"actor":{"status":"candidate"}},
		"steps":[],
		"unknownAuthorityShortcut":"must-not-be-ignored"
	}`))
	if err == nil {
		t.Fatal("unknown fieldを黙って受理しました")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field以外の理由で失敗しました: %v", err)
	}
}

func repoPath() string {
	return filepath.Clean(filepath.Join("..", ".."))
}
