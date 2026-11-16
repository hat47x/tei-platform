package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/threeelement"
)

// The expected envelope contract is written here, independently of the CLI
// code: the read-only CRUD projections must say what their OK covers, and a
// CRUD disagreement must be visible without reading the table. `diff` and
// `rea-*` write separate response shapes and are not part of this claim.
func TestThreeElementCLIConsumerCannotReadUnscopedPass(t *testing.T) {
	model := threeelement.NewThreeElementModel("cli-scope", "Envelope scope")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []threeelement.FunctionOperation{{DataID: "order", Operations: "C"}},
	}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order", Read: true, Source: "declared",
	}}
	file := writeModelFile(t, "scope.json", model)

	for _, command := range []string{"validate", "crud", "cross-reference"} {
		t.Run(command, func(t *testing.T) {
			var stdout strings.Builder
			exit, err := runThreeElementCommand([]string{command, "-model", file}, &stdout)
			if err != nil || exit != 0 {
				t.Fatalf("%s failed: exit %d err %v body %s", command, exit, err, stdout.String())
			}
			var envelope struct {
				OK      bool   `json:"ok"`
				OKScope string `json:"okScope"`
			}
			if err := json.Unmarshal([]byte(stdout.String()), &envelope); err != nil {
				t.Fatalf("invalid JSON: %v: %s", err, stdout.String())
			}
			// An unscoped OK is the defect: a Consumer may read it as full
			// three-element consistency, which this projection cannot prove.
			if envelope.OK && envelope.OKScope == "" {
				t.Fatalf("%s returned an unscoped OK: %s", command, stdout.String())
			}
			if envelope.OKScope == "" {
				t.Fatalf("%s must always name what OK covers: %s", command, stdout.String())
			}
		})
	}
}

// The cross-reference projection must carry the same CRUD review status as the
// other terminals; otherwise a Consumer of this one command alone could accept
// an ambiguous cell as settled.
func TestThreeElementCLICrossReferenceExposesCRUDConflictStatus(t *testing.T) {
	model := threeelement.NewThreeElementModel("cli-xref", "Cross-reference conflict")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []threeelement.FunctionOperation{{DataID: "order", Operations: "C"}},
	}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order", Delete: true, Source: "declared",
	}}
	file := writeModelFile(t, "xref.json", model)

	var stdout strings.Builder
	exit, err := runThreeElementCommand([]string{"cross-reference", "-model", file}, &stdout)
	if err != nil || exit != 0 {
		t.Fatalf("cross-reference failed: exit %d err %v body %s", exit, err, stdout.String())
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(stdout.String()), &body); err != nil {
		t.Fatalf("invalid JSON: %v: %s", err, stdout.String())
	}
	if body["okScope"] != "projection-produced-only" {
		t.Fatalf("cross-reference OK must be scoped: %+v", body)
	}
	result, ok := body["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %+v", body)
	}
	if got := result["crudConflictStatus"]; got != "conflicts-require-review" {
		t.Fatalf("cross-reference hid the review status: %v", got)
	}
	rows, ok := result["functionData"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected one Function/Data row: %+v", result["functionData"])
	}
	row := rows[0].(map[string]any)
	if row["conflictingDeclarations"] != true || row["derivedOperations"] != "C" || row["operations"] != "D" {
		t.Fatalf("projected row must show both intents: %+v", row)
	}
}

// A model with no CRUD disagreement must still be presented as usable. This is
// the positive control that keeps the new status from becoming always-on.
func TestThreeElementCLICleanModelKeepsNoConflictStatus(t *testing.T) {
	model := threeelement.NewThreeElementModel("cli-clean", "Clean model")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []threeelement.FunctionOperation{{DataID: "order", Operations: "C"}},
	}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order", Create: true, Source: "declared",
	}}
	file := writeModelFile(t, "clean.json", model)

	for _, command := range []string{"crud", "cross-reference"} {
		var stdout strings.Builder
		exit, err := runThreeElementCommand([]string{command, "-model", file}, &stdout)
		if err != nil || exit != 0 {
			t.Fatalf("%s failed: exit %d err %v body %s", command, exit, err, stdout.String())
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(stdout.String()), &body); err != nil {
			t.Fatalf("invalid JSON: %v: %s", err, stdout.String())
		}
		result, ok := body["result"].(map[string]any)
		if !ok {
			t.Fatalf("missing result: %+v", body)
		}
		if got := result["crudConflictStatus"]; got != "no-conflicts-detected" {
			t.Fatalf("%s rejected a clean model: %v", command, got)
		}
	}
}

func writeModelFile(t *testing.T, name string, model threeelement.ThreeElementModel) string {
	t.Helper()
	raw, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

// The package-level command test above proves the in-process contract. This
// runs the built binary so a Consumer that only has the executable sees the
// same scoped result.
func TestThreeElementCLIProcessEnvelopeIsScoped(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("no child process support")
	}
	bin := filepath.Join(t.TempDir(), "tei-test.exe")
	if runtime.GOOS != "windows" {
		bin = filepath.Join(t.TempDir(), "tei-test")
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		// A build failure is not evidence about the envelope. Fail rather than
		// skip so a broken build cannot quietly remove the real-process check.
		t.Fatalf("cannot build the CLI binary: %v", err)
	}

	model := threeelement.NewThreeElementModel("cli-process", "Process envelope")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []threeelement.FunctionOperation{{DataID: "order", Operations: "C"}},
	}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order", Read: true, Source: "declared",
	}}
	file := writeModelFile(t, "process.json", model)

	for _, command := range []string{"validate", "crud", "cross-reference"} {
		out, err := exec.Command(bin, "three-element", command, "-model", file).Output()
		if err != nil {
			t.Fatalf("%s in a real process failed: %v (%s)", command, err, out)
		}
		var body map[string]any
		if err := json.Unmarshal(out, &body); err != nil {
			t.Fatalf("%s: invalid JSON: %v: %s", command, err, out)
		}
		if body["ok"] != true || body["okScope"] == "" || body["okScope"] == nil {
			t.Fatalf("%s: OK must be present and scoped: %s", command, out)
		}
		result, ok := body["result"].(map[string]any)
		if !ok {
			t.Fatalf("%s: missing result: %s", command, out)
		}
		if result["crudConflictStatus"] != "conflicts-require-review" {
			t.Fatalf("%s: conflict status missing from a real process: %s", command, out)
		}
	}
}
