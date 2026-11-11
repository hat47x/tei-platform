package syntax_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// This test ties the rule surface to the independent expected fixture in
// plan/fixtures/dsl-syntax-candidates. The fixture is written by hand from the
// existing evaluator/validator contracts; it is not regenerated from this
// parser's output. A parser that silently changed a surface -> meaning mapping
// would still parse, but this test would stop matching the fixture.
func TestRuleAuthoringExpectedFixtureCoversImplementedSurfaces(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "plan", "fixtures", "dsl-syntax-candidates", "rule_authoring_expected_20261111.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture struct {
		Status     string `json:"status"`
		CaseGroups []struct {
			Group string `json:"group"`
			Cases []struct {
				ID       string  `json:"id"`
				Surface  string  `json:"surface"`
				Source   string  `json:"source"`
				Status   *string `json:"status"`
				Expected map[string]any
			} `json:"cases"`
		} `json:"caseGroups"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("fixture must be JSON: %v", err)
	}
	if fixture.Status != "INDEPENDENT_EXPECTED_NOT_OWNER_ACCEPTED" {
		t.Fatalf("fixture status=%q", fixture.Status)
	}

	implemented := 0
	for _, group := range fixture.CaseGroups {
		if group.Group != "same-meaning surfaces" {
			continue
		}
		for _, tc := range group.Cases {
			if tc.Status != nil && *tc.Status == "NOT IMPLEMENTED" {
				continue
			}
			if tc.Source == "" {
				t.Fatalf("%s has no source", tc.ID)
			}
			implemented++
			got, err := ruleSurfaceValues(t, tc.Source)
			if err != nil {
				t.Fatalf("%s: %v", tc.ID, err)
			}
			wantID, _ := tc.Expected["id"].(string)
			wantKind, _ := tc.Expected["kind"].(string)
			wantAttribute, _ := tc.Expected["attribute"].(string)
			wantRelation, _ := tc.Expected["relation"].(string)
			if len(got) != 1 {
				t.Fatalf("%s produced %d validations", tc.ID, len(got))
			}
			value := got[0]
			if string(value.ID) != wantID || string(value.Kind) != wantKind ||
				string(value.Attribute) != wantAttribute || string(value.Relation) != wantRelation {
				t.Fatalf("%s value=%#v want id=%s kind=%s attribute=%s relation=%s",
					tc.ID, value, wantID, wantKind, wantAttribute, wantRelation)
			}
		}
	}
	// Losing the implemented surfaces would let the fixture stop constraining
	// the parser while still passing.
	if implemented < 4 {
		t.Fatalf("only %d implemented surfaces are compared", implemented)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
