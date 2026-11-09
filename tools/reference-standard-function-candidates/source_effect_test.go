package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"tei/internal/workspace"
)

func TestSourceDeclaredEffectReachesExplicitPatternInspection(t *testing.T) {
	const root = "../../examples/first-capability-effect"
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.World.Applications) != 1 {
		t.Fatalf("unexpected loaded applications: %+v", loaded.World.Applications)
	}

	args := []string{
		"-workspace", root,
		"-application", "request_log",
		"-pattern", "request_log.create_request",
	}
	var output bytes.Buffer
	if err := run(args, &output); err != nil {
		t.Fatalf("Source-defined effect did not reach CLI: %v", err)
	}
	var report ExplicitContractReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.DeclaredData != "intake.request" || report.DataOperation != "create" ||
		report.Capability != "intake.request.create" ||
		report.BindingIdentity != "request_log/intake.request.create" ||
		report.Executable || report.Authority != "declared-references-only" ||
		!includes(report.Unverified, "independent-database-read") {
		t.Fatalf("CLI claimed execution or lost Source effect: %+v", report)
	}
}
