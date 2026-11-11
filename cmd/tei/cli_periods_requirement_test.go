package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/operations"
)

// TestAdvanceRefusesAMissingCloseCapability checks that a period which declares
// close-requires is not advanced by the CLI unless the close capability is named.
//
// The declaration was validated (the capability must exist) but never read when the
// period closed, so a period could name a close capability and close without it
// (D-9: a declared value inert at run time).
func TestAdvanceRefusesAMissingCloseCapability(t *testing.T) {
	requirement := operations.PeriodRequirement{
		Period:     "sales.month",
		Capability: "sales.period.close",
	}
	// The CLI ran no capability, so the close is refused.
	if err := operations.CheckCloseRequirement(requirement, operations.PeriodOpen, operations.PeriodClosing, ""); err == nil {
		t.Fatal("a close without the declared capability was accepted")
	}
	// Once the caller names the capability it ran, the close goes ahead.
	if err := operations.CheckCloseRequirement(requirement, operations.PeriodOpen, operations.PeriodClosing, "sales.period.close"); err != nil {
		t.Fatalf("a close with the declared capability was refused: %v", err)
	}
}

// TestPeriodsCommandReadsRequirementsFromTheWorkspace checks the workspace lookup the
// command uses, so a period that declares nothing is not guarded.
func TestPeriodsCommandReadsRequirementsFromTheWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "xtea-lowcode")
	if _, err := os.Stat(root); err != nil {
		t.Skip("the example workspace is not present")
	}
	requirements, err := operations.PeriodRequirementsFromWorkspace(root, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	// The example now declares both requirements, so the lookup sees them. This
	// test asserted the opposite before the example declared them; the assertion
	// moved with the declaration rather than the declaration being held back to
	// keep an old expectation true.
	month, ok := requirements["sales.month"]
	if !ok {
		t.Fatalf("the example declares a requirement but the lookup lost it: %+v", requirements)
	}
	if month.Capability != "sales.close_period" || month.Correction != "sales.correct_order" {
		t.Fatalf("requirement=%+v", month)
	}
	// A period the example does not declare stays absent: the lookup reports what
	// the workspace says, not a default for every period.
	if _, ok := requirements["sales.week"]; ok {
		t.Fatalf("a period the example does not declare was reported: %+v", requirements)
	}
}
