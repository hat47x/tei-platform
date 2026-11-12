package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/threeelement"
)

// The CLI result's OK bit has a narrower meaning than consistency or
// authorization. An explicit conflict must be visible to a JSON consumer,
// even if a read-only table can still be generated without a hard failure.
func TestThreeElementCLIExposesCRUDConflictReviewSeparatelyFromOK(t *testing.T) {
	model := threeelement.NewThreeElementModel("cli-conflict", "CLI conflict fixture")
	model.DataElements = []threeelement.DataElement{
		{ID: "order", Name: "Order"},
		{ID: "invoice", Name: "Invoice"},
	}
	model.FunctionElements = []threeelement.FunctionElement{
		{ID: "submit", Name: "Submit", Operations: []threeelement.FunctionOperation{{DataID: "order", Operations: "R"}}},
		{ID: "bill", Name: "Bill", Operations: []threeelement.FunctionOperation{{DataID: "invoice", Operations: "R"}}},
	}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{
		{ID: "order-c", FunctionID: "submit", DataID: "order", Create: true},
		{ID: "invoice-r", FunctionID: "bill", DataID: "invoice", Read: true},
		{ID: "order-r", FunctionID: "submit", DataID: "order", Read: true},
		{ID: "invoice-u", FunctionID: "bill", DataID: "invoice", Update: true},
	}
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		command string
		scope   string
	}{
		{command: "validate", scope: "structural-diagnostics-only"},
		{command: "crud", scope: "projection-produced-only"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			var stdout strings.Builder
			exit, err := runThreeElementCommand([]string{tc.command, "-model", file}, &stdout)
			if err != nil || exit != 0 {
				t.Fatalf("%s failed: exit %d, err %v, body %s", tc.command, exit, err, stdout.String())
			}
			var envelope threeElementResponse
			if err := json.Unmarshal([]byte(stdout.String()), &envelope); err != nil {
				t.Fatalf("bad JSON: %v: %s", err, stdout.String())
			}
			if !envelope.OK || envelope.OKScope != tc.scope {
				t.Fatalf("OK true must be explicitly scoped for %s: %+v", tc.command, envelope)
			}
			result, ok := envelope.Result.(map[string]any)
			if !ok {
				t.Fatalf("unexpected CLI result: %#v", envelope.Result)
			}
			if got := result["crudConflictStatus"]; got != "conflicts-require-review" {
				t.Fatalf("missing conflict status for %s: %v", tc.command, got)
			}
			entries, ok := result["contradictoryCrudDefinitions"].([]any)
			if !ok || len(entries) != 2 {
				t.Fatalf("expected two independent conflicts for %s: %#v", tc.command, result["contradictoryCrudDefinitions"])
			}
			for _, record := range entries {
				crud, ok := record.(map[string]any)
				if !ok || crud["functionId"] == nil || crud["dataId"] == nil {
					t.Fatalf("conflict must carry Function/Data IDs: %#v", record)
				}
			}
			if tc.command == "validate" {
				if result["okScope"] != tc.scope || result["errorCount"] != float64(0) {
					t.Fatalf("structural diagnostics and conflict review are separate: %+v", result)
				}
			} else {
				if result["structuralOk"] != true {
					t.Fatalf("valid structure should remain separate from conflicts: %+v", result)
				}
				if _, ok := result["rows"]; !ok {
					t.Fatalf("read-only CRUD projection was not returned: %+v", result)
				}
			}
		})
	}
}
