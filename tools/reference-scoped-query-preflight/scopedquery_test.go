package scopedquery

import "testing"

func ok() Input {
	return Input{
		Authorization: Authorization{
			Principal: "user:1", Tenant: "tenant-a", Subject: "role:sales",
			Purpose: "read-orders", PolicyRevision: "rev-7", Scopes: []string{"tenant-a"},
		},
		Realization: Realization{
			Application: "order-entry", Pattern: "list", Query: "recent_orders",
			Data: "Order", Datasource: "pg-main", Provider: "postgres",
			ScopeColumn: "tenant_id",
			Fields:      []string{"order_id", "amount", "created_at"},
			Operations:  []string{"filter", "sort", "limit"},
		},
		Request: Request{
			Filter: "amount > :min", Sort: "created_at desc", Limit: 20,
			Fields: []string{"order_id", "amount"},
		},
		Known: Known{
			Fields:     []string{"order_id", "amount", "created_at"},
			Providers:  []string{"postgres"},
			Operations: []string{"filter", "sort", "limit"},
		},
	}
}

func hasRefusal(r Report, code string) bool {
	for _, it := range r.Decision.Refusals {
		if it.Code == code {
			return true
		}
	}
	return false
}

func clauseKinds(r Report) []string {
	out := []string{}
	for _, c := range r.Decision.Clauses {
		out = append(out, c.Kind)
	}
	return out
}

func TestScopedQueryOrdersScopeFirst(t *testing.T) {
	r := Plan(ok())
	if !r.Decision.Execute {
		t.Fatalf("refusals = %+v", r.Decision.Refusals)
	}
	kinds := clauseKinds(r)
	want := []string{"scope", "filter", "order", "limit"}
	if len(kinds) != len(want) {
		t.Fatalf("clauses = %+v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("clause order = %+v, want %+v", kinds, want)
		}
	}
	if r.Decision.UnscopedFallback {
		t.Error("an implicit unscoped fallback was allowed")
	}
	if r.Decision.PostFilter {
		t.Error("post-filter row removal was allowed")
	}
}

func TestClientScopeNotAuthoritative(t *testing.T) {
	in := ok()
	in.Request.Tenant = "tenant-b"
	r := Plan(in)
	if !hasRefusal(r, "client-scope-not-authoritative") {
		t.Errorf("refusals = %+v", r.Decision.Refusals)
	}
	if r.Decision.Execute {
		t.Error("SQL was allowed with a client-specified tenant")
	}
}

func TestExpiredAuthorizationRefused(t *testing.T) {
	in := ok()
	in.Authorization.Expired = true
	r := Plan(in)
	if !hasRefusal(r, "authorization-expired") {
		t.Errorf("refusals = %+v", r.Decision.Refusals)
	}
}

func TestMissingScopeColumnRefused(t *testing.T) {
	in := ok()
	in.Realization.ScopeColumn = ""
	r := Plan(in)
	if !hasRefusal(r, "scope-column-missing") {
		t.Errorf("refusals = %+v", r.Decision.Refusals)
	}
}

func TestQueryDataMismatchRefused(t *testing.T) {
	in := ok()
	in.Request.Query = "other_query"
	r := Plan(in)
	if !hasRefusal(r, "query-data-mismatch") {
		t.Errorf("refusals = %+v", r.Decision.Refusals)
	}
}

func TestUnsupportedOperationRefused(t *testing.T) {
	for _, op := range []string{"count", "aggregate", "expand", "unknownop"} {
		in := ok()
		in.Request.Operations = []string{op}
		r := Plan(in)
		if !hasRefusal(r, "operation-not-supported") {
			t.Errorf("op %q not refused: %+v", op, r.Decision.Refusals)
		}
	}
}

func TestUnknownFieldRefused(t *testing.T) {
	in := ok()
	in.Request.Fields = []string{"order_id", "secret"}
	r := Plan(in)
	if !hasRefusal(r, "unknown-field") {
		t.Errorf("refusals = %+v", r.Decision.Refusals)
	}
}

func TestProviderNotSupportedHasNoFallback(t *testing.T) {
	in := ok()
	in.Realization.Provider = "mysql"
	r := Plan(in)
	if !hasRefusal(r, "provider-not-supported") {
		t.Errorf("refusals = %+v", r.Decision.Refusals)
	}
	if r.Decision.UnscopedFallback {
		t.Error("an unsupported provider fell back to an unscoped query")
	}
}

func TestPostFilterForbidden(t *testing.T) {
	in := ok()
	in.Request.PostFilterRows = true
	r := Plan(in)
	if !hasRefusal(r, "post-filter-forbidden") {
		t.Errorf("refusals = %+v", r.Decision.Refusals)
	}
}

func TestRefusalIssuesNoSQL(t *testing.T) {
	in := ok()
	in.Authorization.Expired = true
	r := Plan(in)
	if r.Decision.Execute {
		t.Fatal("SQL was allowed after a refusal")
	}
	if len(r.Decision.Clauses) != 0 || len(r.Decision.BoundParams) != 0 {
		t.Errorf("a refusal produced a plan: %+v", r.Decision)
	}
}
