package command

import (
	"context"
	"testing"

	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/semantic"
)

func periodWorld(closedRejects []string) ir.World {
	world := commandWorld()
	world.Semantics[0].Periods = []semantic.Period{{
		ID:            "sales.period",
		Data:          "sales.order",
		BusinessDate:  "sales.order.order_id",
		States:        []string{"open", "closed"},
		ClosedRejects: closedRejects,
	}}
	return world
}

func TestInvokeRejectsClosedPeriod(t *testing.T) {
	state := NewMemoryPeriodState()
	state.SetClosed("sales.period", true)
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      periodWorld([]string{"update"}),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		Periods:    state,
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_PERIOD_CLOSED" {
		t.Fatalf("code=%s", code)
	}
}

func TestInvokeAllowsClosedPeriodWhenEffectIsNotRejected(t *testing.T) {
	state := NewMemoryPeriodState()
	state.SetClosed("sales.period", true)
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      periodWorld([]string{"delete"}),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		Periods:    state,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
}

func TestInvokeReportsPeriodStateFailure(t *testing.T) {
	state := NewMemoryPeriodState()
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      periodWorld([]string{"update"}),
		Executor:   fakeExecutor{},
		Periods:    state,
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_PERIOD_STATE_FAILED" {
		t.Fatalf("code=%s", code)
	}
}

func TestMapPeriodState(t *testing.T) {
	states := NewMapPeriodState(map[string]operations.PeriodState{
		"sales.period": operations.PeriodClosed,
		"other.period": operations.PeriodOpen,
	})
	closed, err := states.Closed(context.Background(), "sales.period")
	if err != nil || !closed {
		t.Fatalf("closed=%t err=%v", closed, err)
	}
	closed, err = states.Closed(context.Background(), "other.period")
	if err != nil || closed {
		t.Fatalf("closed=%t err=%v", closed, err)
	}
	closed, err = states.Closed(context.Background(), "missing.period")
	if err != nil || closed {
		t.Fatalf("closed=%t err=%v", closed, err)
	}
}
