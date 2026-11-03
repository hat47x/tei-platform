package driver_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/operations"
)

func guardSurface() operations.OperationSurface {
	return operations.OperationSurface{Operations: []operations.Operation{
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

func TestGuardPeriodsRefusesAClosedWrite(t *testing.T) {
	rejection, refused := driver.GuardPeriods(guardSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodClosed}, "sales.order.register")
	if !refused {
		t.Fatal("a closed period must refuse the bound capability")
	}
	if rejection.Period != "sales.month" || rejection.State != operations.PeriodClosed {
		t.Fatalf("unexpected rejection: %+v", rejection)
	}
}

func TestGuardPeriodsAllowsAnOpenPeriod(t *testing.T) {
	if _, refused := driver.GuardPeriods(guardSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodOpen}, "sales.order.register"); refused {
		t.Fatal("an open period must not refuse")
	}
}

func TestGuardPeriodsIgnoresCapabilitiesWithoutAPeriod(t *testing.T) {
	if _, refused := driver.GuardPeriods(guardSurface(), map[string]operations.PeriodState{"sales.month": operations.PeriodClosed}, core.MustCanonicalID("sales.note.list")); refused {
		t.Fatal("an operation without a period is not guarded")
	}
}
