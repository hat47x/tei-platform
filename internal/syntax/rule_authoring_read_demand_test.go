package syntax_test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic/validationeval"
	"tei/internal/syntax"
)

// ruleReadDemandProbe records calls to the evaluator's Source interface.
// It does not represent physical Provider I/O or prove a complete data scope.
// Strict mode refuses an undeclared read; permissive mode demonstrates that
// equal result values alone cannot establish equal information demand.
type ruleReadDemandProbe struct {
	strict            bool
	numberAttribute   core.CanonicalID
	numberValue       *big.Rat
	relation          core.CanonicalID
	relationAttribute core.CanonicalID
	relationKeys      []string
	calls             []string
}

func (p *ruleReadDemandProbe) Number(attribute core.CanonicalID) (*big.Rat, error) {
	p.calls = append(p.calls, "Number("+string(attribute)+")")
	if p.strict && attribute != p.numberAttribute {
		return nil, fmt.Errorf("unexpected Number request for %s", attribute)
	}
	return p.numberValue, nil
}

func (p *ruleReadDemandProbe) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	p.calls = append(p.calls, "RelationKeys("+string(relation)+","+string(attribute)+")")
	if p.strict && (relation != p.relation || attribute != p.relationAttribute) {
		return nil, fmt.Errorf("unexpected RelationKeys request for %s/%s", relation, attribute)
	}
	return p.relationKeys, nil
}

func TestRuleAuthoringSourceDemandMatchesIndependentReference(t *testing.T) {
	const source = `namespace sales
rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive
rule sales.order.lines.line_number_unique:
  sales.order_line.line_number is unique per sales.order.lines
`
	file, err := syntax.ParseRuleAuthoringFile(source)
	if err != nil {
		t.Fatalf("parse candidate authoring source: %v", err)
	}
	ns, issues := syntax.RuleNamespace(file)
	if len(issues) != 0 || len(ns.Validations) != 2 {
		t.Fatalf("candidate meanings=%#v, conversion issues=%v", ns.Validations, issues)
	}
	positive, unique := ns.Validations[0], ns.Validations[1]

	const numberCall = "Number(sales.order_line.quantity)"
	const uniqueCall = "RelationKeys(sales.order.lines,sales.order_line.line_number)"
	assertOnlyCall := func(t *testing.T, probe *ruleReadDemandProbe, want string) {
		t.Helper()
		if got := strings.Join(probe.calls, ","); got != want {
			t.Fatalf("Source interface calls=%q, want exactly %q", got, want)
		}
	}

	t.Run("positive reads only its declared numeric attribute", func(t *testing.T) {
		probe := &ruleReadDemandProbe{
			strict: true, numberAttribute: "sales.order_line.quantity", numberValue: big.NewRat(2, 1),
		}
		got, err := validationeval.Evaluate(positive, probe)
		if err != nil || got != validationeval.ResultSatisfied {
			t.Fatalf("result=%q, error=%v", got, err)
		}
		assertOnlyCall(t, probe, numberCall)
	})

	t.Run("unique reads only its declared relation and attribute", func(t *testing.T) {
		probe := &ruleReadDemandProbe{
			strict: true, relation: "sales.order.lines", relationAttribute: "sales.order_line.line_number",
			relationKeys: []string{"1", "1"},
		}
		got, err := validationeval.Evaluate(unique, probe)
		if err != nil || got != validationeval.ResultNotSatisfied {
			t.Fatalf("result=%q, error=%v", got, err)
		}
		assertOnlyCall(t, probe, uniqueCall)
	})

	t.Run("equal positive result does not prove equal read demand", func(t *testing.T) {
		changed := positive
		changed.Attribute = "sales.order_line.line_number" // Also numeric, but a different meaning.
		probe0 := &ruleReadDemandProbe{numberValue: big.NewRat(2, 1)}
		probe1 := &ruleReadDemandProbe{numberValue: big.NewRat(2, 1)}
		got0, err0 := validationeval.Evaluate(positive, probe0)
		got1, err1 := validationeval.Evaluate(changed, probe1)
		if err0 != nil || err1 != nil || got0 != validationeval.ResultSatisfied || got1 != got0 {
			t.Fatalf("permissive stub must produce equal values: %q/%v, %q/%v", got0, err0, got1, err1)
		}
		assertOnlyCall(t, probe0, numberCall)
		assertOnlyCall(t, probe1, "Number(sales.order_line.line_number)")
		strict := &ruleReadDemandProbe{
			strict: true, numberAttribute: "sales.order_line.quantity", numberValue: big.NewRat(2, 1),
		}
		if got, err := validationeval.Evaluate(changed, strict); err == nil {
			t.Fatalf("changed read demand accepted: result=%q, calls=%v", got, strict.calls)
		}
	})

	t.Run("equal uniqueness result does not prove equal relation scope", func(t *testing.T) {
		changed := unique
		changed.Relation = "sales.order.customer" // A different relation, even when keys happen to match.
		permissive := &ruleReadDemandProbe{relationKeys: []string{"1", "1"}}
		got, err := validationeval.Evaluate(changed, permissive)
		if err != nil || got != validationeval.ResultNotSatisfied {
			t.Fatalf("permissive result=%q, error=%v", got, err)
		}
		assertOnlyCall(t, permissive, "RelationKeys(sales.order.customer,sales.order_line.line_number)")
		strict := &ruleReadDemandProbe{
			strict: true, relation: "sales.order.lines", relationAttribute: "sales.order_line.line_number",
			relationKeys: []string{"1", "1"},
		}
		if got, err := validationeval.Evaluate(changed, strict); err == nil {
			t.Fatalf("changed relation demand accepted: result=%q, calls=%v", got, strict.calls)
		}
	})

	// These tests qualify requests at the validationeval.Source boundary only.
	// They do not prove Context completeness, actual Provider reads or Grant.
}
