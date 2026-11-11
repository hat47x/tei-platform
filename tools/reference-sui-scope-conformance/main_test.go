package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadMatrix(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "reference", "sui-scope-conformance", "matrix.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read matrix: %v", err)
	}
	return payload
}

func matrix(t *testing.T) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(loadMatrix(t), &value); err != nil {
		t.Fatalf("decode matrix: %v", err)
	}
	return value
}

func encode(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return payload
}

// M6: 実装した行列は、行・既知の不足・not-run を持つ。
func TestMatrixIsAccepted(t *testing.T) {
	report := Check(loadMatrix(t))
	if !report.OK {
		t.Fatalf("matrix rejected: %+v", report)
	}
	if report.Rows < 14 {
		t.Fatalf("rows = %d, want at least 14", report.Rows)
	}
	if report.Gaps == 0 {
		t.Fatal("matrix declares no known gap")
	}
	if report.NotRun == 0 {
		t.Fatal("matrix records no not-run cell")
	}
	if report.Unsupported == 0 {
		t.Fatal("matrix records no unsupported cell")
	}
}

// 空欄を pass と読み替えない。
func TestEmptyCellIsRejected(t *testing.T) {
	value := matrix(t)
	rows := value["rows"].([]any)
	rows[0].(map[string]any)["tenantCrossing"] = ""
	report := Check(encode(t, value))
	if report.Failure != FailureInvalidMatrix {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidMatrix)
	}
}

// 語彙に無い状態を拒否する。
func TestUnknownCellStateIsRejected(t *testing.T) {
	value := matrix(t)
	rows := value["rows"].([]any)
	rows[0].(map[string]any)["operationBoundary"] = "ok"
	report := Check(encode(t, value))
	if report.Failure != FailureInvalidMatrix {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidMatrix)
	}
}

// 行列は認可の正本ではない。
func TestAuthorityGate(t *testing.T) {
	value := matrix(t)
	value["authority"] = "tei-product-gate"
	report := Check(encode(t, value))
	if report.Failure != FailureAuthorized {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureAuthorized)
	}
}

func TestSchemaGate(t *testing.T) {
	value := matrix(t)
	value["schema"] = "other/v0"
	report := Check(encode(t, value))
	if report.Failure != FailureUnsupportedSchema {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureUnsupportedSchema)
	}
}

func TestUnknownFieldIsRejected(t *testing.T) {
	value := matrix(t)
	value["extra"] = "secret"
	report := Check(encode(t, value))
	if report.Failure != FailureUnsupportedSchema {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureUnsupportedSchema)
	}
}

// 宣言した既知の不足が not-run で示されていない場合を拒否する。
func TestDeclaredGapMustBeNotRun(t *testing.T) {
	value := matrix(t)
	// list 行を pass に変えると、宣言した gap と矛盾する。
	for _, item := range value["rows"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "list" {
			row["tenantCrossing"] = "pass"
			row["operationBoundary"] = "pass"
		}
	}
	report := Check(encode(t, value))
	if report.Failure != FailureInvalidMatrix {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidMatrix)
	}
}

func TestKGramsCannotBeEmpty(t *testing.T) {
	value := matrix(t)
	value["knownGaps"] = []any{}
	report := Check(encode(t, value))
	if report.Failure != FailureInvalidMatrix {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidMatrix)
	}
}

func TestRowKindMustBeUnique(t *testing.T) {
	value := matrix(t)
	rows := value["rows"].([]any)
	rows[1].(map[string]any)["kind"] = rows[0].(map[string]any)["kind"]
	report := Check(encode(t, value))
	if report.Failure != FailureInvalidMatrix {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidMatrix)
	}
	for _, problem := range report.Problems {
		if strings.Contains(problem, "doc-synthetic") {
			t.Fatalf("problem leaked a value: %s", problem)
		}
	}
}
