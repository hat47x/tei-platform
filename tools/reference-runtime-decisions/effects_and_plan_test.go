package decisions

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/runlog"
	"tei/internal/semantic"
)

// updateStep writes the probe row and reports how many times it ran.
func updateStep(t *testing.T, conn *pgx.Conn, calls *int) execution.StepFunc {
	return func(ctx context.Context, operation core.CanonicalID) execution.StepResult {
		*calls++
		if _, err := conn.Exec(ctx, "UPDATE tei_decisions_probe SET status = $1 WHERE id = 1", "closed"); err != nil {
			return execution.StepResult{Operation: string(operation), Status: "unresolved", Detail: err.Error()}
		}
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
}

func lateEffectRule() []semantic.EffectRule {
	return []semantic.EffectRule{{
		ID:            "s.order.stock",
		OnCapability:  "sales.close",
		Stage:         "after-commit",
		EffectKind:    "update",
		EffectData:    "s.order",
		ViaCapability: "sales.close",
	}}
}

func TestAnUnobservableEffectBlocksBeforeTheWrite(t *testing.T) {
	conn, closeConn := probe(t)
	defer closeConn()

	calls := 0
	preconditions := execution.CombinePreconditions(execution.EffectPreconditions(lateEffectRule(), nil))
	_, results, err := execution.RunPlan(context.Background(), probePlan(), []string{"2026-10"}, &memoryJournal{}, updateStep(t, conn, &calls), preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if calls != 0 {
		t.Fatalf("an unobservable effect must stop before the write: calls=%d", calls)
	}
	if len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the effect must block the run: %+v", results)
	}
	if got := status(t, conn); got != "new" {
		t.Fatalf("a blocked run must not change the row: %q", got)
	}
}

func TestACompletedRunIsRefusedWithoutASecondWrite(t *testing.T) {
	conn, closeConn := probe(t)
	defer closeConn()

	journal := &memoryJournal{}
	firstCalls := 0
	if _, results, err := execution.RunPlan(context.Background(), probePlan(), []string{"2026-10"}, journal, updateStep(t, conn, &firstCalls), nil); err != nil || len(results) != 1 || results[0].Status != "resolved" {
		t.Fatalf("the first run must succeed: results=%+v err=%v", results, err)
	}
	if got := status(t, conn); got != "closed" {
		t.Fatalf("the first run must change the row: %q", got)
	}

	secondCalls := 0
	decision, results, err := execution.RunPlan(context.Background(), probePlan(), []string{"2026-10"}, journal, updateStep(t, conn, &secondCalls), nil)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if decision.Action != runlog.Refuse {
		t.Fatalf("a completed run must be refused: %+v", decision)
	}
	if secondCalls != 0 || len(results) != 0 {
		t.Fatalf("a refused run must not write: calls=%d results=%+v", secondCalls, results)
	}
	if got := status(t, conn); got != "closed" {
		t.Fatalf("the row must keep the first run's value: %q", got)
	}
}
