package derivationref

import "fmt"

// numericTypes are the declared semantic types the multiply operator accepts.
// This is a reference-local declaration; it is not a Core type system.
var numericTypes = map[string]bool{"decimal": true, "integer": true}

// ValidateTypedDefinition checks the optional typed contract of a rule. A type
// mismatch is reported separately from an unsupported operator, so a caller can
// tell "the operator does not exist" from "the operator exists but the declared
// types do not fit".
func ValidateTypedDefinition(d Definition) error {
	if d.Operator != "multiply" {
		return fmt.Errorf("%w: %q", ErrOperator, d.Operator)
	}
	if len(d.Inputs) != 2 || len(d.InputTypes) != len(d.Inputs) {
		return fmt.Errorf("%w: %s needs two typed inputs", ErrType, d.ID)
	}
	for _, declared := range d.InputTypes {
		if !numericTypes[declared] {
			return fmt.Errorf("%w: input type %q", ErrType, declared)
		}
	}
	if !numericTypes[d.OutputType] {
		return fmt.Errorf("%w: output type %q", ErrType, d.OutputType)
	}
	return nil
}

// ValidateBinding checks that a binding references the rule and that a declared
// target type agrees with the rule's declared output type. The rule and the
// target stay independent: two bindings may share a rule and differ by target.
func ValidateBinding(d Definition, b Binding) error {
	if b.RuleID != d.ID {
		return fmt.Errorf("%w: binding rule %q, definition %q", ErrBinding, b.RuleID, d.ID)
	}
	if b.Scope == "" {
		return fmt.Errorf("%w: binding scope is required", ErrBinding)
	}
	if b.TargetType != "" && d.OutputType != "" && b.TargetType != d.OutputType {
		return fmt.Errorf("%w: target type %q, rule output %q", ErrType, b.TargetType, d.OutputType)
	}
	return nil
}

// BindingsForRule returns the bindings that reference ruleID, in input order.
func BindingsForRule(bindings []Binding, ruleID string) []Binding {
	var out []Binding
	for _, b := range bindings {
		if b.RuleID == ruleID {
			out = append(out, b)
		}
	}
	return out
}
