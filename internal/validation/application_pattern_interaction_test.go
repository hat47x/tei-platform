package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func applicationPatternInteractionWorld() ir.World {
	w := validApplicationPatternWorld()
	w.Semantics[0].Data[0].Attributes = []semantic.Attribute{
		{ID: "sales.order.order_id", TypeRef: "std.identifier"},
		{ID: "sales.order.customer_id", TypeRef: "customer.customer_id"},
	}
	w.Semantics[1].Data[0].Attributes = []semantic.Attribute{
		{ID: "customer.customer.customer_id", TypeRef: "customer.customer_id"},
		{ID: "customer.customer.name", TypeRef: "std.text"},
	}
	w.Applications[0].Patterns = append(w.Applications[0].Patterns,
		application.Pattern{
			ID:         "order_entry.customer_reference",
			Kind:       application.PatternReference,
			Relation:   "sales.order.customer",
			Datasource: "master",
		},
		application.Pattern{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Items: []application.InteractionItem{
					{Attribute: "sales.order.order_id"},
					{
						Attribute: "sales.order.customer_id",
						Lookup: &application.InteractionLookup{
							Pattern:         "order_entry.customer_reference",
							TargetAttribute: "customer.customer.customer_id",
						},
					},
				},
				Navigations: []application.InteractionNavigation{{From: "order_entry.recent_orders"}},
			},
		},
	)
	return w
}

func interactionItem(t *testing.T, w *ir.World, attribute core.CanonicalID) *application.InteractionItem {
	t.Helper()
	for i := range w.Applications[0].Patterns {
		pattern := &w.Applications[0].Patterns[i]
		if pattern.ID != "order_entry.order_record" || pattern.Interaction == nil {
			continue
		}
		for j := range pattern.Interaction.Items {
			if pattern.Interaction.Items[j].Attribute == attribute {
				return &pattern.Interaction.Items[j]
			}
		}
	}
	t.Fatalf("interaction item %s not found", attribute)
	return nil
}

func TestValidateApplicationPatternsAcceptsRecordInteraction(t *testing.T) {
	w := applicationPatternInteractionWorld()

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("valid record interaction diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsUnknownInteractionItem(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications[0].Patterns[2].Interaction.Items = append(
		w.Applications[0].Patterns[2].Interaction.Items,
		application.InteractionItem{Attribute: "sales.order.missing"},
	)

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_ITEM") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsDuplicateInteractionItem(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications[0].Patterns[2].Interaction.Items = append(
		w.Applications[0].Patterns[2].Interaction.Items,
		application.InteractionItem{Attribute: "sales.order.order_id"},
	)

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_ITEM_DUPLICATE") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsLookupTypeMismatch(t *testing.T) {
	w := applicationPatternInteractionWorld()
	interactionItem(t, &w, "sales.order.customer_id").Lookup.TargetAttribute = "customer.customer.name"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_LOOKUP_TYPE_MISMATCH") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsLookupNotReference(t *testing.T) {
	w := applicationPatternInteractionWorld()
	interactionItem(t, &w, "sales.order.customer_id").Lookup.Pattern = "order_entry.recent_orders"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_LOOKUP_NOT_REFERENCE") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsNavigationNotBrowse(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications[0].Patterns[2].Interaction.Navigations[0].From = "order_entry.order_record"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_NAVIGATION_NOT_BROWSE") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsInteractionOnBrowse(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications[0].Patterns[0].Interaction = &application.Interaction{
		Items: []application.InteractionItem{{Attribute: "sales.order.order_id"}},
	}

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsUnknownLookupPattern(t *testing.T) {
	w := applicationPatternInteractionWorld()
	interactionItem(t, &w, "sales.order.customer_id").Lookup.Pattern = "order_entry.missing"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_LOOKUP_PATTERN") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsLookupApplicationMismatch(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications = append(w.Applications, application.Application{
		ID:        "other_app",
		Databases: []application.DatabaseRequirement{{Alias: "master", Access: database.AccessRead}},
		Patterns: []application.Pattern{{
			ID:         "other_app.customer_reference",
			Kind:       application.PatternReference,
			Relation:   "sales.order.customer",
			Datasource: "master",
		}},
	})
	interactionItem(t, &w, "sales.order.customer_id").Lookup.Pattern = "other_app.customer_reference"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_LOOKUP_APPLICATION_MISMATCH") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsUnknownLookupTargetAttribute(t *testing.T) {
	w := applicationPatternInteractionWorld()
	interactionItem(t, &w, "sales.order.customer_id").Lookup.TargetAttribute = "customer.customer.missing"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_LOOKUP_ATTRIBUTE") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsNavigationApplicationMismatch(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications = append(w.Applications, application.Application{
		ID:        "other_app",
		Databases: []application.DatabaseRequirement{{Alias: "sales", Access: database.AccessRead}},
		Patterns: []application.Pattern{{
			ID:    "other_app.recent_orders",
			Kind:  application.PatternBrowse,
			Data:  "sales.order",
			Query: "order_entry.recent_orders",
		}},
	})
	w.Applications[0].Patterns[2].Interaction.Navigations[0].From = "other_app.recent_orders"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_NAVIGATION_APPLICATION_MISMATCH") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsNavigationDataMismatch(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications[0].Patterns[0].Data = "customer.customer"

	ds := validateApplicationPatterns(w)
	if diagnosticByCode(ds, "APPLICATION_PATTERN_INTERACTION_NAVIGATION_DATA_MISMATCH") == nil {
		t.Fatalf("diagnostics=%#v", ds)
	}
}
