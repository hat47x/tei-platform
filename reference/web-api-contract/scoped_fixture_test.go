package webapicontract

import (
	"errors"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

func scopedPublication(t *testing.T) PublishedBrowse {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ResolvePublishedBrowse(loaded.World,
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.recent_orders"),
		map[core.CanonicalID]FieldPolicy{
			core.MustCanonicalID("sales.order.order_id"):    {Select: true, Filter: true, Sort: true},
			core.MustCanonicalID("sales.order.customer_id"): {Select: true},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

func scopedSupport(pub PublishedBrowse, plan ReadPlan) ProviderReadSupport {
	orderID := core.MustCanonicalID("sales.order.order_id")
	customerID := core.MustCanonicalID("sales.order.customer_id")
	return ProviderReadSupport{
		QueryID:        pub.QueryID,
		PrincipalID:    plan.PrincipalID,
		RowScopeID:     plan.ScopeID,
		PolicyRevision: plan.PolicyRevision,
		EnforcesScope:  true,
		SupportsLimit:  true,
		ProjectsFields: map[core.CanonicalID]bool{orderID: true, customerID: true},
		Filters:        map[core.CanonicalID]map[string]bool{orderID: {"eq": true, "gt": true}},
		Sorts:          map[core.CanonicalID]bool{orderID: true},
	}
}

func fixtureScope() FixtureScope {
	return FixtureScope{
		PrincipalID:    "subject-order-editor",
		RowScopeID:     "tenant-a-only",
		PolicyRevision: "r17",
		Tenants:        map[string]bool{"tenant-a": true},
	}
}

func fixtureOrders() []FixtureRow {
	orderID := core.MustCanonicalID("sales.order.order_id")
	customerID := core.MustCanonicalID("sales.order.customer_id")
	total := core.MustCanonicalID("sales.order.total")
	return []FixtureRow{
		{TenantID: "tenant-b", Values: map[core.CanonicalID]any{orderID: int64(100), customerID: int64(21), total: int64(99000)}},
		{TenantID: "tenant-a", Values: map[core.CanonicalID]any{orderID: int64(3), customerID: int64(11), total: int64(3000)}},
		{TenantID: "tenant-b", Values: map[core.CanonicalID]any{orderID: int64(1), customerID: int64(22), total: int64(100)}},
		{TenantID: "tenant-a", Values: map[core.CanonicalID]any{orderID: int64(9), customerID: int64(12), total: int64(9000)}},
	}
}

func TestFixturePhysicallyScopesBeforeFilterSortLimitAndProjection(t *testing.T) {
	pub := scopedPublication(t)
	ctx := TrustedReadContext{
		PrincipalID: "subject-order-editor", RowScopeID: "tenant-a-only", PolicyRevision: "r17",
	}
	for _, tc := range []struct {
		name   string
		params url.Values
		want   [][]any
	}{
		{
			name:   "descending and filter",
			params: url.Values{"$select": {"orderId"}, "$filter": {"orderId gt 2"}, "$orderby": {"orderId desc"}, "$top": {"1"}},
			want:   [][]any{{int64(9)}},
		},
		{
			name:   "ascending before limit",
			params: url.Values{"$select": {"orderId"}, "$orderby": {"orderId asc"}, "$top": {"1"}},
			want:   [][]any{{int64(3)}},
		},
		{
			name:   "project allowed attributes only",
			params: url.Values{"$select": {"orderId,customerId"}, "$orderby": {"orderId asc"}, "$top": {"2"}},
			want:   [][]any{{int64(3), int64(11)}, {int64(9), int64(12)}},
		},
		{
			name:   "filter on other tenant's order returns no rows",
			params: url.Values{"$select": {"orderId"}, "$filter": {"orderId eq 100"}},
			want:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := pub.BindRead(tc.params, ctx)
			if err != nil {
				t.Fatal(err)
			}
			result, err := ExecuteScopedFixtureRead(pub, plan, scopedSupport(pub, plan),
				fixtureScope(), fixtureOrders())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Rows, tc.want) {
				t.Fatalf("incorrect row-scope ordering, got=%v want=%v", result.Rows, tc.want)
			}
			if !reflect.DeepEqual(result.Columns, plan.Selected) {
				t.Fatalf("unselected column escaped: %v", result.Columns)
			}
			for _, col := range result.Columns {
				if col == "total" {
					t.Fatal("raw sensitive column escaped public projection")
				}
			}
		})
	}
}

func TestFixtureRefusesScopeImpersonationAndUnsupportedOperations(t *testing.T) {
	pub := scopedPublication(t)
	ctx := TrustedReadContext{
		PrincipalID: "subject-order-editor", RowScopeID: "tenant-a-only", PolicyRevision: "r17",
	}
	plan, err := pub.BindRead(url.Values{"$select": {"orderId"}, "$filter": {"orderId gt 2"}, "$orderby": {"orderId asc"}}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	type scenario struct {
		name    string
		change  func(*ReadPlan, *ProviderReadSupport, *FixtureScope, *[]FixtureRow)
	}
	scenarios := []scenario{
		{"different principal", func(_ *ReadPlan, _ *ProviderReadSupport, s *FixtureScope, _ *[]FixtureRow) { s.PrincipalID = "other" }},
		{"different scope", func(_ *ReadPlan, _ *ProviderReadSupport, s *FixtureScope, _ *[]FixtureRow) { s.RowScopeID = "tenant-b" }},
		{"stale policy", func(_ *ReadPlan, _ *ProviderReadSupport, s *FixtureScope, _ *[]FixtureRow) { s.PolicyRevision = "stale" }},
		{"missing tenant authority", func(_ *ReadPlan, _ *ProviderReadSupport, s *FixtureScope, _ *[]FixtureRow) { s.Tenants = nil }},
		{"all tenant grants false", func(_ *ReadPlan, _ *ProviderReadSupport, s *FixtureScope, _ *[]FixtureRow) { s.Tenants["tenant-a"] = false }},
		{"empty tenant id", func(_ *ReadPlan, _ *ProviderReadSupport, s *FixtureScope, _ *[]FixtureRow) { s.Tenants[""] = true }},
		{"unsupported provider filter", func(_ *ReadPlan, support *ProviderReadSupport, _ *FixtureScope, _ *[]FixtureRow) {
			support.Filters = nil
		}},
		{"unsupported provider sort", func(_ *ReadPlan, support *ProviderReadSupport, _ *FixtureScope, _ *[]FixtureRow) {
			support.Sorts = nil
		}},
		{"unknown scope row", func(_ *ReadPlan, _ *ProviderReadSupport, _ *FixtureScope, rows *[]FixtureRow) {
			*rows = append(*rows, FixtureRow{TenantID: "", Values: map[core.CanonicalID]any{}})
		}},
		{"forged unsupported operator", func(p *ReadPlan, support *ProviderReadSupport, _ *FixtureScope, _ *[]FixtureRow) {
			p.Filter.Operator = "internal"
			support.Filters[core.MustCanonicalID("sales.order.order_id")]["internal"] = true
		}},
		{"forged incompatible filter literal", func(p *ReadPlan, _ *ProviderReadSupport, _ *FixtureScope, _ *[]FixtureRow) {
			p.Filter.Value = "'unexpected string'"
		}},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			working := plan
			working.Filter = &Filter{FieldID: plan.Filter.FieldID, Operator: plan.Filter.Operator, Value: plan.Filter.Value}
			support := scopedSupport(pub, working)
			scope := fixtureScope()
			rows := fixtureOrders()
			tc.change(&working, &support, &scope, &rows)
			out, err := ExecuteScopedFixtureRead(pub, working, support, scope, rows)
			if !errors.Is(err, ErrRejected) || len(out.Columns) != 0 || len(out.Rows) != 0 {
				t.Fatalf("unauthorized fixture execution escaped: result=%+v err=%v", out, err)
			}
		})
	}
}
