package command

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func splitPeriodWorld(closingDay int) ir.World {
	world := commandWorld()
	// A split period needs a business date input, so the fixture declares one and
	// binds it to the pattern.
	world.Semantics[0].Data[0].Attributes = append(world.Semantics[0].Data[0].Attributes,
		semantic.Attribute{ID: "sales.order.order_date", TypeRef: "std.date"})
	world.Semantics[0].Capabilities[0].Inputs = append(world.Semantics[0].Capabilities[0].Inputs,
		semantic.Parameter{Name: "order_date", Type: "std.date"})
	world.Applications[0].Patterns[0].Interaction.Commands[0].Inputs = append(
		world.Applications[0].Patterns[0].Interaction.Commands[0].Inputs,
		application.CommandInputBinding{Name: "order_date", Attribute: "sales.order.order_date"})
	world.Semantics[0].Periods = []semantic.Period{{
		ID:            "sales.period",
		Data:          "sales.order",
		BusinessDate:  "sales.order.order_date",
		States:        []string{"open", "closed"},
		ClosedRejects: []string{"update"},
		ClosingDay:    closingDay,
	}}
	return world
}

// halfState is a period state that understands halves, so the scoped path can be
// exercised.
type halfState struct {
	closed map[string]bool
}

func (s halfState) Closed(_ context.Context, periodID core.CanonicalID) (bool, error) {
	return s.closed[string(periodID)], nil
}

func (s halfState) ClosedScope(_ context.Context, scope string) (bool, error) {
	return s.closed[scope], nil
}

func splitRequest(date string) Request {
	return Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.submit",
		Inputs:      map[string]string{"order": "ORD-1", "order_date": date},
	}
}

// TestInvokeClosesOneHalfOfASplitMonth checks that closing the first half refuses
// a command whose business date is in that half, and allows one in the second.
// X-TEA read one month sequence for both halves (getMSeqOfDate,
// D:Session.java L2471–2484), so the two dates were the same period.
func TestInvokeClosesOneHalfOfASplitMonth(t *testing.T) {
	state := halfState{closed: map[string]bool{"sales.period#1": true}}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      splitPeriodWorld(20),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		Periods:    state,
	}
	_, err := invoker.Invoke(context.Background(), splitRequest("2026-05-20"))
	if code := executionCode(t, err); code != "COMMAND_PERIOD_CLOSED" {
		t.Fatalf("code=%s", code)
	}
	// The same month, the other half: allowed, because that half is open.
	if _, err := invoker.Invoke(context.Background(), splitRequest("2026-05-21")); err != nil {
		t.Fatalf("the second half was refused: %v", err)
	}
}

// monthOnlyState implements only the base PeriodState, so it cannot answer for a
// half month.
type monthOnlyState struct{}

func (monthOnlyState) Closed(context.Context, core.CanonicalID) (bool, error) { return false, nil }

// TestInvokeRefusesASplitPeriodWithoutHalfState checks that a state which only knows
// whole months does not answer for a half. Reading the month's closure as the
// half's closure would allow or refuse the wrong half.
func TestInvokeRefusesASplitPeriodWithoutHalfState(t *testing.T) {
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      splitPeriodWorld(20),
		Executor:   fakeExecutor{},
		Periods:    monthOnlyState{},
	}
	_, err := invoker.Invoke(context.Background(), splitRequest("2026-05-21"))
	if code := executionCode(t, err); code != "COMMAND_PERIOD_SCOPE_UNAVAILABLE" {
		t.Fatalf("code=%s", code)
	}
}

// TestMemoryPeriodStateAnswersForHalfMonths checks that the reference state can be
// told a half's closure, so a caller can drive the split path without a provider.
func TestMemoryPeriodStateAnswersForHalfMonths(t *testing.T) {
	state := NewMemoryPeriodState()
	state.SetScopeClosed("sales.period#1", true)
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      splitPeriodWorld(20),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		Periods:    state,
	}
	_, err := invoker.Invoke(context.Background(), splitRequest("2026-05-20"))
	if code := executionCode(t, err); code != "COMMAND_PERIOD_CLOSED" {
		t.Fatalf("code=%s", code)
	}
	if _, err := invoker.Invoke(context.Background(), splitRequest("2026-05-21")); err != nil {
		t.Fatalf("the open half was refused: %v", err)
	}
}

// TestInvokeRefusesASplitPeriodWhoseDateIsNotADate checks that a split period
// whose business date does not read as a date is refused rather than read as one
// half. The fixture binds the order identifier to the date, which is the shape a
// wrong declaration takes.
func TestInvokeRefusesASplitPeriodWhoseDateIsNotADate(t *testing.T) {
	world := splitPeriodWorld(20)
	world.Semantics[0].Periods[0].BusinessDate = "sales.order.order_id"
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   fakeExecutor{},
		Periods:    halfState{closed: map[string]bool{}},
	}
	_, err := invoker.Invoke(context.Background(), splitRequest("2026-05-21"))
	if code := executionCode(t, err); code != "COMMAND_PERIOD_SCOPE_UNKNOWN" {
		t.Fatalf("code=%s", code)
	}
}

// TestInvokeKeepsWholeMonthPeriodsUnchanged checks that a period without a closing
// day still asks the state by period ID, so an existing store keeps working.
func TestInvokeKeepsWholeMonthPeriodsUnchanged(t *testing.T) {
	state := halfState{closed: map[string]bool{"sales.period": true}}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      splitPeriodWorld(0),
		Executor:   fakeExecutor{},
		Periods:    state,
	}
	_, err := invoker.Invoke(context.Background(), splitRequest("2026-05-21"))
	if code := executionCode(t, err); code != "COMMAND_PERIOD_CLOSED" {
		t.Fatalf("code=%s", code)
	}
}
