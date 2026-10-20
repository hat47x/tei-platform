// Command tei-ddl-entry-pgcheck applies the DDL that reference-ddl-entry and
// the PostgreSQL Provider render to a real, disposable PostgreSQL database and
// reports what the catalog shows afterwards (#1803). It drops and recreates the
// schemas public and tei_ddl_pgcheck, so point it only at a throwaway database.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	dbprovider "tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
	ddlentry "tei/tools/reference-ddl-entry"
)

type stepResult struct {
	Case         string                `json:"case"`
	Statements   []string              `json:"statements,omitempty"`
	Unresolved   []ddlentry.Unresolved `json:"unresolved,omitempty"`
	ExecError    string                `json:"execError,omitempty"`
	FailedSQL    string                `json:"failedSQL,omitempty"`
	Constraints  []string              `json:"constraints,omitempty"`
	Indexes      []string              `json:"indexes,omitempty"`
	ResidualDiff *schemadiff.Diff      `json:"residualDiff,omitempty"`
	ResidualSQL  []string              `json:"residualSQL,omitempty"`
	ResidualErr  string                `json:"residualRenderError,omitempty"`
	Converged    bool                  `json:"converged"`
	RerunError   string                `json:"rerunError,omitempty"`
}

var (
	ctx context.Context
	db  *sql.DB
)

func main() {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dsn := os.Getenv("TEI_TEST_POSTGRES_DSN")
	if dsn == "" || len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: TEI_TEST_POSTGRES_DSN=... tei-ddl-entry-pgcheck <workspace>")
		os.Exit(2)
	}
	var err error
	if db, err = postgresdb.OpenSQL(dsn); err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
	var version string
	_ = db.QueryRowContext(ctx, "SHOW server_version").Scan(&version)

	loaded, err := workspace.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	var results []stepResult

	// A: reference/workspace sales_core, statements in ddl-entry output order.
	results = append(results, runEntry("A reference/workspace sales_core (ddl-entry order)", loaded.World, "sales_core", true))

	// B: synthetic mappings with PK / unique (named, unnamed) / FK (named,
	// unnamed, composite), where alpha references beta (later in name order).
	syn := ir.World{Persistence: syntheticMappings("tei_ddl_pgcheck.")}
	results = append(results, runEntry("B1 synthetic, ddl-entry per-table order", syn, "pgcheck", true))
	results = append(results, runWholeDiff("B2 synthetic, one diff for all tables (Provider order)", syn, "pgcheck"))

	// C: same mappings with unqualified table names (as reference/workspace).
	unq := ir.World{Persistence: syntheticMappings("")}
	results = append(results, runEntry("C synthetic, unqualified table names", unq, "pgcheck", false))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"serverVersion": version, "results": results})
}

func reset() {
	if _, err := db.ExecContext(ctx, `DROP SCHEMA IF EXISTS tei_ddl_pgcheck CASCADE; DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public; CREATE SCHEMA tei_ddl_pgcheck;`); err != nil {
		panic(err)
	}
}

func runEntry(name string, world ir.World, database string, apply bool) stepResult {
	reset()
	res, err := ddlentry.Render(world, database, environment.ProviderPostgres)
	if err != nil {
		return stepResult{Case: name, ExecError: "render: " + err.Error()}
	}
	var statements []string
	for _, t := range res.Tables {
		statements = append(statements, t.Statements...)
	}
	out := stepResult{Case: name, Statements: statements, Unresolved: res.Unresolved}
	if !apply {
		return out
	}
	return applyAndObserve(out, world, database)
}

func runWholeDiff(name string, world ir.World, database string) stepResult {
	reset()
	desired, err := realization.RealizeDesired(world.Persistence, database, environment.ProviderPostgres)
	if err != nil {
		panic(err)
	}
	diff, err := schemadiff.Compare(desired, physical.SchemaSnapshot{Database: desired.Database})
	if err != nil {
		panic(err)
	}
	statements, err := migration.Render(environment.ProviderPostgres, diff)
	if err != nil {
		return stepResult{Case: name, ExecError: "render: " + err.Error()}
	}
	return applyAndObserve(stepResult{Case: name, Statements: statements}, world, database)
}

func applyAndObserve(out stepResult, world ir.World, database string) stepResult {
	for _, s := range out.Statements {
		if _, err := db.ExecContext(ctx, s); err != nil {
			out.ExecError, out.FailedSQL = err.Error(), s
			return out
		}
	}
	out.Constraints = queryLines(`SELECT n.nspname||'.'||c.relname||' '||con.contype::text||' '||con.conname||' '||pg_get_constraintdef(con.oid)
FROM pg_constraint con JOIN pg_class c ON c.oid=con.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname IN ('public','tei_ddl_pgcheck') AND con.contype IN ('p','u','f') ORDER BY 1`)
	out.Indexes = queryLines(`SELECT schemaname||'.'||indexname||' '||indexdef FROM pg_indexes WHERE schemaname IN ('public','tei_ddl_pgcheck') ORDER BY 1`)

	desired, err := realization.RealizeDesired(world.Persistence, database, environment.ProviderPostgres)
	if err != nil {
		panic(err)
	}
	provider := postgres.Provider{Catalog: postgres.SQLCatalogReader{DB: db}, Now: time.Now}
	instance := environment.DatabaseInstance{ID: core.MustCanonicalID("pgcheck.local"), Definition: desired.Database, Provider: environment.ProviderPostgres}
	actual, err := dbprovider.Introspect(ctx, provider, instance)
	if err != nil {
		panic(err)
	}
	residual, err := schemadiff.Compare(desired, actual)
	if err != nil {
		panic(err)
	}
	out.Converged = len(residual.Changes) == 0 && len(residual.Unresolved) == 0
	if !out.Converged {
		out.ResidualDiff = &residual
		if sqls, err := migration.Render(environment.ProviderPostgres, residual); err != nil {
			out.ResidualErr = err.Error()
		} else {
			out.ResidualSQL = sqls
		}
	}
	// Re-run the same script on the populated database (idempotency is not claimed).
	for _, s := range out.Statements {
		if _, err := db.ExecContext(ctx, s); err != nil {
			out.RerunError = err.Error() + " :: " + s
			break
		}
	}
	return out
}

func queryLines(q string) []string {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			panic(err)
		}
		out = append(out, s)
	}
	return out
}

func col(table, name string, req bool) persistence.ColumnMapping {
	n := persistence.Optional
	if req {
		n = persistence.Required
	}
	return persistence.ColumnMapping{Attribute: core.CanonicalID("s." + table + "." + name), Column: name, Storage: persistence.StorageVarchar, Nullability: n}
}

func attr(table, name string) core.CanonicalID { return core.CanonicalID("s." + table + "." + name) }

// syntheticMappings: alpha -> beta (FK to a table later in name order),
// gamma -> alpha (named FK), gamma has a composite PK, unique constraints are
// both named and unnamed.
func syntheticMappings(prefix string) []persistence.Mapping {
	return []persistence.Mapping{
		{
			ID: "m.alpha", Database: "pgcheck", SemanticData: "s.alpha", Table: prefix + "alpha",
			Columns:           []persistence.ColumnMapping{col("alpha", "code", true), col("alpha", "label", true), col("alpha", "beta_code", false)},
			PrimaryKey:        []core.CanonicalID{attr("alpha", "code")},
			UniqueConstraints: []persistence.UniqueConstraintMapping{{Name: "alpha_label_named_uq", Attributes: []core.CanonicalID{attr("alpha", "label")}}},
			ForeignKeys:       []persistence.ForeignKeyMapping{{Attributes: []core.CanonicalID{attr("alpha", "beta_code")}, ReferencedMapping: "m.beta", ReferencedAttributes: []core.CanonicalID{attr("beta", "code")}}},
		},
		{
			ID: "m.beta", Database: "pgcheck", SemanticData: "s.beta", Table: prefix + "beta",
			Columns:           []persistence.ColumnMapping{col("beta", "code", true), col("beta", "name", true)},
			PrimaryKey:        []core.CanonicalID{attr("beta", "code")},
			UniqueConstraints: []persistence.UniqueConstraintMapping{{Attributes: []core.CanonicalID{attr("beta", "name")}}},
		},
		{
			ID: "m.gamma", Database: "pgcheck", SemanticData: "s.gamma", Table: prefix + "gamma",
			Columns:           []persistence.ColumnMapping{col("gamma", "code", true), col("gamma", "seq", true), col("gamma", "alpha_code", true), col("gamma", "note", false)},
			PrimaryKey:        []core.CanonicalID{attr("gamma", "code"), attr("gamma", "seq")},
			UniqueConstraints: []persistence.UniqueConstraintMapping{{Attributes: []core.CanonicalID{attr("gamma", "alpha_code"), attr("gamma", "note")}}},
			ForeignKeys:       []persistence.ForeignKeyMapping{{Name: "gamma_alpha_named_fk", Attributes: []core.CanonicalID{attr("gamma", "alpha_code")}, ReferencedMapping: "m.alpha", ReferencedAttributes: []core.CanonicalID{attr("alpha", "code")}}},
		},
	}
}
