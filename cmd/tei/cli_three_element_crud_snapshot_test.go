package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/threeelement"
)

func TestThreeElementCRUDSnapshotCLIIsBoundAndReadOnly(t *testing.T) {
	model := threeelement.NewThreeElementModel("snapshot", "Snapshot")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []threeelement.FunctionOperation{
			{DataID: "order", Operations: "R"},
			{DataID: "order", Operations: "U"},
		},
	}}
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	exit, err := runThreeElementCommand([]string{"crud-snapshot", "-model", path}, &out)
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v body=%s", exit, err, out.String())
	}
	var got struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		OKScope       string `json:"okScope"`
		Result        struct {
			SchemaVersion    string                        `json:"schemaVersion"`
			ModelFingerprint string                        `json:"modelFingerprint"`
			Rows             []threeelement.CRUDTable      `json:"rows"`
			Origins          []threeelement.CRUDCellOrigin `json:"origins"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != "tei.cli.three-element.crud-snapshot/v0" ||
		got.Result.SchemaVersion != "tei.three-element.crud-snapshot/v0" ||
		!got.OK || got.OKScope != "projection-produced-only" {
		t.Fatalf("unexpected response contract: %+v", got)
	}
	if got.Result.ModelFingerprint != threeelement.ModelFingerprint(model) ||
		len(got.Result.Rows) != 1 || len(got.Result.Origins) != 1 ||
		len(got.Result.Origins[0].Facts) != 2 ||
		!got.Result.Rows[0].Read || !got.Result.Rows[0].Update {
		t.Fatalf("lost CRUD cell or original declarations: %+v", got.Result)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(raw) {
		t.Fatal("read-only command changed its model file")
	}
}

func TestThreeElementCRUDSnapshotCLIRejectsMissingModel(t *testing.T) {
	var out strings.Builder
	exit, err := runThreeElementCommand([]string{"crud-snapshot"}, &out)
	if err != nil || exit != 2 {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if !strings.Contains(out.String(), "INVALID_ARGUMENT") {
		t.Fatal(out.String())
	}
}
