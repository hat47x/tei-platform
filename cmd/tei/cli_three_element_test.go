package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/threeelement"
)

var fixtureBytes = func() []byte {
	model := threeelement.NewThreeElementModel("test-model", "Test Model")
	model.DataElements = []threeelement.DataElement{
		{
			ID:   "test.data",
			Name: "Test Data",
			Attributes: []threeelement.DataAttribute{
				{ID: "id", Name: "ID", Type: "identifier", Key: true},
			},
		},
	}
	raw, _ := json.Marshal(model)
	return raw
}()

func TestThreeElementValidateCommand(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"validate", "-model", modelPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK validation, got %+v", resp.Result)
	}
	// The schemaVersion is part of the response contract, so a rename here is
	// a breaking change and not a cosmetic edit.
	if resp.SchemaVersion != "tei.cli.three-element.validate/v0" {
		t.Fatalf("schemaVersion = %q", resp.SchemaVersion)
	}
	// The report carries its own identifier: it is the model schema the
	// three-element source declares, not the CLI envelope's.
	if result, ok := resp.Result.(map[string]any); ok {
		if result["schemaVersion"] != "tei.three-element.model/v0" {
			t.Fatalf("result schemaVersion = %v", result["schemaVersion"])
		}
	}
}

func TestThreeElementCRUDCommand(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"crud", "-model", modelPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK response, got %+v", resp.Result)
	}
	if resp.SchemaVersion != "tei.cli.three-element.crud/v0" {
		t.Fatalf("schemaVersion = %q", resp.SchemaVersion)
	}
	// The CRUD table carries the model's own identifier, which is the
	// three-element source schema rather than the CLI envelope's.
	if result, ok := resp.Result.(map[string]any); ok {
		if result["schemaVersion"] != "tei.three-element.crud/v0" {
			t.Fatalf("result schemaVersion = %v", result["schemaVersion"])
		}
	}
}

// The cross-reference subcommand has its own envelope identifier and a
// per-result one. Both are response contracts, so both are pinned here.
func TestThreeElementCrossReferenceCommand(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"cross-reference", "-model", modelPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.SchemaVersion != "tei.cli.three-element.cross-reference/v0" {
		t.Fatalf("schemaVersion = %q", resp.SchemaVersion)
	}
	if result, ok := resp.Result.(map[string]any); ok {
		if result["schemaVersion"] != "tei.three-element.cross-reference/v0" {
			t.Fatalf("result schemaVersion = %v", result["schemaVersion"])
		}
	}
}

func TestThreeElementDiffCommand(t *testing.T) {
	root := t.TempDir()
	before := filepath.Join(root, "before.json")
	after := filepath.Join(root, "after.json")
	if err := os.WriteFile(before, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(after, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"diff", "-before", before, "-after", after}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK response, got %+v", resp.Result)
	}
	if resp.SchemaVersion != "tei.cli.three-element.diff/v0" {
		t.Fatalf("schemaVersion = %q", resp.SchemaVersion)
	}
}

// The top-level command answers with its own identifier when a subcommand is
// missing, so that failure is machine-readable before the subcommand runs.
func TestThreeElementWithoutASubcommandNamesItsSchema(t *testing.T) {
	var out strings.Builder
	code, err := runThreeElementCommand(nil, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 2 {
		t.Fatalf("expected exit 2, got %d: %s", code, out.String())
	}
	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.SchemaVersion != "tei.cli.three-element/v0" {
		t.Fatalf("schemaVersion = %q", resp.SchemaVersion)
	}
	if resp.OK {
		t.Fatal("a missing subcommand must not report ok")
	}
}

func TestThreeElementCrossReferenceCommand(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"cross-reference", "-model", modelPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK response, got %+v", resp.Result)
	}
}
