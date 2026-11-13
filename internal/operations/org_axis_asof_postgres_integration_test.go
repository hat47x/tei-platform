package operations

import (
    "context"
    "database/sql"
    "os"
    "path/filepath"
    "reflect"
    "testing"
    "time"

    _ "github.com/jackc/pgx/v5/stdlib"
    postgresquery "tei/internal/driver/queryruntime/postgres"
    "tei/internal/driver/queryresult"
    "tei/internal/workspace"
)

// This opt-in test runs the same native .tei Query against disposable PostgreSQL.
// It only verifies interval filtering for one row per Membership occurrence.
// Existing PK(membership_id) cannot preserve multiple known-time revisions of
// the SAME occurrence; that remains a separate, unaccepted design decision.
func TestOrgAxisAsOfActualPostgresOptIn(t *testing.T) {
    dsn := os.Getenv("TEI_ORG_AXIS_PG_DSN")
    if dsn == "" { t.Skip("real PostgreSQL as-of read NOT RUN: TEI_ORG_AXIS_PG_DSN unset") }
    const isolated = "tei_org_axis_e2e"
    if os.Getenv("TEI_ORG_AXIS_PG_CONFIRM_ISOLATED") != isolated {
        t.Fatal("isolated database confirmation absent; refusing to create fixture")
    }
    ctx, stop := context.WithTimeout(context.Background(), 60*time.Second)
    defer stop()
    db, err := sql.Open("pgx", dsn)
    if err != nil { t.Fatal(err) }
    defer db.Close()
    db.SetMaxOpenConns(1)
    if err := db.PingContext(ctx); err != nil { t.Fatal(err) }
    var dbName string
    if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); err != nil || dbName != isolated {
        t.Fatalf("refusing to change database %q: %v", dbName, err)
    }
    var preexisting sql.NullString
    if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.org_memberships')::text").Scan(&preexisting); err != nil {
        t.Fatal(err)
    }
    if preexisting.Valid { t.Fatal("refusing to modify pre-existing public.org_memberships") }
    ddl := "CREATE TABLE public.org_memberships (" +
        "membership_id varchar PRIMARY KEY, org_unit_id varchar NOT NULL, allocation numeric(8,2) NOT NULL, " +
        "valid_from timestamptz NOT NULL, valid_to timestamptz, known_from timestamptz NOT NULL, known_to timestamptz)"
    if _, err := db.ExecContext(ctx, ddl); err != nil { t.Fatal(err) }
    defer func() {
        cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        if _, err := db.ExecContext(cleanup, "DROP TABLE public.org_memberships"); err != nil {
            t.Errorf("disposable fixture cleanup failed: %v", err)
        }
    }()
    date := func(s string) any {
        if s == "" { return nil }
        tm, err := time.Parse("2006-01-02", s)
        if err != nil { t.Fatal(err) }
        return tm
    }
    // Each row has its own occurrence identity; this does NOT model historical
    // correction of A2 with one stable identity. These are literal test inputs.
    records := [][7]string{
        {"A1","D1","1.00","2027-01-01","","2027-04-01",""},
        {"A2","D1","0.60","2027-01-01","2027-06-01","2027-04-01",""},
        {"A3","D2","0.40","2027-01-01","","2027-04-01",""},
        {"A4","D2","1.00","2027-01-01","","2027-07-02",""},
        {"A5","D1","0.80","2027-06-15","2027-06-16","2027-01-01","2027-07-02"},
        {"A6","D2","1.20","2027-07-01","","2027-06-01",""},
        {"A7","D1","0.70","2027-01-01","","2027-07-03",""},
        {"A8","D2","0.50","2027-01-01","","2027-01-01","2027-07-01"},
        {"A9","D1","0.10","2027-01-01","","2027-01-01",""},
    }
    insert := "INSERT INTO public.org_memberships " +
        "(membership_id,org_unit_id,allocation,valid_from,valid_to,known_from,known_to) VALUES ($1,$2,$3,$4,$5,$6,$7)"
    for _, r := range records {
        if _, err := db.ExecContext(ctx, insert, r[0],r[1],r[2],date(r[3]),date(r[4]),date(r[5]),date(r[6])); err != nil {
            t.Fatalf("insert %s: %v", r[0], err)
        }
    }
    loaded, err := workspace.Load(filepath.Join("..","..","reference","org-staged-axis"))
    if err != nil { t.Fatal(err) }
    if len(loaded.World.Queries) != 1 || len(loaded.World.Applications) != 1 {
        t.Fatal("unexpected native Query/Application cardinality")
    }
    q := loaded.World.Queries[0]
    ds, ok := loaded.World.Applications[0].DatabaseByAliasExact(q.Datasource)
    if !ok || ds.TargetDatabase != "org_store" { t.Fatal("native datasource mismatch") }
    executor := postgresquery.Executor{DB: db, Database: ds.TargetDatabase}
    for _, tc := range []struct{
        title,validAt,knownAt,minimum string
        want []string
    }{
        {"valid and known boundary", "2027-06-15","2027-07-01","0.40",[]string{"A1","A3","A5"}},
        {"late known source", "2027-06-15","2027-07-03","0.40",[]string{"A1","A3","A4","A7"}},
        {"before valid end and higher allocation", "2027-05-31","2027-07-01","0.60",[]string{"A1","A2"}},
        {"at valid end is excluded", "2027-06-16","2027-07-01","0.40",[]string{"A1","A3"}},
    } {
        t.Run(tc.title, func(t *testing.T) {
            actual, err := executor.ExecuteBrowseQuery(ctx,q,ds,map[string]any{
                "valid_from":date(tc.validAt),"known_from":date(tc.knownAt),"allocation_min":tc.minimum,
            })
            if err != nil { t.Fatal(err) }
            mapped, err := queryresult.Map(q,actual.Raw)
            if err != nil { t.Fatal(err) }
            if ids:=orgAxisResultIDs(mapped.Rows); !reflect.DeepEqual(ids,tc.want) {
                t.Fatalf("actual=%v expected=%v",ids,tc.want)
            }
        })
    }
}
