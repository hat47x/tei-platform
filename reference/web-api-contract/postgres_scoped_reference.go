package webapicontract

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/query"
)

// PostgresCustomerScope is injected by a trusted authorization authority.
// CustomerID is a physical row restriction for this one reference query;
// it is not a general tenant/RBAC policy or proof of authenticated identity.
type PostgresCustomerScope struct {
	PrincipalID    string
	RowScopeID     string
	PolicyRevision string
	CustomerID     int64
}

// ScopedRecentOrdersPostgres is an explicitly authored realization for a
// single canonical Browse Query. It never rewrites arbitrary native SQL.
// This reference is NOT an authorization provider or a production web handler.
type ScopedRecentOrdersPostgres struct {
	DB       *sql.DB
	Database core.CanonicalID
	Scope    PostgresCustomerScope
}

var _ ScopedCanonicalExecutor = ScopedRecentOrdersPostgres{}

// This canonical source is the only native Query shape that this explicit
// physical realization has been reviewed against. Authoring changes fail
// closed until a new realization is reviewed.
const recentOrdersCanonicalShape = "SELECT o.order_id, o.customer_id, o.ordered_at, o.status, o.total_amount FROM public.orders AS o WHERE o.ordered_at >= :from ORDER BY o.ordered_at DESC"

func (e ScopedRecentOrdersPostgres) ExecuteScopedBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	plan ReadPlan,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	if ctx == nil || e.DB == nil ||
		e.Database != core.CanonicalID("sales_core") ||
		datasource.TargetDatabase != e.Database ||
		datasource.Alias != "sales" ||
		q.ID != core.CanonicalID("order_entry.recent_orders") ||
		q.Application != core.CanonicalID("order_entry") ||
		q.Datasource != "sales" ||
		q.Dialect != environment.ProviderPostgres ||
		strings.Join(strings.Fields(q.Source), " ") != recentOrdersCanonicalShape {
		return driver.BrowseQueryResult{}, ErrScopedQueryContract
	}
	scope := e.Scope
	if scope.PrincipalID == "" || scope.PrincipalID != plan.PrincipalID ||
		scope.RowScopeID == "" || scope.RowScopeID != plan.ScopeID ||
		scope.PolicyRevision == "" || scope.PolicyRevision != plan.PolicyRevision ||
		scope.CustomerID <= 0 || plan.Limit < 1 || plan.Limit > 100 ||
		len(plan.Selected) == 0 {
		return driver.BrowseQueryResult{}, ErrRejected
	}
	// The canonical Query maps five raw columns, but a Browse user must
	// never cause unselected sensitive values to be fetched merely because
	// Driver mapping expects all five column identities. Static, reviewed
	// expressions replace unselected columns with typed NULL aliases.
	// All names and expressions come from this hard-coded realization.
	physical := []struct {
		name  string
		expr  string
		alias string
	}{
		{"orderId", "o.order_id", "order_id"},
		{"customerId", "o.customer_id", "customer_id"},
		{"orderedAt", "o.ordered_at", "ordered_at"},
		{"status", "o.status", "status"},
		{"total", "o.total_amount", "total_amount"},
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
	selectItems := make([]string, len(physical))
	for i, column := range physical {
		if selected[column.name] {
			selectItems[i] = column.expr
		} else {
			selectItems[i] = "NULL AS " + column.alias
		}
	}
	if len(parameters) != 1 {
		return driver.BrowseQueryResult{}, ErrRejected
	}
	from, ok := parameters["from"].(time.Time)
	if !ok || from.IsZero() {
		return driver.BrowseQueryResult{}, ErrRejected
	}
	// No source SQL concatenation or caller-controlled identifiers. The
	// row scope is the FIRST bound predicate; client filtering is additional.
	statement := "SELECT " + strings.Join(selectItems, ", ") +
		" FROM public.orders AS o WHERE o.customer_id = $1 AND o.ordered_at >= $2"
	args := []any{scope.CustomerID, from}
	if plan.Filter != nil {
		if plan.Filter.FieldID != "orderId" {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		var operator string
		switch plan.Filter.Operator {
		case "eq":
			operator = "="
		case "gt":
			operator = ">"
		default:
			return driver.BrowseQueryResult{}, ErrRejected
		}
		orderID, err := strconv.ParseInt(plan.Filter.Value, 10, 64)
		if err != nil || orderID < 0 {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		statement += " AND o.order_id " + operator + " $3"
		args = append(args, orderID)
	}
	orderBy := "o.ordered_at DESC"
	if plan.Order != nil {
		if plan.Order.FieldID != "orderId" {
			return driver.BrowseQueryResult{}, ErrRejected
		}
		orderBy = "o.order_id ASC"
		if plan.Order.Descending {
			orderBy = "o.order_id DESC"
		}
	}
	args = append(args, plan.Limit)
	statement += fmt.Sprintf(" ORDER BY %s LIMIT $%d", orderBy, len(args))
	rows, err := e.DB.QueryContext(ctx, statement, args...)
	if err != nil {
		return driver.BrowseQueryResult{}, ErrScopedQueryUnavailable
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return driver.BrowseQueryResult{}, ErrScopedQueryUnavailable
	}
	raw := queryresult.RawResult{Columns: append([]string(nil), columns...)}
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return driver.BrowseQueryResult{}, ErrScopedQueryUnavailable
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = append([]byte(nil), bytes...)
			}
		}
		raw.Rows = append(raw.Rows, values)
		if len(raw.Rows) > plan.Limit {
			return driver.BrowseQueryResult{}, ErrScopedQueryContract
		}
	}
	if err := rows.Err(); err != nil {
		return driver.BrowseQueryResult{}, ErrScopedQueryUnavailable
	}
	return driver.BrowseQueryResult{Raw: raw}, nil
}
