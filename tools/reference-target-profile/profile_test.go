package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func selectionPaths(t *testing.T) (workspace, contract, fixtures, candidates string) {
	t.Helper()
	root := repoRoot(t)
	base := filepath.Join(root, "tools", "reference-generator-selection", "testdata")
	return filepath.Join(root, "reference", "workspace"),
		filepath.Join(base, "line-amount", "contract.json"),
		filepath.Join(base, "line-amount", "fixtures.json"),
		filepath.Join(base, "candidates")
}

func TestLoadsServerAndEmbeddedProfiles(t *testing.T) {
	for _, name := range []string{"server", "embedded"} {
		profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", name+".json"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if profile.ID != name {
			t.Fatalf("%s: id=%q", name, profile.ID)
		}
	}
}

func TestValidateRejectsUnknownOrIncompleteProfiles(t *testing.T) {
	valid := Profile{
		Schema: Schema, ID: "p", Version: "1", ProcessModel: ProcessOneShot,
		Storage: StorageNone, Network: NetworkNone, Update: UpdateReplace,
		MemoryLimitBytes: 1024, Observability: []string{"stdout"},
	}
	cases := []struct {
		name string
		edit func(*Profile)
	}{
		{"schema", func(p *Profile) { p.Schema = "other" }},
		{"id", func(p *Profile) { p.ID = "" }},
		{"process", func(p *Profile) { p.ProcessModel = "thread" }},
		{"storage", func(p *Profile) { p.Storage = "tape" }},
		{"network", func(p *Profile) { p.Network = "sometimes" }},
		{"update", func(p *Profile) { p.Update = "maybe" }},
		{"memory", func(p *Profile) { p.MemoryLimitBytes = 0 }},
		{"observability", func(p *Profile) { p.Observability = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile := valid
			tc.edit(&profile)
			if err := Validate(profile); err == nil {
				t.Fatalf("invalid profile was accepted: %+v", profile)
			}
		})
	}
}

func TestRunRejectsDeclaredUnsupportedObservability(t *testing.T) {
	workspace, contract, fixtures, candidates := selectionPaths(t)
	var profile Profile
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatal(err)
	}
	profile.Observability = append(profile.Observability, "audit-log")
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(t.TempDir(), "server-audit.json")
	if err := os.WriteFile(profilePath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Run(context.Background(), RunOptions{
		WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures,
		CandidatesDir: candidates, ProfilePath: profilePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatalf("profile with unsupported observation channel passed: %+v", receipt)
	}
	if receipt.Observation == nil || len(receipt.Observation.Unsupported) == 0 || !strings.Contains(strings.Join(receipt.Observation.Unsupported, ","), "audit-log") {
		t.Fatalf("unsupported observation channel was not recorded: %+v", receipt.Observation)
	}
}

func TestLongRunningProfileRejectsOneshotCandidate(t *testing.T) {
	workspace, contract, fixtures, candidates := selectionPaths(t)
	var profile Profile
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatal(err)
	}
	profile.ID = "long-running-with-oneshot"
	profile.GeneratorID = "template"
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(t.TempDir(), "long-running-oneshot.json")
	if err := os.WriteFile(profilePath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Run(context.Background(), RunOptions{
		WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures,
		CandidatesDir: candidates, ProfilePath: profilePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || receipt.ProcessModel == nil || receipt.ProcessModel.Passed {
		t.Fatalf("one-shot candidate passed a long-running profile: %+v", receipt.ProcessModel)
	}
	if receipt.ProcessModel.Mode != ProcessLongRunning || receipt.ProcessModel.Reason == "" {
		t.Fatalf("long-running failure was not explained: %+v", receipt.ProcessModel)
	}
}

func TestRunAppliesBatchProcessModel(t *testing.T) {
	workspace, contract, fixtures, candidates := selectionPaths(t)
	profilePath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "batch.json")
	receipt, err := Run(context.Background(), RunOptions{
		WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures,
		CandidatesDir: candidates, ProfilePath: profilePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || receipt.ProcessModel == nil || !receipt.ProcessModel.Passed {
		t.Fatalf("batch profile did not pass: %+v", receipt)
	}
	if receipt.ProcessModel.Mode != ProcessBatch || receipt.ProcessModel.Processes != 1 ||
		receipt.ProcessModel.Requests != 10 || receipt.ProcessModel.Responses != 10 {
		t.Fatalf("unexpected batch process evidence: %+v", receipt.ProcessModel)
	}
	if receipt.Enforcement["processModel"] != "batch:single-process" {
		t.Fatalf("batch enforcement is wrong: %+v", receipt.Enforcement)
	}
}

func TestRunAppliesProfilesAndKeepsSelection(t *testing.T) {
	workspace, contract, fixtures, candidates := selectionPaths(t)
	previous, had := os.LookupEnv("GOMEMLIMIT")
	defer func() {
		if had {
			_ = os.Setenv("GOMEMLIMIT", previous)
		} else {
			_ = os.Unsetenv("GOMEMLIMIT")
		}
	}()
	for _, name := range []string{"server", "embedded"} {
		profilePath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", name+".json")
		receipt, err := Run(context.Background(), RunOptions{
			WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures,
			CandidatesDir: candidates, ProfilePath: profilePath,
		})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !receipt.Passed || !receipt.Selection.Passed {
			t.Fatalf("%s: profile run did not pass: %+v", name, receipt)
		}
		if receipt.Profile.ID != name {
			t.Fatalf("%s: profile id=%q", name, receipt.Profile.ID)
		}
		if !strings.Contains(receipt.Enforcement["memory"], "GOMEMLIMIT") {
			t.Fatalf("%s: memory enforcement is not recorded: %+v", name, receipt.Enforcement)
		}
		if receipt.Enforcement["observability"] != "observed" {
			t.Fatalf("%s: observability was not observed: %+v", name, receipt.Enforcement)
		}
		if receipt.Observation == nil || len(receipt.Observation.Observed) != len(receipt.Profile.Observability) {
			t.Fatalf("%s: unexpected observation evidence: %+v", name, receipt.Observation)
		}
		if receipt.Observation.Cases != 10 || receipt.Observation.CasesWithStdout != 10 || receipt.Observation.CasesWithExitCode != 10 {
			t.Fatalf("%s: observation case counts are incomplete: %+v", name, receipt.Observation)
		}
		if receipt.ProcessModel == nil || !receipt.ProcessModel.Passed {
			t.Fatalf("%s: process model was not enforced: %+v", name, receipt.ProcessModel)
		}
		switch name {
		case "server":
			if receipt.ProcessModel.Mode != ProcessLongRunning || receipt.ProcessModel.Processes != 1 ||
				receipt.ProcessModel.Requests != 10 || receipt.ProcessModel.Responses != 10 {
				t.Fatalf("server process model evidence is wrong: %+v", receipt.ProcessModel)
			}
			if receipt.Enforcement["processModel"] != "long-running:single-process" {
				t.Fatalf("server process model enforcement is wrong: %+v", receipt.Enforcement)
			}
		case "embedded":
			if receipt.ProcessModel.Mode != ProcessOneShot || receipt.ProcessModel.Processes != 10 {
				t.Fatalf("embedded process model evidence is wrong: %+v", receipt.ProcessModel)
			}
			if receipt.Enforcement["processModel"] != "oneshot:process-per-case" {
				t.Fatalf("embedded process model enforcement is wrong: %+v", receipt.Enforcement)
			}
		}
		if receipt.Profile.GeneratorID == "" {
			t.Fatalf("%s: profile did not select a generator", name)
		}
		if len(receipt.Selection.Qualified) != 1 || receipt.Selection.Qualified[0] != receipt.Profile.GeneratorID {
			t.Fatalf("%s: qualified=%v want %s", name, receipt.Selection.Qualified, receipt.Profile.GeneratorID)
		}
		for _, candidate := range receipt.Selection.Candidates {
			if candidate.ID != receipt.Profile.GeneratorID && !strings.Contains(candidate.RejectReason, "not selected by target profile") {
				t.Fatalf("%s: unselected candidate %s was not rejected by the profile: %+v", name, candidate.ID, candidate)
			}
		}
	}
}
