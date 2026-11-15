package operations

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// evaluatorReadProbe is a test-only synthetic Source, not a product Provider.
// It refuses undeclared read targets and incomplete relation scopes rather
// than returning plausible values for a different rule or an unknown set.
type evaluatorReadProbe struct {
	attribute core.CanonicalID
	relation  core.CanonicalID
	number    *big.Rat
	keys      []string
	complete  bool
	reads     []string
}

func (s *evaluatorReadProbe) Number(attribute core.CanonicalID) (*big.Rat, error) {
	s.reads = append(s.reads, "Number:"+string(attribute))
	if attribute != s.attribute || s.relation != "" {
		return nil, fmt.Errorf("unexpected numeric read target %s", attribute)
	}
	if !s.complete {
		return nil, errors.New("numeric input not obtained")
	}
	return s.number, nil
}

func (s *evaluatorReadProbe) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	s.reads = append(s.reads, "RelationKeys:"+string(relation)+"/"+string(attribute))
	if relation != s.relation || attribute != s.attribute || s.relation == "" {
		return nil, fmt.Errorf("unexpected scoped read %s / %s", relation, attribute)
	}
	if !s.complete {
		return nil, errors.New("relation input scope not obtained")
	}
	return s.keys, nil
}

// This is the first product-selected Markdown Rule -> existing Go Evaluator
// handoff. Workspace selection and typed meaning are real; numeric/key inputs
// remain a deliberately strict test double, with no Driver, Provider or Effect.
func TestSelectedMarkdownRulesDriveEvaluatorReadFootprint(t *testing.T) {
	root := filepath.Join("..", "workspace", "testdata", "tei-md-rule")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("selected Rule Markdown Load: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v0" ||
		loaded.AuthoringSources[0].Definitions != 2 {
		t.Fatalf("product Rule Reader is not selected: %+v", loaded.AuthoringSources)
	}
	idx, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("Source Index from same Load: %v", err)
	}

	// The IDs, read targets, inputs and results here are fixed expectations.
	// They are not generated from the canonical output being checked.
	cases := []struct {
		name, id, kind, attribute, relation string
		number *big.Rat
		keys []string
		complete bool
		want validationeval.Result
		wantRead string
		wantError bool
	}{
		{"positive-two", "sales.order_line.quantity_positive", "positive",
			"sales.order_line.quantity", "", big.NewRat(2, 1), nil, true,
			validationeval.ResultSatisfied, "Number:sales.order_line.quantity", false},
		{"positive-zero", "sales.order_line.quantity_positive", "positive",
			"sales.order_line.quantity", "", big.NewRat(0, 1), nil, true,
			validationeval.ResultNotSatisfied, "Number:sales.order_line.quantity", false},
		{"positive-missing", "sales.order_line.quantity_positive", "positive",
			"sales.order_line.quantity", "", nil, nil, false,
			"", "Number:sales.order_line.quantity", true},
		{"unique-complete", "sales.order.lines.line_number_unique", "relation-unique",
			"sales.order_line.line_number", "sales.order.lines", nil, []string{"1", "2"}, true,
			validationeval.ResultSatisfied, "RelationKeys:sales.order.lines/sales.order_line.line_number", false},
		{"unique-duplicate", "sales.order.lines.line_number_unique", "relation-unique",
			"sales.order_line.line_number", "sales.order.lines", nil, []string{"1", "1"}, true,
			validationeval.ResultNotSatisfied, "RelationKeys:sales.order.lines/sales.order_line.line_number", false},
		{"unique-scope-unknown", "sales.order.lines.line_number_unique", "relation-unique",
			"sales.order_line.line_number", "sales.order.lines", nil, nil, false,
			"", "RelationKeys:sales.order.lines/sales.order_line.line_number", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subject := sourceindex.Subject{Kind: "semantic-validation", ID: tc.id}
			inspection, err := InspectionFromLoad(loaded, idx, subject)
			if err != nil {
				t.Fatalf("selected Rule was not inspected: %v", err)
			}
			if inspection.Subject != subject || inspection.Effect != "read-only" ||
				inspection.SourceStatus != "indexed" || len(inspection.Sources.References) != 1 {
				t.Fatalf("cannot qualify selected Rule read: %+v", inspection)
			}
			var rule semantic.Validation
			if err := json.Unmarshal(inspection.Canonical, &rule); err != nil {
				t.Fatalf("decode real canonical Rule: %v", err)
			}
			if string(rule.ID) != tc.id || string(rule.Kind) != tc.kind ||
				string(rule.Attribute) != tc.attribute || string(rule.Relation) != tc.relation {
				t.Fatalf("selected Rule changed expected read targets: %+v", rule)
			}
			probe := &evaluatorReadProbe{
				attribute: core.CanonicalID(tc.attribute),
				relation:  core.CanonicalID(tc.relation),
				number: tc.number, keys: tc.keys, complete: tc.complete,
			}
			got, err := validationeval.Evaluate(rule, probe)
			if (err != nil) != tc.wantError {
				t.Fatalf("result=%q error=%v, want error=%t", got, err, tc.wantError)
			}
			if tc.wantError {
				if got != "" {
					t.Fatalf("unavailable input was promoted to business outcome: %q", got)
				}
			} else if got != tc.want {
				t.Fatalf("result=%q, want %q", got, tc.want)
			}
			if len(probe.reads) != 1 || probe.reads[0] != tc.wantRead {
				t.Fatalf("wrong actual input read footprint: %v want %q", probe.reads, tc.wantRead)
			}
		})
	}
}
