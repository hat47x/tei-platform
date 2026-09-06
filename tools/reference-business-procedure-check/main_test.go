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
		t.Fatal("Reference authoring fixtureがCanonical Procedure identityを使用しています")
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

func TestPatternOwnedMeaningIsDerivedWithoutFixtureDuplication(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range f.Steps {
		switch step.ID {
		case "BP-01":
			if len(step.Data) != 0 {
				t.Fatalf("BP-01がPattern-owned Dataをfixtureへ重複保持しています: %+v", step.Data)
			}
		case "BP-02", "BP-03":
			if len(step.Relations) != 0 {
				t.Fatalf("%sがPattern-owned Relationをfixtureへ重複保持しています: %+v", step.ID, step.Relations)
			}
		case "BP-07":
			if len(step.Capabilities) != 0 {
				t.Fatalf("BP-07がPattern-owned Capabilityをfixtureへ重複保持しています: %+v", step.Capabilities)
			}
		}
	}

	result := evaluate(loaded.World, f)
	rows := map[string]patternMeaningRow{}
	for _, row := range result.BusinessActionPatternMeaningRows {
		rows[row.StepID] = row
	}
	if rows["BP-01"].Data != "sales.order" {
		t.Fatalf("BP-01 DataがPatternから導出されていません: %+v", rows["BP-01"])
	}
	if rows["BP-02"].Relation != "sales.order.customer" || rows["BP-02"].Datasource != "master" {
		t.Fatalf("BP-02 Relation/DatasourceがPatternから導出されていません: %+v", rows["BP-02"])
	}
	if rows["BP-03"].Relation != "sales.order.lines" {
		t.Fatalf("BP-03 RelationがPatternから導出されていません: %+v", rows["BP-03"])
	}
	if rows["BP-07"].Capability != "sales.order.submit" {
		t.Fatalf("BP-07 CapabilityがPatternから導出されていません: %+v", rows["BP-07"])
	}
}

func TestFixtureDecodeRejectsUnknownFields(t *testing.T) {
	_, err := decodeFixture(strings.NewReader(`{
		"schemaVersion":"business-procedure-reference-v0",
		"artifactRole":"reference-authoring-fixture",
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

func TestFixtureDecodeRejectsUnsupportedSchemaVersion(t *testing.T) {
	_, err := decodeFixture(strings.NewReader(`{
		"schemaVersion":"business-procedure-reference-v1",
		"artifactRole":"reference-authoring-fixture",
		"canonical":false,
		"application":"order_entry",
		"procedure":{"id":"p","identityStatus":"experiment-local","purpose":"p","trigger":{"status":"candidate"},"actor":{"status":"candidate"}},
		"steps":[]
	}`))
	if err == nil {
		t.Fatal("unsupported schemaVersionを黙って受理しました")
	}
	if !strings.Contains(err.Error(), "unsupported schemaVersion") {
		t.Fatalf("schemaVersion以外の理由で失敗しました: %v", err)
	}
}

func TestFixtureDecodeRejectsWrongArtifactRole(t *testing.T) {
	_, err := decodeFixture(strings.NewReader(`{
		"schemaVersion":"business-procedure-reference-v0",
		"artifactRole":"canonical-authority",
		"canonical":false,
		"application":"order_entry",
		"procedure":{"id":"p","identityStatus":"experiment-local","purpose":"p","trigger":{"status":"candidate"},"actor":{"status":"candidate"}},
		"steps":[]
	}`))
	if err == nil {
		t.Fatal("別artifact roleを黙って受理しました")
	}
	if !strings.Contains(err.Error(), "unsupported artifactRole") {
		t.Fatalf("artifactRole以外の理由で失敗しました: %v", err)
	}
}

func repoPath() string {
	return filepath.Clean(filepath.Join("..", ".."))
}
