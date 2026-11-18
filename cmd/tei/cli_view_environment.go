package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
	"tei/internal/database/persistence"
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
	// the Data declares, encoded as a JSON array when the key has more than one
	// attribute. It is never the position of the row.
	Key    string                      `json:"key"`
	Fields map[string]destinationField `json:"fields"`
}

// destinationDeclaration is what the Workspace's header-detail Contract Query
// binding states about how many rows its Query returns.
//
// It is the author's statement, not an observation. This entry reads it and
// reports it, and does not verify it: the parent scope, the executed Query and
// the rows actually fetched are not cross-checked, so the declaration is never
// used as evidence for a rule. Reporting it is what makes "declared" and
// "checked" distinguishable; without the field a reader cannot tell whether
// coverage was measured or merely promised.
type destinationDeclaration struct {
	// CoversRelation is the Relation the binding says the Query returns every
	// row of, for the header record whose values were handed over as inputs.
	CoversRelation core.CanonicalID `json:"coversRelation,omitempty"`
	// CoversSome is true when the binding says the Query may return only some
	// rows (a search, a page).
	CoversSome bool `json:"coversSome,omitempty"`
	// Checked is false: this entry did not confirm the declaration. It is
	// present and always false so a reader is never left to infer a check that
	// did not happen.
	Checked bool `json:"checked"`
}

// destinationRead is the read-only runtime result of `tei view -environment`.
// It names no Application, Pattern, Environment or Data of a particular
// Workspace: everything is resolved from the Workspace that was loaded.
// Coverage is complete only when the binding states `covers <Relation>` and
// the Query that ran matches the Workspace's relation storage (see
// driver.QualifyHeaderDetailCoverage); otherwise it is partial and the report
// says why. A witnessed duplicate invalidates uniqueness either way; the
// absence of duplicates establishes it only when coverage is complete.
//
// The binding's own `covers` statement is reported separately in Declaration.
// It is not evidence on its own: Declaration.Checked is always false, and the
// qualification above is what decides Coverage.
type destinationRead struct {
	Application          core.CanonicalID             `json:"application"`
	Pattern              core.CanonicalID             `json:"pattern"`
	Environment          core.CanonicalID             `json:"environment"`
	Relation             core.CanonicalID             `json:"relation"`
	KeyAttributes        []core.CanonicalID           `json:"keyAttributes"`
	Coverage             string                       `json:"coverage"`
	CoverageDeclared     string                       `json:"coverageDeclared"`
	CoverageBasis        string                       `json:"coverageBasis,omitempty"`
	CoverageReason       string                       `json:"coverageReason,omitempty"`
	Declaration          *destinationDeclaration      `json:"declaration,omitempty"`
	Rows                 []destinationRow             `json:"rows"`
	PerRow               map[string]map[string]string `json:"perRow"`
	PerRelation          map[string]string            `json:"perRelation"`
	PerRowUndecided      map[string]map[string]string `json:"perRowUndecided,omitempty"`
	PerRelationUndecided map[string]string            `json:"perRelationUndecided,omitempty"`
}

// postgresPhysicalReadAllowsComplete verifies that the physical table used by
// the selected Query cannot silently hide rows through row-level security or a
// view/inheritance expansion. This is a conservative runtime qualification,
// not a replacement for the Workspace's static Query/storage proof.
//
// The catalog query runs in the same repeatable-read transaction as the data
// Query, so a separate connection or a later snapshot cannot qualify a read.
// Unknown tables and metadata failures leave the result partial.
func postgresPhysicalReadAllowsComplete(
	ctx context.Context,
	tx *sql.Tx,
	world ir.World,
	resolution driver.ResolvedHeaderDetailContractQuery,
) (bool, string) {
	var physicalTable string
	for _, mapping := range world.Persistence {
		if mapping.Database != resolution.Binding.Target.Database ||
			mapping.SemanticData != resolution.Relation.Target.ID {
			continue
		}
		if physicalTable != "" {
			return false, "the PostgreSQL persistence table is ambiguous"
		}
		physicalTable = mapping.Table
	}
	if physicalTable == "" {
		return false, "the PostgreSQL persistence table was not identified"
	}
	var kind string
	var rlsEnabled, rlsForced, hasChildren bool
	err := tx.QueryRowContext(ctx, `SELECT c.relkind::text, c.relrowsecurity,
		c.relforcerowsecurity, c.relhassubclass
		FROM pg_catalog.pg_class AS c
		WHERE c.oid = pg_catalog.to_regclass($1)`, physicalTable).
		Scan(&kind, &rlsEnabled, &rlsForced, &hasChildren)
	if err != nil {
		return false, "the physical PostgreSQL table could not be verified"
	}
	if kind != "r" || hasChildren {
		return false, "the physical PostgreSQL source is not a plain table without inherited rows"
	}
	if rlsEnabled || rlsForced {
		return false, "the PostgreSQL table enables row-level security and may hide relation rows"
	}
	return true, ""
}

// destinationDeclarationFrom reports the binding's own statement about how many
// rows its Query returns. Checked is false by construction: this entry does not
// cross-check the statement against the parent scope, the executed Query or the
// rows it received, so the value must not be read as an observation.
//
// An undeclared binding yields nil, which is distinct from a declaration that
// was made and not checked.
//
// A binding that states both "every row of this Relation" and "only some rows"
// is refused instead of reported. The static validator rejects it
// (HEADER_DETAIL_BINDING_COVERAGE_CONFLICT), but this read path runs no
// validator, so refusing here is the only place the contradiction can stop the
// read. Labelling it and continuing would answer rules against a binding that
// contradicts itself about the very thing the rule depends on.
//
// A binding whose `covers` names a Relation other than the one this read returns
// is refused for the same reason. The static validator reports it as
// HEADER_DETAIL_BINDING_COVERS_OTHER_RELATION, and this path runs no validator:
// measured before this check existed, the view accepted and reported a
// declaration about another Relation, which reads as a statement about the rows
// on the page when it is a statement about a different Relation entirely.
func destinationDeclarationFrom(binding realization.HeaderDetailContractQueryBinding, readRelation core.CanonicalID) (*destinationDeclaration, error) {
	if binding.CoversRelation != "" && binding.CoversSome {
		return nil, fmt.Errorf(
			"the header-detail Contract Query binding states both that it returns every row of %s and that it returns only some rows; this read cannot be taken as either",
			binding.CoversRelation,
		)
	}
	if binding.CoversRelation != "" && binding.CoversRelation != readRelation {
		return nil, fmt.Errorf(
			"the header-detail Contract Query binding states that it returns every row of %s, but this read returns the rows of %s; the declaration is about a different Relation",
			binding.CoversRelation,
			readRelation,
		)
	}
	if binding.CoversRelation == "" && !binding.CoversSome {
		return nil, nil
	}
	return &destinationDeclaration{
		CoversRelation: binding.CoversRelation,
		CoversSome:     binding.CoversSome,
		Checked:        false,
	}, nil
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
	source, err := environmentHeaderSource(world, prepared, inputs)
	if err != nil {
		return destinationRead{}, fmt.Errorf("%w: %v", errViewEnvironmentInput, err)
	}
	// Reject a binding whose source attribute or Query input is invalid before
	// opening a database connection. The ordinary runtime still prepares the
	// same binding again; this check only moves rejection before external I/O.
	if _, err := resolver.PrepareHeaderDetailContractQuery(appID, patternID, source, 0); err != nil {
		return destinationRead{}, fmt.Errorf("invalid header-detail Contract Query input binding: %w", err)
	}
	// Refuse a self-contradictory coverage declaration before opening the
	// database. The view path does not run static validation, and a read-only
	// query is still external I/O that an invalid binding must not trigger.
	declaration, err := destinationDeclarationFrom(prepared.Binding, prepared.Relation.Relation.ID)
	if err != nil {
		return destinationRead{}, err
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
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
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
	// Neither a row count, an SQL query without LIMIT, nor the binding's
	// declaration alone proves complete relation coverage. Only a Query that
	// matches the Workspace's relation storage does; anything else is partial.
	qualification := resolver.QualifyHeaderDetailCoverage(execution.Execution.Resolution, execution.Executor.Instance.Provider)
	// Matching the declared SQL to persistence does not prove that PostgreSQL
	// allowed the current role to see all matching rows. RLS, a view or an
	// inherited relation can remove rows without changing the authored Query.
	// Resolve the physical table on this same read-only transaction/snapshot.
	if qualification.Complete {
		if ok, reason := postgresPhysicalReadAllowsComplete(ctx, tx, world, execution.Execution.Resolution); !ok {
			qualification.Complete = false
			qualification.Basis = ""
			qualification.Reason = reason
		}
	}
	coverage := validationeval.CoveragePartial
	if qualification.Complete {
		coverage = validationeval.CoverageComplete
	}
	detail, err := validationread.FromHeaderDetail(world.Semantics, execution.Execution, coverage)
	if err != nil {
		return destinationRead{}, err
	}
	rules, err := resolver.ResolveApplicationValidations(appID)
	if err != nil {
		return destinationRead{}, err
	}

	result := destinationRead{
		Application:      appID,
		Pattern:          patternID,
		Environment:      envID,
		Relation:         prepared.Relation.Relation.ID,
		KeyAttributes:    keyAttributes,
		Coverage:         string(coverage),
		CoverageDeclared: qualification.Declared,
		CoverageBasis:    qualification.Basis,
		CoverageReason:   qualification.Reason,
		Declaration:      declaration,
		Rows:             make([]destinationRow, 0, len(execution.Execution.Result.Rows)),
		PerRow:           map[string]map[string]string{},
		PerRelation:      map[string]string{},
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

// environmentHeaderSource uses the Contract Query input bindings and the
// selected source Data's storage declarations. Numeric-looking text is never
// converted to an integer unless its column explicitly declares int64.
func environmentHeaderSource(world ir.World, resolved driver.ResolvedHeaderDetailContractQuery, given map[string]string) (queryresult.MappedResult, error) {
	declared := resolved.Binding.Inputs
	if len(declared) == 0 {
		return queryresult.MappedResult{}, fmt.Errorf("the Contract Query does not declare a source input binding")
	}
	source := queryresult.MappedResult{Rows: [][]any{{}}}
	names := make([]string, 0, len(declared))
	for _, input := range declared {
		if input.Input == "" || input.SourceAttribute == "" || slices.Contains(names, string(input.Input)) {
			return queryresult.MappedResult{}, fmt.Errorf("the Contract Query source input binding is incomplete or duplicated")
		}
		names = append(names, string(input.Input))
	}
	for _, input := range declared {
		raw, ok := given[string(input.Input)]
		if !ok || raw == "" {
			return queryresult.MappedResult{}, fmt.Errorf("the Contract Query needs -input %s=<value>; its inputs are %s", input.Input, strings.Join(names, ", "))
		}
		value, err := environmentHeaderInputValue(world, resolved.Database.ID, resolved.Relation.Source.ID, input.SourceAttribute, raw)
		if err != nil {
			return queryresult.MappedResult{}, fmt.Errorf("cannot read -input %s: %w", input.Input, err)
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

// environmentHeaderInputValue does not use the appearance of a literal to
// select its Go type. A physical int64 column gives an explicit conversion
// rule; text-like and unknown storage retain the original string. Conversion
// failure is reported rather than silently changing or truncating the input.
func environmentHeaderInputValue(world ir.World, databaseID, dataID, attributeID core.CanonicalID, raw string) (any, error) {
	found := false
	var storage persistence.StorageType
	for _, mapping := range world.Persistence {
		if mapping.Database != databaseID || mapping.SemanticData != dataID {
			continue
		}
		for _, column := range mapping.Columns {
			if column.Attribute != attributeID {
				continue
			}
			if found {
				return nil, fmt.Errorf("source attribute %s has multiple persistence column mappings", attributeID)
			}
			found = true
			storage = column.Storage
		}
	}
	if !found {
		// An external source need not declare physical storage. Keep the
		// input as written; a SQL/provider boundary may reject its type.
		return raw, nil
	}
	switch storage {
	case persistence.StorageInt64:
		number, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("source attribute %s needs an int64 value", attributeID)
		}
		return number, nil
	case persistence.StorageVarchar, persistence.StorageText, persistence.StorageDateTimeTZ,
		persistence.StorageDecimal, persistence.StorageJSON:
		return raw, nil
	default:
		return nil, fmt.Errorf("source attribute %s has unsupported storage type %s", attributeID, storage)
	}
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
	if len(parts) == 1 {
		return parts[0], nil
	}
	// JSON preserves the individual components: ["a/b","c"] differs from
	// ["a","b/c"]. Delimiter concatenation would collapse these keys.
	encoded, err := json.Marshal(parts)
	if err != nil {
		return "", fmt.Errorf("cannot encode a composite row key: %w", err)
	}
	return string(encoded), nil
}
