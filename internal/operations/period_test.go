package operations

import (
	"testing"

	"tei/internal/database"
	"tei/internal/semantic"
)

func periodWorld() OperationSurface {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Semantics[0].Periods = []semantic.Period{{
		ID:            "sales.month",
		Data:          "sales.order",
		BusinessDate:  "sales.order.order_id",
		States:        []string{"open", "closing", "closed"},
		ClosedRejects: []string{"create", "update"},
	}}
	return DeriveOperationSurface(world, "")
}

func TestCheckPeriodsRefusesWritesWhenClosed(t *testing.T) {
	surface := periodWorld()

	rejections := CheckPeriods(surface, map[string]PeriodState{"sales.month": PeriodClosed})
	if len(rejections) != 2 {
		t.Fatalf("create and update must be refused: %+v", rejections)
	}
	for _, rejection := range rejections {
		if rejection.Period != "sales.month" || rejection.State != PeriodClosed || rejection.Detail == "" {
			t.Fatalf("unexpected rejection: %+v", rejection)
		}
	}
}

func TestCheckPeriodsAllowsWritesWhenOpen(t *testing.T) {
	surface := periodWorld()
	if rejections := CheckPeriods(surface, map[string]PeriodState{"sales.month": PeriodOpen}); len(rejections) != 0 {
		t.Fatalf("an open period refuses nothing: %+v", rejections)
	}
}

func TestCheckPeriodsRefusesWritesWhileClosing(t *testing.T) {
	surface := periodWorld()
	if rejections := CheckPeriods(surface, map[string]PeriodState{"sales.month": PeriodClosing}); len(rejections) != 2 {
		t.Fatalf("a closing period must refuse the writes it rejects: %+v", rejections)
	}
}

func TestCheckPeriodsIgnoresUnknownState(t *testing.T) {
	surface := periodWorld()
	if rejections := CheckPeriods(surface, map[string]PeriodState{}); len(rejections) != 0 {
		t.Fatalf("an unknown state must not be treated as closed: %+v", rejections)
	}
}
