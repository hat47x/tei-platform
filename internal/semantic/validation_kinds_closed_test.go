package semantic

import "testing"

// The rule kinds are closed. A new rule is added as a declared operator with
// named inputs (ValidationOperator), not as a new kind: a kind reaches the
// parser, this list, the validator, the evaluator, the row reader and the
// derived tools at once. The decision and the way to add a rule are in
// plan/design/RULE_NORMAL_FORM_AND_OPERATOR_SIGNATURE_DECISION_2026-11.md.
//
// If this test fails because a kind was added, do not extend the list here.
// Declare an operator instead. If the decision itself has to change, change
// that document first and name the reason there.
func TestValidationKindsAreClosed(t *testing.T) {
	closed := []ValidationKind{
		"positive",
		"relation-unique",
		"operator",
		"not-after",
		"no-overlap",
		"required-if",
	}
	if len(ValidationKinds) != len(closed) {
		t.Fatalf("ValidationKinds has %d kinds, the closed list has %d: %v", len(ValidationKinds), len(closed), ValidationKinds)
	}
	for i, kind := range closed {
		if ValidationKinds[i] != kind {
			t.Fatalf("ValidationKinds[%d] = %q, the closed list has %q", i, ValidationKinds[i], kind)
		}
	}
	if ValidationKind("at-most").Valid() {
		t.Fatal("a kind outside the closed list is valid")
	}
}
