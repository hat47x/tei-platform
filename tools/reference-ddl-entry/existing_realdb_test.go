package ddlentry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/tools/internal/postgresdb"
)

// These tests cover #1803 P4 on a real PostgreSQL: adding a primary key, a
// unique constraint and a foreign key to tables that already hold rows. The
// tables are first created by the fresh-create script without its constraint
// statements, which reproduces a database made before the constraints were
// declared.

func existingTablesWorld() (ir.World, string) {
	col := func(table, name string) persistence.ColumnMapping {
		return persistence.ColumnMapping{
			Attribute: core.CanonicalID("s." + table + "." + name), Column: name,
			Storage: persistence.StorageVarchar, Nullability: persistence.Required,
		}
	}
	attr := func(table, name string) core.CanonicalID { return core.CanonicalID("s." + table + "." + name) }
	parent := persistence.Mapping{
		ID: "m.parent", Database: "store", SemanticData: "s.parent", Table: "public.parent",
		Columns:    []persistence.ColumnMapping{col("parent", "code")},
		PrimaryKey: []core.CanonicalID{attr("parent", "code")},
	}
	child := persistence.Mapping{
		ID: "m.child", Database: "store", SemanticData: "s.child", Table: "public.child",
		Columns:    []persistence.ColumnMapping{col("child", "id"), col("child", "parent_code"), col("child", "label")},
		PrimaryKey: []core.CanonicalID{attr("child", "id")},
		UniqueConstraints: []persistence.UniqueConstraintMapping{
			{Attributes: []core.CanonicalID{attr("child", "label")}},
		},
		ForeignKeys: []persistence.ForeignKeyMapping{{
			Attributes: []core.CanonicalID{attr("child", "parent_code")}, ReferencedMapping: "m.parent",
			ReferencedAttributes: []core.CanonicalID{attr("parent", "code")},
		}},
	}
	return ir.World{Persistence: []persistence.Mapping{child, parent}}, "store"
}

type existingFixture struct {
	db       *sql.DB
	dsn      string
	provider postgres.Provider
	instance environment.DatabaseInstance
	desired  physical.SchemaSnapshot
}

// newExistingFixture creates the declared tables without any constraint and
// then runs setup (rows, column changes) on them.
func newExistingFixture(t *testing.T, adminDSN string, setup ...string) existingFixture {
	t.Helper()
	world, databaseID := existingTablesWorld()
	res, err := Render(world, databaseID, environment.ProviderPostgres)
	if err != nil || len(res.Unresolved) != 0 {
		t.Fatalf("render: %v %+v", err, res.Unresolved)
	}
	db, dsn := throwawayDBWithDSN(t, adminDSN)
	ctx := context.Background()
	for _, statement := range res.Script {
		if strings.Contains(statement, "PRIMARY KEY") || strings.Contains(statement, " UNIQUE ") || strings.Contains(statement, "FOREIGN KEY") {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}
	for _, statement := range setup {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("setup %q: %v", statement, err)
		}
	}
	desired, err := realization.RealizeDesired(world.Persistence, databaseID, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	return existingFixture{
		db: db, dsn: dsn, desired: desired,
		provider: postgres.Provider{Catalog: postgres.SQLCatalogReader{DB: db}, Now: func() time.Time { return time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC) }},
		instance: environment.DatabaseInstance{
			ID: core.MustCanonicalID("ddl_entry.postgres.local"), Definition: core.CanonicalID(databaseID),
			Provider: environment.ProviderPostgres,
		},
	}
}

func throwawayDBWithDSN(t *testing.T, adminDSN string) (*sql.DB, string) {
	t.Helper()
	db := throwawayDB(t, adminDSN)
	var name string
	if err := db.QueryRow(`SELECT current_database()`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(adminDSN)
	if err != nil {
		t.Fatal("invalid DSN")
	}
	u.Path = "/" + name
	return db, u.String()
}

// observeAndDiff observes the schema with provider p, compares it with the
// declared schema and, when count is set, counts the existing data.
func (f existingFixture) observeAndDiff(t *testing.T, p postgres.Provider, count bool) (schemadiff.Diff, postgres.SchemaObservation) {
	t.Helper()
	ctx := context.Background()
	observation, err := p.ObserveSchema(ctx, f.instance)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := schemadiff.Compare(f.desired, observation.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if count {
		if observation, err = p.ObserveConstraintData(ctx, observation, diff); err != nil {
			t.Fatal(err)
		}
	}
	return diff, observation
}

func changeKinds(diff schemadiff.Diff) string {
	var kinds []string
	for _, c := range diff.Changes {
		kinds = append(kinds, string(c.Kind)+" "+c.Table)
	}
	return strings.Join(kinds, ", ")
}

func TestRealPostgresExistingTablesWithCleanDataGetConstraints(t *testing.T) {
	f := newExistingFixture(t, realDSN(t),
		`INSERT INTO "public"."parent" VALUES ('P1'), ('P2')`,
		`INSERT INTO "public"."child" VALUES ('c1', 'P1', 'x'), ('c2', 'P2', 'y'), ('c3', 'P1', 'z')`,
	)
	diff, observation := f.observeAndDiff(t, f.provider, true)
	if got := changeKinds(diff); got != "change-primary-key public.child, add-unique-constraint public.child, add-foreign-key public.child, change-primary-key public.parent" {
		t.Fatalf("diff before = %s", got)
	}
	for _, check := range observation.ConstraintData {
		if !check.Observed {
			t.Fatalf("check not observed: %+v", check)
		}
	}
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range statements {
		if _, err := f.db.Exec(statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}
	after, _ := f.observeAndDiff(t, f.provider, false)
	if len(after.Changes) != 0 || len(after.Unresolved) != 0 {
		raw, _ := json.MarshalIndent(after, "", "  ")
		t.Fatalf("observed schema did not converge:\n%s\nstatements:\n%s", raw, strings.Join(statements, "\n"))
	}
	if got := strings.Join(constraintNames(t, f.db, "p"), ","); got != "child_pkey,parent_pkey" {
		t.Fatalf("primary keys = %s", got)
	}
	if got := strings.Join(constraintNames(t, f.db, "u"), ",") + "|" + strings.Join(constraintNames(t, f.db, "f"), ","); got != "child_label_key|child_parent_code_fkey" {
		t.Fatalf("constraints = %s", got)
	}
}

// EX-01: rows violating the constraints refuse the whole diff with the counts,
// and nothing is executed.
func TestRealPostgresExistingTablesWithViolationsAreRefusedWithCounts(t *testing.T) {
	f := newExistingFixture(t, realDSN(t),
		`ALTER TABLE "public"."child" ALTER COLUMN "id" DROP NOT NULL`,
		`INSERT INTO "public"."parent" VALUES ('P1'), ('P2')`,
		`INSERT INTO "public"."child" VALUES
		  ('c1', 'P1', 'a'), ('c1', 'P2', 'b'),
		  (NULL, 'P1', 'c'),
		  ('c2', 'P9', 'x'), ('c3', 'P8', 'x'), ('c4', 'P1', 'x'),
		  ('c5', 'P2', 'd')`,
	)
	diff, observation := f.observeAndDiff(t, f.provider, true)
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil {
		t.Fatalf("statements=%v err=%v", statements, err)
	}
	for _, want := range []string{
		"primary-key public.child (id): existing-data-violations: 2 row(s) share 1 duplicated key value(s), 1 row(s) have NULL in a key column",
		"unique-constraint public.child (label): existing-data-violations: 3 row(s) share 1 duplicated key value(s)",
		"foreign-key public.child (parent_code): existing-data-violations: 2 row(s) have no referenced row",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in %v", want, err)
		}
	}
	// The clean primary key on parent is executable on its own, but the diff is
	// refused as a whole.
	for _, addition := range blocked.Readiness.Additions {
		if addition.Check.Table == "public.parent" && !addition.Executable {
			t.Fatalf("parent addition = %+v", addition)
		}
	}
	// Counts only: no stored value appears in the refusal.
	for _, value := range []string{"c1", "P9", "P8", "'x'"} {
		if strings.Contains(err.Error(), value) {
			t.Fatalf("refusal leaks value %s: %v", value, err)
		}
	}
	if n := len(constraintNames(t, f.db, "p")) + len(constraintNames(t, f.db, "u")) + len(constraintNames(t, f.db, "f")); n != 0 {
		t.Fatalf("constraints were created: %d", n)
	}
}

// EX-02 / DE-10: without the counts, or without the privilege to read the
// rows, the additions are unresolved, not zero.
func TestRealPostgresExistingTablesWithoutObservationAreUnresolved(t *testing.T) {
	adminDSN := realDSN(t)
	admin, err := postgresdb.OpenSQL(adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	role := fmt.Sprintf("tei_p4_reader_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE ROLE "` + role + `" LOGIN PASSWORD 'tei_p4_reader'`); err != nil {
		t.Fatalf("create role: %v", err)
	}
	// Registered before the database, so it runs after the database is dropped.
	t.Cleanup(func() { admin.Exec(`DROP ROLE IF EXISTS "` + role + `"`) })

	f := newExistingFixture(t, adminDSN,
		`INSERT INTO "public"."parent" VALUES ('P1')`,
		`INSERT INTO "public"."child" VALUES ('c1', 'P1', 'x')`,
		`GRANT USAGE ON SCHEMA "public" TO "`+role+`"`,
		`GRANT SELECT ON "public"."parent" TO "`+role+`"`,
	)

	t.Run("not counted", func(t *testing.T) {
		diff, observation := f.observeAndDiff(t, f.provider, false)
		_, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
		var blocked *postgres.ConstraintDataReadinessError
		if !errors.As(err, &blocked) || len(blocked.Readiness.Additions) != 4 {
			t.Fatalf("err=%v", err)
		}
		for _, addition := range blocked.Readiness.Additions {
			if addition.Reason != postgres.ConstraintDataUnobserved {
				t.Fatalf("addition=%+v", addition)
			}
		}
	})

	t.Run("no privilege to read the rows", func(t *testing.T) {
		u, err := url.Parse(f.dsn)
		if err != nil {
			t.Fatal("invalid DSN")
		}
		u.User = url.UserPassword(role, "tei_p4_reader")
		reader, err := postgresdb.OpenSQL(u.String())
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		limited := f.provider
		limited.Catalog = postgres.SQLCatalogReader{DB: reader}
		diff, observation := f.observeAndDiff(t, limited, true)
		_, err = postgres.RenderMigrationSQLWithObservation(diff, observation)
		var blocked *postgres.ConstraintDataReadinessError
		if !errors.As(err, &blocked) {
			t.Fatalf("err=%v", err)
		}
		unobserved := 0
		for _, addition := range blocked.Readiness.Additions {
			if addition.Check.Table == "public.child" {
				if addition.Reason != postgres.ConstraintDataUnobserved || !strings.Contains(addition.Detail, "permission denied") {
					t.Fatalf("addition=%+v", addition)
				}
				unobserved++
			}
		}
		if unobserved != 3 {
			t.Fatalf("unobserved child additions = %d", unobserved)
		}
	})
}
