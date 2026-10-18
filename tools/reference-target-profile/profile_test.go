package targetprofile

import (
	"context"
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
		if len(receipt.Selection.Qualified) != 2 {
			t.Fatalf("%s: qualified=%v", name, receipt.Selection.Qualified)
		}
	}
}
