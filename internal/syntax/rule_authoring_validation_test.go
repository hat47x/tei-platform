package syntax_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/stdlib"
	"tei/internal/syntax"
	"tei/internal/validation"
)

// These cases connect the rule authoring surface to the existing typed meaning:
// semantic.Validation, validation.Validate, and validationeval.Evaluate. No
// evaluator, kind, or validation rule is added here.

func mustRuleNamespace(t *testing.T, src string) semantic.Namespace {
	t.Helper()
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ns, issues := syntax.RuleNamespace(file)
	if len(issues) != 0 {
		t.Fatalf("conversion issues: %v", issues)
	}
	return ns
}

// S2 primary case: the same authored intent becomes the same
// semantic.Validation value the existing grammar produces for the reference
// workspace.
func TestRuleNamespaceMatchesExistingValidationValues(t *testing.T) {
	ns := mustRuleNamespace(t, ruleAuthoringBoth)
	if len(ns.Validations) != 2 {
		t.Fatalf("validations=%d", len(ns.Validations))
	}

	positive := ns.Validations[0]
	wantPositive := semantic.Validation{
		ID:        core.MustCanonicalID("sales.order_line.quantity_positive"),
		Symbol:    "quantity_positive",
		Kind:      semantic.ValidationPositive,
		Attribute: core.MustCanonicalID("sales.order_line.quantity"),
	}
	if positive != wantPositive {
		t.Fatalf("positive=%#v want %#v", positive, wantPositive)
	}

	unique := ns.Validations[1]
	wantUnique := semantic.Validation{
		ID:        core.MustCanonicalID("sales.order.lines.line_number_unique"),
		Symbol:    "line_number_unique",
		Kind:      semantic.ValidationRelationUnique,
		Attribute: core.MustCanonicalID("sales.order_line.line_number"),
		Relation:  core.MustCanonicalID("sales.order.lines"),
	}
	if unique != wantUnique {
		t.Fatalf("unique=%#v want %#v", unique, wantUnique)
	}
}

// The existing grammar is the independent side of that comparison: it must
// produce exactly the same two values from the reference workspace source.
func TestExistingValidationGrammarProducesTheSameValues(t *testing.T) {
	existing := `namespace sales
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}
validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}`
	ns, err := syntax.NewParser(existing).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse existing: %v", err)
	}
	converted := mustRuleNamespace(t, ruleAuthoringBoth)
	if len(ns.Validations) != len(converted.Validations) {
		t.Fatalf("counts differ: %d vs %d", len(ns.Validations), len(converted.Validations))
	}
	for i := range ns.Validations {
		if ns.Validations[i] != converted.Validations[i] {
			t.Fatalf("validation %d differs:\nexisting %#v\nrule     %#v", i, ns.Validations[i], converted.Validations[i])
		}
	}
}

// stubSource is the existing evaluator's boundary, implemented by the test.
// It is not a provider and not a second evaluator.
type stubSource struct {
	number    *big.Rat
	numberErr error
	keys      []string
	keysErr   error
}

func (s stubSource) Number(core.CanonicalID) (*big.Rat, error) { return s.number, s.numberErr }
func (s stubSource) RelationKeys(core.CanonicalID, core.CanonicalID) ([]string, error) {
	return s.keys, s.keysErr
}

// M-01..M-03 and M-09: the existing evaluator decides. The surface must not
// pre-chew NULL, unavailable, or empty into a pass.
func TestRuleValidationUsesTheExistingEvaluator(t *testing.T) {
	ns := mustRuleNamespace(t, ruleAuthoringBoth)
	positive := ns.Validations[0]
	unique := ns.Validations[1]
	number := func(value *big.Rat) stubSource { return stubSource{number: value} }
	keys := func(values ...string) stubSource { return stubSource{keys: values} }

	cases := []struct {
		name    string
		rule    semantic.Validation
		src     stubSource
		want    validationeval.Result
		wantErr bool
	}{
		{"EX-01 positive 2", positive, number(big.NewRat(2, 1)), validationeval.ResultSatisfied, false},
		{"EX-02 zero", positive, number(big.NewRat(0, 1)), validationeval.ResultNotSatisfied, false},
		{"EX-03 negative", positive, number(big.NewRat(-1, 1)), validationeval.ResultNotSatisfied, false},
		{"EX-04 null is not zero", positive, stubSource{}, "", true},
		{"EX-05 unavailable is an error", positive, stubSource{numberErr: errors.New("unavailable")}, "", true},
		{"EX-06 unique 1,2", unique, keys("1", "2"), validationeval.ResultSatisfied, false},
		{"EX-07 duplicate 1,1", unique, keys("1", "1"), validationeval.ResultNotSatisfied, false},
		{"EX-08 empty scope is satisfied", unique, keys(), validationeval.ResultSatisfied, false},
		{"EX-09 unavailable keys is an error", unique, stubSource{keysErr: errors.New("unavailable")}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validationeval.Evaluate(tc.rule, tc.src)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v want error=%t", err, tc.wantErr)
			}
			if !tc.wantErr && result != tc.want {
				t.Fatalf("result=%q want %q", result, tc.want)
			}
		})
	}
}

// ruleValidatorWorld builds a Workspace the existing validator can check. The
// candidate validations are merged into the Data namespace rather than added as
// a second namespace: a Workspace loader appends a parsed file into the
// namespace it already loaded, and two namespace records would themselves be a
// duplicate.
func ruleValidatorWorld(validations ...semantic.Validation) ir.World {
	sales := semantic.Namespace{
		ID: "sales",
		Types: []semantic.Type{
			{ID: "sales.order_status", Symbol: "OrderStatus", Kind: semantic.TypeAlias, Underlying: "std.code"},
		},
		Data: []semantic.Data{
			{ID: "sales.order_line", Symbol: "OrderLine",
				Attributes: []semantic.Attribute{
					{ID: "sales.order_line.quantity", Symbol: "quantity", TypeRef: "std.decimal"},
					{ID: "sales.order_line.line_number", Symbol: "lineNumber", TypeRef: "std.integer"},
				}},
			{ID: "sales.order", Symbol: "Order",
				Attributes: []semantic.Attribute{
					{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "std.identifier"},
					{ID: "sales.order.customer_id", Symbol: "customerId", TypeRef: "std.identifier"},
				},
				Relations: []semantic.Relation{
					{ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
					{ID: "sales.order.customer", Symbol: "customer", Target: "sales.order", Cardinality: semantic.CardinalityOne},
				}},
		},
		Validations: validations,
	}
	return ir.World{Semantics: []semantic.Namespace{stdlib.SemanticNamespace(), sales}}
}

// M-03/M-04/M-05/M-08: the existing validator decides, and the codes are the
// existing ones rather than a parallel vocabulary.
func TestRuleValidationReachesTheExistingValidator(t *testing.T) {
	check := func(t *testing.T, src string, wantCode string) {
		t.Helper()
		diagnostics := validation.Validate(ruleValidatorWorld(mustRuleNamespace(t, src).Validations...))
		if wantCode == "" {
			for _, d := range diagnostics {
				if d.Severity == validation.Error {
					t.Fatalf("unexpected error diagnostic: %s %s", d.Code, d.Message)
				}
			}
			return
		}
		for _, d := range diagnostics {
			if d.Code == wantCode {
				return
			}
		}
		t.Fatalf("missing %s in %v", wantCode, diagnostics)
	}

	t.Run("EX-15 clean model across both grammars has no error", func(t *testing.T) {
		check(t, ruleAuthoringBoth, "")
	})
	t.Run("EX-11 positive on an identifier type", func(t *testing.T) {
		check(t, "namespace sales\nrule sales.order_line.quantity_positive:\n  sales.order.order_id is positive\n",
			"SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE")
	})
	t.Run("EX-10 unique over a one relation", func(t *testing.T) {
		check(t, "namespace sales\nrule sales.order.lines.line_number_unique:\n  sales.order_line.line_number is unique per sales.order.customer\n",
			"SEMANTIC_VALIDATION_RELATION_NOT_MANY")
	})
	t.Run("EX-10b unique over an unknown relation", func(t *testing.T) {
		check(t, "namespace sales\nrule sales.order.lines.line_number_unique:\n  sales.order_line.line_number is unique per sales.order.nope\n",
			"UNKNOWN_SEMANTIC_VALIDATION_RELATION")
	})
	t.Run("N-02 unknown attribute", func(t *testing.T) {
		check(t, "namespace sales\nrule sales.order_line.quantity_positive:\n  sales.order_line.quantity2 is positive\n",
			"UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE")
	})
	t.Run("EX-12 duplicate id across both grammars", func(t *testing.T) {
		existing := semantic.Validation{
			ID: "sales.order_line.quantity_positive", Symbol: "quantity_positive",
			Kind: semantic.ValidationPositive, Attribute: "sales.order_line.quantity",
		}
		ns := mustRuleNamespace(t, "namespace sales\nrule sales.order_line.quantity_positive:\n  sales.order_line.quantity is positive\n")
		w := ruleValidatorWorld(existing)
		w.Semantics[1].Validations = append(w.Semantics[1].Validations, ns.Validations...)
		diagnostics := validation.Validate(w)
		for _, d := range diagnostics {
			if d.Code == "DUPLICATE_SEMANTIC_VALIDATION" {
				return
			}
		}
		t.Fatalf("the two grammars produced one identity silently: %v", diagnostics)
	})
}

// M-07: a bare attribute name is ambiguous. The conversion must refuse it
// rather than pick a Data by proximity.
func TestRuleConversionRefusesBareReferences(t *testing.T) {
	src := "namespace sales\nrule sales.q: quantity is positive\n"
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ns, issues := syntax.RuleNamespace(file)
	if len(ns.Validations) != 0 {
		t.Fatalf("ambiguous reference produced a validation: %#v", ns.Validations)
	}
	if len(issues) != 1 {
		t.Fatalf("issues=%v", issues)
	}
	issue := issues[0]
	if got := src[issue.Span.Start.Offset:issue.Span.End.Offset]; got != "quantity" {
		t.Fatalf("issue span=%q", got)
	}
	if !strings.Contains(issue.Msg, "not a qualified canonical id") {
		t.Fatalf("message=%q", issue.Msg)
	}
}

// A bad rule must not hide a good one: every issue is reported, and a rule with
// an issue contributes no validation, so a half-resolved rule never reaches the
// validator as plausible meaning.
func TestRuleConversionReportsEveryIssue(t *testing.T) {
	src := `namespace sales
rule sales.a: bare is positive
rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive
`
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ns, issues := syntax.RuleNamespace(file)
	if len(issues) != 1 {
		t.Fatalf("issues=%v", issues)
	}
	if len(ns.Validations) != 1 || ns.Validations[0].ID != "sales.order_line.quantity_positive" {
		t.Fatalf("good rule was dropped with the bad one: %#v", ns.Validations)
	}
}
