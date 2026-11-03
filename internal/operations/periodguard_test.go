package operations

import (
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
