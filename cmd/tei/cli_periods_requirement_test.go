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
	// The example declares no close-requires, so it is absent rather than given a
	// default capability.
	if _, ok := requirements["sales.month"]; ok {
		t.Fatalf("the example declares no requirement but one was reported: %+v", requirements)
	}
}
