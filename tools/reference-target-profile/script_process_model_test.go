package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func TestScriptLongRunningProfiles(t *testing.T) {
	for _, language := range []string{generatorselection.LanguagePython, generatorselection.LanguageJavaScript} {
		t.Run(language, func(t *testing.T) {
			_, runtime := generatorselection.ResolveScriptRuntime(language)
			if !runtime.Available {
				t.Skip(language + " runtime unavailable")
			}
			workspace, contract, fixtures, _ := selectionPaths(t)
			candidates := filepath.Join(t.TempDir(), "candidates")
			strategy := language + "-long-running-template"
			generated, err := generatorselection.Generate(contract, strategy, filepath.Join(candidates, strategy))
			if err != nil {
				t.Fatal(err)
			}
			profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "server.json"))
			if err != nil {
				t.Fatal(err)
			}
			profile.ID, profile.GeneratorID = language, strategy
			profilePath := filepath.Join(t.TempDir(), "profile.json")
			run := func(t *testing.T) Receipt {
				t.Helper()
				raw, err := json.Marshal(profile)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(profilePath, raw, 0600); err != nil {
					t.Fatal(err)
				}
				receipt, err := Run(context.Background(), RunOptions{WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates, ProfilePath: profilePath})
				if err != nil {
					t.Fatal(err)
				}
				return receipt
			}
			t.Run("single-process", func(t *testing.T) {
				receipt := run(t)
				evidence := receipt.ProcessModel
				if !receipt.Passed || evidence == nil || !evidence.Passed || evidence.Processes != 1 || evidence.Requests != 10 || evidence.Responses != 10 || len(evidence.Cases) != 10 {
					t.Fatalf("incomplete long-running evidence: %+v", receipt)
				}
				if evidence.Runtime == nil || !evidence.Runtime.Available || evidence.Runtime.Language != language || evidence.Runtime.Version == "" || !reflect.DeepEqual(evidence.Runtime, receipt.Selection.Candidates[0].Runtime) {
					t.Fatalf("interpreter evidence differs from selection: %+v", evidence.Runtime)
				}
				if !strings.HasPrefix(receipt.Enforcement["memory"], "not-enforced:") || receipt.Enforcement["processModel"] != "long-running:single-process" {
					t.Fatalf("inaccurate enforcement: %+v", receipt.Enforcement)
				}
			})
			t.Run("missing-runtime", func(t *testing.T) {
				t.Setenv("PATH", "")
				receipt := run(t)
				selected := selectedCandidate(receipt)
				if receipt.Passed || receipt.ProcessModel.Passed || receipt.ProcessModel.Processes != 0 || selected.Runtime == nil || selected.Runtime.Available || !strings.Contains(selected.RejectReason, "runtime unavailable") {
					t.Fatalf("unavailable interpreter was accepted: %+v", receipt)
				}
			})
			t.Run("oneshot-is-not-long-running", func(t *testing.T) {
				oneshot := language + "-template"
				if _, err := generatorselection.Generate(contract, oneshot, filepath.Join(candidates, oneshot)); err != nil {
					t.Fatal(err)
				}
				profile.GeneratorID = oneshot
				defer func() { profile.GeneratorID = strategy }()
				receipt := run(t)
				if !receipt.Selection.Passed || receipt.Passed || receipt.ProcessModel.Passed || receipt.ProcessModel.Processes != 1 || receipt.ProcessModel.Responses >= receipt.ProcessModel.Requests {
					t.Fatalf("oneshot script passed long-running probe: %+v", receipt)
				}
			})
			t.Run("rejected-source-is-not-executed", func(t *testing.T) {
				source := filepath.Join(generated.Path, generated.Manifest.Source)
				raw, err := os.ReadFile(source)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(source, append(raw, ' '), 0600); err != nil {
					t.Fatal(err)
				}
				receipt := run(t)
				if receipt.Passed || receipt.ProcessModel.Processes != 0 || !strings.Contains(selectedCandidate(receipt).RejectReason, "source digest mismatch") {
					t.Fatalf("rejected source reached probe: %+v", receipt)
				}
			})
		})
	}
}

func TestGoAndScriptLongRunningProfilesHaveEqualSemantics(t *testing.T) {
	for _, language := range []string{generatorselection.LanguagePython, generatorselection.LanguageJavaScript} {
		if _, runtime := generatorselection.ResolveScriptRuntime(language); !runtime.Available {
			t.Skip(language + " runtime unavailable")
		}
	}
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	var profiles []string
	for _, strategy := range []string{"long-running-template", "python-long-running-template", "javascript-long-running-template"} {
		if _, err := generatorselection.Generate(contract, strategy, filepath.Join(candidates, strategy)); err != nil {
			t.Fatal(err)
		}
		profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "server.json"))
		if err != nil {
			t.Fatal(err)
		}
		profile.ID, profile.GeneratorID = strategy, strategy
		raw, err := json.Marshal(profile)
		if err != nil {
			t.Fatal(err)
		}
		profilePath := filepath.Join(t.TempDir(), "profile.json")
		if err := os.WriteFile(profilePath, raw, 0600); err != nil {
			t.Fatal(err)
		}
		profiles = append(profiles, profilePath)
	}
	comparison, err := Compare(context.Background(), CompareOptions{WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates, ProfilePaths: profiles})
	if err != nil {
		t.Fatal(err)
	}
	if !comparison.Passed || !comparison.SemanticsEqual || len(comparison.Cases) != 10 || comparison.GrantsRuntimeAuthority || comparison.ProvesRealTarget {
		t.Fatalf("long-running semantics differ: %+v", comparison)
	}
	if !strings.Contains(strings.Join(comparison.EnforcementDifferences, " "), "memory:") {
		t.Fatalf("missing non-Go memory limitation: %+v", comparison.EnforcementDifferences)
	}
}
