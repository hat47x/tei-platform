package main

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// hookViewWorld declares a record pattern with the hooks the CLI would consult
// before a reference command.
func hookViewWorld(hooks ...application.Hook) ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: hooks,
				},
			}},
		}},
	}
}

// TestRunDeclaredHooksRunsTheBeforeChangePhases checks that the phases a
// reference run can reach, with the reads the caller supplied, are executed and
// reported in the runner order.
func TestRunDeclaredHooksRunsTheBeforeChangePhases(t *testing.T) {
	world := hookViewWorld(
		application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.status"}},
		application.Hook{Phase: "before-read"},
	)
	report, err := runDeclaredHooks(context.Background(), world, "order_entry", "order_entry.order_record",
		map[string]string{"sales.order.status": "open"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no hook report for a pattern that declares hooks")
	}
	want := []string{"before-read", "before-update"}
	if len(report.Executed) != len(want) {
		t.Fatalf("executed=%v want=%v", report.Executed, want)
	}
	for i, phase := range want {
		if report.Executed[i] != phase {
			t.Fatalf("executed=%v want=%v", report.Executed, want)
		}
	}
}

// TestRunDeclaredHooksSkipsTheAfterChangePhases checks that an after-change hook
// is not reported as executed: the reference run has no after-change point.
func TestRunDeclaredHooksSkipsTheAfterChangePhases(t *testing.T) {
	world := hookViewWorld(application.Hook{Phase: "after-update"})
	report, err := runDeclaredHooks(context.Background(), world, "order_entry", "order_entry.order_record", nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no hook report for a pattern that declares an after-change hook")
	}
	if len(report.Executed) != 0 {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "after-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if report.Reasons["after-update"] == "" {
		t.Fatalf("the skip has no stated reason: %+v", report.Reasons)
	}
}

// TestRunDeclaredHooksRecordsAMissingReadAsNotEvaluated checks that a declared
// read the caller did not supply makes the phase "not evaluated" rather than a
// pass, and that the reason names the attribute.
func TestRunDeclaredHooksRecordsAMissingReadAsNotEvaluated(t *testing.T) {
	world := hookViewWorld(application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.status"}})
	report, err := runDeclaredHooks(context.Background(), world, "order_entry", "order_entry.order_record", nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no report")
	}
	if len(report.Executed) != 0 || len(report.Warned) != 0 || report.Refused != "" {
		t.Fatalf("a missing read was reported as run: %+v", report)
	}
	if len(report.NotEvaluated) != 1 || report.NotEvaluated[0] != "before-update" {
		t.Fatalf("notEvaluated=%v", report.NotEvaluated)
	}
	if reason := report.Reasons["before-update"]; reason == "" {
		t.Fatal("the skip does not name the missing attribute")
	}
}

// TestRunDeclaredHooksRefusesARefusedPhase checks that a hook the reference rule
// refuses stops the run and names the phase.
func TestRunDeclaredHooksRefusesARefusedPhase(t *testing.T) {
	world := hookViewWorld(application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.quantity"}})
	rule := hook.ReferenceRule{Positive: []core.CanonicalID{"sales.order.quantity"}}
	report, err := runDeclaredHooksWithRule(context.Background(), world, "order_entry", "order_entry.order_record",
		map[string]string{"sales.order.quantity": "0"}, rule)
	if err == nil {
		t.Fatal("a non-positive quantity did not stop the run")
	}
	var notSatisfied *hook.NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("err=%v", err)
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if report.Outcome != "value-rejected" {
		t.Fatalf("outcome=%s", report.Outcome)
	}
}

// TestRunDeclaredHooksWithoutHooksReportsNothing checks that a pattern with no
// hook produces no report, so the run does not claim it consulted one.
func TestRunDeclaredHooksWithoutHooksReportsNothing(t *testing.T) {
	report, err := runDeclaredHooks(context.Background(), hookViewWorld(), "order_entry", "order_entry.order_record", nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report != nil {
		t.Fatalf("report=%+v", report)
	}
}

// TestRunDeclaredHooksRecordsAnUnsupportedPhase checks that a phase outside the
// runner order is recorded instead of being dropped by the loop.
func TestRunDeclaredHooksRecordsAnUnsupportedPhase(t *testing.T) {
	world := hookViewWorld(application.Hook{Phase: "during-update"})
	report, err := runDeclaredHooks(context.Background(), world, "order_entry", "order_entry.order_record", nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil || len(report.Executed) != 0 {
		t.Fatalf("report=%+v", report)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "during-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if report.Reasons["during-update"] == "" {
		t.Fatal("the unsupported phase has no stated reason")
	}
	if report.Note == "" {
		t.Fatal("the report does not state its boundary")
	}
}
