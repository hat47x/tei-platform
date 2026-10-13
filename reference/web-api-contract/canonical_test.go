package webapicontract

import (
	"errors"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/workspace"
)

func orderWorld(t *testing.T) (string, error) {
	// This function intentionally returns no parsed World to keep the tests
	// about the actual Workspace source and its canonical resolver.
	return filepath.Join("..", "workspace"), nil
}

func TestActualOrderEntryBrowseProjection(t *testing.T) {
	root, _ := orderWorld(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := driver.NewResolver(loaded.World).ResolvePattern(
		core.MustCanonicalID("order_entry"), core.MustCanonicalID("order_entry.recent_orders"))
	if err != nil || resolved.Query == nil || resolved.Data == nil {
		t.Fatalf("existing browse resolution not available: %+v %v", resolved, err)
	}
	if resolved.Query.ID != core.MustCanonicalID("order_entry.recent_orders") ||
		resolved.Data.ID != core.MustCanonicalID("sales.order") {
		t.Fatalf("unexpected canonical binding: %+v", resolved)
	}

	// Runtime authorization owns these decisions. The query result mapping
	// NEVER grants access; total is a real mapped attribute, intentionally
	// inaccessible to this synthetic principal.
	grants := map[core.CanonicalID]FieldPolicy{
		core.MustCanonicalID("sales.order.order_id"): {Select: true, Filter: true, Sort: true},
		core.MustCanonicalID("sales.order.customer_id"): {Select: true},
		core.MustCanonicalID("sales.order.total"): {Select: false, Filter: false, Sort: false},
	}
	pub, err := ResolvePublishedBrowse(loaded.World, core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.recent_orders"), grants)
	if err != nil {
		t.Fatal(err)
	}
	if pub.QueryID != resolved.Query.ID || pub.DataID != resolved.Data.ID {
		t.Fatalf("invented query identity: %+v", pub)
	}
	if len(pub.Fields) != 2 {
		t.Fatalf("unexpected public metadata projection: %+v", pub.Fields)
	}
	if !reflect.DeepEqual([]string{pub.Fields[0].Name, pub.Fields[1].Name},
		[]string{"orderId", "customerId"}) {
		t.Fatalf("field names must come from Semantic symbols: %+v", pub.Fields)
	}
	if pub.Fields[0].AttributeID != core.MustCanonicalID("sales.order.order_id") {
		t.Fatal("raw SQL column was substituted for canonical attribute identity")
	}
	ctx := trusted()
	plan, err := pub.BindRead(url.Values{"$orderby": {"orderId desc"}, "$filter": {"orderId eq 'ORD-1'"}}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Selected, []string{"orderId", "customerId"}) {
		t.Fatalf("unauthorized implicit projection: %+v", plan.Selected)
	}
	for _, request := range []url.Values{
		{"$select": {"total"}}, {"$filter": {"total gt 100"}},
		{"$orderby": {"total"}}, {"$select": {"order_id"}},
	} {
		if _, err := pub.BindRead(request, ctx); !errors.Is(err, ErrRejected) {
			t.Fatalf("forbidden field/query escaped: %v: %v", request, err)
		}
	}
}

func TestActualOrderEntryFailsClosedWhenModelOrGrantsConflict(t *testing.T) {
	root, _ := orderWorld(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	app := core.MustCanonicalID("order_entry")
	pattern := core.MustCanonicalID("order_entry.recent_orders")
	cases := []struct {
		name    string
		app     core.CanonicalID
		pattern core.CanonicalID
		grants  map[core.CanonicalID]FieldPolicy
	}{
		{"no grants", app, pattern, nil},
		{"only filter no projection", app, pattern, map[core.CanonicalID]FieldPolicy{
			core.MustCanonicalID("sales.order.order_id"): {Filter: true},
		}},
		{"unmapped attribute from same Data", app, pattern, map[core.CanonicalID]FieldPolicy{
			core.MustCanonicalID("sales.order.order_id"): {Select: true},
			core.MustCanonicalID("sales.order.external_id"): {Filter: true},
		}},
		{"unknown attribute", app, pattern, map[core.CanonicalID]FieldPolicy{
			core.MustCanonicalID("sales.order.order_id"): {Select: true},
			core.MustCanonicalID("sales.order.private"): {Sort: true},
		}},
		{"wrong application", core.MustCanonicalID("repair_intake"), pattern,
			map[core.CanonicalID]FieldPolicy{core.MustCanonicalID("sales.order.order_id"): {Select: true}}},
		{"non-browse pattern", app, core.MustCanonicalID("order_entry.order_record"),
			map[core.CanonicalID]FieldPolicy{core.MustCanonicalID("sales.order.order_id"): {Select: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolvePublishedBrowse(loaded.World, tc.app, tc.pattern, tc.grants)
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("expected refusal, got %v", err)
			}
		})
	}
}
