package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func profilePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", name+".json")
}

func TestCompareServerAndEmbeddedProfilesHaveEqualSemantics(t *testing.T) {
	workspace, contract, fixtures, candidates := selectionPaths(t)
	receipt, err := Compare(context.Background(), CompareOptions{
		WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures,
		CandidatesDir: candidates, ProfilePaths: []string{profilePath(t, "server"), profilePath(t, "embedded")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || !receipt.SemanticsEqual {
		t.Fatalf("profiles did not produce equal semantics: %+v", receipt)
	}
	if len(receipt.Cases) != 10 {
		t.Fatalf("unexpected case count: %d", len(receipt.Cases))
	}
	for _, item := range receipt.Cases {
		if !item.Equal {
			t.Fatalf("case %s was not equal: %+v", item.CaseID, item)
		}
		if len(item.Results) != 2 {
			t.Fatalf("case %s has %d profile results", item.CaseID, len(item.Results))
		}
	}
	if len(receipt.ProfileDifferences) == 0 {
		t.Fatal("profile declaration differences were not recorded")
	}
	if len(receipt.UnsupportedDifferences) == 0 || !strings.Contains(strings.Join(receipt.UnsupportedDifferences, " "), "rolling-update") {
		t.Fatalf("unsupported differences were not recorded: %+v", receipt.UnsupportedDifferences)
	}
	if receipt.GrantsRuntimeAuthority || receipt.ProvesRealTarget {
		t.Fatal("comparison claimed authority or a real target")
	}
}

func TestCompareRequiresTwoProfiles(t *testing.T) {
	_, err := Compare(context.Background(), CompareOptions{ProfilePaths: []string{profilePath(t, "server")}})
	if err == nil || !strings.Contains(err.Error(), "at least two") {
		t.Fatalf("single profile was accepted: %v", err)
	}
}

func TestCompareFailsWhenOneProfileSelectsFailingGenerator(t *testing.T) {
	workspace, contract, fixtures, candidates := selectionPaths(t)
	var profile Profile
	raw, err := os.ReadFile(profilePath(t, "server"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatal(err)
	}
	profile.ID = "failing-generator"
	profile.GeneratorID = "dangerous"
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	failingPath := filepath.Join(t.TempDir(), "failing.json")
	if err := os.WriteFile(failingPath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Compare(context.Background(), CompareOptions{
		WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures,
		CandidatesDir: candidates, ProfilePaths: []string{failingPath, profilePath(t, "embedded")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatalf("comparison passed with a failing generator: %+v", receipt)
	}
}

func TestCompareCasesDetectsMissingProfileResult(t *testing.T) {
	left := Receipt{
		Profile: Profile{ID: "a", GeneratorID: "g"},
		Selection: generatorselection.Receipt{Candidates: []generatorselection.CandidateReceipt{{
			ID: "g", Results: []generatorselection.CaseResult{{
				CaseID: "c1", Passed: true,
				Got: generatorselection.Result{State: "present", Amount: "1.00"},
			}},
		}}},
	}
	right := Receipt{
		Profile: Profile{ID: "b", GeneratorID: "g"},
		Selection: generatorselection.Receipt{Candidates: []generatorselection.CandidateReceipt{{
			ID: "g",
		}}},
	}
	cases, equal := compareCases([]Receipt{left, right})
	if equal || len(cases) != 1 || cases[0].Equal {
		t.Fatalf("missing profile result was treated as equal: %+v", cases)
	}
	if cases[0].Results[1].Present {
		t.Fatalf("missing result was marked present: %+v", cases[0].Results[1])
	}
}
