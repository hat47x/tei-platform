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
}

// The CLI envelope carries one schemaVersion per subcommand, and a caller
// branches on it. These identifiers were written without any test naming them,
// so each is asserted here through the command that writes it (#4032).
func TestThreeElementCommandsAdvertiseTheirSchema(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(t *testing.T, want string, args ...string) {
		t.Helper()
		var out strings.Builder
		code, err := runThreeElementCommand(args, &out)
		if err != nil || code != 0 {
			t.Fatalf("%v: code=%d err=%v body=%s", args, code, err, out.String())
		}
		var resp threeElementResponse
		if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
			t.Fatalf("%v: invalid JSON: %v", args, err)
		}
		if resp.SchemaVersion != want {
			t.Fatalf("%v: schemaVersion=%q want %q", args, resp.SchemaVersion, want)
		}
	}

	run(t, "tei.cli.three-element.validate/v0", "validate", "-model", modelPath)
	run(t, "tei.cli.three-element.crud/v0", "crud", "-model", modelPath)
	run(t, "tei.cli.three-element.cross-reference/v0", "cross-reference", "-model", modelPath)
	// The result body carries its own identifier, nested under the envelope.
	for _, inner := range []struct {
		command string
		want    string
	}{
		{"crud", "tei.three-element.crud/v0"},
		{"cross-reference", "tei.three-element.cross-reference/v0"},
	} {
		var out strings.Builder
		code, err := runThreeElementCommand([]string{inner.command, "-model", modelPath}, &out)
		if err != nil || code != 0 {
			t.Fatalf("%s: code=%d err=%v", inner.command, code, err)
		}
		var nested struct {
			Result struct {
				SchemaVersion string `json:"schemaVersion"`
			} `json:"result"`
		}
		if jsonErr := json.Unmarshal([]byte(out.String()), &nested); jsonErr != nil {
			t.Fatalf("%s: invalid JSON: %v", inner.command, jsonErr)
		}
		if nested.Result.SchemaVersion != inner.want {
			t.Fatalf("%s: result schemaVersion=%q want %q", inner.command, nested.Result.SchemaVersion, inner.want)
		}
	}
	run(t, "tei.cli.three-element.diff/v0", "diff", "-before", modelPath, "-after", modelPath)

	// The failure envelope has its own identifier.
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"no-such-subcommand"}, &out)
	if err == nil && code == 0 {
		t.Fatalf("an unknown subcommand was accepted: %s", out.String())
	}
	var resp threeElementResponse
	if jsonErr := json.Unmarshal([]byte(out.String()), &resp); jsonErr != nil {
		t.Fatalf("invalid JSON: %v", jsonErr)
	}
	if resp.SchemaVersion != "tei.cli.three-element/v0" {
		t.Fatalf("failure schemaVersion=%q", resp.SchemaVersion)
	}
}
