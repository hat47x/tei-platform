//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
	"tei/internal/query"
)

// postgresBrowseExecutor runs the accepted browse Query's own SQL text
// (order_entry.recent_orders, from the Source) against PostgreSQL. It is the
// opt-in counterpart of referenceBrowseExecutor, which returns fixed rows.
//
// It does not decide permission and does not infer tables from the SQL: the
// Driver has already resolved the Query and the Datasource, and this adapter
// only checks that they are the ones it was written for.
type postgresBrowseExecutor struct {
	db *sql.DB
}

// namedParameter matches :name but not the :: cast operator.
var namedParameter = regexp.MustCompile(`(^|[^:]):([A-Za-z_][A-Za-z0-9_]*)`)

// bindNamedParameters rewrites :name to $n and returns the argument list. A
// name without a supplied value, or a value without a name, is an error, so a
// Query is never run with an unbound or silently ignored parameter.
func bindNamedParameters(source string, parameters map[string]any) (string, []any, error) {
	var (
		args  []any
		order = map[string]int{}
		used  = map[string]bool{}
		err   error
	)
	bound := namedParameter.ReplaceAllStringFunc(source, func(match string) string {
		groups := namedParameter.FindStringSubmatch(match)
		name := groups[2]
		value, ok := parameters[name]
		if !ok {
			err = errors.Join(err, fmt.Errorf("query parameter %q has no value", name))
			return match
		}
		used[name] = true
		position, seen := order[name]
		if !seen {
			args = append(args, value)
			position = len(args)
			order[name] = position
		}
		return fmt.Sprintf("%s$%d", groups[1], position)
	})
	if err != nil {
		return "", nil, err
	}
	for name := range parameters {
		if !used[name] {
			return "", nil, fmt.Errorf("query parameter %q is not used by the Query", name)
		}
	}
	return bound, args, nil
}

func (e postgresBrowseExecutor) ExecuteBrowseQuery(
	ctx context.Context,
	source query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	if source.ID != core.MustCanonicalID("order_entry.recent_orders") {
		return driver.BrowseQueryResult{}, fmt.Errorf("unsupported browse Query %s", source.ID)
	}
	if strings.TrimSpace(string(source.Dialect)) != "postgres" {
		return driver.BrowseQueryResult{}, fmt.Errorf("browse Query dialect %q is not postgres", source.Dialect)
	}
	if datasource.Alias != core.MustLocalName("sales") ||
		datasource.TargetDatabase != core.MustCanonicalID("sales_core") ||
		datasource.Access != database.AccessReadWrite {
		return driver.BrowseQueryResult{}, fmt.Errorf("unexpected browse Datasource: %+v", datasource)
	}
	statement, args, err := bindNamedParameters(source.Source, parameters)
	if err != nil {
		return driver.BrowseQueryResult{}, err
	}

	rows, err := e.db.QueryContext(ctx, statement, args...)
	if err != nil {
		// The error reaches the HTTP client; keep connection detail out of it.
		return driver.BrowseQueryResult{}, errors.New("browse query failed")
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return driver.BrowseQueryResult{}, errors.New("browse query failed")
	}
	var collected [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for index := range values {
			targets[index] = &values[index]
		}
		if err := rows.Scan(targets...); err != nil {
			return driver.BrowseQueryResult{}, errors.New("browse query failed")
		}
		for index, value := range values {
			if raw, isBytes := value.([]byte); isBytes {
				values[index] = string(raw)
			}
		}
		collected = append(collected, values)
	}
	if err := rows.Err(); err != nil {
		return driver.BrowseQueryResult{}, errors.New("browse query failed")
	}
	return driver.BrowseQueryResult{
		Raw:     queryresult.RawResult{Columns: columns, Rows: collected},
		Payload: map[string]any{"source": "postgres"},
	}, nil
}

// newPostgresBrowseReadRunner is the normal read runner with only the browse
// executor replaced. The customer and order-line reads keep their existing
// executors.
func newPostgresBrowseReadRunner(world ir.World, db *sql.DB) *referenceReadScenarioRunner {
	runner := newReferenceReadScenarioRunner(world)
	runner.browse = postgresBrowseExecutor{db: db}
	return runner
}
