package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeCatalogValidationCLIWorkspace(t *testing.T, model string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: catalog_validation_cli\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// The catalog keeps answering ok:true / exit 0 for a Workspace that does not
// validate, because it is a projection being read rather than an acceptance
// decision. The response must still say the projection is over an invalid
// Workspace, otherwise a caller reading only the catalog cannot tell (#3674).
func TestCLIModelCatalogReportsValidationOfAnInvalidWorkspace(t *testing.T) {
	root := writeCatalogValidationCLIWorkspace(t, `namespace scn

data X {
    attribute a type scn.nothere
}
`)
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", root}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model catalog refused a readable Workspace: exit=%d err=%v response=%s", code, err, stdout.String())
	}
	var response modelCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Result == nil {
		t.Fatalf("the catalog envelope changed for an invalid Workspace: %+v", response)
	}
	if response.Result.Validation.OK {
		t.Fatal("the catalog reported an invalid Workspace as valid")
	}
	if response.Result.Validation.ErrorCount == 0 {
		t.Fatalf("errorCount is zero beside validation.ok=false: %+v", response.Result.Validation)
	}

	// The other command over the same Workspace must disagree on success and
	// agree on the verdict, so a reader cannot get two different answers.
	var validateOut bytes.Buffer
	validateCode, validateErr := validateWorkspaceJSON(&validateOut, root)
	if validateErr != nil || validateCode != 1 {
		t.Fatalf("validate did not fail on the same Workspace: exit=%d err=%v out=%s", validateCode, validateErr, validateOut.String())
	}
}

// A valid Workspace reports validation.ok:true, so the field distinguishes the
// two states rather than always being false.
func TestCLIModelCatalogReportsAValidWorkspace(t *testing.T) {
	root := writeCatalogValidationCLIWorkspace(t, `namespace scn

data RuleCandidate {
    attribute sourceVersion type std.text
}
`)
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", root}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model catalog failed: exit=%d err=%v response=%s", code, err, stdout.String())
	}
	var response modelCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result == nil || !response.Result.Validation.OK || response.Result.Validation.ErrorCount != 0 {
		t.Fatalf("a valid Workspace was not reported as valid: %+v", response.Result)
	}
}
