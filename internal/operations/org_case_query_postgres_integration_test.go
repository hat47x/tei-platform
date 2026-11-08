package operations

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/workspace"
)

// verifyNativeCaseReadWithPostgres is opt-in. It uses the very same native
// CaseRecord Query + Browse Pattern loaded by the E1f Go characterization, not a second SQL
// interpretation or a new Source fixture. It is NOT a Function Data Context,
// scoped authorization, Host Grant or product-level read certification.
func verifyNativeCaseReadWithPostgres(t *testing.T, loaded workspace.Result) {
	t.Helper()
	dsn := os.Getenv("TEI_ORG_CASE_PG_DSN")
	if dsn == "" {
		t.Skip("physical organization CaseRecord PostgreSQL read NOT RUN: TEI_ORG_CASE_PG_DSN unset")
	}
	if os.Getenv("TEI_ORG_CASE_PG_CONFIRM_ISOLATED") != "tei_org_case_e2e" {
		t.Fatal("refusing to create a CaseRecord table without explicit disposable database confirmation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize PostgreSQL driver")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("isolated PostgreSQL connection unavailable")
	}
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); err != nil ||
		dbName != "tei_org_case_e2e" {
		t.Fatalf("refusing to create fixture in unexpected database: %q", dbName)
	}
	var preexisting sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.org_case_records')::text").Scan(&preexisting); err != nil {
		t.Fatal("cannot determine whether CaseRecord fixture table exists")
	}
	if preexisting.Valid {
		t.Fatal("refusing to modify or drop existing public.org_case_records")
	}
	var version string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&version); err != nil {
		t.Fatal("cannot determine actual PostgreSQL server version")
	}
	t.Logf("actual PostgreSQL CaseRecord read: db=%s server_version=%s", dbName, version)
	if _, err := db.ExecContext(ctx,
		"CREATE TABLE public.org_case_records (case_id VARCHAR PRIMARY KEY, handling_unit_id VARCHAR NOT NULL)"); err != nil {
		t.Fatal("cannot create isolated CaseRecord fixture")
	}
	defer func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := db.ExecContext(cleanupCtx, "DROP TABLE public.org_case_records"); err != nil {
			t.Errorf("cannot clean up disposable CaseRecord fixture: %v", err)
		}
	}()
	for _, entry := range [][2]string{{"C2", "D2"}, {"C1", "D1"}} {
		if _, err := db.ExecContext(ctx,
			"INSERT INTO public.org_case_records (case_id, handling_unit_id) VALUES ($1, $2)",
			entry[0], entry[1]); err != nil {
			t.Fatal("cannot seed isolated CaseRecord fixture")
		}
	}
	if len(loaded.World.Queries) != 1 || len(loaded.World.Applications) != 1 {
		t.Fatalf("ambiguous native CaseRecord query or application: %+v", loaded.World.Queries)
	}
	q := loaded.World.Queries[0]
	app := loaded.World.Applications[0]
	ds, ok := app.DatabaseByAliasExact(q.Datasource)
	if !ok || ds.TargetDatabase != core.CanonicalID("org_store") {
		t.Fatalf("native SQL datasource not resolved uniquely: %+v", ds)
	}
	executor := postgresquery.Executor{DB: db, Database: ds.TargetDatabase}
	resolver := driver.NewResolver(loaded.World)
	for _, tc := range []struct {
		name   string
		caseID string
		want   [][]any
	}{
		{name: "actual C1 handlingUnit D1", caseID: "C1", want: [][]any{{"C1", "D1"}}},
		{name: "actual C2 handlingUnit D2", caseID: "C2", want: [][]any{{"C2", "D2"}}},
		{name: "actual missing C3 is zero physical rows", caseID: "C3", want: [][]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			execution, err := resolver.ExecuteBrowseQuery(ctx, "org_view", "org_view.case_browse",
				executor, map[string]any{"case_id": tc.caseID})
			if err != nil {
				t.Fatalf("native Case Pattern/Query did not execute on actual PostgreSQL: %v", err)
			}
			if execution.Resolution.Query == nil || execution.Resolution.Query.ID != q.ID ||
				execution.Resolution.Datasource == nil ||
				execution.Resolution.Datasource.TargetDatabase != ds.TargetDatabase ||
				execution.Resolution.Data == nil ||
				execution.Resolution.Data.ID != "organization.case_record" {
				t.Fatalf("real DB result was not attributed to the native Case Pattern/Query/Data: %+v",
					execution.Resolution)
			}
			if !reflect.DeepEqual(execution.Result.Mapped.Rows, tc.want) {
				t.Fatalf("physical Case business identity/handlingUnit mismatch: actual=%+v want=%+v",
					execution.Result.Mapped.Rows, tc.want)
			}
		})
	}
}
