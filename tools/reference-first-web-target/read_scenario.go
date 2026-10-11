package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/realization"
)

type readScenarioInput struct {
	From     string
	RowIndex int
}

type readScenarioResult struct {
	ScenarioID       string                     `json:"scenarioId"`
	Environment      string                     `json:"environment"`
	ApplicationID    string                     `json:"applicationId"`
	From             string                     `json:"from"`
	SelectedRowIndex int                        `json:"selectedRowIndex"`
	BrowseRows       []semanticRowProjection    `json:"browseRows"`
	SelectedOrder    semanticRowProjection      `json:"selectedOrder"`
	CustomerRows     []semanticRowProjection    `json:"customerRows"`
	OrderLineRows    []semanticRowProjection    `json:"orderLineRows"`
	RuntimeBindings  []runtimeBindingProjection `json:"runtimeBindings"`
	AuthorityLimits  []string                   `json:"authorityLimits"`
	Unresolved       []string                   `json:"unresolved"`
}

type semanticRowProjection struct {
	Attributes []semanticValueProjection `json:"attributes"`
}

type semanticValueProjection struct {
	Attribute string `json:"attribute"`
	Value     any    `json:"value"`
}

type runtimeBindingProjection struct {
	Pattern            string `json:"pattern"`
	Environment        string `json:"environment,omitempty"`
	Instance           string `json:"instance,omitempty"`
	Provider           string `json:"provider,omitempty"`
	ExecutorCapability string `json:"executorCapability,omitempty"`
	ContractQuery      string `json:"contractQuery,omitempty"`
	Datasource         string `json:"datasource,omitempty"`
	Database           string `json:"database,omitempty"`
	Access             string `json:"access,omitempty"`
	Note               string `json:"note,omitempty"`
}

type readScenarioRunner interface {
	Run(context.Context, readScenarioInput) (readScenarioResult, error)
}

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

// referenceContractQueryExecutors is the development runtime composition shared by every
// Reference scenario runner, so the Executor-to-instance binding is written once.
// The Executors are synthetic fixtures, not connections to a real Database.
func referenceContractQueryExecutors() driver.ContractQueryExecutorMap {
	return driver.ContractQueryExecutorMap{
		{
			Environment: core.MustCanonicalID("development"),
			Instance:    core.MustCanonicalID("master_data.dev"),
			Capability:  core.MustCanonicalID("database.query.oracle"),
		}: referenceContractQueryExecutor{provider: environment.ProviderOracle},
		{
			Environment: core.MustCanonicalID("development"),
			Instance:    core.MustCanonicalID("sales_core.dev"),
			Capability:  core.MustCanonicalID("database.query.postgres"),
		}: referenceContractQueryExecutor{provider: environment.ProviderPostgres},
	}
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

func projectEnvironmentBinding(
	pattern string,
	environmentID core.CanonicalID,
	executor realization.ResolvedContractQueryExecutor,
) runtimeBindingProjection {
	return runtimeBindingProjection{
		Pattern:            pattern,
		Environment:        string(environmentID),
		Instance:           string(executor.Instance.ID),
		Provider:           string(executor.Instance.Provider),
		ExecutorCapability: string(executor.Capability.Capability.ID),
		ContractQuery:      executor.Binding.Target.Identity(),
	}
}

func projectMappedRows(source queryresult.MappedResult) ([]semanticRowProjection, error) {
	result := make([]semanticRowProjection, len(source.Rows))
	for rowIndex, row := range source.Rows {
		if len(row) != len(source.Columns) {
			return nil, fmt.Errorf(
				"mapped result row %d has %d values for %d columns",
				rowIndex,
				len(row),
				len(source.Columns),
			)
		}
		projected := semanticRowProjection{}
		for columnIndex, column := range source.Columns {
			projected.Attributes = append(projected.Attributes, semanticValueProjection{
				Attribute: string(column.Attribute),
				Value:     row[columnIndex],
			})
		}
		result[rowIndex] = projected
	}
	return result, nil
}

func projectContractRows(source []driver.ContractQueryRow) []semanticRowProjection {
	result := make([]semanticRowProjection, len(source))
	for rowIndex, row := range source {
		keys := make([]string, 0, len(row.Attributes))
		values := make(map[string]any, len(row.Attributes))
		for attribute, value := range row.Attributes {
			key := string(attribute)
			keys = append(keys, key)
			values[key] = value
		}
		sort.Strings(keys)
		for _, key := range keys {
			result[rowIndex].Attributes = append(result[rowIndex].Attributes, semanticValueProjection{
				Attribute: key,
				Value:     values[key],
			})
		}
	}
	return result
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

type referenceContractQueryExecutor struct {
	provider environment.DatabaseProvider
}

func (e referenceContractQueryExecutor) DatabaseProvider() environment.DatabaseProvider {
	return e.provider
}

func (e referenceContractQueryExecutor) ExecuteContractQuery(
	_ context.Context,
	target realization.ContractQueryTarget,
	queryContract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	switch target.Identity() {
	case "master_data/master_data.customer/find_by_id":
		if e.provider != environment.ProviderOracle {
			return driver.ContractQueryResult{}, fmt.Errorf("customer executor provider=%s", e.provider)
		}
		if queryContract.Name != "find_by_id" {
			return driver.ContractQueryResult{}, fmt.Errorf("unexpected customer Query %s", queryContract.Name)
		}
		id, ok := parameters["id"]
		if !ok {
			return driver.ContractQueryResult{}, fmt.Errorf("customer id parameter is required")
		}
		return driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{{
				Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("customer.customer.customer_id"): id,
					core.MustCanonicalID("customer.customer.name"):        "Reference Customer " + fmt.Sprint(id),
				},
			}},
			Payload: map[string]any{"fixture": "oracle-customer-read"},
		}, nil

	case "sales_core/sales_core.order_line/find_by_order":
		if e.provider != environment.ProviderPostgres {
			return driver.ContractQueryResult{}, fmt.Errorf("order-line executor provider=%s", e.provider)
		}
		orderIDValue, ok := parameters["order_id"]
		if !ok {
			return driver.ContractQueryResult{}, fmt.Errorf("order_id parameter is required")
		}
		orderID, ok := orderIDValue.(int64)
		if !ok {
			return driver.ContractQueryResult{}, fmt.Errorf("order_id parameter has unexpected type %T", orderIDValue)
		}
		if queryContract.Name != "find_by_order" {
			return driver.ContractQueryResult{}, fmt.Errorf("unexpected order-line Query %s", queryContract.Name)
		}
		firstQuantity := "2"
		if orderID == 1002 {
			firstQuantity = "0"
		}
		return driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("sales.order_line.order_line_id"): int64(5001),
					core.MustCanonicalID("sales.order_line.line_number"):   1,
					core.MustCanonicalID("sales.order_line.item_code"):     "ITEM-A",
					core.MustCanonicalID("sales.order_line.quantity"):      firstQuantity,
					core.MustCanonicalID("sales.order_line.unit_price"):    "4000.00",
				}},
				{Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("sales.order_line.order_line_id"): int64(5002),
					core.MustCanonicalID("sales.order_line.line_number"):   2,
					core.MustCanonicalID("sales.order_line.item_code"):     "ITEM-B",
					core.MustCanonicalID("sales.order_line.quantity"):      "1",
					core.MustCanonicalID("sales.order_line.unit_price"):    "4500.00",
				}},
			},
			Payload: map[string]any{
				"fixture": "postgres-order-line-read",
				"orderId": orderID,
			},
		}, nil
	default:
		return driver.ContractQueryResult{}, fmt.Errorf("unsupported Contract Query target %s", target.Identity())
	}
}
