package execution_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/operations"
	"tei/internal/runlog"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// writeRunPlanWorkspace builds a small workspace whose capability is bound to a
// declared period and to an effect that runs outside the transaction.
func writeRunPlanWorkspace(t *testing.T, withLateEffect bool) (string, *semantic.Namespace) {
	t.Helper()
	root := t.TempDir()
	write := func(path, content string) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: w\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - databases\n")
	rules := ""
	if withLateEffect {
		rules = `
effect-rule s.order.stock {
    on capability s.order.close
    stage after-commit
    effect update s.order
    via capability s.order.close
}
`
	}
	write("semantic/s/x.tei", `namespace s

data Order {
    attribute order_date type std.date
    attribute status type std.code
}

capability s.order.close {
    input order s.order
    outcome closed s.order
    effect update s.order
}

period s.month {
    data s.order
    business-date s.order.order_date
    states open closing closed
    closed-rejects create update
}
`+rules)
	write("applications/app/application.tei", `application app {
    uses semantic s

    database main {
        target main_db
        access read-write
    }
}
`)
	write("databases/main-db/database.tei", "database main_db {\n    kind relational\n    ownership managed-here\n    uses semantic s\n}\n")
	write("databases/main-db/persistence/order.tei", `persist main_db.order {
    database main_db
    semantic s.order
    table public.orders

    column s.order.order_date -> order_date storage date required
    column s.order.status -> status storage varchar required

    primary-key s.order.order_date
}
`)

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for i := range loaded.World.Semantics {
		if loaded.World.Semantics[i].ID == "s" {
			return root, &loaded.World.Semantics[i]
		}
	}
	t.Fatal("the semantic namespace is missing")
	return "", nil
}

func TestRunPlanStopsWhenTheWorkspacePeriodIsClosed(t *testing.T) {
	root, _ := writeRunPlanWorkspace(t, false)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	surface := operations.DeriveOperationSurface(loaded.World, "app")
	if len(surface.Operations) == 0 {
		t.Fatalf("the surface derived nothing: %+v", surface.Gaps)
	}

	plan := semantic.RunPlan{ID: "s.monthly_close", Operations: []core.CanonicalID{"s.order.close"}, Autonomy: "unattended"}
	preconditions := execution.CombinePreconditions(
		execution.PeriodPreconditions(surface, map[string]operations.PeriodState{"s.month": operations.PeriodClosed}),
		execution.EffectPreconditions(nil, nil),
	)

	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if called || len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the closed period must block the plan: called=%v results=%+v", called, results)
	}
	if journal.records[len(journal.records)-1].Status != runlog.Blocked {
		t.Fatalf("the run must be recorded blocked: %+v", journal.records)
	}
}

func TestRunPlanCompletesWhenTheWorkspacePeriodIsOpen(t *testing.T) {
	root, _ := writeRunPlanWorkspace(t, false)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	surface := operations.DeriveOperationSurface(loaded.World, "app")
	plan := semantic.RunPlan{ID: "s.monthly_close", Operations: []core.CanonicalID{"s.order.close"}, Autonomy: "unattended"}
	preconditions := execution.CombinePreconditions(
		execution.PeriodPreconditions(surface, map[string]operations.PeriodState{"s.month": operations.PeriodOpen}),
		execution.EffectPreconditions(nil, nil),
	)
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 1 || results[0].Status != "resolved" {
		t.Fatalf("an open period must let the plan run: %+v", results)
	}
	if journal.records[len(journal.records)-1].Status != runlog.Completed {
		t.Fatalf("the run must complete: %+v", journal.records)
	}
}

func TestRunPlanStopsWhenTheWorkspaceEffectIsUnobservable(t *testing.T) {
	root, namespace := writeRunPlanWorkspace(t, true)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	surface := operations.DeriveOperationSurface(loaded.World, "app")
	rules := namespace.EffectRules
	if len(rules) != 1 {
		t.Fatalf("the workspace must declare one effect rule: %+v", rules)
	}

	plan := semantic.RunPlan{ID: "s.monthly_close", Operations: []core.CanonicalID{"s.order.close"}, Autonomy: "unattended"}
	preconditions := execution.CombinePreconditions(
		execution.PeriodPreconditions(surface, map[string]operations.PeriodState{"s.month": operations.PeriodOpen}),
		execution.EffectPreconditions(rules, nil),
	)
	journal := &runJournal{}
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("an unobservable effect must block the plan: %+v", results)
	}
	if !strings.Contains(results[0].Detail, "s.order.stock") {
		t.Fatalf("the reason must name the effect rule: %q", results[0].Detail)
	}
}
