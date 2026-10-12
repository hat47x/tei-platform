package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/xeadsource"
)

func commandContractFixture() []byte {
	return []byte(`{
  "constructMappings": {
    "xead": {
      "System": {
        "classification": ["source-metadata"],
        "extension": "reference.compat.xead.source"
      }
    }
  },
  "sourceProfiles": {
    "xead": {
      "rootKind": "System",
      "supportedFormatVersions": ["1.2"]
    }
  }
}`)
}

func TestCommandOutputDefaultsToPublicAccounting(t *testing.T) {
	source := []byte(`<System FormatVersion="1.2">
  <Table ID="private-table"/>
  <Function ID="private-function"><IOTable ID="private-io" TableID="private-table" OpR="+"/></Function>
  <Task ID="private-task"><TaskAction ID=""><TaskFunctionIO FunctionID="private-function" IOID="0"/></TaskAction></Task>
</System>`)
	output, err := buildCommandOutput(xeadsource.FormatXead, source, commandContractFixture(), false)
	if err != nil {
		t.Fatal(err)
	}
	if output.LocalProjection != nil || output.LocalOutputNotice != "" {
		t.Fatalf("default command leaked local projection: %+v", output)
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-table", "private-function", "private-task", "private-io", "fromToken", "sourceTokens"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("default public output leaked %q: %s", forbidden, encoded)
		}
	}
	if output.Accounting.TaskActionMissingIDCount != 1 ||
		output.Accounting.PotentialActionFunctionTablePathCount != 1 {
		t.Fatalf("unexpected accounting: %+v", output.Accounting)
	}
}

func TestCommandOutputRequiresExplicitLocalProjection(t *testing.T) {
	source := []byte(`<System FormatVersion="1.2"><Task ID="T1"><TaskAction ID=""/></Task></System>`)
	output, err := buildCommandOutput(xeadsource.FormatXead, source, commandContractFixture(), true)
	if err != nil {
		t.Fatal(err)
	}
	if output.LocalProjection == nil || output.LocalOutputNotice == "" {
		t.Fatalf("explicit local output missing: %+v", output)
	}
}


func TestCommandVerifiesSavedLocalOutputWithoutRevealingIDs(t *testing.T) {
	source := []byte(`<System FormatVersion="1.2">
  <Table ID="private-table-471"/>
  <Function ID="private-function-582">
    <IOTable ID="private-io-693" TableID="private-table-471" OpR="+"/>
  </Function>
  <Task ID="private-task-804">
    <TaskAction ID="private-action-915">
      <TaskFunctionIO FunctionID="private-function-582" IOID="0"/>
    </TaskAction>
  </Task>
  <Task ID="private-task-without-actions"/>
</System>`)
	contract := commandContractFixture()
	saved, err := buildCommandOutput(xeadsource.FormatXead, source, contract, true)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := buildVerifiedCommandOutput(xeadsource.FormatXead, source, contract, data, false)
	if err != nil {
		t.Fatal(err)
	}
	if verified.LocalProjection != nil || verified.LocalOutputNotice != "" {
		t.Fatalf("verified public output included private projection: %+v", verified)
	}
	if !reflect.DeepEqual(verified.Accounting, saved.Accounting) {
		t.Fatalf("reconciled source accounting differed from fresh command: %+v", verified.Accounting)
	}
	encoded, err := json.Marshal(verified)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-table-471", "private-function-582", "private-io-693",
		"private-task-804", "private-action-915", "fromToken", "sourceTokens"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("source identity %q escaped into public output: %s", secret, encoded)
		}
	}
	verifiedLocal, err := buildVerifiedCommandOutput(xeadsource.FormatXead, source, contract, data, true)
	if err != nil || verifiedLocal.LocalProjection == nil || verifiedLocal.LocalOutputNotice == "" {
		t.Fatalf("explicit local inspection failed: %+v / %v", verifiedLocal, err)
	}

	// The saved aggregate is not authoritative. A caller may have changed
	// the envelope even while leaving its underlying projection intact.
	saved.Accounting.TaskCount = 999
	editedEnvelope, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := buildVerifiedCommandOutput(xeadsource.FormatXead, source, contract, editedEnvelope, false)
	if err != nil || checked.Accounting.TaskCount != 2 {
		t.Fatalf("saved aggregate was trusted instead of rederived: %+v / %v", checked, err)
	}
}

func TestCommandRejectsUnboundSavedProjection(t *testing.T) {
	source := []byte(`<System FormatVersion="1.2">
  <Table ID="T1"/>
  <Function ID="F1"><IOTable TableID="T1" OpR="+"/></Function>
  <Task ID="T1"><TaskAction ID="A1"><TaskFunctionIO FunctionID="F1" IOID="0"/></TaskAction></Task>
</System>`)
	contract := commandContractFixture()
	saved, err := buildCommandOutput(xeadsource.FormatXead, source, contract, true)
	if err != nil {
		t.Fatal(err)
	}
	encode := func(value commandOutput) []byte {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	savedBytes := encode(saved)
	unknownEnvelope := append(append([]byte(nil), savedBytes[:len(savedBytes)-1]...), []byte(`,"futureMeaning":true}`)...)
	trailingJSON := append(append([]byte(nil), savedBytes...), []byte(` {}`)...)
	for _, tc := range []struct {
		name     string
		source   []byte
		contract []byte
		data     []byte
	}{
		{"changed XML", append(append([]byte(nil), source...), ' '), contract, encode(saved)},
		{"different contract bytes", source, append(append([]byte(nil), contract...), ' '), encode(saved)},
		{"missing projection", source, contract, encode(commandOutput{Accounting: saved.Accounting})},
		{"unknown envelope fields", source, contract, unknownEnvelope},
		{"trailing JSON value", source, contract, trailingJSON},
		{"invalid JSON", source, contract, []byte(`{"localProjection":`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if output, err := buildVerifiedCommandOutput(xeadsource.FormatXead, tc.source, tc.contract, tc.data, false); err == nil {
				t.Fatalf("unverified source/projection was published: %+v", output)
			}
		})
	}
}


func TestCLIRechecksSavedProjectionAgainstSourceFiles(t *testing.T) {
	source := []byte(`<System FormatVersion="1.2">
  <Table ID="secret-table-1"/>
  <Function ID="secret-function-2"><IOTable ID="secret-io-3" TableID="secret-table-1" OpR="+"/></Function>
  <Task ID="secret-task-4"><TaskAction ID="secret-action-5">
    <TaskFunctionIO FunctionID="secret-function-2" IOID="0"/>
  </TaskAction></Task>
</System>`)
	contract := commandContractFixture()
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "source.xead")
	contractFile := filepath.Join(dir, "contract.json")
	projectionFile := filepath.Join(dir, "saved-local-output.json")
	for file, value := range map[string][]byte{
		inputFile: source,
		contractFile: contract,
	} {
		if err := os.WriteFile(file, value, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Compile and invoke the real CLI instead of only exercising its Go
	// helper, so that the saved-output flag and public redaction are tested.
	binary := filepath.Join(dir, "reference-xead-cross-view")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build reference CLI: %v\n%s", err, output)
	}
	save := exec.Command(binary, "-input", inputFile, "-contract", contractFile, "-local-projection")
	saved, err := save.CombinedOutput()
	if err != nil {
		t.Fatalf("save local projection: %v\n%s", err, saved)
	}
	if err := os.WriteFile(projectionFile, saved, 0o600); err != nil {
		t.Fatal(err)
	}
	verify := func() ([]byte, error) {
		return exec.Command(binary, "-input", inputFile, "-contract", contractFile,
			"-verify-projection", projectionFile).CombinedOutput()
	}
	verified, err := verify()
	if err != nil {
		t.Fatalf("verify saved result: %v\n%s", err, verified)
	}
	var output commandOutput
	if err := json.Unmarshal(verified, &output); err != nil {
		t.Fatal(err)
	}
	if output.LocalProjection != nil || output.Accounting.TaskActionCount != 1 {
		t.Fatalf("wrong public result: %+v", output)
	}
	for _, secret := range []string{"secret-table-1", "secret-function-2", "secret-action-5", "sourceTokens"} {
		if strings.Contains(string(verified), secret) {
			t.Fatalf("private source identifier leaked: %q", secret)
		}
	}

	if err := os.WriteFile(inputFile, append(source, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	if bad, err := verify(); err == nil {
		t.Fatalf("stale saved projection survived XML byte change: %s", bad)
	}
}
