package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/threeelement"
)

func TestThreeElementCRUDCellCLIConsumesPriorSnapshotWithoutWriting(t *testing.T) {
	model := threeelement.NewThreeElementModel("order", "Order")
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
	var snapshot strings.Builder
	exit, err := runThreeElementCommand([]string{"crud-snapshot", "-model", path}, &snapshot)
	if err != nil || exit != 0 {
		t.Fatalf("snapshot exit=%d err=%v response=%s", exit, err, snapshot.String())
	}
	var envelope struct {
		Result struct {
			ModelFingerprint string `json:"modelFingerprint"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(snapshot.String()), &envelope); err != nil {
		t.Fatal(err)
	}
	var selected strings.Builder
	exit, err = runThreeElementCommand([]string{
		"crud-cell", "-model", path,
		"-expected-fingerprint", envelope.Result.ModelFingerprint,
		"-function", "submit", "-data", "order",
	}, &selected)
	if err != nil || exit != 0 {
		t.Fatalf("inspection exit=%d err=%v response=%s", exit, err, selected.String())
	}
	var got struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		OKScope       string `json:"okScope"`
		Result        struct {
			ModelFingerprint    string                            `json:"modelFingerprint"`
			SourceQualification string                            `json:"sourceQualification"`
			Row                 threeelement.CRUDTable            `json:"row"`
			Facts               []threeelement.CRUDFactOccurrence `json:"facts"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(selected.String()), &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.SchemaVersion != "tei.cli.three-element.crud-cell/v0" ||
		got.OKScope != "model-snapshot-cell-inspected-only" ||
		got.Result.ModelFingerprint != envelope.Result.ModelFingerprint ||
		got.Result.SourceQualification != "model-local-only" ||
		!got.Result.Row.Read || !got.Result.Row.Update || len(got.Result.Facts) != 2 {
		t.Fatalf("selected cell lost evidence or overstated qualifications: %+v", got)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(raw) {
		t.Fatal("read-only inspection changed the model")
	}
	model.FunctionElements[0].Operations[1].Operations = "D"
	changed, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	var stale strings.Builder
	exit, err = runThreeElementCommand([]string{
		"crud-cell", "-model", path,
		"-expected-fingerprint", envelope.Result.ModelFingerprint,
		"-function", "submit", "-data", "order",
	}, &stale)
	if err != nil || exit != 2 || !strings.Contains(stale.String(), "STALE_SNAPSHOT") {
		t.Fatalf("older origins were silently reused: exit=%d err=%v body=%s", exit, err, stale.String())
	}
}

func TestThreeElementCRUDCellCLIRefusesMissingAndUnknownSelection(t *testing.T) {
	var out strings.Builder
	exit, err := runThreeElementCommand([]string{"crud-cell", "-model", "model.json"}, &out)
	if err != nil || exit != 2 || !strings.Contains(out.String(), "INVALID_ARGUMENT") {
		t.Fatalf("missing expected fingerprint or pair not rejected: exit=%d err=%v body=%s", exit, err, out.String())
	}
	model := threeelement.NewThreeElementModel("model", "Model")
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	exit, err = runThreeElementCommand([]string{
		"crud-cell", "-model", path, "-expected-fingerprint", threeelement.ModelFingerprint(model),
		"-function", "missing", "-data", "missing",
	}, &out)
	if err != nil || exit != 2 || !strings.Contains(out.String(), "CRUD_CELL_NOT_FOUND") {
		t.Fatalf("missing pair not rejected: exit=%d err=%v body=%s", exit, err, out.String())
	}
}
