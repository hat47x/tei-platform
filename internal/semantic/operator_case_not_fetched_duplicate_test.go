package semantic_test

import (
	"testing"

	"tei/internal/semantic"
)

// A cell nobody read states a word, which is not a decimal, so two such cells
// state the same example when they are the same word. A decimal and the word are
// different examples, because one states a number and the other states that no
// number was read.
//
// The expectations are drawn from section 8.3's table, which fixes the writing
// per state rather than per type, not from the code.
func TestTwoNotReadCellsStateTheSameExample(t *testing.T) {
	left := semantic.OperatorCase{Inputs: map[string]string{
		"value": "not-fetched", "limit": "not-fetched"}}
	right := semantic.OperatorCase{Inputs: map[string]string{
		"value": "not-fetched", "limit": "not-fetched"}}
	if !semantic.SameOperatorCaseStates(left, right) {
		t.Fatal("the same not-read cells were read as different examples")
	}
}

// A not-read cell and a decimal cell are different examples, so a table that
// states one of each is not a table that repeats itself.
func TestANotReadCellAndADecimalAreDifferentExamples(t *testing.T) {
	left := semantic.OperatorCase{Inputs: map[string]string{"value": "5", "limit": "10"}}
	right := semantic.OperatorCase{Inputs: map[string]string{"value": "not-fetched", "limit": "10"}}
	if semantic.SameOperatorCaseStates(left, right) {
		t.Fatal("a decimal and a not-read cell were read as the same example")
	}
}

// The presence words keep their own comparison: a not-read cell is not the same
// example as either of them, because it states a third thing.
func TestANotReadCellIsNotTheSameExampleAsAPresenceWord(t *testing.T) {
	for _, word := range []string{semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent} {
		left := semantic.OperatorCase{Inputs: map[string]string{"value": word}}
		right := semantic.OperatorCase{Inputs: map[string]string{"value": "not-fetched"}}
		if semantic.SameOperatorCaseStates(left, right) {
			t.Fatalf("the presence word %q was read as the same example as a not-read cell", word)
		}
	}
}
