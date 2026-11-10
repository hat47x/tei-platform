package main

import (
	"testing"

	"tei/internal/operations"
)

// TestAdvancePeriodMovesOneStep checks that advancing goes one place at a time and
// treats an unstated period as open, so an operator does not have to open a period
// before closing it.
func TestAdvancePeriodMovesOneStep(t *testing.T) {
	next, err := advancePeriod("sales.month", "", "/tmp/state.json")
	if err != nil {
		t.Fatal(err)
	}
	if next != operations.PeriodClosing {
		t.Fatalf("next=%s", next)
	}
	next, err = advancePeriod("sales.month", operations.PeriodClosing, "/tmp/state.json")
	if err != nil {
		t.Fatal(err)
	}
	if next != operations.PeriodClosed {
		t.Fatalf("next=%s", next)
	}
}

// TestAdvancePeriodRefusesAClosedPeriod checks that a closed period has no further
// step, so a repeated advance is not mistaken for a fresh closing.
func TestAdvancePeriodRefusesAClosedPeriod(t *testing.T) {
	if _, err := advancePeriod("sales.month", operations.PeriodClosed, "/tmp/state.json"); err == nil {
		t.Fatal("a closed period was advanced")
	}
}

// TestAdvancePeriodRequiresAFile checks that advancing needs a file to write, so the
// new state is kept rather than forgotten.
func TestAdvancePeriodRequiresAFile(t *testing.T) {
	if _, err := advancePeriod("sales.month", operations.PeriodOpen, ""); err == nil {
		t.Fatal("advancing without a state file was accepted")
	}
	if _, err := advancePeriod("", operations.PeriodOpen, "/tmp/state.json"); err == nil {
		t.Fatal("advancing without a period was accepted")
	}
}
