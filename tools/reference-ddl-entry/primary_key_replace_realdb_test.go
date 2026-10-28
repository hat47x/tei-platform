package ddlentry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/realization"
	"tei/internal/environment"
	"tei/internal/ir"
)

// These tests cover #1803 P5 on a real PostgreSQL: replacing the primary key of
// a table that already holds rows. The existing key is created with a name that
// is not PostgreSQL's default, so a renderer that assumed "<table>_pkey" fails.

const legacyPrimaryKeyName = "t_legacy_key_pk"

func primaryKeyWorld(primaryKey ...string) (ir.World, string) {
	attr := func(name string) core.CanonicalID { return core.CanonicalID("s.t." + name) }
	col := func(name string) persistence.ColumnMapping {
		return persistence.ColumnMapping{Attribute: attr(name), Column: name, Storage: persistence.StorageVarchar, Nullability: persistence.Required}
	}
	var key []core.CanonicalID
	for _, name := range primaryKey {
		key = append(key, attr(name))
	}
	return ir.World{Persistence: []persistence.Mapping{{
		ID: "m.t", Database: "store", SemanticData: "s.t", Table: "public.t",
		Columns:    []persistence.ColumnMapping{col("a"), col("b")},
		PrimaryKey: key,
	}}}, "store"
}

// newPrimaryKeyFixture creates public.t with the key (a) under a non-default
// name, then runs setup. The declared key is primaryKey.
func newPrimaryKeyFixture(t *testing.T, adminDSN string, primaryKey []string, setup ...string) existingFixture {
	t.Helper()
	world, databaseID := primaryKeyWorld(primaryKey...)
	res, err := Render(world, databaseID, environment.ProviderPostgres)
	if err != nil || len(res.Unresolved) != 0 {
		t.Fatalf("render: %v %+v", err, res.Unresolved)
	}
	db, dsn := throwawayDBWithDSN(t, adminDSN)
	ctx := context.Background()
	for _, statement := range res.Script {
		if strings.Contains(statement, "PRIMARY KEY") {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}
	all := append([]string{`ALTER TABLE "public"."t" ADD CONSTRAINT "` + legacyPrimaryKeyName + `" PRIMARY KEY ("a")`}, setup...)
	for _, statement := range all {
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
		provider: postgres.Provider{Catalog: postgres.SQLCatalogReader{DB: db}, Now: func() time.Time { return time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC) }},
		instance: environment.DatabaseInstance{
			ID: core.MustCanonicalID("ddl_entry.postgres.local"), Definition: core.CanonicalID(databaseID),
			Provider: environment.ProviderPostgres,
		},
	}
}

func primaryKeyConstraint(t *testing.T, f existingFixture) (name string, columns string) {
	t.Helper()
	err := f.db.QueryRow(`SELECT con.conname, string_agg(att.attname, ',' ORDER BY k.ord)
FROM pg_catalog.pg_constraint con
JOIN LATERAL unnest(con.conkey) WITH ORDINALITY AS k(attnum, ord) ON true
JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.attnum
WHERE con.contype = 'p' AND con.conrelid = 'public.t'::regclass GROUP BY con.conname`).Scan(&name, &columns)
	if err != nil {
		t.Fatal(err)
	}
	return name, columns
}

func TestRealPostgresPrimaryKeyReplacementUsesObservedNameAndConverges(t *testing.T) {
	f := newPrimaryKeyFixture(t, realDSN(t), []string{"a", "b"},
		`INSERT INTO "public"."t" VALUES ('1', 'x'), ('2', 'x'), ('3', 'y')`,
	)
	if name, _ := primaryKeyConstraint(t, f); name != legacyPrimaryKeyName {
		t.Fatalf("fixture key name = %s", name)
	}
	diff, observation := f.observeAndDiff(t, f.provider, true)
	if got := changeKinds(diff); got != "change-primary-key public.t" {
		t.Fatalf("diff = %s", got)
	}
	check := observation.ConstraintData[0]
	if !check.Observed || check.PrimaryKeyName != legacyPrimaryKeyName || len(check.ReferencingForeignKeys) != 0 {
		t.Fatalf("check = %+v", check)
	}
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 || !strings.Contains(statements[0], `DROP CONSTRAINT "`+legacyPrimaryKeyName+`"`) || strings.Contains(statements[0], "t_pkey") {
		t.Fatalf("statements = %v", statements)
	}
	for _, statement := range statements {
		if _, err := f.db.Exec(statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}
	after, _ := f.observeAndDiff(t, f.provider, false)
	if len(after.Changes) != 0 || len(after.Unresolved) != 0 {
		raw, _ := json.MarshalIndent(after, "", "  ")
		t.Fatalf("observed schema did not converge:\n%s", raw)
	}
	// The new key is added unnamed (design 4.2 option C): PostgreSQL's own name.
	if name, columns := primaryKeyConstraint(t, f); name != "t_pkey" || columns != "a,b" {
		t.Fatalf("primary key = %s (%s)", name, columns)
	}
}

// The name is read at each observation: a key renamed after the plan makes the
// freshly rendered statement differ from the planned one, which the apply
// boundary (ApplyPostgresWithFreshObservation) refuses.
func TestRealPostgresPrimaryKeyReplacementFollowsTheCurrentCatalogName(t *testing.T) {
	f := newPrimaryKeyFixture(t, realDSN(t), []string{"a", "b"}, `INSERT INTO "public"."t" VALUES ('1', 'x')`)
	diff, observation := f.observeAndDiff(t, f.provider, true)
	planned, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`ALTER TABLE "public"."t" RENAME CONSTRAINT "` + legacyPrimaryKeyName + `" TO "t_renamed_pk"`); err != nil {
		t.Fatal(err)
	}
	diff, observation = f.observeAndDiff(t, f.provider, true)
	fresh, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(planned, "") == strings.Join(fresh, "") || !strings.Contains(fresh[0], `"t_renamed_pk"`) {
		t.Fatalf("planned=%v fresh=%v", planned, fresh)
	}
	for _, statement := range planned {
		if _, err := f.db.Exec(statement); err == nil {
			t.Fatal("a stale constraint name must not execute")
		}
	}
}

func TestRealPostgresPrimaryKeyReplacementWithDuplicatesIsRefusedWithCounts(t *testing.T) {
	f := newPrimaryKeyFixture(t, realDSN(t), []string{"b"},
		`ALTER TABLE "public"."t" ALTER COLUMN "b" DROP NOT NULL`,
		`INSERT INTO "public"."t" VALUES ('1', 'x'), ('2', 'x'), ('3', 'x'), ('4', 'y'), ('5', 'y'), ('6', 'z'), ('7', NULL)`,
	)
	diff, observation := f.observeAndDiff(t, f.provider, true)
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil {
		t.Fatalf("statements=%v err=%v", statements, err)
	}
	want := "primary-key-replace public.t (b): existing-data-violations: 5 row(s) share 2 duplicated key value(s), 1 row(s) have NULL in a key column"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("missing %q in %v", want, err)
	}
	for _, value := range []string{"'x'", "'y'", "x,", "NULL'"} {
		if strings.Contains(err.Error(), value) {
			t.Fatalf("refusal leaks a value: %v", err)
		}
	}
	if name, columns := primaryKeyConstraint(t, f); name != legacyPrimaryKeyName || columns != "a" {
		t.Fatalf("old key changed: %s (%s)", name, columns)
	}
}

func TestRealPostgresPrimaryKeyReplacementReferencedByForeignKeyIsRefusedNamingIt(t *testing.T) {
	f := newPrimaryKeyFixture(t, realDSN(t), []string{"a", "b"},
		`INSERT INTO "public"."t" VALUES ('1', 'x')`,
		`CREATE TABLE "public"."refs" (id text PRIMARY KEY, ta text)`,
		`ALTER TABLE "public"."refs" ADD CONSTRAINT refs_ta_custom_fk FOREIGN KEY (ta) REFERENCES "public"."t" (a)`,
	)
	diff, observation := f.observeAndDiff(t, f.provider, true)
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil {
		t.Fatalf("statements=%v err=%v", statements, err)
	}
	if !strings.Contains(err.Error(), string(postgres.ConstraintDataReferencedByForeignKeys)) || !strings.Contains(err.Error(), `foreign key "refs_ta_custom_fk" on public.refs (ta)`) {
		t.Fatalf("err=%v", err)
	}
	if name, columns := primaryKeyConstraint(t, f); name != legacyPrimaryKeyName || columns != "a" {
		t.Fatalf("old key changed: %s (%s)", name, columns)
	}
}

func TestRealPostgresPrimaryKeyReplacementWithoutObservationIsUnresolved(t *testing.T) {
	f := newPrimaryKeyFixture(t, realDSN(t), []string{"a", "b"}, `INSERT INTO "public"."t" VALUES ('1', 'x')`)
	diff, observation := f.observeAndDiff(t, f.provider, false)
	_, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || blocked.Readiness.Additions[0].Reason != postgres.ConstraintDataUnobserved {
		t.Fatalf("err=%v", err)
	}
	if _, err := postgres.RenderMigrationSQL(diff); err == nil {
		t.Fatal("RenderMigrationSQL must not render a primary-key replacement")
	}
}

// A statement that fails (here a duplicate written after the count) leaves the
// old primary key in place: the drop and the add are one ALTER TABLE.
func TestRealPostgresFailedPrimaryKeyReplacementLeavesOldKeyIntact(t *testing.T) {
	f := newPrimaryKeyFixture(t, realDSN(t), []string{"b"},
		`INSERT INTO "public"."t" VALUES ('1', 'x'), ('2', 'y')`,
	)
	diff, observation := f.observeAndDiff(t, f.provider, true)
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`INSERT INTO "public"."t" VALUES ('3', 'x')`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range statements {
		if _, err := f.db.Exec(statement); err == nil {
			t.Fatalf("duplicate key must fail %q", statement)
		}
	}
	if name, columns := primaryKeyConstraint(t, f); name != legacyPrimaryKeyName || columns != "a" {
		t.Fatalf("old key lost after a failed statement: %s (%s)", name, columns)
	}
	var rows int
	if err := f.db.QueryRow(`SELECT count(*) FROM "public"."t"`).Scan(&rows); err != nil || rows != 3 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
}
