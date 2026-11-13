package syntax_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// This test reads the S0 design fixture independently of the candidate
// parser's output. It checks the integrity of the declared cases, then
// consumes the same evaluation expectations as the existing stub evaluator
// tests. It does not assert Provider completeness, Product Workspace
// selection, owner acceptance, or all 34 cases being executed.
type ruleExpectedCase struct {
	ID        string          `json:"id"`
	Surface   string          `json:"surface"`
	Source    string          `json:"source"`
	Input     json.RawMessage `json:"input"`
	Expected  json.RawMessage `json:"expected"`
	Status    string          `json:"status"`
	Note      string          `json:"note"`
	Forbidden string          `json:"forbidden"`
}

type ruleExpectedFixture struct {
	Kind        string   `json:"kind"`
	Status      string   `json:"status"`
	Revision    string   `json:"revision"`
	ParentIssue string   `json:"parentIssue"`
	S0Issue     string   `json:"s0Issue"`
	Basis       string   `json:"basis"`
	Sources     []string `json:"sources"`
	Notes       []string `json:"notes"`
	CaseGroups  []struct {
		Group string             `json:"group"`
		Cases []ruleExpectedCase `json:"cases"`
	} `json:"caseGroups"`
}

func loadRuleExpectedFixture(t *testing.T) ruleExpectedFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repositoryRoot(t), "plan", "fixtures", "dsl-syntax-candidates", "rule_authoring_expected_20261111.json"))
	if err != nil {
		t.Fatalf("read S0 fixture: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var fixture ruleExpectedFixture
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatalf("S0 fixture has an unknown field or invalid JSON: %v", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("S0 fixture must contain exactly one JSON object, trailing decode=%v", err)
	}
	return fixture
}

func TestRuleExpectedFixtureCatalogIntegrity(t *testing.T) {
	fixture := loadRuleExpectedFixture(t)
	if fixture.Kind != "tei-dsl-syntax-candidate-expected-fixture" || fixture.Status != "INDEPENDENT_EXPECTED_NOT_OWNER_ACCEPTED" {
		t.Fatalf("fixture identity/status: %q / %q", fixture.Kind, fixture.Status)
	}
	// Minimum inventory is a guard against silently dropping a case group,
	// not evidence that every case is exercised by a Go test.
	minimum := map[string]int{
		"same-meaning surfaces":            6,
		"evaluation distinction":           9,
		"static diagnostics":               7,
		"negative references and presence": 12,
	}
	seenCases := map[string]bool{}
	seenGroups := map[string]bool{}
	for _, group := range fixture.CaseGroups {
		min, known := minimum[group.Group]
		if !known || seenGroups[group.Group] {
			t.Fatalf("unknown or repeated case group %q", group.Group)
		}
		seenGroups[group.Group] = true
		if len(group.Cases) < min {
			t.Fatalf("group %q has %d cases, expected at least %d", group.Group, len(group.Cases), min)
		}
		for _, c := range group.Cases {
			if c.ID == "" || seenCases[c.ID] {
				t.Fatalf("empty or duplicate S0 case ID %q", c.ID)
			}
			seenCases[c.ID] = true
			if len(c.Expected) == 0 || bytes.Equal(bytes.TrimSpace(c.Expected), []byte("null")) {
				t.Fatalf("%s has no expected value", c.ID)
			}
			if c.Status != "" && c.Status != "NOT IMPLEMENTED" {
				t.Fatalf("%s has unrecognized implementation status %q", c.ID, c.Status)
			}
			if group.Group == "same-meaning surfaces" && c.Source == "" {
				t.Fatalf("%s has no authored surface", c.ID)
			}
			if c.ID == "SF-06" {
				// An Operator call without an authored rule ID must not
				// borrow the identity of SF-A/B from the expected fixture.
				var expected map[string]json.RawMessage
				if err := json.Unmarshal(c.Expected, &expected); err != nil {
					t.Fatalf("SF-06 expected: %v", err)
				}
				if c.Status != "NOT IMPLEMENTED" || !bytes.Equal(bytes.TrimSpace(expected["id"]), []byte("null")) {
					t.Fatal("SF-06 must keep unresolved rule identity and NOT IMPLEMENTED status")
				}
			}
		}
	}
	if len(seenGroups) != len(minimum) {
		t.Fatalf("S0 fixture has %d groups, expected %d", len(seenGroups), len(minimum))
	}
}

func TestRuleExpectedFixtureEvaluationCases(t *testing.T) {
	fixture := loadRuleExpectedFixture(t)
	rules := mustRuleNamespace(t, ruleAuthoringBoth).Validations
	if len(rules) != 2 {
		t.Fatalf("test fixture produced %d validations, expected two", len(rules))
	}
	run := 0
	for _, group := range fixture.CaseGroups {
		if group.Group != "evaluation distinction" {
			continue
		}
		for _, c := range group.Cases {
			c := c
			t.Run(c.ID, func(t *testing.T) {
				var input map[string]json.RawMessage
				if err := json.Unmarshal(c.Input, &input); err != nil || len(input) != 1 {
					t.Fatalf("%s input must have exactly one field: %v", c.ID, err)
				}
				var expected struct {
					Result string `json:"result"`
				}
				if err := json.Unmarshal(c.Expected, &expected); err != nil || expected.Result == "" {
					t.Fatalf("%s expected result: %v", c.ID, err)
				}
				var rule semantic.Validation
				src := stubSource{}
				if q, ok := input["quantity"]; ok {
					rule = rules[0]
					switch {
					case bytes.Equal(bytes.TrimSpace(q), []byte("null")):
						// A nil number represents unevaluable NULL in the stub.
					default:
						var value string
						if err := json.Unmarshal(q, &value); err != nil {
							t.Fatalf("%s quantity: %v", c.ID, err)
						}
						if value == "unavailable" {
							src.numberErr = errors.New("unavailable")
						} else {
							n, ok := new(big.Rat).SetString(value)
							if !ok {
								t.Fatalf("%s has invalid rational number %q", c.ID, value)
							}
							src.number = n
						}
					}
				} else if keys, ok := input["line_number"]; ok {
					rule = rules[1]
					if strings.HasPrefix(strings.TrimSpace(string(keys)), "[") {
						if err := json.Unmarshal(keys, &src.keys); err != nil {
							t.Fatalf("%s relation keys: %v", c.ID, err)
						}
						// The empty array is a completely observed empty
						// *stub* scope. No actual Provider read is implied.
					} else {
						var value string
						if err := json.Unmarshal(keys, &value); err != nil || value != "unavailable" {
							t.Fatalf("%s unexpected relation input %s: %v", c.ID, keys, err)
						}
						src.keysErr = errors.New("unavailable")
					}
				} else {
					t.Fatalf("%s has unsupported evaluation input", c.ID)
				}
				got, err := validationeval.Evaluate(rule, src)
				if expected.Result == "error" {
					if err == nil {
						t.Fatalf("%s expected evaluation error, got %q", c.ID, got)
					}
				} else if err != nil || string(got) != expected.Result {
					t.Fatalf("%s got %q / %v, expected %q", c.ID, got, err, expected.Result)
				}
			})
			run++
		}
	}
	if run < 9 {
		t.Fatalf("only %d evaluation cases are connected to the S0 fixture", run)
	}
}
