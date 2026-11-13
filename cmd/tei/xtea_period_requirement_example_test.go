package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/workspace"
)

// この試験は、例のWorkspaceが期間の close-requires と correction を宣言できることを
// 確かめます。
//
// これまで `tei view` はこの2つを投影していましたが、例は宣言していませんでした。
// 宣言が無いので、投影の試験は合成の世界だけを扱い、例では一度も動いていません。
// ここでは例そのものへ宣言を足し、validate・view・運用の3つが同じ宣言を見ることを
// 確かめます。

const exampleRoot = "../../examples/xtea-lowcode"

// TestExampleDeclaresThePeriodRequirements checks that the example workspace names
// the capability a close must run and the one a reopen must run, and that both
// resolve.
func TestExampleDeclaresThePeriodRequirements(t *testing.T) {
	if _, err := os.Stat(filepath.Join(exampleRoot, "semantic", "sales")); err != nil {
		t.Skip("the example workspace is not present")
	}
	loaded, err := workspace.Load(exampleRoot)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	requirements := operations.PeriodRequirements(loaded.World)
	month, ok := requirements["sales.month"]
	if !ok {
		t.Fatal("the example period declares no requirement")
	}
	if month.Capability != "sales.close_period" {
		t.Fatalf("close-requires=%q", month.Capability)
	}
	if month.Correction != "sales.correct_order" {
		t.Fatalf("correction=%q", month.Correction)
	}
}

// TestExampleProjectsThePeriodRequirements checks that `tei view` reads the same
// declaration the operations path reads, so a reader of the view sees the
// requirement the CLI enforces.
func TestExampleProjectsThePeriodRequirements(t *testing.T) {
	if _, err := os.Stat(filepath.Join(exampleRoot, "semantic", "sales")); err != nil {
		t.Skip("the example workspace is not present")
	}
	var out bytes.Buffer
	if code, err := runViewCommand([]string{"-workspace", exampleRoot, "-application", "order_entry", "-format", "text"}, &out); code != 0 || err != nil {
		t.Fatalf("view failed: code=%d err=%v\n%s", code, err, out.String())
	}
	text := out.String()
	if !strings.Contains(text, "period sales.month close-requires sales.close_period") {
		t.Fatalf("the view does not print close-requires:\n%s", text)
	}
	if !strings.Contains(text, "period sales.month correction via sales.correct_order") {
		t.Fatalf("the view does not print correction:\n%s", text)
	}
}

// TestExampleCloseIsRefusedWithoutTheDeclaredCapability checks that the declaration
// the example makes is enforced by the operations path: advancing the period without
// the declared capability is refused.
func TestExampleCloseIsRefusedWithoutTheDeclaredCapability(t *testing.T) {
	if _, err := os.Stat(filepath.Join(exampleRoot, "semantic", "sales")); err != nil {
		t.Skip("the example workspace is not present")
	}
	loaded, err := workspace.Load(exampleRoot)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	requirement, ok := operations.PeriodRequirements(loaded.World)["sales.month"]
	if !ok {
		t.Fatal("the example period declares no requirement")
	}
	// The CLI ran nothing, so the close is refused.
	err = operations.CheckCloseRequirement(requirement, operations.PeriodOpen, operations.PeriodClosing, "")
	if err == nil {
		t.Fatal("the example period closed without its declared capability")
	}
	// The same close with the declared capability is accepted.
	if err := operations.CheckCloseRequirement(requirement, operations.PeriodOpen, operations.PeriodClosing, "sales.close_period"); err != nil {
		t.Fatalf("the declared capability was refused: %v", err)
	}
	// And a reopen needs the correction capability, not the close one.
	if err := operations.CheckReopenRequirement(requirement, operations.PeriodClosed, operations.PeriodOpen, "sales.close_period"); err == nil {
		t.Fatal("a reopen through the close capability was accepted")
	}
	if err := operations.CheckReopenRequirement(requirement, operations.PeriodClosed, operations.PeriodOpen, "sales.correct_order"); err != nil {
		t.Fatalf("the declared correction capability was refused: %v", err)
	}
}
