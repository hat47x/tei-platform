package execution_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/effectlog"
	"tei/internal/execution"
	"tei/internal/operations"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/runlog"
	"tei/internal/semantic"
)

// allowObserver confirms any effect; nil observer means nothing is observed.
type allowObserver struct{}

func (allowObserver) ObserveEffect(context.Context, effectlog.Record) (bool, string, error) {
	return true, "observed", nil
}

func allPreconditionSurface() operations.OperationSurface {
	return operations.OperationSurface{Operations: []operations.Operation{{
		ID:            "app.create.sales.order",
		Kind:          "create",
		Target:        "sales.order",
		Binding:       "sales.close",
		Period:        "s.month",
		PeriodRejects: []string{"create"},
	}}}
}

func allPreconditionRules() []semantic.EffectRule {
	return []semantic.EffectRule{{
		ID:            "s.order.stock",
		OnCapability:  "sales.close",
		Stage:         "after-commit",
		EffectKind:    "update",
		EffectData:    "s.order",
		ViaCapability: "sales.close",
	}}
}

func allPreconditionCatalog() plugin.Catalog {
	return plugin.Catalog{Plugins: []plugin.Definition{{
		ID:      "tei.demo",
		Version: "1.0.0",
		Provides: []plugin.Capability{{
			ID:          "sales.close",
			Permissions: []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}},
		}},
	}}}
}

func allPreconditionPlan() semantic.RunPlan {
	return semantic.RunPlan{ID: "sales.close_run", Operations: []core.CanonicalID{"sales.close"}, Autonomy: "unattended"}
}

func combinedPreconditions(period operations.PeriodState, observer execution.EffectObserver, granted []permission.Spec) execution.PreconditionFunc {
	return execution.CombinePreconditions(
		execution.PeriodPreconditions(allPreconditionSurface(), map[string]operations.PeriodState{"s.month": period}),
		execution.EffectPreconditions(allPreconditionRules(), observer),
		execution.PermissionPreconditions(execution.RequirementsFromCatalog(allPreconditionCatalog()), granted),
	)
}

func runWithPreconditions(t *testing.T, preconditions execution.PreconditionFunc) ([]execution.StepResult, runlog.Record, bool) {
	t.Helper()
	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	_, results, err := execution.RunPlan(context.Background(), allPreconditionPlan(), []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return results, journal.records[len(journal.records)-1], called
}

func TestRunPlanBlocksOnThePeriodEvenWhenEverythingElseIsReady(t *testing.T) {
	granted := []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}}
	results, last, called := runWithPreconditions(t, combinedPreconditions(operations.PeriodClosed, allowObserver{}, granted))
	if called || len(results) != 1 || results[0].Status != "blocked" || !strings.Contains(results[0].Detail, "closed") {
		t.Fatalf("the closed period must block: %+v", results)
	}
	if last.Status != runlog.Blocked {
		t.Fatalf("the run must be recorded blocked: %+v", last)
	}
}

func TestRunPlanBlocksOnAnUnobservableEffect(t *testing.T) {
	granted := []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}}
	results, _, called := runWithPreconditions(t, combinedPreconditions(operations.PeriodOpen, nil, granted))
	if called || len(results) != 1 || results[0].Status != "blocked" || !strings.Contains(results[0].Detail, "s.order.stock") {
		t.Fatalf("the unobservable effect must block: %+v", results)
	}
}

func TestRunPlanBlocksOnAnUngrantedPermission(t *testing.T) {
	results, _, called := runWithPreconditions(t, combinedPreconditions(operations.PeriodOpen, allowObserver{}, nil))
	if called || len(results) != 1 || results[0].Status != "blocked" || !strings.Contains(results[0].Detail, "db:5432") {
		t.Fatalf("the ungranted permission must block: %+v", results)
	}
}

func TestRunPlanRunsWhenEveryPreconditionIsMet(t *testing.T) {
	granted := []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}}
	results, last, called := runWithPreconditions(t, combinedPreconditions(operations.PeriodOpen, allowObserver{}, granted))
	if !called || len(results) != 1 || results[0].Status != "resolved" {
		t.Fatalf("every precondition is met, so the step must run: %+v", results)
	}
	if last.Status != runlog.Completed {
		t.Fatalf("the run must complete: %+v", last)
	}
}
