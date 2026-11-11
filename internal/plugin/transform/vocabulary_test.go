package transform

import "testing"

// A transform value's kind is what a reader switches on, and an operation name is
// what a plan declares. Both are wire vocabularies (#4032).
func TestTransformVocabulariesAreCovered(t *testing.T) {
	t.Run("value kinds", func(t *testing.T) {
		kinds := []ValueKind{ValueNull, ValueBool, ValueNumber, ValueString, ValueArray, ValueObject, ValueBytes}
		want := []string{"null", "bool", "number", "string", "array", "object", "bytes"}
		if len(kinds) != len(want) {
			t.Fatalf("a value kind was dropped: %d values", len(kinds))
		}
		seen := map[string]bool{}
		for i, kind := range kinds {
			if string(kind) != want[i] {
				t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
			}
			if seen[string(kind)] {
				t.Fatalf("value kind %q is declared twice", kind)
			}
			seen[string(kind)] = true
		}
	})

	t.Run("operations", func(t *testing.T) {
		operations := []Op{
			OpInput, OpLiteral, OpSelectField, OpConstructObject, OpMapArray,
			OpAssertEach, OpSortArrayBy, OpReplaceField, OpOmitField, OpAssertEqual, OpAssertEnum,
		}
		want := []string{
			"input", "literal", "select-field", "construct-object", "map-array",
			"assert-each", "sort-array-by", "replace-field", "omit-field", "assert-equal", "assert-enum",
		}
		if len(operations) != len(want) {
			t.Fatalf("an operation was dropped: %d values", len(operations))
		}
		seen := map[string]bool{}
		for i, operation := range operations {
			if string(operation) != want[i] {
				t.Fatalf("operations[%d]=%q want %q", i, operation, want[i])
			}
			if seen[string(operation)] {
				t.Fatalf("operation %q is declared twice", operation)
			}
			seen[string(operation)] = true
		}
	})
}
