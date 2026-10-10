package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/tools/internal/rmtevidence"
)

func TestRMTSelectedNodeCLIIsReplayableAndRedacted(t *testing.T) {
	dir := t.TempDir()
	source := []byte(`<System FormatVersion="1.2"><Table ID="fictional_item_xyz"/><NovelConstruct Detail="fictional-value"/></System>`)
	input := filepath.Join(dir, "sample.xead")
	contractPath := filepath.Join(dir, "contract.json")
	if err := os.WriteFile(input, source, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contractPath, classificationContractFixture(), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(extra ...string) ([]byte, error) {
		args := append([]string{"run", ".", "-format", "xead", "-input", input, "-contract", contractPath}, extra...)
		return exec.Command("go", args...).CombinedOutput()
	}
	output, err := run("-candidate-evidence-node", "2")
	if err != nil {
		t.Fatalf("CLI: %s: %v", output, err)
	}
	var value map[string]any
	if err := json.Unmarshal(output, &value); err != nil {
		t.Fatal(err)
	}
	if value["schema"] != rmtevidence.Schema {
		t.Fatalf("not typed Evidence: %s", output)
	}
	verified, err := rmtevidence.Decode(output)
	if err != nil {
		t.Fatal(err)
	}
	observed, err := Observe(FormatXead, source)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTranscriptionCandidateEvidence(FormatXead, observed, classificationContractFixture(), verified); err != nil {
		t.Fatal(err)
	}
	for _, item := range []string{"fictional_item_xyz", "NovelConstruct", "fictional-value", "/System[1]", "localCandidates"} {
		if strings.Contains(string(output), item) {
			t.Fatalf("source values leaked in Evidence: %s", item)
		}
	}
	output, err = run("-candidate-evidence-node", "3")
	if err != nil {
		t.Fatalf("CLI unknown node: %s: %v", output, err)
	}
	unknown, err := rmtevidence.Decode(output)
	if err != nil {
		t.Fatal(err)
	}
	if unknown.MappingStatus != rmtevidence.MappingUnmapped || unknown.SourceConstructKnown || unknown.SourceConstruct != "" {
		t.Fatalf("unrecognized node claimed known construct: %+v", unknown)
	}
	if strings.Contains(string(output), "NovelConstruct") {
		t.Fatal("unknown node name leaked")
	}
}

func TestRMTSelectedNodeCLIRejectsUnsupportedMode(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "sample.xead")
	if err := os.WriteFile(input, []byte(`<System FormatVersion="1.2"><Table ID="one"/></System>`), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("go", "run", ".", "-format", "xead", "-input", input, "-candidate-evidence-node", "2").CombinedOutput()
	if err == nil {
		t.Fatalf("missing contract accepted: %s", output)
	}
}
