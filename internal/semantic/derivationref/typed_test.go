package derivationref

import (
	"errors"
	"reflect"
	"testing"
)

func typedRule() Definition {
	return Definition{
		ID:         "line-amount",
		Inputs:     []string{"quantity", "unit-price"},
		InputTypes: []string{"decimal", "decimal"},
		Operator:   "multiply",
		OutputType: "decimal",
		Pure:       true,
	}
}

func TestTypedDefinitionAcceptsNumericMultiply(t *testing.T) {
	rule := typedRule()
	if err := ValidateTypedDefinition(rule); err != nil {
		t.Fatal(err)
	}
	integer := rule
	integer.InputTypes = []string{"integer", "decimal"}
	if err := ValidateTypedDefinition(integer); err != nil {
		t.Fatalf("integer input should be numeric: %v", err)
	}
}

func TestTypedDefinitionRejectsTypeMismatch(t *testing.T) {
	rule := typedRule()
	rule.InputTypes = []string{"decimal", "text"}
	err := ValidateTypedDefinition(rule)
	if !errors.Is(err, ErrType) {
		t.Fatalf("input mismatch: got %v, want ErrType", err)
	}
	if errors.Is(err, ErrOperator) {
		t.Fatalf("type mismatch must not be reported as an operator failure: %v", err)
	}

	output := typedRule()
	output.OutputType = "text"
	if err := ValidateTypedDefinition(output); !errors.Is(err, ErrType) {
		t.Fatalf("output mismatch: got %v, want ErrType", err)
	}
}

func TestTypedDefinitionRejectsUnknownOperator(t *testing.T) {
	rule := typedRule()
	rule.Operator = "sqrt"
	if err := ValidateTypedDefinition(rule); !errors.Is(err, ErrOperator) {
		t.Fatalf("got %v, want ErrOperator", err)
	}
}

func TestTypedDefinitionRejectsArityAndMissingTypes(t *testing.T) {
	oneInput := typedRule()
	oneInput.Inputs = []string{"quantity"}
	oneInput.InputTypes = []string{"decimal"}
	if err := ValidateTypedDefinition(oneInput); !errors.Is(err, ErrType) {
		t.Fatalf("one input: got %v, want ErrType", err)
	}

	untyped := typedRule()
	untyped.InputTypes = nil
	if err := ValidateTypedDefinition(untyped); !errors.Is(err, ErrType) {
		t.Fatalf("missing types: got %v, want ErrType", err)
	}
}

func TestBindingRejectsTargetTypeMismatch(t *testing.T) {
	rule := typedRule()
	bad := Binding{RuleID: rule.ID, TargetID: "line.amount", Scope: "semantic", TargetType: "text"}
	if err := ValidateBinding(rule, bad); !errors.Is(err, ErrType) {
		t.Fatalf("got %v, want ErrType", err)
	}
	good := Binding{RuleID: rule.ID, TargetID: "line.amount", Scope: "semantic", TargetType: "decimal"}
	if err := ValidateBinding(rule, good); err != nil {
		t.Fatalf("matching target type was rejected: %v", err)
	}
}

func TestBindingRejectsWrongRuleAndEmptyScope(t *testing.T) {
	rule := typedRule()
	wrongRule := Binding{RuleID: "other", TargetID: "line.amount", Scope: "semantic"}
	if err := ValidateBinding(rule, wrongRule); !errors.Is(err, ErrBinding) {
		t.Fatalf("wrong rule: got %v, want ErrBinding", err)
	}
	noScope := Binding{RuleID: rule.ID, TargetID: "line.amount"}
	if err := ValidateBinding(rule, noScope); !errors.Is(err, ErrBinding) {
		t.Fatalf("empty scope: got %v, want ErrBinding", err)
	}
}

func TestTwoBindingsShareOneRuleWithoutChangingIt(t *testing.T) {
	rule := typedRule()
	bindings := []Binding{
		{RuleID: rule.ID, TargetID: "line.amount", Scope: "semantic", TargetType: "decimal"},
		{RuleID: rule.ID, TargetID: "query.total", Scope: "query", TargetType: "decimal"},
	}
	for _, binding := range bindings {
		if err := ValidateBinding(rule, binding); err != nil {
			t.Fatalf("binding %+v: %v", binding, err)
		}
	}
	if got := BindingsForRule(bindings, rule.ID); len(got) != 2 {
		t.Fatalf("BindingsForRule = %+v", got)
	}

	// Changing one binding's target type affects only that binding, not the rule.
	changed := bindings[1]
	changed.TargetType = "text"
	if err := ValidateBinding(rule, changed); !errors.Is(err, ErrType) {
		t.Fatalf("changed binding: got %v, want ErrType", err)
	}
	if err := ValidateBinding(rule, bindings[0]); err != nil {
		t.Fatalf("the other binding must stay valid: %v", err)
	}
	if !reflect.DeepEqual(rule, typedRule()) {
		t.Fatalf("the definition changed: %+v", rule)
	}
}

func TestValidateGraphStillAcceptsUntypedDefinitions(t *testing.T) {
	// The typed fields are optional; existing untyped callers keep working.
	defs := []Definition{{ID: "a", Pure: true, Operator: "multiply"}}
	if err := ValidateGraph(defs, nil); err != nil {
		t.Fatalf("untyped definition was rejected: %v", err)
	}
}
