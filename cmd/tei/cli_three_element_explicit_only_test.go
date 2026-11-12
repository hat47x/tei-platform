package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/threeelement"
)

// Dispatch through the shipped CLI command, not a projection test helper.
// A valid explicit-only CRUD mapping must appear in both user-facing views.
func TestThreeElementCLIIncludesExplicitOnlyCRUDAndCrossReference(t *testing.T) {
	model := threeelement.NewThreeElementModel("cli-explicit-only", "Explicit-only CLI fixture")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order data"}}
	model.BusinessElements = []threeelement.BusinessElement{{ID: "order", Name: "Business order"}}
	model.FunctionElements = []threeelement.FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{{
		ID: "submit-order", FunctionID: "submit", DataID: "order", Create: true, Source: "declared",
	}}
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cmd       string
		resultKey string
	}{
		{cmd: "crud", resultKey: "rows"},
		{cmd: "cross-reference", resultKey: "functionData"},
	} {
		t.Run(tc.cmd, func(t *testing.T) {
			var stdout strings.Builder
			exit, err := runThreeElementCommand([]string{tc.cmd, "-model", file}, &stdout)
			if err != nil || exit != 0 {
				t.Fatalf("CLI failed: exit=%d err=%v body=%s", exit, err, stdout.String())
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(stdout.String()), &body); err != nil {
				t.Fatalf("invalid JSON: %v: %s", err, stdout.String())
			}
			if body["ok"] != true {
				t.Fatalf("read-only CLI failed unexpectedly: %+v", body)
			}
			result, ok := body["result"].(map[string]any)
			if !ok {
				t.Fatalf("missing result: %+v", body)
			}
			rows, ok := result[tc.resultKey].([]any)
			if !ok || len(rows) != 1 {
				t.Fatalf("explicit-only mapping missing from %s: %+v", tc.cmd, result)
			}
			row, ok := rows[0].(map[string]any)
			if !ok || row["functionId"] != "submit" || row["functionName"] != "Submit" ||
				row["dataId"] != "order" || row["dataName"] != "Order data" ||
				row["source"] != "declared" {
				t.Fatalf("wrong qualified CRUD row: %+v", rows)
			}
			if tc.cmd == "crud" {
				if body["okScope"] != "projection-produced-only" ||
					result["crudConflictStatus"] != "no-conflicts-detected" ||
					row["create"] != true {
					t.Fatalf("CRUD projection lost claim scope or flags: %+v", body)
				}
			} else if row["operations"] != "C" {
				t.Fatalf("cross-reference did not carry explicit CRUD: %+v", row)
			}
		})
	}
}

// The UI may render a read-only projection, but a conflict indicator must
// remain visible even when the operation flags are last-writer projected.
func TestThreeElementCLIExplicitOnlyConflictIsNotSilentlyAccepted(t *testing.T) {
	model := threeelement.NewThreeElementModel("cli-explicit-conflict", "Conflict fixture")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{
		{ID: "first", FunctionID: "submit", DataID: "order", Create: true},
		{ID: "second", FunctionID: "submit", DataID: "order", Read: true},
	}
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "conflicting.json")
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cmd       string
		resultKey string
	}{
		{cmd: "crud", resultKey: "rows"},
		{cmd: "cross-reference", resultKey: "functionData"},
	} {
		t.Run(tc.cmd, func(t *testing.T) {
			var stdout strings.Builder
			code, err := runThreeElementCommand([]string{tc.cmd, "-model", file}, &stdout)
			if code != 0 || err != nil {
				t.Fatalf("read-only projection must remain available: code=%d err=%v", code, err)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(stdout.String()), &body); err != nil {
				t.Fatal(err)
			}
			result := body["result"].(map[string]any)
			rows := result[tc.resultKey].([]any)
			if len(rows) != 1 || rows[0].(map[string]any)["conflictingDeclarations"] != true {
				t.Fatalf("unresolved CRUD conflict was presented as accepted: %s", stdout.String())
			}
			if tc.cmd == "crud" && result["crudConflictStatus"] != "conflicts-require-review" {
				t.Fatalf("CLI projection hid the Owner-review status: %s", stdout.String())
			}
		})
	}
}
