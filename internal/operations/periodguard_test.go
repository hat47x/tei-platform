package operations

import (
	"context"
	"testing"

	"tei/internal/core"
)

func guardSurface() OperationSurface {
	return OperationSurface{Operations: []Operation{
		{
			ID:            "app.create.sales.order",
			Kind:          "create",
			Target:        "sales.order",
			Binding:       "sales.order.register",
			Period:        "sales.month",
			PeriodRejects: []string{"create", "update"},
		},
		{ID: "app.list.sales.note", Kind: "list", Target: "sales.note", Binding: "sales.note.list"},
	}}
}

func TestGuardCapabilityRefusesAClosedWrite(t *testing.T) {
	rejection, refused := GuardCapability(guardSurface(), map[string]PeriodState{"sales.month": PeriodClosed}, core.MustCanonicalID("sales.order.register"))
	if !refused {
		t.Fatal("a closed period must refuse the bound capability")
	}
	if rejection.Period != "sales.month" || rejection.State != PeriodClosed {
		t.Fatalf("unexpected rejection: %+v", rejection)
	}
}

func TestGuardCapabilityAllowsAnOpenPeriod(t *testing.T) {
	if _, refused := GuardCapability(guardSurface(), map[string]PeriodState{"sales.month": PeriodOpen}, core.MustCanonicalID("sales.order.register")); refused {
		t.Fatal("an open period must not refuse")
	}
}

func TestGuardCapabilityIgnoresCapabilitiesWithoutAPeriod(t *testing.T) {
	if _, refused := GuardCapability(guardSurface(), map[string]PeriodState{"sales.month": PeriodClosed}, core.MustCanonicalID("sales.note.list")); refused {
		t.Fatal("an operation without a period is not guarded")
	}
}

type fixedSource struct {
	states map[string]PeriodState
	err    error
}

func (s fixedSource) PeriodStates(context.Context, string) (map[string]PeriodState, error) {
	return s.states, s.err
}

func TestGuardCapabilityWithAsksTheSource(t *testing.T) {
	surface := guardSurface()
	_, refused, err := GuardCapabilityWith(context.Background(), fixedSource{states: map[string]PeriodState{"sales.month": PeriodClosed}}, surface, "app", core.MustCanonicalID("sales.order.register"))
	if err != nil {
		t.Fatalf("source error: %v", err)
	}
	if !refused {
		t.Fatal("a closed period from the source must refuse")
	}

	_, refused, err = GuardCapabilityWith(context.Background(), fixedSource{}, surface, "app", core.MustCanonicalID("sales.order.register"))
	if err != nil || refused {
		t.Fatalf("a source with no state must not refuse: refused=%v err=%v", refused, err)
	}
}

func TestGuardCapabilityWithReportsASourceError(t *testing.T) {
	_, _, err := GuardCapabilityWith(context.Background(), fixedSource{err: context.DeadlineExceeded}, guardSurface(), "app", core.MustCanonicalID("sales.order.register"))
	if err == nil {
		t.Fatal("a source error must be returned as an error, not as a refusal")
	}
}

func TestGuardCapabilityWithWithoutASourceDoesNothing(t *testing.T) {
	if _, refused, err := GuardCapabilityWith(context.Background(), nil, guardSurface(), "app", core.MustCanonicalID("sales.order.register")); err != nil || refused {
		t.Fatalf("no source means no guard: refused=%v err=%v", refused, err)
	}
}
