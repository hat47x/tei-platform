package webapicontract

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/workspace"
)

type scopedCanonicalExecutorFunc func(
	context.Context,
	query.EmbeddedSQL,
	application.DatabaseRequirement,
	ReadPlan,
	map[string]any,
) (driver.BrowseQueryResult, error)

func (f scopedCanonicalExecutorFunc) ExecuteScopedBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	plan ReadPlan,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	return f(ctx, q, datasource, plan, parameters)
}

func bridgeWorld(t *testing.T) driver.Resolver {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	return driver.NewResolver(loaded.World)
}

func TestRealDriverRequiresScopedProviderAndProjectsOnlyPublishedColumns(t *testing.T) {
	pub := scopedPublication(t)
	ctx := TrustedReadContext{
		PrincipalID: "subject-order-viewer", RowScopeID: "tenant-a-only", PolicyRevision: "r17",
	}
	plan, err := AuthorizeBrowseRead(context.Background(), fixtureAuthorization(t), pub,
		url.Values{
			"$select": {"orderId,customerId"},
			"$filter": {"orderId gt 2"},
			"$orderby": {"orderId asc"},
			"$top": {"2"},
		}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	provider := scopedCanonicalExecutorFunc(func(
		_ context.Context,
		q query.EmbeddedSQL,
		datasource application.DatabaseRequirement,
		actual ReadPlan,
		parameters map[string]any,
	) (driver.BrowseQueryResult, error) {
		called++
		if q.ID != pub.QueryID || datasource.Alias != "sales" ||
			actual.PrincipalID != ctx.PrincipalID ||
			actual.ScopeID != ctx.RowScopeID || actual.PolicyRevision != ctx.PolicyRevision ||
			actual.Filter == nil || actual.Filter.Operator != "gt" ||
			actual.Order == nil || actual.Order.FieldID != "orderId" ||
			actual.Limit != 2 || parameters["from"] != "fixture-query-parameter" {
			t.Fatalf("Driver did not pass the trusted plan or canonical Query: q=%+v scope=%+v datasource=%+v", q.ID, actual, datasource)
		}
		// The fixture provider is explicitly pretending it has physically
		// applied the plan. This verifies the Driver mapping and projection
		// boundary only; the provider's row security remains unproven.
		return driver.BrowseQueryResult{Raw: queryresult.RawResult{
			Columns: []string{"total_amount", "status", "ordered_at", "customer_id", "order_id"},
			Rows: [][]any{
				{int64(3000), "OPEN", "2026-10-01", int64(11), int64(3)},
				{int64(9000), "READY", "2026-10-02", int64(12), int64(9)},
			},
		}}, nil
	})
	out, err := ExecuteScopedCanonicalRead(context.Background(), bridgeWorld(t), pub, plan,
		scopedSupport(pub, plan), provider, map[string]any{"from": "fixture-query-parameter"})
	if err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("actual Driver did not invoke scoped provider exactly once: %d", called)
	}
	if !reflect.DeepEqual(out.Columns, []string{"orderId", "customerId"}) ||
		!reflect.DeepEqual(out.Rows, [][]any{{int64(3), int64(11)}, {int64(9), int64(12)}}) {
		t.Fatalf("unpublished native SQL columns or incorrect mapping: %+v", out)
	}
}

func TestRealDriverRefusesUnscopedFallbackAndInvalidResult(t *testing.T) {
	pub := scopedPublication(t)
	ctx := TrustedReadContext{
		PrincipalID: "subject-order-viewer", RowScopeID: "tenant-a-only", PolicyRevision: "r17",
	}
	plan, err := pub.BindRead(url.Values{"$select": {"orderId"}, "$top": {"1"}}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	support := scopedSupport(pub, plan)
	seen := 0
	provider := scopedCanonicalExecutorFunc(func(
		context.Context, query.EmbeddedSQL, application.DatabaseRequirement, ReadPlan, map[string]any,
	) (driver.BrowseQueryResult, error) {
		seen++
		return driver.BrowseQueryResult{Raw: queryresult.RawResult{
			Columns: []string{"order_id"}, Rows: [][]any{{int64(1)}},
		}}, nil
	})
	for _, tc := range []struct {
		name string
		pub  PublishedBrowse
		plan ReadPlan
		supp ProviderReadSupport
		exec ScopedCanonicalExecutor
		want error
	}{
		{"missing executor", pub, plan, support, nil, ErrScopedQueryUnavailable},
		{"wrong scope", pub, plan, func() ProviderReadSupport {
			bad := support
			bad.RowScopeID = "other"
			return bad
		}(), provider, ErrRejected},
		{"provider lacks filter ability", pub, func() ReadPlan {
			bad := plan
			bad.Filter = &Filter{FieldID: "orderId", Operator: "eq", Value: "1"}
			return bad
		}(), func() ProviderReadSupport {
			bad := support
			bad.Filters = nil
			return bad
		}(), provider, ErrRejected},
		{"wrong canonical Query", func() PublishedBrowse {
			bad := pub
			bad.QueryID = core.MustCanonicalID("other.query")
			return bad
		}(), plan, func() ProviderReadSupport {
			bad := support
			bad.QueryID = core.MustCanonicalID("other.query")
			return bad
		}(), provider, ErrScopedQueryUnavailable},
		{"wrong canonical Data", func() PublishedBrowse {
			bad := pub
			bad.DataID = core.MustCanonicalID("other.data")
			return bad
		}(), plan, support, provider, ErrScopedQueryUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := seen
			out, err := ExecuteScopedCanonicalRead(context.Background(), bridgeWorld(t),
				tc.pub, tc.plan, tc.supp, tc.exec, nil)
			if !errors.Is(err, tc.want) || len(out.Columns) != 0 || len(out.Rows) != 0 {
				t.Fatalf("invalid dispatch escaped: result=%+v err=%v want=%v", out, err, tc.want)
			}
			if seen != before && tc.name != "wrong canonical Data" {
				t.Fatalf("provider was invoked before rejection: %s", tc.name)
			}
		})
	}
	// Provider raw rows missing the expected result mapping MUST NOT
	// be presented as an empty or partial public result.
	out, err := ExecuteScopedCanonicalRead(context.Background(), bridgeWorld(t),
		pub, plan, support, provider, nil)
	if !errors.Is(err, ErrScopedQueryUnavailable) || len(out.Columns) != 0 {
		t.Fatalf("invalid provider mapping was published: %+v err=%v", out, err)
	}
}
