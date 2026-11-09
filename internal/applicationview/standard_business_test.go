package applicationview_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// standardBusinessWorld declares one Data per binder kind, each carrying the
// roles its binder reads.
func standardBusinessWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "master",
				Data: []semantic.Data{
					{
						ID: "master.tax_rate",
						Attributes: []semantic.Attribute{
							{ID: "master.tax_rate.applies_from", TypeRef: "std.date", Role: "tax-applies-from"},
							{ID: "master.tax_rate.applies_until", TypeRef: "std.date", Role: "tax-applies-until"},
							{ID: "master.tax_rate.rate", TypeRef: "std.decimal", Role: "tax-value"},
							{ID: "master.tax_rate.rounding", TypeRef: "std.text", Role: "tax-rounding"},
							{ID: "master.tax_rate.category", TypeRef: "std.code", Role: "tax-category"},
						},
					},
					{
						ID: "master.holiday",
						Attributes: []semantic.Attribute{
							{ID: "master.holiday.holiday_on", TypeRef: "std.date", Role: "holiday-date"},
						},
					},
					{
						ID: "master.setting",
						Attributes: []semantic.Attribute{
							{ID: "master.setting.key", TypeRef: "std.identifier", Role: "setting-key"},
							{ID: "master.setting.value", TypeRef: "std.text", Role: "setting-value"},
						},
					},
					{
						ID: "master.other",
						Attributes: []semantic.Attribute{
							{ID: "master.other.name", TypeRef: "std.text", Role: "name"},
						},
					},
				},
			},
		},
		Applications: []application.Application{
			{
				ID:           "order_entry",
				SemanticUses: []core.CanonicalID{"master"},
				Patterns: []application.Pattern{
					{ID: "order_entry.order_record", Kind: application.PatternRecord, Data: "master.other"},
				},
			},
		},
	}
}

// TestBuildApplicationProjectsStandardBusiness checks that the application view
// lists the standard business Data and the attributes each binder reads, so a
// reader does not have to search the semantic files.
func TestBuildApplicationProjectsStandardBusiness(t *testing.T) {
	view, err := applicationview.BuildApplication(standardBusinessWorld(), "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.StandardBusiness) != 3 {
		t.Fatalf("standard business=%+v", view.StandardBusiness)
	}
	kinds := map[core.CanonicalID]applicationview.StandardBusinessRef{}
	for _, ref := range view.StandardBusiness {
		kinds[ref.Kind] = ref
	}
	tax, ok := kinds["tax"]
	if !ok || tax.Data != "master.tax_rate" {
		t.Fatalf("tax ref=%+v", kinds)
	}
	// The roles are listed in the binder's order, required before optional.
	if len(tax.Attributes) != 5 || tax.Attributes[2] != "master.tax_rate.rate" {
		t.Fatalf("tax attributes=%+v", tax.Attributes)
	}
	holiday, ok := kinds["holiday"]
	if !ok || holiday.Data != "master.holiday" || len(holiday.Attributes) != 1 {
		t.Fatalf("holiday ref=%+v", kinds)
	}
	setting, ok := kinds["setting"]
	if !ok || setting.Data != "master.setting" || len(setting.Attributes) != 2 {
		t.Fatalf("setting ref=%+v", kinds)
	}
	// A Data with no standard business role is not listed.
	for _, ref := range view.StandardBusiness {
		if ref.Data == "master.other" {
			t.Fatalf("a data without a binder role was listed: %+v", ref)
		}
	}
}

// TestStandardBusinessSkipsADataWithNoBinderRole checks that a Data whose
// attributes carry display roles only is not projected, so the list stays about
// the standard business binders.
func TestStandardBusinessSkipsADataWithNoBinderRole(t *testing.T) {
	world := standardBusinessWorld()
	// Keep only the Data that carries display roles, which is the last one.
	world.Semantics[0].Data = world.Semantics[0].Data[len(world.Semantics[0].Data)-1:]
	world.Semantics[0].Data[0].ID = "master.other"
	view, err := applicationview.BuildApplication(world, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.StandardBusiness) != 0 {
		t.Fatalf("standard business=%+v", view.StandardBusiness)
	}
}

// TestStandardBusinessListsOnlyRolesTheBinderReads checks that an extra
// attribute with another role is not listed under the binder, so an incomplete
// declaration shows as a short list rather than a padded one.
func TestStandardBusinessListsOnlyRolesTheBinderReads(t *testing.T) {
	world := standardBusinessWorld()
	world.Semantics[0].Data[0].Attributes = append(world.Semantics[0].Data[0].Attributes,
		semantic.Attribute{ID: "master.tax_rate.note", TypeRef: "std.text", Role: "note"})
	view, err := applicationview.BuildApplication(world, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range view.StandardBusiness {
		if ref.Kind != "tax" {
			continue
		}
		for _, attribute := range ref.Attributes {
			if attribute == "master.tax_rate.note" {
				t.Fatalf("a display role was listed under the binder: %+v", ref)
			}
		}
	}
}
