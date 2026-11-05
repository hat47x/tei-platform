package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type lineNumberOracleScenario struct {
	Steps []struct {
		ID                string            `json:"id"`
		Operation         string            `json:"operation"`
		Fixture           string            `json:"fixture"`
		ExpectValidations map[string]string `json:"expectValidations"`
	} `json:"steps"`
}

type lineNumberOracleFixture struct {
	Numbers      map[string]string `json:"numbers"`
	RelationKeys []struct {
		Relation  string   `json:"relation"`
		Attribute string   `json:"attribute"`
		Keys      []string `json:"keys"`
	} `json:"relationKeys"`
}

func independentLineNumberUnique(keys []string) bool {
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, duplicate := seen[key]; duplicate {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

// oracleMatchesLineNumberCase evaluates a frozen, independent finite expectation.
// Candidate values are compared to this expectation, not used to derive it.
func oracleMatchesLineNumberCase(keys []string, wantUnique bool, candidateState string) bool {
	if independentLineNumberUnique(keys) != wantUnique {
		return false
	}
	wantState := "not-satisfied"
	if wantUnique {
		wantState = "satisfied"
	}
	return candidateState == wantState
}

// TestAILineNumberIndependentOracle checks only a public, finite relation-key
// property. It does not claim that the business meaning or actual Provider is
// independently accepted. The existing Scenario CLI test runs the real steps.
func TestAILineNumberIndependentOracle(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "driver-scenarios")
	raw, err := os.ReadFile(filepath.Join(root, "ai-candidate-line-number-uniqueness.json"))
	if err != nil {
		t.Fatal(err)
	}
	var scenario lineNumberOracleScenario
	if err := json.Unmarshal(raw, &scenario); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		stepID     string
		fixture    string
		keys       []string
		wantUnique bool
	}{
		{"ai-line-number-unique", "fixtures/order_valid.json", []string{"1", "2"}, true},
		{"ai-line-number-duplicate", "fixtures/order_duplicate_line_numbers.json", []string{"1", "1"}, false},
	}
	if len(scenario.Steps) != len(tests) {
		t.Fatalf("required steps: got %d, want %d", len(scenario.Steps), len(tests))
	}
	seen := make(map[string]bool, len(tests))
	for _, tc := range tests {
		t.Run(tc.stepID, func(t *testing.T) {
			var step *struct {
				ID                string            `json:"id"`
				Operation         string            `json:"operation"`
				Fixture           string            `json:"fixture"`
				ExpectValidations map[string]string `json:"expectValidations"`
			}
			for i := range scenario.Steps {
				if scenario.Steps[i].ID == tc.stepID {
					step = &scenario.Steps[i]
				}
			}
			if step == nil || seen[tc.stepID] {
				t.Fatalf("missing or duplicate candidate step %q", tc.stepID)
			}
			seen[tc.stepID] = true
			if step.Operation != "evaluate-validations" || step.Fixture != tc.fixture {
				t.Fatalf("changed operation or input binding: %+v", step)
			}
			fixtureBytes, err := os.ReadFile(filepath.Join(root, tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			var fixture lineNumberOracleFixture
			if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
				t.Fatal(err)
			}
			if fixture.Numbers["sales.order_line.quantity"] != "2.5" || fixture.Numbers["sales.order_line.line_number"] != "1" ||
				len(fixture.RelationKeys) != 1 || fixture.RelationKeys[0].Relation != "sales.order.lines" ||
				fixture.RelationKeys[0].Attribute != "sales.order_line.line_number" ||
				!reflect.DeepEqual(fixture.RelationKeys[0].Keys, tc.keys) {
				t.Fatalf("test fixture changed independently fixed inputs: %+v", fixture)
			}
			observed := independentLineNumberUnique(fixture.RelationKeys[0].Keys)
			if observed != tc.wantUnique {
				t.Fatalf("independent duplicate-key oracle: got %v, want %v", observed, tc.wantUnique)
			}
			wantState := "not-satisfied"
			if tc.wantUnique {
				wantState = "satisfied"
			}
			if step.ExpectValidations["sales.order.lines.line_number_unique"] != wantState {
				t.Fatalf("candidate expectation disagrees with independent property: %q", step.ExpectValidations["sales.order.lines.line_number_unique"])
			}
			// Negative controls: a flipped expectation and a changed key set
			// must both disagree with the frozen independent expectation.
			wrongState := "satisfied"
			wrongKeys := []string{"1", "2"}
			if tc.wantUnique {
				wrongState = "not-satisfied"
				wrongKeys = []string{"1", "1"}
			}
			if oracleMatchesLineNumberCase(fixture.RelationKeys[0].Keys, tc.wantUnique, wrongState) {
				t.Fatal("flipped Candidate expectation mutant was accepted")
			}
			if oracleMatchesLineNumberCase(wrongKeys, tc.wantUnique, wantState) {
				t.Fatal("changed relation-keys input mutant was accepted")
			}
		})
	}
}
