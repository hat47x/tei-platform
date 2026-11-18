package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

const studioAuthoringManifest = "workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n"

func makeStudioAuthoringWorkspace(t *testing.T, fields string) (string, string) {
	t.Helper()
	root := t.TempDir()
	relative := "semantic/studio/inspection-response.tei.md"
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(studioAuthoringManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	document := "# Studio response\n\n<!-- tei: data InspectionResponse in studio -->\n" +
		"| attribute | type | required | display |\n" +
		"| --- | --- | --- | --- |\n" +
		"| schemaVersion | std.text | yes | Schema |\n" + fields
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func TestAuthoringDataControlsGeneratedReaderWithoutGeneratorChanges(t *testing.T) {
	root, sourcePath := makeStudioAuthoringWorkspace(t, "| snapshotFingerprint | std.text | yes | Snapshot |\n")
	originalTransform := read(t, transformFile)
	originalContract := read(t, contractFile)
	effective, binding, err := bindAuthoringData(root, "studio.inspection_response", originalContract)
	if err != nil {
		t.Fatal(err)
	}
	if binding.DataID != "studio.inspection_response" ||
		binding.SourcePath != "semantic/studio/inspection-response.tei.md" ||
		binding.InputContractDigest != digest(originalContract) ||
		len(binding.RequiredAttributeIDs) != 2 ||
		binding.ModelRevision == "" || binding.SourceFingerprint == "" {
		t.Fatalf("binding lacks exact Data, revision, or attribute identity: %+v", binding)
	}
	var contract struct {
		RequiredResponseFields []requiredResponseField `json:"requiredResponseFields"`
	}
	if err := json.Unmarshal(effective, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.RequiredResponseFields) != 2 ||
		contract.RequiredResponseFields[0] != (requiredResponseField{"schemaVersion", "string"}) ||
		contract.RequiredResponseFields[1] != (requiredResponseField{"snapshotFingerprint", "string"}) {
		t.Fatalf("baseline or added requirement differs: %+v", contract.RequiredResponseFields)
	}
	before, _, err := generate(originalTransform, effective)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(before, []byte("[\"snapshotFingerprint\", \"string\"]")) {
		t.Fatal("model-required response field did not reach generated Reader")
	}
	if err := os.WriteFile(sourcePath, []byte("# Studio response\n\n<!-- tei: data InspectionResponse in studio -->\n"+
		"| attribute | type | required | display |\n"+
		"| --- | --- | --- | --- |\n"+
		"| schemaVersion | std.text | yes | Schema |\n"+
		"| snapshotFingerprint | std.text | no | Snapshot |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	effectiveNext, nextBinding, err := bindAuthoringData(root, "studio.inspection_response", originalContract)
	if err != nil {
		t.Fatal(err)
	}
	after, _, err := generate(originalTransform, effectiveNext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) ||
		bytes.Contains(after, []byte("[\"snapshotFingerprint\", \"string\"]")) {
		t.Fatal("Data required flag change did not change generated Reader")
	}
	if nextBinding.ModelRevision == binding.ModelRevision ||
		nextBinding.SourceFingerprint == binding.SourceFingerprint {
		t.Fatal("Data Source revision change was not recorded")
	}
	if !bytes.Equal(originalTransform, read(t, transformFile)) {
		t.Fatal("Generator Source bytes changed")
	}
}

func TestAuthoringBindingUsesRealChildNoClobberAndReportsSource(t *testing.T) {
	root, _ := makeStudioAuthoringWorkspace(t, "| snapshotFingerprint | std.text | yes | Snapshot |\n")
	artifact := filepath.Join(t.TempDir(), "reader.ts")
	args := []string{
		"-transform", transformFile,
		"-contract", contractFile,
		"-artifact-out", artifact,
		"-authoring-root", root,
		"-authoring-data", "studio.inspection_response",
	}
	var stdout bytes.Buffer
	if err := run(args, &stdout); err != nil {
		t.Fatal(err)
	}
	var got report
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if got.Authoring == nil || got.Authoring.DataID != "studio.inspection_response" ||
		got.Authoring.SourceFingerprint == "" ||
		got.ContractBytesDigest == got.Authoring.InputContractDigest ||
		got.ArtifactDigest != digest(written) {
		t.Fatalf("child did not report actual derived contract and output: %+v", got)
	}
	if !bytes.Contains(written, []byte("[\"snapshotFingerprint\", \"string\"]")) {
		t.Fatal("child did not write the model-derived Reader")
	}
	if err := run(args, &bytes.Buffer{}); err == nil {
		t.Fatal("child overwrote an existing derived Reader")
	}
}

func TestAuthoringBindingFailsClosedBeforeWriting(t *testing.T) {
	for name, item := range map[string]struct {
		fields string
		id     string
	}{
		"unsupported type":      {"| snapshotFingerprint | std.decimal | yes | Snapshot |\n", "studio.inspection_response"},
		"unresolved alias":      {"| snapshotFingerprint | custom.text | yes | Snapshot |\n", "studio.inspection_response"},
		"unknown subject":       {"| snapshotFingerprint | std.text | yes | Snapshot |\n", "studio.other"},
		"no required attribute": {"| schemaVersion | std.text | no | Schema |\n", "studio.inspection_response"},
	} {
		t.Run(name, func(t *testing.T) {
			root, _ := makeStudioAuthoringWorkspace(t, item.fields)
			artifact := filepath.Join(t.TempDir(), "rejected.ts")
			err := run([]string{
				"-transform", transformFile, "-contract", contractFile,
				"-artifact-out", artifact,
				"-authoring-root", root, "-authoring-data", item.id,
			}, &bytes.Buffer{})
			if err == nil {
				t.Fatal("invalid authoring Data produced an artifact")
			}
			if _, err := os.Stat(artifact); !os.IsNotExist(err) {
				t.Fatalf("rejected authoring input left an artifact: %v", err)
			}
		})
	}
	root, _ := makeStudioAuthoringWorkspace(t, "| snapshotFingerprint | std.text | yes | Snapshot |\n")
	for _, only := range [][]string{
		{"-authoring-root", root},
		{"-authoring-data", "studio.inspection_response"},
	} {
		err := run(append([]string{"-transform", transformFile, "-contract", contractFile,
			"-artifact-out", filepath.Join(t.TempDir(), "missing.ts")}, only...), &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "must be provided together") {
			t.Fatalf("partial authoring selection was accepted: %v", err)
		}
	}
}

func TestAuthoringRequiredFieldsAreReturnedByTheRealInspectOperation(t *testing.T) {
	root, _ := makeStudioAuthoringWorkspace(t, "| snapshotFingerprint | std.text | yes | Snapshot |\n")
	effective, _, err := bindAuthoringData(root, "studio.inspection_response", read(t, contractFile))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		RequiredResponseFields []requiredResponseField `json:"requiredResponseFields"`
	}
	if err := json.Unmarshal(effective, &contract); err != nil {
		t.Fatal(err)
	}
	inspection, err := operations.InspectSubject("../../reference/workspace",
		sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(inspection)
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	for _, field := range contract.RequiredResponseFields {
		encoded, ok := response[field.Name]
		if !ok {
			t.Fatalf("generated requirement %q is missing from real InspectSubject", field.Name)
		}
		var value any
		if err := json.Unmarshal(encoded, &value); err != nil {
			t.Fatal(err)
		}
		if _, ok := value.(string); !ok || field.Type != "string" {
			t.Fatalf("required response %q is not the declared string: %s", field.Name, encoded)
		}
	}
}

func TestAuthoringCannotBorrowInspectShapeForAnotherBridge(t *testing.T) {
	root, _ := makeStudioAuthoringWorkspace(t, "| snapshotFingerprint | std.text | yes | Snapshot |\n")
	other := bytes.Replace(read(t, contractFile),
		[]byte("main.StudioBridge.InspectSubject"), []byte("other.Bridge.Read"), 1)
	artifact := filepath.Join(t.TempDir(), "incorrect-bridge.ts")
	template := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(template, other, 0o644); err != nil {
		t.Fatal(err)
	}
	err := run([]string{
		"-transform", transformFile, "-contract", template, "-artifact-out", artifact,
		"-authoring-root", root, "-authoring-data", "studio.inspection_response",
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "read-only InspectSubject Operation") {
		t.Fatalf("unrelated bridge was accepted: %v", err)
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("unrelated bridge left a generated artifact: %v", err)
	}
}

func TestAuthoringRejectsPropertiesAbsentOrIncompatibleWithInspectSubject(t *testing.T) {
	for _, name := range []string{"inventedToken", "subject", "workspace"} {
		t.Run(name, func(t *testing.T) {
			row := "| " + name + " | std.text | yes | Invalid |\n"
			root, _ := makeStudioAuthoringWorkspace(t, row)
			artifact := filepath.Join(t.TempDir(), "invalid-response.ts")
			err := run([]string{
				"-transform", transformFile, "-contract", contractFile,
				"-artifact-out", artifact,
				"-authoring-root", root, "-authoring-data", "studio.inspection_response",
			}, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "InspectSubject response") {
				t.Fatalf("non-string or unknown InspectSubject property accepted: %v", err)
			}
			if _, err := os.Stat(artifact); !os.IsNotExist(err) {
				t.Fatalf("invalid response generated an artifact: %v", err)
			}
		})
	}
}
