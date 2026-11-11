package applicationview_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// periodRequirementViewWorld is an application whose period declares both the
// capability a close must run and the one a reopen must run.
func periodRequirementViewWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
		}},
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Periods: []semantic.Period{{
				ID:            "sales.month",
				Data:          "sales.order",
				BusinessDate:  "sales.order.order_date",
				States:        []string{"open", "closing", "closed"},
				CloseRequires: "sales.period.close",
				Correction:    "sales.period.correct",
			}},
		}},
	}
}

// TestApplicationViewProjectsThePeriodRequirements checks that the capabilities a
// period requires are visible in `tei view`. Both were carried in the Command plan
// but not in the application view, so a reader of the view could not see which
// capability a close or a reopen needs — the Command boundary knew and the
// projection did not.
func TestApplicationViewProjectsThePeriodRequirements(t *testing.T) {
	view, err := applicationview.BuildApplication(periodRequirementViewWorld(), "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Periods) != 1 {
		t.Fatalf("periods=%+v", view.Periods)
	}
	period := view.Periods[0]
	if period.CloseRequires != "sales.period.close" {
		t.Fatalf("closeRequires=%q", period.CloseRequires)
	}
	if period.Correction != "sales.period.correct" {
		t.Fatalf("correction=%q", period.Correction)
	}
}

// TestApplicationViewLeavesAnUndeclaredRequirementEmpty checks the other direction:
// a period that declares neither leaves both fields empty, so a reader sees "no
// requirement" as an absence rather than a default capability.
func TestApplicationViewLeavesAnUndeclaredRequirementEmpty(t *testing.T) {
	world := periodRequirementViewWorld()
	world.Semantics[0].Periods[0].CloseRequires = ""
	world.Semantics[0].Periods[0].Correction = ""
	view, err := applicationview.BuildApplication(world, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if view.Periods[0].CloseRequires != "" || view.Periods[0].Correction != "" {
		t.Fatalf("a period with no requirement was given one: %+v", view.Periods[0])
	}
}
