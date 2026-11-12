// reference-browse-read runs ONE browse Pattern of a Workspace against ONE PostgreSQL
// database through the Environment binding, and prints the resulting Function Data Context
// as JSON. It is a read-only reference Consumer entry, not the `tei` CLI and not a Host.
//
// The database must be explicitly confirmed as disposable: -confirm-isolated has to equal
// current_database(). The connection string is read from an environment variable named by
// -dsn-env and is never printed.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/functioncontext"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

const schemaVersion = "tei.reference-browse-read.v0"

const scope = "reference Consumer entry: one browse Pattern, one PostgreSQL, read-only transaction. " +
	"Binding is host-asserted; coverage is never established; not the tei CLI."

type response struct {
	OK            bool                     `json:"ok"`
	SchemaVersion string                   `json:"schemaVersion"`
	Scope         string                   `json:"scope"`
	Context       *functioncontext.Context `json:"context,omitempty"`
	Error         *failure                 `json:"error,omitempty"`
}

type failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type paramFlags []string

func (p *paramFlags) String() string     { return strings.Join(*p, ",") }
func (p *paramFlags) Set(v string) error { *p = append(*p, v); return nil }

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Getenv))
}

func run(args []string, out io.Writer, getenv func(string) string) int {
	r := response{SchemaVersion: schemaVersion, Scope: scope}
	fail := func(code string, err error) int {
		r.OK = false
		r.Error = &failure{Code: code, Message: err.Error()}
		_ = encode(out, r)
		return 1
	}

	fs := flag.NewFlagSet("reference-browse-read", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	workspacePath := fs.String("workspace", "", "Workspace root")
	environmentID := fs.String("environment", "", "Environment ID")
	applicationID := fs.String("application", "", "Application ID")
	patternID := fs.String("pattern", "", "browse Pattern ID")
	dsnEnv := fs.String("dsn-env", "", "name of the environment variable holding the PostgreSQL DSN")
	confirm := fs.String("confirm-isolated", "", "must equal current_database() of the disposable database")
	var params paramFlags
	fs.Var(&params, "param", "name=type:value, type is time (RFC3339), text or int; repeatable")
	if err := fs.Parse(args); err != nil {
		return fail("INVALID_ARGUMENT", err)
	}
	for name, v := range map[string]string{"workspace": *workspacePath, "environment": *environmentID,
		"application": *applicationID, "pattern": *patternID, "dsn-env": *dsnEnv, "confirm-isolated": *confirm} {
		if v == "" {
			return fail("INVALID_ARGUMENT", fmt.Errorf("-%s is required", name))
		}
	}
	for _, name := range []string{*environmentID, *applicationID, *patternID} {
		if _, err := core.ParseCanonicalID(name); err != nil {
			return fail("INVALID_ARGUMENT", err)
		}
	}
	values, shapes, err := parseParams(params)
	if err != nil {
		return fail("INVALID_PARAMETER", err)
	}

	loaded, err := workspace.Load(*workspacePath)
	if err != nil {
		return fail("WORKSPACE_LOAD_FAILED", err)
	}
	for _, d := range validation.Validate(loaded.World) {
		if d.Severity == validation.Error {
			return fail("WORKSPACE_INVALID", fmt.Errorf("%s: %s", d.Code, d.Message))
		}
	}
	envID, appID, patID := core.MustCanonicalID(*environmentID), core.MustCanonicalID(*applicationID), core.MustCanonicalID(*patternID)
	resolver := driver.NewResolver(loaded.World)
	// Resolve the binding before the DSN is read or any connection is opened.
	instance, err := resolver.ResolveBrowseQueryInstance(envID, appID, patID)
	if err != nil {
		return fail("BINDING_REFUSED", err)
	}

	dsn := getenv(*dsnEnv)
	if dsn == "" {
		return fail("CONNECTION_UNAVAILABLE", fmt.Errorf("environment variable %s is empty", *dsnEnv))
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		return fail("CONNECTION_UNAVAILABLE", errors.New("cannot open the database (details withheld)"))
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return fail("CONNECTION_UNAVAILABLE", errors.New("cannot start a read-only transaction (details withheld)"))
	}
	defer func() { _ = tx.Rollback() }()

	observed, err := postgresquery.ObserveConnection(ctx, tx)
	if err != nil {
		return fail("OBSERVATION_FAILED", err)
	}
	switch observed.Database {
	case "", "postgres", "template0", "template1":
		return fail("DATABASE_NOT_CONFIRMED", fmt.Errorf("refusing database %q: a dedicated disposable database is required", observed.Database))
	}
	if observed.Database != *confirm {
		return fail("DATABASE_NOT_CONFIRMED", fmt.Errorf("-confirm-isolated %q does not equal current_database() %q", *confirm, observed.Database))
	}
	if !observed.ReadOnly {
		return fail("DATABASE_NOT_CONFIRMED", errors.New("the transaction is not read-only"))
	}

	registry := driver.BrowseQueryExecutorMap{
		{Environment: envID, Instance: instance.ID}: postgresquery.Executor{DB: tx, Database: instance.Definition},
	}
	execution, err := resolver.ExecuteBrowseQueryInEnvironment(ctx, envID, appID, patID, registry, values)
	if err != nil {
		return fail("READ_FAILED", err)
	}
	declared := map[string]bool{}
	for _, in := range execution.Execution.Resolution.Query.Inputs {
		declared[string(in.Name)] = true
	}
	for i := range shapes {
		shapes[i].DeclaredByQuery = declared[shapes[i].Name]
	}
	fc, err := functioncontext.From(functioncontext.Input{Execution: execution, Connection: observed, Parameters: shapes})
	if err != nil {
		return fail("CONTEXT_FAILED", err)
	}
	r.OK, r.Context = true, &fc
	if err := encode(out, r); err != nil {
		return 1
	}
	return 0
}

// parseParams turns explicit "name=type:value" arguments into typed values. There is no
// type inference: an untagged value, an unknown tag or a repeated name is refused.
func parseParams(raw []string) (map[string]any, []functioncontext.ParameterShape, error) {
	values := map[string]any{}
	var shapes []functioncontext.ParameterShape
	for _, item := range raw {
		name, typed, ok := strings.Cut(item, "=")
		if !ok || name == "" {
			return nil, nil, fmt.Errorf("parameter %q is not name=type:value", item)
		}
		if _, dup := values[name]; dup {
			return nil, nil, fmt.Errorf("parameter %q is given twice", name)
		}
		tag, text, ok := strings.Cut(typed, ":")
		if !ok {
			return nil, nil, fmt.Errorf("parameter %q has no type tag", name)
		}
		switch tag {
		case "time":
			t, err := time.Parse(time.RFC3339Nano, text)
			if err != nil {
				return nil, nil, fmt.Errorf("parameter %q is not RFC3339: %w", name, err)
			}
			values[name] = t
		case "text":
			values[name] = text
		case "int":
			n, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("parameter %q is not an integer: %w", name, err)
			}
			values[name] = n
		default:
			return nil, nil, fmt.Errorf("parameter %q has unknown type tag %q", name, tag)
		}
		shapes = append(shapes, functioncontext.ParameterShape{Name: name, GoType: fmt.Sprintf("%T", values[name])})
	}
	return values, shapes, nil
}

func encode(out io.Writer, r response) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
