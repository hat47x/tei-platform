package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/validation"
)

// These tests add an operator that no production code knows. The contract is
// a document under semantic/, the implementation is a function in this file.
// No parser, enum, validator, or evaluator is edited to add it.

const (
	teiMDOperatorDeclarations = "semantic/sales/zz-operators.tei.md"
	teiMDOperatorRules        = "semantic/sales/discount-rules.tei.md"
)

func teiMDOperatorWorkspace(t *testing.T) string {
	t.Helper()
	root := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, root, "semantic/sales/order.tei",
		"    attribute itemName type std.text\n",
		"    attribute itemName type std.text\n    attribute discount type std.decimal\n")
	// The declaring document sorts after the using one, so a reader that
	// needed declarations first would fail here.
	writeTEIMDTestFile(t, root, teiMDOperatorDeclarations,
		"# 値引きの検査に使う演算子\n\n0以上であることを確かめる。0は認める。\n\n"+
			"```tei\nnamespace sales\n\noperator sales.non_negative: takes number\n```\n")
	writeTEIMDTestFile(t, root, teiMDOperatorRules,
		"# 値引きの規則\n\n値引きは負の数にしない。値引きなしの0は正しい入力である。\n\n"+
			"```tei\nnamespace sales\n\nrule sales.order_line.discount_non_negative:\n  sales.order_line.discount is sales.non_negative\n```\n")
	return root
}

func operatorsOf(result Result) []semantic.Operator {
	var out []semantic.Operator
	for _, ns := range result.World.Semantics {
		out = append(out, ns.Operators...)
	}
	return out
}

func diagnosticCodes(result Result) []string {
	var codes []string
	for _, diagnostic := range validation.Validate(result.World) {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

type fixedNumberSource struct {
	value *big.Rat
	calls []string
}

func (s *fixedNumberSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	s.calls = append(s.calls, "Number("+string(attribute)+")")
	return s.value, nil
}

func (s *fixedNumberSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	s.calls = append(s.calls, "RelationKeys")
	return nil, errors.New("not expected")
}

func TestTEIMDOperatorDeclaredInADocumentDecidesARule(t *testing.T) {
	root := teiMDOperatorWorkspace(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if codes := diagnosticCodes(loaded); len(codes) != 0 {
		t.Fatalf("validator rejects the declared operator: %v", codes)
	}

	want := semantic.Operator{ID: "sales.non_negative", Symbol: "non_negative", Input: semantic.OperatorInputNumber}
	declared := operatorsOf(loaded)
	if len(declared) != 1 || !reflect.DeepEqual(declared[0], want) {
		t.Fatalf("operators=%+v, want %+v", declared, want)
	}
	rule := loadedValidation(t, loaded, "sales.order_line.discount_non_negative")
	if rule.Kind != semantic.ValidationOperator || rule.Operator != want.ID || rule.Attribute != "sales.order_line.discount" {
		t.Fatalf("rule=%+v", rule)
	}

	// The implementation a selected plugin would supply.
	calls := 0
	operators := validationeval.Operators{
		Declared: declared,
		Implementations: []validationeval.OperatorImplementation{{
			Operator: want.ID, Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) { calls++; return value.Sign() >= 0, nil },
		}},
	}
	for value, expected := range map[int64]validationeval.Result{
		300: validationeval.ResultSatisfied,
		0:   validationeval.ResultSatisfied,
		-1:  validationeval.ResultNotSatisfied,
	} {
		source := &fixedNumberSource{value: big.NewRat(value, 1)}
		got, err := validationeval.EvaluateWithOperators(rule, source, operators)
		if err != nil || got != expected {
			t.Fatalf("discount %d: %q %v, want %q", value, got, err, expected)
		}
		if strings.Join(source.calls, ",") != "Number(sales.order_line.discount)" {
			t.Fatalf("discount %d: read %v", value, source.calls)
		}
	}
	if calls != 3 {
		t.Fatalf("implementation calls=%d, want 3", calls)
	}
	// Without the implementation the rule has no answer. It is not skipped.
	if got, err := validationeval.EvaluateWithOperators(rule, &fixedNumberSource{value: big.NewRat(1, 1)},
		validationeval.Operators{Declared: declared}); err == nil || got != "" {
		t.Fatalf("decided without an implementation: %q %v", got, err)
	}

	// The built-in rules of the same Workspace are untouched.
	positive := loadedValidation(t, loaded, "sales.order_line.quantity_positive")
	if positive.Kind != semantic.ValidationPositive || positive.Operator != "" {
		t.Fatalf("built-in rule changed: %+v", positive)
	}

	// The declaration is findable where it was written.
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	document := readTEIMDTestFile(t, root, teiMDOperatorDeclarations)
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-operator", ID: "sales.non_negative"})
	if len(refs) != 1 || refs[0].ArtifactPath != teiMDOperatorDeclarations ||
		refs[0].Range.Start.Offset != bytes.Index(document, []byte("sales.non_negative")) {
		t.Fatalf("operator source reference=%+v", refs)
	}
}

func TestTEIMDOperatorWorldWithoutOperatorsIsUnchanged(t *testing.T) {
	// Adding the operator mechanism must not change what an existing
	// Workspace serializes to: no empty operator list, no operator field.
	loaded, err := Load(copyTEIMDRuleWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(loaded.World.Semantics)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"operators"`) || strings.Contains(string(raw), `"operator"`) {
		t.Fatalf("a Workspace that declares no operator serializes operator fields: %s", raw)
	}
	// With an operator the same serialization does carry it, so the check
	// above is not passing because the fields are never written.
	withOperator, err := Load(teiMDOperatorWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(withOperator.World.Semantics)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"operators":[{"id":"sales.non_negative"`) ||
		!strings.Contains(string(raw), `"operator":"sales.non_negative"`) {
		t.Fatalf("operator data is missing from the serialized World: %s", raw)
	}
}

func TestTEIMDOperatorNegatives(t *testing.T) {
	t.Run("rule uses an operator no document declares", func(t *testing.T) {
		root := teiMDOperatorWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDOperatorRules, "is sales.non_negative", "is sales.non_negativ")
		loaded, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if codes := diagnosticCodes(loaded); len(codes) != 1 || codes[0] != "UNKNOWN_SEMANTIC_VALIDATION_OPERATOR" {
			t.Fatalf("codes=%v", codes)
		}
	})

	t.Run("number operator on a text attribute", func(t *testing.T) {
		root := teiMDOperatorWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDOperatorRules, "sales.order_line.discount is", "sales.order_line.item_name is")
		loaded, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if codes := diagnosticCodes(loaded); len(codes) != 1 || codes[0] != "SEMANTIC_VALIDATION_OPERATOR_REQUIRES_NUMERIC_ATTRIBUTE" {
			t.Fatalf("codes=%v", codes)
		}
	})

	t.Run("an attribute written where the operator belongs", func(t *testing.T) {
		root := teiMDOperatorWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDOperatorRules, "is sales.non_negative", "is sales.order_line.quantity")
		loaded, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if codes := diagnosticCodes(loaded); len(codes) != 1 || codes[0] != "UNKNOWN_SEMANTIC_VALIDATION_OPERATOR" {
			t.Fatalf("an attribute was accepted as an operator: %v", codes)
		}
	})

	t.Run("operator declared in two documents", func(t *testing.T) {
		root := teiMDOperatorWorkspace(t)
		writeTEIMDTestFile(t, root, "semantic/sales/zzz-copy.tei.md",
			"```tei\nnamespace sales\noperator sales.non_negative: takes number\n```\n")
		_, err := Load(root)
		if err == nil || !strings.Contains(err.Error(), "already declared at "+teiMDOperatorDeclarations) {
			t.Fatalf("the later declaration was accepted: %v", err)
		}
	})

	t.Run("same symbol in another namespace is another operator", func(t *testing.T) {
		root := teiMDOperatorWorkspace(t)
		writeTEIMDTestFile(t, root, "semantic/purchase/operators.tei.md",
			"```tei\nnamespace purchase\noperator purchase.non_negative: takes number\n```\n")
		loaded, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(operatorsOf(loaded)) != 2 {
			t.Fatalf("operators=%+v", operatorsOf(loaded))
		}
		// The sales rule still names the sales operator, and an implementation
		// of the purchase operator does not decide it.
		rule := loadedValidation(t, loaded, "sales.order_line.discount_non_negative")
		got, err := validationeval.EvaluateWithOperators(rule, &fixedNumberSource{value: big.NewRat(1, 1)}, validationeval.Operators{
			Declared: operatorsOf(loaded),
			Implementations: []validationeval.OperatorImplementation{{
				Operator: "purchase.non_negative", Input: semantic.OperatorInputNumber,
				Number: func(*big.Rat) (bool, error) { return true, nil },
			}},
		})
		if err == nil || got != "" {
			t.Fatalf("an operator of the same name in another namespace decided the rule: %q %v", got, err)
		}
	})
}
