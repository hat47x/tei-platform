package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/environment"
	"tei/internal/functioncontext"
	"tei/internal/ir"
)

// browseReadResult is the `view -execute` result for a browse Pattern. It carries
// the Function Data Context (what was read, under which conditions, how much of
// the target set it covers) and no connection details.
type browseReadResult struct {
	Kind        string                  `json:"kind"`
	Application core.CanonicalID        `json:"application"`
	Pattern     core.CanonicalID        `json:"pattern"`
	DataContext functioncontext.Context `json:"dataContext"`
}

var dsnNameUnsafe = regexp.MustCompile(`[^A-Za-z0-9]`)

// browseDSNVariable names the environment variable that holds the connection
// string of one Database instance. The DSN is never a flag (it would leak into
// process listings and shell history) and never falls back to a test variable.
func browseDSNVariable(instance core.CanonicalID) string {
	return "TEI_DSN_" + strings.ToUpper(dsnNameUnsafe.ReplaceAllString(string(instance), "_"))
}

// browseInputs decodes -input pairs by the Query's declared input types.
// An undeclared name or an unsupported type stops the read; nothing is guessed.
func browseInputs(inputs []queryInput, pairs map[string]string) (map[string]any, string, error) {
	declared := map[string]core.CanonicalID{}
	for _, input := range inputs {
		declared[input.name] = input.typ
	}
	out := make(map[string]any, len(pairs))
	for name, raw := range pairs {
		typ, ok := declared[name]
		if !ok {
			return nil, "INVALID_ARGUMENT", fmt.Errorf("-input %s is not an input of the pattern's query", name)
		}
		switch typ {
		case "std.integer":
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return nil, "INVALID_ARGUMENT", fmt.Errorf("-input %s must be an integer", name)
			}
			out[name] = n
		case "std.code", "std.text", "std.identifier":
			out[name] = raw
		default:
			return nil, "INPUT_TYPE_UNSUPPORTED", fmt.Errorf("-input %s has type %s, which view -execute does not decode", name, typ)
		}
	}
	return out, "", nil
}

type queryInput struct {
	name string
	typ  core.CanonicalID
}

// openReadOnlyPostgres opens one connection whose every session is read-only
// (set as a connection parameter, then confirmed with SHOW) before any statement runs.
func openReadOnlyPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("connection string for the Database instance is not valid")
	}
	if config.RuntimeParams == nil {
		config.RuntimeParams = map[string]string{}
	}
	config.RuntimeParams["default_transaction_read_only"] = "on"
	db := stdlib.OpenDB(*config)
	db.SetMaxOpenConns(1)
	var readOnly string
	if err := db.QueryRowContext(ctx, "SHOW transaction_read_only").Scan(&readOnly); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cannot confirm a read-only PostgreSQL session")
	}
	if readOnly != "on" {
		_ = db.Close()
		return nil, fmt.Errorf("refusing to run: the PostgreSQL session is not read-only")
	}
	return db, nil
}

// runViewBrowseRead executes a browse Pattern through the Environment binding and
// reports its Function Data Context. Failures are failures, never an empty result.
func runViewBrowseRead(
	world ir.World,
	applicationID, patternID, environmentID core.CanonicalID,
	pairs map[string]string,
	report cliViewReport,
	out io.Writer,
	fail func(code, message string, exit int) (int, error),
) (int, error) {
	resolver := driver.NewResolver(world)
	env, instance, resolved, err := resolver.ResolveBrowseQueryInstance(environmentID, applicationID, patternID)
	if err != nil {
		return fail("VIEW_FAILED", err.Error(), 1)
	}
	_ = env
	if resolved.Pattern.Kind != application.PatternBrowse {
		return fail("INVALID_ARGUMENT", "view -execute without -command requires a browse pattern", 2)
	}
	declared := make([]queryInput, 0, len(resolved.Query.Inputs))
	for _, input := range resolved.Query.Inputs {
		declared = append(declared, queryInput{name: string(input.Name), typ: input.Type})
	}
	requested, code, err := browseInputs(declared, pairs)
	if err != nil {
		exit := 2
		if code == "INPUT_TYPE_UNSUPPORTED" {
			exit = 1
		}
		return fail(code, err.Error(), exit)
	}
	if instance.Provider != environment.ProviderPostgres {
		return fail("PROVIDER_EXECUTOR_UNAVAILABLE", fmt.Sprintf("no read executor for provider %s", instance.Provider), 1)
	}
	variable := browseDSNVariable(instance.ID)
	dsn := os.Getenv(variable)
	if strings.TrimSpace(dsn) == "" {
		return fail("DATASOURCE_CONNECTION_UNAVAILABLE", fmt.Sprintf("connection for Database instance %s is not configured: set %s", instance.ID, variable), 1)
	}
	if other := conflictingDSNVariable(world, environmentID, instance.ID); other != "" {
		return fail("DATASOURCE_CONNECTION_UNAVAILABLE", fmt.Sprintf("Database instances %s and %s map to the same variable %s", instance.ID, other, variable), 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := openReadOnlyPostgres(ctx, dsn)
	if err != nil {
		return fail("DATASOURCE_CONNECTION_UNAVAILABLE", err.Error(), 1)
	}
	defer db.Close()

	executor := postgresquery.Executor{DB: db, Database: instance.Definition}
	execution, err := resolver.ExecuteBrowseQueryInEnvironment(ctx, environmentID, applicationID, patternID, executor, requested)
	if err != nil {
		return fail("BROWSE_READ_FAILED", err.Error(), 1)
	}
	var parameterNames []string
	if payload, ok := execution.Execution.Result.Payload.(postgresquery.Result); ok {
		parameterNames = payload.ParameterNames
	}
	dataContext, err := functioncontext.FromBrowseExecution(execution, requested, parameterNames, time.Now())
	if err != nil {
		return fail("BROWSE_READ_FAILED", err.Error(), 1)
	}
	report.OK = true
	report.Result = browseReadResult{Kind: "browse-read", Application: applicationID, Pattern: patternID, DataContext: dataContext}
	return writeViewReport(out, report, 0)
}

// conflictingDSNVariable returns another instance of the Environment whose
// variable name is identical, so one connection string cannot serve two instances.
func conflictingDSNVariable(world ir.World, environmentID, instance core.CanonicalID) core.CanonicalID {
	for _, env := range world.Environments {
		if env.ID != environmentID {
			continue
		}
		for _, other := range env.Instances {
			if other.ID != instance && browseDSNVariable(other.ID) == browseDSNVariable(instance) {
				return other.ID
			}
		}
	}
	return ""
}
