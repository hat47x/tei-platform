package webapicontract

import (
	"context"
	"database/sql"
	"reflect"
	"strconv"
	"strings"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/environment"
	"tei/internal/query"
)

// SQLTenantScopeRequest is built from a trusted, authorized ReadPlan. It is
// not a client-controlled SQL fragment, tenant ID, or proof of authentication.
type SQLTenantScopeRequest struct {
	PrincipalID    string
	RowScopeID     string
	PolicyRevision string
	QueryID        core.CanonicalID
}

// SQLTenantScopeGrant is issued by an external authority. The reference
// executor verifies binding identities but CANNOT verify the authority's
// authenticity or revocation without a production-backed implementation.
type SQLTenantScopeGrant struct {
	SQLTenantScopeRequest
	TenantID string
}

// SQLTenantScopeAuthority must resolve tenant membership from the policy
// source of truth; implementations must not trust HTTP-supplied tenant IDs.
type SQLTenantScopeAuthority interface {
	ResolveSQLTenantScope(context.Context, SQLTenantScopeRequest) (SQLTenantScopeGrant, error)
}

// ScopedPostgresOrdersReference is a dedicated, explicit PostgreSQL
// realization of order_entry.recent_orders for a separate fixture table.
// It is NOT an adapter for arbitrary Canonical native SQL and MUST NOT be
// installed as the production provider without a real TenantScopeAuthority,
// audited PostgreSQL schema and database integration tests.
//
// It delegates physical named-parameter binding and database/sql execution
// to TEI's existing PostgreSQL BrowseQueryExecutor.
type ScopedPostgresOrdersReference struct {
	DB        *sql.DB
	Authority SQLTenantScopeAuthority
	Database  core.CanonicalID
}

var _ ScopedCanonicalExecutor = ScopedPostgresOrdersReference{}

const scopedReferenceFromSQL = `FROM public.tei_web_scoped_orders_reference AS o
WHERE o.tenant_id = :tei_scope_tenant
  AND o.ordered_at >= :from`

// The fixture uses an explicitly authored query, not a rewrite of q.Source.
// The only generated fragments are allowlisted comparison/order operators.
func (e ScopedPostgresOrdersReference) ExecuteScopedBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	ds application.DatabaseRequirement,
	plan ReadPlan,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	if ctx == nil || e.DB == nil {
		return driver.BrowseQueryResult{}, ErrScopedQueryUnavailable
	}
	if e.Authority == nil {
		return driver.BrowseQueryResult{}, ErrAuthorizationUnavailable
	}
	if authority := reflect.ValueOf(e.Authority); authority.Kind() == reflect.Pointer && authority.IsNil() {
		return driver.BrowseQueryResult{}, ErrAuthorizationUnavailable
	}
	if e.Database != "sales_core" || ds.TargetDatabase != e.Database ||
		ds.Alias != "sales" || q.ID != "order_entry.recent_orders" ||
		q.Application != "order_entry" || q.Datasource != "sales" ||
		q.Dialect != environment.ProviderPostgres ||
		plan.PrincipalID == "" || plan.ScopeID == "" ||
		plan.PolicyRevision == "" || plan.Limit < 1 || plan.Limit > 100 ||
		len(plan.Selected) == 0 || len(parameters) != 1 {
		return driver.BrowseQueryResult{}, ErrRejected
	}

	expectedColumns := []query.ResultColumnMapping{
		{Column: "order_id", Attribute: "sales.order.order_id"},
		{Column: "customer_id", Attribute: "sales.order.customer_id"},
		{Column: "ordered_at", Attribute: "sales.order.ordered_at"},
		{Column: "status", Attribute: "sales.order.status"},
		{Column: "total_amount", Attribute: "sales.order.total"},
	}
	// This dedicated realization is pinned to the CURRENT authored Query
	// meaning; a future native Query change requires an explicit review.
	const expectedCanonicalSource = "SELECT o.order_id, o.customer_id, o.ordered_at, o.status, o.total_amount FROM public.orders AS o WHERE o.ordered_at >= :from ORDER BY o.ordered_at DESC"
	if strings.Join(strings.Fields(q.Source), " ") != expectedCanonicalSource {
		return driver.BrowseQueryResult{}, ErrRejected
	}
	if !reflect.DeepEqual(q.ResultColumns, expectedColumns) {
		return driver.BrowseQueryResult{}, ErrRejected
	}
	from, ok := parameters["from"].(time.Time)
	if !ok || from.IsZero() {
		return driver.BrowseQueryResult{}, ErrRejected
	}
	selected := make(map[string]bool, len(plan.Selected))
	for _, name := range plan.Selected {
		switch name {
		case "orderId", "customerId", "orderedAt", "status", "total":
		default:
			return driver.BrowseQueryResult{}, ErrRejected
		}
		if selected[name] {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		selected[name] = true
	}
	// Keep the canonical raw column aliases while preventing the database
	// from fetching any field omitted from the authorized projection.
	// These expressions/aliases are fixed by this reviewed realization.
	physicalColumns := []struct {
		name       string
		expression string
		alias      string
	}{
		{"orderId", "o.order_id", "order_id"},
		{"customerId", "o.customer_id", "customer_id"},
		{"orderedAt", "o.ordered_at", "ordered_at"},
		{"status", "o.status", "status"},
		{"total", "o.total_amount", "total_amount"},
	}
	selectParts := make([]string, 0, len(physicalColumns))
	for _, column := range physicalColumns {
		if selected[column.name] {
			selectParts = append(selectParts, column.expression)
		} else {
			selectParts = append(selectParts, "NULL AS "+column.alias)
		}
	}

	// Validate all client-driven operations before contacting the scope
	// authority or database. Values are never interpolated into SQL text.
	source := "SELECT\n    " + strings.Join(selectParts, ",\n    ") + "\n" + scopedReferenceFromSQL
	args := map[string]any{
		"from":    from,
		"tei_top": plan.Limit,
	}
	if plan.Filter != nil {
		if plan.Filter.FieldID != "orderId" {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		physicalOperator, ok := map[string]string{
			"eq": "=", "ne": "<>", "gt": ">",
			"ge": ">=", "lt": "<", "le": "<=",
		}[plan.Filter.Operator]
		if !ok || !scopedDecimalDigits(plan.Filter.Value) {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		value, err := strconv.ParseInt(plan.Filter.Value, 10, 64)
		if err != nil {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		source += "\n  AND o.order_id " + physicalOperator + " :tei_order_filter"
		args["tei_order_filter"] = value
	}
	orderClause := "o.ordered_at DESC, o.order_id DESC"
	if plan.Order != nil {
		if plan.Order.FieldID != "orderId" {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		if plan.Order.Descending {
			orderClause = "o.order_id DESC"
		} else {
			orderClause = "o.order_id ASC"
		}
	}
	source += "\nORDER BY " + orderClause + "\nLIMIT :tei_top"

	request := SQLTenantScopeRequest{
		PrincipalID:    plan.PrincipalID,
		RowScopeID:     plan.ScopeID,
		PolicyRevision: plan.PolicyRevision,
		QueryID:        q.ID,
	}
	grant, err := e.Authority.ResolveSQLTenantScope(ctx, request)
	if err != nil {
		return driver.BrowseQueryResult{}, ErrAuthorizationUnavailable
	}
	if grant.SQLTenantScopeRequest != request ||
		strings.TrimSpace(grant.TenantID) == "" {
		return driver.BrowseQueryResult{}, ErrAuthorizationContract
	}
	// The tenant is a positional bound value, not a SQL identifier, predicate
	// expression or browser-supplied value.
	args["tei_scope_tenant"] = grant.TenantID
	realization := q
	realization.Source = source
	return (postgresquery.Executor{
		DB: e.DB, Database: e.Database,
	}).ExecuteBrowseQuery(ctx, realization, ds, args)
}

func scopedDecimalDigits(raw string) bool {
	if raw == "" {
		return false
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] < '0' || raw[i] > '9' {
			return false
		}
	}
	return true
}
