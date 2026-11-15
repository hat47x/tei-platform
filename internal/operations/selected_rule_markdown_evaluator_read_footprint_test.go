package operations

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
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
		number                              *big.Rat
		keys                                []string
		complete                            bool
		want                                validationeval.Result
		wantRead                            string
		wantError                           bool
	}{
		{"positive-two", "sales.order_line.quantity_positive", "positive",
			"sales.order_line.quantity", "", big.NewRat(2, 1), nil, true,
			validationeval.ResultSatisfied, "Number:sales.order_line.quantity", false},
		{"positive-zero", "sales.order_line.quantity_positive", "positive",
			"sales.order_line.quantity", "", big.NewRat(0, 1), nil, true,
			validationeval.ResultNotSatisfied, "Number:sales.order_line.quantity", false},
		{"positive-null", "sales.order_line.quantity_positive", "positive",
			"sales.order_line.quantity", "", nil, nil, true,
			"", "Number:sales.order_line.quantity", true},
		{"positive-missing", "sales.order_line.quantity_positive", "positive",
			"sales.order_line.quantity", "", nil, nil, false,
			"", "Number:sales.order_line.quantity", true},
		{"unique-complete", "sales.order.lines.line_number_unique", "relation-unique",
			"sales.order_line.line_number", "sales.order.lines", nil, []string{"1", "2"}, true,
			validationeval.ResultSatisfied, "RelationKeys:sales.order.lines/sales.order_line.line_number", false},
		{"unique-duplicate", "sales.order.lines.line_number_unique", "relation-unique",
			"sales.order_line.line_number", "sales.order.lines", nil, []string{"1", "1"}, true,
			validationeval.ResultNotSatisfied, "RelationKeys:sales.order.lines/sales.order_line.line_number", false},
		{"unique-empty-complete", "sales.order.lines.line_number_unique", "relation-unique",
			"sales.order_line.line_number", "sales.order.lines", nil, []string{}, true,
			validationeval.ResultSatisfied, "RelationKeys:sales.order.lines/sales.order_line.line_number", false},
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
				number:    tc.number,
				keys:      tc.keys,
				complete:  tc.complete,
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


// fixedRuleMutationSource is one unchanged input dataset for both Rule versions.
// A changed result is therefore attributable to the authored Rule's read target,
// not to editing the data or substituting a different Provider.
type fixedRuleMutationSource struct {
	reads []core.CanonicalID
}

func (s *fixedRuleMutationSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	s.reads = append(s.reads, attribute)
	switch attribute {
	case "sales.order_line.quantity":
		return big.NewRat(0, 1), nil
	case "sales.order_line.line_number":
		return big.NewRat(2, 1), nil
	default:
		return nil, fmt.Errorf("unexpected attribute %s", attribute)
	}
}

func (s *fixedRuleMutationSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	return nil, fmt.Errorf("unexpected relation read %s / %s", relation, attribute)
}

// A single edit to the selected Markdown Rule must change the Canonical
// dependency, the Evaluator's actual Source read and the business result.
// Merely updating the display text or replaying a fixed Go function fails.
func TestSelectedMarkdownRuleEditChangesActualEvaluatorRead(t *testing.T) {
	sourceRoot := filepath.Join("..", "workspace", "testdata", "tei-md-rule")
	root := t.TempDir()
	for _, rel := range []string{
		"tei.workspace.yaml",
		"semantic/sales/order.tei",
		"semantic/sales/order-line-rules.tei.md",
	} {
		raw, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read product Rule fixture %s: %v", rel, err)
		}
		dest := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	subject := sourceindex.Subject{
		Kind: "semantic-validation", ID: "sales.order_line.quantity_positive",
	}
	readSelected := func() (SubjectInspection, string) {
		t.Helper()
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("load selected Rule: %v", err)
		}
		if len(loaded.AuthoringSources) != 1 ||
			loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
			loaded.AuthoringSources[0].Definitions != 2 {
			t.Fatalf("Rule language selection changed: %+v", loaded.AuthoringSources)
		}
		index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatal(err)
		}
		inspection, err := InspectionFromLoad(loaded, index, subject)
		if err != nil {
			t.Fatalf("inspect selected Rule: %v", err)
		}
		if len(inspection.Sources.References) != 1 ||
			len(inspection.Sources.Artifacts) != 1 {
			t.Fatalf("Rule lacks one exact Source occurrence: %+v", inspection.Sources)
		}
		return inspection, sourceindex.ModelRevision(index)
	}
	evaluate := func(inspection SubjectInspection, wantAttribute core.CanonicalID, want validationeval.Result) {
		t.Helper()
		var rule semantic.Validation
		if err := json.Unmarshal(inspection.Canonical, &rule); err != nil {
			t.Fatal(err)
		}
		if rule.ID != core.CanonicalID(subject.ID) ||
			rule.Kind != semantic.ValidationPositive || rule.Attribute != wantAttribute {
			t.Fatalf("authored Rule did not change its canonical dependency: %+v", rule)
		}
		src := &fixedRuleMutationSource{}
		got, err := validationeval.Evaluate(rule, src)
		if err != nil || got != want {
			t.Fatalf("evaluated %s: got %q err=%v, want %q", wantAttribute, got, err, want)
		}
		if len(src.reads) != 1 || src.reads[0] != wantAttribute {
			t.Fatalf("actual read target %v, want only %s", src.reads, wantAttribute)
		}
	}

	before, beforeRevision := readSelected()
	evaluate(before, "sales.order_line.quantity", validationeval.ResultNotSatisfied)

	path := filepath.Join(root, "semantic", "sales", "order-line-rules.tei.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const original = "sales.order_line.quantity is positive"
	const replacement = "sales.order_line.line_number is positive"
	if strings.Count(string(raw), original) != 1 {
		t.Fatal("the selected Rule's original expression is not unique")
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), original, replacement, 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	after, afterRevision := readSelected()
	evaluate(after, "sales.order_line.line_number", validationeval.ResultSatisfied)
	if beforeRevision == afterRevision ||
		before.SourceIndexFingerprint == after.SourceIndexFingerprint ||
		before.WorldFingerprint == after.WorldFingerprint ||
		before.Sources.Artifacts[0].Fingerprint == after.Sources.Artifacts[0].Fingerprint {
		t.Fatal("selected Rule edit did not change its Source revision and typed World")
	}
	// The ID text precedes the edited predicate. Same Subject/byte location
	// alone cannot qualify the old interpretation of a changed Rule.
	if before.Subject != after.Subject ||
		before.Sources.References[0].Range != after.Sources.References[0].Range {
		t.Fatal("Rule identifier was moved or silently retargeted")
	}
}
