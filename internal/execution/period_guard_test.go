package execution_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/execution"
	"tei/internal/operations"
)

type fakeInvoker struct {
	called bool
	result driver.CapabilityInvocation
	err    error
}

func (f *fakeInvoker) InvokeCapability(context.Context, core.CanonicalID, core.CanonicalID, driver.BindingAdapterResolver, any) (driver.CapabilityInvocation, error) {
	f.called = true
	return f.result, f.err
}

type fakeSource struct {
	states map[string]operations.PeriodState
	err    error
}

func (s fakeSource) PeriodStates(context.Context, string) (map[string]operations.PeriodState, error) {
	return s.states, s.err
}

func guardSurface() operations.OperationSurface {
	return operations.OperationSurface{Operations: []operations.Operation{{
		ID:            "app.create.sales.order",
		Kind:          "create",
		Target:        "sales.order",
		Binding:       "sales.order.register",
		Period:        "sales.month",
		PeriodRejects: []string{"create", "update"},
	}}}
}

func TestInvokeRefusesAClosedPeriodWithoutCallingTheDriver(t *testing.T) {
	invoker := &fakeInvoker{}
	guard := execution.PeriodGuard{Surface: guardSurface(), Source: fakeSource{states: map[string]operations.PeriodState{"sales.month": operations.PeriodClosed}}}

	_, rejection, err := execution.Invoke(context.Background(), invoker, guard, "app", "sales.order.register", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rejection == nil {
		t.Fatal("a closed period must be reported as a refusal")
	}
	if invoker.called {
		t.Fatal("a refused capability must not reach the driver")
	}
}

func TestInvokeRunsWhenThePeriodIsOpen(t *testing.T) {
	invoker := &fakeInvoker{result: driver.CapabilityInvocation{}}
	guard := execution.PeriodGuard{Surface: guardSurface(), Source: fakeSource{states: map[string]operations.PeriodState{"sales.month": operations.PeriodOpen}}}

	_, rejection, err := execution.Invoke(context.Background(), invoker, guard, "app", "sales.order.register", nil, nil)
	if err != nil || rejection != nil {
		t.Fatalf("an open period must run: rejection=%+v err=%v", rejection, err)
	}
	if !invoker.called {
		t.Fatal("the driver must be called")
	}
}

func TestInvokeWithoutASourceDoesNotGuard(t *testing.T) {
	invoker := &fakeInvoker{}
	if _, rejection, err := execution.Invoke(context.Background(), invoker, execution.PeriodGuard{Surface: guardSurface()}, "app", "sales.order.register", nil, nil); err != nil || rejection != nil {
		t.Fatalf("no source means no guard: rejection=%+v err=%v", rejection, err)
	}
	if !invoker.called {
		t.Fatal("the driver must be called")
	}
}

func TestInvokeReportsASourceErrorWithoutCallingTheDriver(t *testing.T) {
	invoker := &fakeInvoker{}
	guard := execution.PeriodGuard{Surface: guardSurface(), Source: fakeSource{err: errors.New("state unavailable")}}

	_, _, err := execution.Invoke(context.Background(), invoker, guard, "app", "sales.order.register", nil, nil)
	if err == nil {
		t.Fatal("a source error must be returned")
	}
	if invoker.called {
		t.Fatal("an unevaluated guard must not reach the driver")
	}
}
