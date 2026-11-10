package operations_test

import (
	"errors"
	"testing"

	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/semantic"
)

func requirementWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Periods: []semantic.Period{
			{ID: "sales.month", CloseRequires: "sales.period.close", Correction: "sales.period.correct"},
			{ID: "sales.week"},
		},
	}}}
}

// TestPeriodRequirementsReadsTheDeclaration checks that the close capability a period
// names is read. It was validated before but never read, so a period could close
// without running it (the D-9 shape).
func TestPeriodRequirementsReadsTheDeclaration(t *testing.T) {
	requirements := operations.PeriodRequirements(requirementWorld())
	month, ok := requirements["sales.month"]
	if !ok {
		t.Fatal("the declared requirement was not read")
	}
	if month.Capability != "sales.period.close" || month.Correction != "sales.period.correct" {
		t.Fatalf("requirement=%+v", month)
	}
	// A period that declares nothing is absent, so "no requirement" is an absence
	// rather than a default capability.
	if _, ok := requirements["sales.week"]; ok {
		t.Fatal("a period with no requirement was reported")
	}
}

// TestCheckCloseRequirementRefusesAnUnrunClose checks that a close which did not run
// the declared capability is refused.
func TestCheckCloseRequirementRefusesAnUnrunClose(t *testing.T) {
	requirement := operations.PeriodRequirement{
		Period:     "sales.month",
		Capability: "sales.period.close",
	}
	err := operations.CheckCloseRequirement(requirement, operations.PeriodOpen, operations.PeriodClosing, "sales.other")
	if err == nil {
		t.Fatal("a close without the declared capability was accepted")
	}
	if !errors.Is(err, operations.ErrRequirementNotRun) {
		t.Fatalf("the refusal is not the requirement error: %v", err)
	}
	// The same close with the declared capability is accepted.
	if err := operations.CheckCloseRequirement(requirement, operations.PeriodOpen, operations.PeriodClosing, "sales.period.close"); err != nil {
		t.Fatalf("a close with the declared capability was refused: %v", err)
	}
	// The step that finishes the close needs the same capability.
	if err := operations.CheckCloseRequirement(requirement, operations.PeriodClosing, operations.PeriodClosed, ""); err == nil {
		t.Fatal("finishing the close without the capability was accepted")
	}
}

// TestCheckCloseRequirementLeavesAnUndeclaredPeriodAlone checks that a period which
// declares nothing is not guarded: there is no requirement to run.
func TestCheckCloseRequirementLeavesAnUndeclaredPeriodAlone(t *testing.T) {
	if err := operations.CheckCloseRequirement(operations.PeriodRequirement{Period: "sales.week"},
		operations.PeriodOpen, operations.PeriodClosing, ""); err != nil {
		t.Fatalf("a period with no requirement was refused: %v", err)
	}
}
