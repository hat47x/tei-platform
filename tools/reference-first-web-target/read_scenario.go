//go:build !tei_repair_only

package main

import (
	"context"
	"fmt"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
	"tei/internal/query"
)

type referenceReadScenarioRunner struct {
	resolver      driver.Resolver
	environmentID core.CanonicalID
	applicationID core.CanonicalID
	browse        driver.BrowseQueryExecutor
	contracts     driver.ContractQueryExecutorRegistry
}

func newReferenceReadScenarioRunner(world ir.World) *referenceReadScenarioRunner {
	return &referenceReadScenarioRunner{
		resolver:      driver.NewResolver(world),
		environmentID: core.MustCanonicalID("development"),
		applicationID: core.MustCanonicalID("order_entry"),
		browse:        referenceBrowseExecutor{},
		contracts:     referenceContractQueryExecutors(),
	}
}

// ApplicationID returns the Application this runner executes. The server compares it with the
// projected Application so a runner never serves another Application's manifest.
func (r *referenceReadScenarioRunner) ApplicationID() string {
	return string(r.applicationID)
}

func (r *referenceReadScenarioRunner) Run(ctx context.Context, input readScenarioInput) (readScenarioResult, error) {
	if input.From == "" {
		return readScenarioResult{}, fmt.Errorf("from is required")
	}
	if input.RowIndex < 0 {
		return readScenarioResult{}, fmt.Errorf("row index must be non-negative")
	}
	from, err := time.Parse(time.RFC3339, input.From)
	if err != nil {
		return readScenarioResult{}, fmt.Errorf("from must be RFC3339: %w", err)
	}

	browse, err := r.resolver.ExecuteBrowseQuery(
		ctx,
		r.applicationID,
		core.MustCanonicalID("order_entry.recent_orders"),
		r.browse,
		map[string]any{"from": from},
	)
	if err != nil {
		return readScenarioResult{}, err
	}
	if input.RowIndex >= len(browse.Result.Mapped.Rows) {
		return readScenarioResult{}, fmt.Errorf(
			"selected row %d is outside browse result: rows=%d",
			input.RowIndex,
			len(browse.Result.Mapped.Rows),
		)
	}

	record, err := r.resolver.ResolvePattern(r.applicationID, core.MustCanonicalID("order_entry.order_record"))
	if err != nil {
		return readScenarioResult{}, err
	}
	if record.Data == nil || record.Data.ID != core.MustCanonicalID("sales.order") {
		return readScenarioResult{}, fmt.Errorf("order_record did not resolve sales.order")
	}

	customer, err := r.resolver.ExecuteReferenceContractQueryInEnvironment(
		ctx,
		r.environmentID,
		r.applicationID,
		core.MustCanonicalID("order_entry.customer_reference"),
		browse.Result.Mapped,
		input.RowIndex,
		r.contracts,
	)
	if err != nil {
		return readScenarioResult{}, err
	}

	lines, err := r.resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		r.environmentID,
		r.applicationID,
		core.MustCanonicalID("order_entry.order_lines"),
		browse.Result.Mapped,
		input.RowIndex,
		r.contracts,
	)
	if err != nil {
		return readScenarioResult{}, err
	}

	browseRows, err := projectMappedRows(browse.Result.Mapped)
	if err != nil {
		return readScenarioResult{}, err
	}
	selectedOrder := browseRows[input.RowIndex]

	customerRows := projectContractRows(customer.Execution.Result.Rows)
	orderLineRows := projectContractRows(lines.Execution.Result.Rows)

	return readScenarioResult{
		ScenarioID:       "order-entry.read-only-web-slice",
		Environment:      string(r.environmentID),
		ApplicationID:    string(r.applicationID),
		From:             input.From,
		SelectedRowIndex: input.RowIndex,
		BrowseRows:       browseRows,
		SelectedOrder:    selectedOrder,
		CustomerRows:     customerRows,
		OrderLineRows:    orderLineRows,
		RuntimeBindings: []runtimeBindingProjection{
			{
				Pattern:    "order_entry.recent_orders",
				Datasource: string(browse.Resolution.Datasource.Alias),
				Database:   string(browse.Resolution.Datasource.TargetDatabase),
				Access:     string(browse.Resolution.Datasource.Access),
				Note:       "BrowseQueryExecutor boundary does not select an Environment instance in current Driver API",
			},
			projectEnvironmentBinding("order_entry.customer_reference", customer.Environment.ID, customer.Executor),
			projectEnvironmentBinding("order_entry.order_lines", lines.Environment.ID, lines.Executor),
		},
		AuthorityLimits: []string{
			"synthetic-runtime-fixture-is-not-canonical-data",
			"runtime-row-index-is-selection-state-not-business-identity",
			"runtime-row-or-attribute-value-is-not-business-identity-proof",
			"explicit-result-mapping-is-semantic-correspondence-not-business-key-proof",
			"scenario-application-environment-and-binding-identities-are-runtime-context-not-business-data-identity",
			"browse-runtime-environment-binding-is-not-inferred",
			"reference-and-header-detail-runtime-binding-use-driver-environment-resolution",
			"read-slice-does-not-imply-authorization-or-command-readiness",
			"http-interaction-does-not-define-semantic-outcome",
		},
		Unresolved: []string{
			"standalone-read-endpoint-is-reference-diagnostic-without-authentication-gate",
			"standalone-read-endpoint-does-not-invoke-command",
			"actual-functional-evidence-not-yet-acquired",
			"visual-evidence-not-yet-acquired",
		},
	}, nil
}

type referenceBrowseExecutor struct{}

func (referenceBrowseExecutor) ExecuteBrowseQuery(
	_ context.Context,
	source query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	if source.ID != core.MustCanonicalID("order_entry.recent_orders") {
		return driver.BrowseQueryResult{}, fmt.Errorf("unsupported browse Query %s", source.ID)
	}
	if datasource.Alias != core.MustLocalName("sales") ||
		datasource.TargetDatabase != core.MustCanonicalID("sales_core") ||
		datasource.Access != database.AccessReadWrite {
		return driver.BrowseQueryResult{}, fmt.Errorf("unexpected browse Datasource: %+v", datasource)
	}
	from, ok := parameters["from"].(time.Time)
	if !ok {
		return driver.BrowseQueryResult{}, fmt.Errorf("browse parameter from must be time.Time: %T", parameters["from"])
	}
	candidates := []struct {
		orderID    int64
		customerID int64
		orderedAt  time.Time
		status     string
		total      string
	}{
		{1001, 11, time.Date(2026, 9, 18, 9, 30, 0, 0, time.UTC), "draft", "12500.00"},
		{1002, 12, time.Date(2026, 9, 18, 8, 15, 0, 0, time.UTC), "draft", "8300.00"},
		{1003, 13, time.Date(2026, 9, 18, 7, 30, 0, 0, time.UTC), "draft", "6200.00"},
	}
	rows := make([][]any, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.orderedAt.Before(from) {
			continue
		}
		rows = append(rows, []any{
			candidate.orderID,
			candidate.customerID,
			candidate.orderedAt,
			candidate.status,
			candidate.total,
		})
	}
	return driver.BrowseQueryResult{
		Raw: queryresult.RawResult{
			Columns: []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"},
			Rows:    rows,
		},
		Payload: map[string]any{
			"fixture": "first-web-target-read-only-slice",
		},
	}, nil
}

