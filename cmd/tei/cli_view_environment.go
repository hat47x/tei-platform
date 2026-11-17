package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
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
	"tei/internal/realization"
	"tei/internal/semantic/validationeval"
)

// destinationField records what the query actually returned, not an inferred
// value for an attribute absent from its result.
type destinationField struct {
	State string `json:"state"`
	Value any    `json:"value,omitempty"`
}

type destinationRow struct {
	// Key identifies the row by the primary key the persistence mapping of
	// the Data declares, joined with "/" when the key has more than one
	// attribute. It is never the position of the row.
	Key    string                      `json:"key"`
	Fields map[string]destinationField `json:"fields"`
}

// destinationRead is the read-only runtime result of `tei view -environment`.
// It names no Application, Pattern, Environment or Data of a particular
// Workspace: everything is resolved from the Workspace that was loaded.
// Because the current Contract Query binding does not certify that every
// relation row was fetched, it conservatively reports partial coverage.
// A witnessed duplicate still invalidates uniqueness, but absence of
// duplicates alone cannot establish it.
type destinationRead struct {
	Application          core.CanonicalID             `json:"application"`
	Pattern              core.CanonicalID             `json:"pattern"`
	Environment          core.CanonicalID             `json:"environment"`
	Relation             core.CanonicalID             `json:"relation"`
	KeyAttributes        []core.CanonicalID           `json:"keyAttributes"`
	Coverage             string                       `json:"coverage"`
	Rows                 []destinationRow             `json:"rows"`
	PerRow               map[string]map[string]string `json:"perRow"`
	PerRelation          map[string]string            `json:"perRelation"`
	PerRowUndecided      map[string]map[string]string `json:"perRowUndecided,omitempty"`
	PerRelationUndecided map[string]string            `json:"perRelationUndecided,omitempty"`
}

func destinationDecision(d validationeval.Decision) (answer, reason string) {
	if d.Undecided != nil {
		return "undecided", d.Undecided.String()
	}
	return string(d.Result), ""
}

// readEnvironmentRows reads the detail rows of one header-detail Pattern
// from the PostgreSQL instance the Environment binds, and answers every rule
// the Application uses. It reuses the Workspace's Pattern, Environment,
// Contract Query, selected executor and Application rules. It does not
// publish a general-purpose database connection from the CLI.
func readEnvironmentRows(
	world ir.World,
	appID, patternID, envID core.CanonicalID,
	inputs map[string]string,
) (destinationRead, error) {
	resolver := driver.NewResolver(world)
	prepared, err := resolver.ResolveHeaderDetailContractQuery(appID, patternID)
	if err != nil {
		return destinationRead{}, err
	}
	source, err := environmentHeaderSource(prepared.Binding.Inputs, inputs)
	if err != nil {
		return destinationRead{}, fmt.Errorf("%w: %v", errViewEnvironmentInput, err)
	}
	keyAttributes, err := environmentRowKey(world, prepared.Binding.Target.Database, prepared.Relation.Target.ID)
	if err != nil {
		return destinationRead{}, err
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
	execution, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx, envID, appID, patternID, source, 0, registry,
	)
	if err != nil {
		// External PostgreSQL errors may contain connection details. Do not
		// put an unredacted driver error into the public JSON report.
		return destinationRead{}, fmt.Errorf("Contract Query execution failed; check the environment binding and database")
	}
	// Neither row count nor an SQL query without LIMIT proves complete
	// relation coverage. An unknown coverage string is rejected by Decide;
	// until a binding certifies completeness, report the conservative partial
	// state and preserve its undecided reasons.
	detail, err := validationread.FromHeaderDetail(world.Semantics, execution.Execution, validationeval.CoveragePartial)
	if err != nil {
		return destinationRead{}, err
	}
	rules, err := resolver.ResolveApplicationValidations(appID)
	if err != nil {
		return destinationRead{}, err
	}

	result := destinationRead{
		Application:   appID,
		Pattern:       patternID,
		Environment:   envID,
		Relation:      prepared.Relation.Relation.ID,
		KeyAttributes: keyAttributes,
		Coverage:      string(validationeval.CoveragePartial),
		Rows:          make([]destinationRow, 0, len(execution.Execution.Result.Rows)),
		PerRow:        map[string]map[string]string{},
		PerRelation:   map[string]string{},
	}
	ids := make([]string, 0, len(execution.Execution.Result.Rows))
	for _, row := range execution.Execution.Result.Rows {
		key, err := environmentRowKeyValue(row.Attributes, keyAttributes)
		if err != nil {
			return destinationRead{}, err
		}
		if _, duplicate := result.PerRow[key]; duplicate {
			return destinationRead{}, fmt.Errorf("the Contract Query returned two rows with the key %s", key)
		}
		ids = append(ids, key)
		result.PerRow[key] = map[string]string{}
		item := destinationRow{Key: key, Fields: map[string]destinationField{}}
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
			result.PerRelation[string(rule.ID)] = answer
			if reason != "" {
				if result.PerRelationUndecided == nil {
					result.PerRelationUndecided = map[string]string{}
				}
				result.PerRelationUndecided[string(rule.ID)] = reason
			}
			continue
		}
		if len(decisions.Rows) != len(ids) {
			return destinationRead{}, fmt.Errorf("rule %s did not return one answer for each observed row", rule.ID)
		}
		for i, decision := range decisions.Rows {
			answer, reason := destinationDecision(decision)
			result.PerRow[ids[i]][string(rule.ID)] = answer
			if reason != "" {
				if result.PerRowUndecided == nil {
					result.PerRowUndecided = map[string]map[string]string{}
				}
				if result.PerRowUndecided[ids[i]] == nil {
					result.PerRowUndecided[ids[i]] = map[string]string{}
				}
				result.PerRowUndecided[ids[i]][string(rule.ID)] = reason
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return destinationRead{}, fmt.Errorf("read-only PostgreSQL transaction did not complete")
	}
	return result, nil
}

// errViewEnvironmentInput marks an -input the Contract Query cannot take. It
// is an argument error of the caller, not a failure of the read.
var errViewEnvironmentInput = errors.New("invalid -input")

// environmentHeaderSource builds the header record the detail is read for,
// from -input name=value pairs. Every input the binding declares must be
// given, and nothing else. An integer literal is handed over as an integer,
// anything else as text: the binding does not state a storage type for the
// header value, so no other conversion is attempted.
func environmentHeaderSource(declared []realization.HeaderDetailContractQueryInputBinding, given map[string]string) (queryresult.MappedResult, error) {
	source := queryresult.MappedResult{Rows: [][]any{{}}}
	names := make([]string, 0, len(declared))
	for _, input := range declared {
		names = append(names, string(input.Input))
	}
	for _, input := range declared {
		text, ok := given[string(input.Input)]
		if !ok || text == "" {
			return queryresult.MappedResult{}, fmt.Errorf("the Contract Query needs -input %s=<value>; its inputs are %s", input.Input, strings.Join(names, ", "))
		}
		var value any = text
		if number, err := strconv.ParseInt(text, 10, 64); err == nil {
			value = number
		}
		source.Columns = append(source.Columns, queryresult.Column{Raw: string(input.Input), Attribute: input.SourceAttribute})
		source.Rows[0] = append(source.Rows[0], value)
	}
	for name := range given {
		if !slices.Contains(names, name) {
			return queryresult.MappedResult{}, fmt.Errorf("-input %s is not an input of the Contract Query; its inputs are %s", name, strings.Join(names, ", "))
		}
	}
	return source, nil
}

// environmentRowKey returns the primary key the persistence mapping of data
// in database declares. A row is pointed to by that key in the result, so a
// Data without exactly one mapping with a primary key is refused rather
// than pointed to by its position.
func environmentRowKey(world ir.World, database, data core.CanonicalID) ([]core.CanonicalID, error) {
	var key []core.CanonicalID
	found := 0
	for _, mapping := range world.Persistence {
		if mapping.Database == database && mapping.SemanticData == data {
			key = mapping.PrimaryKey
			found++
		}
	}
	if found != 1 || len(key) == 0 {
		return nil, fmt.Errorf("Data %s needs exactly one persistence mapping with a primary key in database %s to point to its rows; found %d", data, database, found)
	}
	return key, nil
}

func environmentRowKeyValue(attributes map[core.CanonicalID]any, key []core.CanonicalID) (string, error) {
	parts := make([]string, 0, len(key))
	for _, attribute := range key {
		value, held := attributes[attribute]
		if !held || value == nil {
			return "", fmt.Errorf("the Contract Query did not return the key attribute %s", attribute)
		}
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, "/"), nil
}
