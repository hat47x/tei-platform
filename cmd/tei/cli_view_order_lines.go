package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/driver/validationread"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/semantic/validationeval"
)

// destinationField records what the query actually returned, not an inferred
// value for an attribute absent from its result.
type destinationField struct {
	State string `json:"state"`
	Value any    `json:"value,omitempty"`
}

type destinationRow struct {
	OrderLineID string                      `json:"orderLineId"`
	Fields      map[string]destinationField `json:"fields"`
}

// destinationRead is the first read-only runtime result of tei view. A rule
// over the whole relation is never reported satisfied just because the query
// returned no duplicate: the current binding does not certify completeness.
type destinationRead struct {
	Application       core.CanonicalID             `json:"application"`
	Pattern           core.CanonicalID             `json:"pattern"`
	Environment       core.CanonicalID             `json:"environment"`
	Coverage          string                       `json:"coverage"`
	Rows              []destinationRow             `json:"rows"`
	PerLine           map[string]map[string]string `json:"perLine"`
	PerOrder          map[string]string            `json:"perOrder"`
	PerLineUndecided  map[string]map[string]string `json:"perLineUndecided,omitempty"`
	PerOrderUndecided map[string]string            `json:"perOrderUndecided,omitempty"`
}

func destinationDecision(d validationeval.Decision) (answer, reason string) {
	if d.Undecided != nil {
		return "undecided", d.Undecided.String()
	}
	return string(d.Result), ""
}

// readDestinationOrderLines is deliberately limited to NOW.md's first
// destination. It reuses the actual Workspace's Pattern, Environment,
// Contract Query, selected executor and Application rules. It does not
// publish a general-purpose database connection from the CLI.
func readDestinationOrderLines(
	world ir.World,
	appID, patternID, envID core.CanonicalID,
	inputs map[string]string,
) (destinationRead, error) {
	if appID != "order_entry" || patternID != "order_entry.order_lines" || envID != "development" {
		return destinationRead{}, fmt.Errorf("runtime view currently supports only order_entry.order_lines in development")
	}
	if len(inputs) != 1 {
		return destinationRead{}, fmt.Errorf("runtime view requires exactly one -input order_id=<integer>")
	}
	rawID, ok := inputs["order_id"]
	if !ok {
		return destinationRead{}, fmt.Errorf("runtime view requires -input order_id=<integer>")
	}
	orderID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || orderID <= 0 {
		return destinationRead{}, fmt.Errorf("order_id must be a positive 64-bit integer")
	}

	resolver := driver.NewResolver(world)
	prepared, err := resolver.ResolveHeaderDetailContractQuery(appID, patternID)
	if err != nil {
		return destinationRead{}, err
	}
	if len(prepared.Binding.Inputs) != 1 ||
		prepared.Binding.Inputs[0].SourceAttribute != "sales.order.order_id" {
		return destinationRead{}, fmt.Errorf("order header input is not explicitly bound to sales.order.order_id")
	}

	var selected *environment.Environment
	for i := range world.Environments {
		if world.Environments[i].ID == envID {
			if selected != nil {
				return destinationRead{}, fmt.Errorf("environment %s is declared more than once", envID)
			}
			selected = &world.Environments[i]
		}
	}
	if selected == nil {
		return destinationRead{}, fmt.Errorf("environment %s is not declared", envID)
	}
	var instanceID core.CanonicalID
	for _, binding := range selected.Bindings {
		if binding.Application == appID && binding.Alias == prepared.Datasource.Alias {
			if instanceID != "" {
				return destinationRead{}, fmt.Errorf("database binding is ambiguous for %s", prepared.Datasource.Alias)
			}
			instanceID = binding.Instance
		}
	}
	instance, ok := selected.InstanceByID(instanceID)
	if !ok || instance.Provider != environment.ProviderPostgres || instance.Definition != prepared.Binding.Target.Database {
		return destinationRead{}, fmt.Errorf("the selected database is not an explicitly bound PostgreSQL instance")
	}
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		return destinationRead{}, fmt.Errorf("TEI_VIEW_PG_DSN must be set outside the Workspace")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return destinationRead{}, fmt.Errorf("PostgreSQL connection initialization failed")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return destinationRead{}, fmt.Errorf("PostgreSQL connection failed")
	}
	// A Workspace's SQL realization is selected by its Contract binding.
	// Run it inside a server-enforced read-only transaction, not merely under
	// a CLI option named "read". Never allow an edited realization to write.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return destinationRead{}, fmt.Errorf("cannot begin a read-only PostgreSQL transaction")
	}
	defer tx.Rollback()

	executor := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: world.ContractQuerySQLRealizations,
		Runner:       postgresquery.Runner{DB: tx},
	}
	registry := driver.ContractQueryExecutorMap{}
	for _, binding := range world.ContractQueryExecutorBindings {
		if binding.Environment != envID || binding.Instance != instance.ID || binding.Target != prepared.Binding.Target {
			continue
		}
		registry[driver.ContractQueryExecutorRuntimeKey{
			Environment: binding.Environment,
			Instance:    binding.Instance,
			Capability:  binding.Executor,
		}] = executor
	}
	source := queryresult.MappedResult{
		Columns: []queryresult.Column{{Raw: "order_id", Attribute: "sales.order.order_id"}},
		Rows:    [][]any{{orderID}},
	}
	execution, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx, envID, appID, patternID, source, 0, registry,
	)
	if err != nil {
		// External PostgreSQL errors may contain connection details. Do not
		// put an unredacted driver error into the public JSON report.
		return destinationRead{}, fmt.Errorf("Contract Query execution failed; check the environment binding and database")
	}
	// Neither the row count nor the SQL's lack of LIMIT proves complete
	// relation coverage. The binding must say so before it can be used.
	detail, err := validationread.FromHeaderDetail(world.Semantics, execution.Execution, "")
	if err != nil {
		return destinationRead{}, err
	}
	rules, err := resolver.ResolveApplicationValidations(appID)
	if err != nil {
		return destinationRead{}, err
	}

	result := destinationRead{
		Application: appID,
		Pattern:     patternID,
		Environment: envID,
		Coverage:    "unverified",
		Rows:        make([]destinationRow, 0, len(execution.Execution.Result.Rows)),
		PerLine:     map[string]map[string]string{},
		PerOrder:    map[string]string{},
	}
	ids := make([]string, 0, len(execution.Execution.Result.Rows))
	for _, row := range execution.Execution.Result.Rows {
		rawID, exists := row.Attributes["sales.order_line.order_line_id"]
		id, typed := rawID.(int64)
		if !exists || !typed {
			return destinationRead{}, fmt.Errorf("the Contract Query did not return a typed order line identity")
		}
		key := strconv.FormatInt(id, 10)
		if _, duplicate := result.PerLine[key]; duplicate {
			return destinationRead{}, fmt.Errorf("the Contract Query returned a duplicate order line identity")
		}
		ids = append(ids, key)
		result.PerLine[key] = map[string]string{}
		item := destinationRow{OrderLineID: key, Fields: map[string]destinationField{}}
		for _, attribute := range execution.Execution.Resolution.Relation.Target.Attributes {
			value, held := row.Attributes[attribute.ID]
			field := destinationField{State: "not-fetched"}
			if held {
				if value == nil {
					field.State = "null"
				} else {
					field.State, field.Value = "present", value
				}
			}
			item.Fields[string(attribute.ID)] = field
		}
		result.Rows = append(result.Rows, item)
	}
	for _, rule := range rules {
		decisions, err := detail.Decide(rule, validationeval.Operators{})
		if err != nil {
			return destinationRead{}, fmt.Errorf("cannot decide rule %s: %w", rule.ID, err)
		}
		if decisions.Relation != nil {
			answer, reason := destinationDecision(*decisions.Relation)
			result.PerOrder[string(rule.ID)] = answer
			if reason != "" {
				if result.PerOrderUndecided == nil {
					result.PerOrderUndecided = map[string]string{}
				}
				result.PerOrderUndecided[string(rule.ID)] = reason
			}
			continue
		}
		if len(decisions.Rows) != len(ids) {
			return destinationRead{}, fmt.Errorf("rule %s did not return one answer for each observed row", rule.ID)
		}
		for i, decision := range decisions.Rows {
			answer, reason := destinationDecision(decision)
			result.PerLine[ids[i]][string(rule.ID)] = answer
			if reason != "" {
				if result.PerLineUndecided == nil {
					result.PerLineUndecided = map[string]map[string]string{}
				}
				if result.PerLineUndecided[ids[i]] == nil {
					result.PerLineUndecided[ids[i]] = map[string]string{}
				}
				result.PerLineUndecided[ids[i]][string(rule.ID)] = reason
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return destinationRead{}, fmt.Errorf("read-only PostgreSQL transaction did not complete")
	}
	return result, nil
}
