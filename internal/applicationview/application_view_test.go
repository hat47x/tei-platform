package applicationview_test

import (
	"testing"

	"tei/internal/applicationview"
	"tei/internal/semantic"
)

func TestBuildApplicationCarriesPeriods(t *testing.T) {
	world := viewWorld()
	world.Semantics[0].Periods = []semantic.Period{{
		ID:           "sales.period",
		Data:         "sales.order",
		BusinessDate: "sales.order.order_id",
		States:       []string{"open", "closed"},
	}}
	view, err := applicationview.BuildApplication(world, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Periods) != 1 || view.Periods[0].ID != "sales.period" || len(view.Periods[0].States) != 2 {
		t.Fatalf("periods=%+v", view.Periods)
	}
}
