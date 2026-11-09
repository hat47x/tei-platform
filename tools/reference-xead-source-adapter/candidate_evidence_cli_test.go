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
	contractPath := filepath.Join(dir, "contract.json")
	if err := os.WriteFile(contractPath, classificationContractFixture(), 0o600); err != nil {
		t.Fatal(err)
	}
	base := []string{"run", ".", "-format", "xead", "-input", input}
	cases := []struct {
		name string
		args []string
	}{
		{"missing contract", []string{"-candidate-evidence-node", "2"}},
		{"zero ordinal", []string{"-contract", contractPath, "-candidate-evidence-node", "0"}},
		{"negative ordinal", []string{"-contract", contractPath, "-candidate-evidence-node", "-1"}},
		{"out of range", []string{"-contract", contractPath, "-candidate-evidence-node", "400"}},
		{"conflicting local mode", []string{"-contract", contractPath, "-candidate-evidence-node", "2", "-local-candidates"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append(append([]string{}, base...), tc.args...)
			output, err := exec.Command("go", args...).CombinedOutput()
			if err == nil {
				t.Fatalf("invalid node evidence CLI mode accepted: %s", output)
			}
		})
	}
}
