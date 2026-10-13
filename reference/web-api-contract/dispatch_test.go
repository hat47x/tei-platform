package webapicontract

import (
	"errors"
	"net/url"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

func TestReadDispatchRejectsUnsupportedNativeQueryOperations(t *testing.T) {
	root, _ := orderWorld(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	id := core.MustCanonicalID("sales.order.order_id")
	pub, err := ResolvePublishedBrowse(
		loaded.World, core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.recent_orders"),
		map[core.CanonicalID]FieldPolicy{id: {Select: true, Filter: true, Sort: true}},
	)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pub.BindRead(url.Values{
		"$filter":  {"orderId eq 'ORD-1'"},
		"$orderby": {"orderId desc"},
		"$top":     {"5"},
	}, trusted())
	if err != nil {
		t.Fatal(err)
	}
	complete := func() ProviderReadSupport {
		return ProviderReadSupport{
			QueryID: pub.QueryID, PolicyRevision: plan.PolicyRevision,
			EnforcesScope: true, SupportsLimit: true,
			ProjectsFields: map[core.CanonicalID]bool{id: true},
			Filters:        map[core.CanonicalID]map[string]bool{id: {"eq": true}},
			Sorts:          map[core.CanonicalID]bool{id: true},
		}
	}
	// Synthetic complete support is deliberately not a proof that the real
	// native-SQL Query can enforce these operations.
	if err := CheckReadDispatch(pub, plan, complete()); err != nil {
		t.Fatalf("explicit support should pass a syntactic preflight: %v", err)
	}
	cases := []struct {
		name   string
		change func(*ProviderReadSupport)
	}{
		{"no scope enforcement", func(s *ProviderReadSupport) { s.EnforcesScope = false }},
		{"unsupported pagination", func(s *ProviderReadSupport) { s.SupportsLimit = false }},
		{"wrong query provider", func(s *ProviderReadSupport) { s.QueryID = core.MustCanonicalID("other.query") }},
		{"stale policy", func(s *ProviderReadSupport) { s.PolicyRevision = "stale" }},
		{"no projection", func(s *ProviderReadSupport) { s.ProjectsFields[id] = false }},
		{"no filter", func(s *ProviderReadSupport) { delete(s.Filters, id) }},
		{"no filter operator", func(s *ProviderReadSupport) { s.Filters[id] = map[string]bool{"ne": true} }},
		{"no sort", func(s *ProviderReadSupport) { s.Sorts[id] = false }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			support := complete()
			tc.change(&support)
			if err := CheckReadDispatch(pub, plan, support); !errors.Is(err, ErrRejected) {
				t.Fatalf("unsupported operation was accepted: %v", err)
			}
		})
	}
	// An ordinary native SQL Query without an OData-aware executor does NOT
	// support automatic dynamic filtering/sorting/scoping.
	if err := CheckReadDispatch(pub, plan, ProviderReadSupport{QueryID: pub.QueryID}); !errors.Is(err, ErrRejected) {
		t.Fatal("native Query without an explicit trusted execution plan was dispatched")
	}
}

func TestReadDispatchRechecksGrantsForForgedPlans(t *testing.T) {
	id := core.MustCanonicalID("sales.order.order_id")
	q := core.MustCanonicalID("order_entry.recent_orders")
	pub := PublishedBrowse{
		QueryID: q,
		Fields: []PublishedField{{Name: "orderId", AttributeID: id,
			Policy: FieldPolicy{Select: true}}},
	}
	support := ProviderReadSupport{
		QueryID: q, PolicyRevision: "rev1",
		EnforcesScope: true, SupportsLimit: true,
		ProjectsFields: map[core.CanonicalID]bool{id: true},
		Filters:        map[core.CanonicalID]map[string]bool{id: {"eq": true}},
		Sorts:          map[core.CanonicalID]bool{id: true},
	}
	base := ReadPlan{Selected: []string{"orderId"}, Limit: 50, ScopeID: "row-scope", PolicyRevision: "rev1"}
	if err := CheckReadDispatch(pub, base, support); err != nil {
		t.Fatalf("expected valid preflight: %v", err)
	}
	for name, modify := range map[string]func(*ReadPlan){
		"filter without field grant": func(p *ReadPlan) { p.Filter = &Filter{FieldID: "orderId", Operator: "eq", Value: "x"} },
		"sort without field grant": func(p *ReadPlan) { p.Order = &Ordering{FieldID: "orderId"} },
		"duplicate projection":     func(p *ReadPlan) { p.Selected = []string{"orderId", "orderId"} },
		"unknown projection":       func(p *ReadPlan) { p.Selected = []string{"private"} },
		"empty scope":              func(p *ReadPlan) { p.ScopeID = "" },
		"out of bounds limit":      func(p *ReadPlan) { p.Limit = 9999 },
	} {
		t.Run(name, func(t *testing.T) {
			forged := base
			modify(&forged)
			if err := CheckReadDispatch(pub, forged, support); !errors.Is(err, ErrRejected) {
				t.Fatalf("forged request accepted: %v", err)
			}
		})
	}
}
