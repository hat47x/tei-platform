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
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/internal/ir"
)

// These tests cover #1803 (name-only differences) on a real PostgreSQL. The
// tables are created with constraints under names other than the declared ones,
// which reproduces a database made by another tool. Unique and foreign key names
// that the definition declares are renamed from the observed name; names the
// realization generated for an unnamed definition are not compared.

func renameWorld(uniqueName, foreignKeyName string) (ir.World, string) {
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
			{Name: uniqueName, Attributes: []core.CanonicalID{attr("child", "label")}},
		},
		ForeignKeys: []persistence.ForeignKeyMapping{{
			Name: foreignKeyName, Attributes: []core.CanonicalID{attr("child", "parent_code")}, ReferencedMapping: "m.parent",
			ReferencedAttributes: []core.CanonicalID{attr("parent", "code")},
		}},
	}
	return ir.World{Persistence: []persistence.Mapping{child, parent}}, "store"
}

// newRenameFixture creates the declared tables, adds a unique constraint and a
// foreign key to child under legacyUnique and legacyForeignKey, inserts rows,
// and runs setup.
func newRenameFixture(t *testing.T, uniqueName, foreignKeyName, legacyUnique, legacyForeignKey string, setup ...string) existingFixture {
	t.Helper()
	world, databaseID := renameWorld(uniqueName, foreignKeyName)
	res, err := Render(world, databaseID, environment.ProviderPostgres)
	if err != nil || len(res.Unresolved) != 0 {
		t.Fatalf("render: %v %+v", err, res.Unresolved)
	}
	db, dsn := throwawayDBWithDSN(t, realDSN(t))
	ctx := context.Background()
	for _, statement := range res.Script {
		if strings.Contains(statement, " UNIQUE ") || strings.Contains(statement, "FOREIGN KEY") {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}
	all := []string{
		`ALTER TABLE "public"."child" ADD CONSTRAINT "` + legacyUnique + `" UNIQUE ("label")`,
		`ALTER TABLE "public"."child" ADD CONSTRAINT "` + legacyForeignKey + `" FOREIGN KEY ("parent_code") REFERENCES "public"."parent" ("code")`,
		`INSERT INTO "public"."parent" VALUES ('p1')`,
		`INSERT INTO "public"."child" VALUES ('c1', 'p1', 'one'), ('c2', 'p1', 'two')`,
	}
	for _, statement := range append(all, setup...) {
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
		provider: postgres.Provider{Catalog: postgres.SQLCatalogReader{DB: db}, Now: func() time.Time { return time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC) }},
		instance: environment.DatabaseInstance{
			ID: core.MustCanonicalID("ddl_entry.postgres.local"), Definition: core.CanonicalID(databaseID),
			Provider: environment.ProviderPostgres,
		},
	}
}

func (f existingFixture) relationExists(t *testing.T, name string) bool {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM pg_catalog.pg_class WHERE relnamespace = 'public'::regnamespace AND relname = $1`, name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func renameChangeSummary(diff schemadiff.Diff) string {
	var parts []string
	for _, c := range diff.Changes {
		parts = append(parts, string(c.Kind)+" "+c.Table+" "+c.BeforeName)
	}
	return strings.Join(parts, "; ")
}

func TestRealPostgresDeclaredConstraintNamesAreRenamedAndConverge(t *testing.T) {
	f := newRenameFixture(t, "child_label_declared_uq", "child_parent_declared_fk", "legacy_label_key", "legacy_parent_fkey")
	diff, observation := f.observeAndDiff(t, f.provider, true)
	if len(diff.Changes) != 2 {
		t.Fatalf("diff = %s", renameChangeSummary(diff))
	}
	for _, c := range diff.Changes {
		if c.Kind != schemadiff.RenameConstraint {
			t.Fatalf("diff = %s", renameChangeSummary(diff))
		}
	}
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(statements, "\n")
	for _, want := range []string{
		`ALTER TABLE "public"."child" RENAME CONSTRAINT "legacy_label_key" TO "child_label_declared_uq";`,
		`ALTER TABLE "public"."child" RENAME CONSTRAINT "legacy_parent_fkey" TO "child_parent_declared_fk";`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in\n%s", want, joined)
		}
	}
	if !f.relationExists(t, "legacy_label_key") {
		t.Fatal("fixture: the legacy unique index is missing")
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
	// Renaming a unique constraint renames its index too.
	if f.relationExists(t, "legacy_label_key") || !f.relationExists(t, "child_label_declared_uq") {
		t.Fatal("the unique constraint's index was not renamed with it")
	}
	var rows int
	if err := f.db.QueryRow(`SELECT count(*) FROM "public"."child"`).Scan(&rows); err != nil || rows != 2 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
	// The primary key is never renamed.
	var pk string
	if err := f.db.QueryRow(`SELECT conname FROM pg_catalog.pg_constraint WHERE contype = 'p' AND conrelid = 'public.child'::regclass`).Scan(&pk); err != nil || pk != "child_pkey" {
		t.Fatalf("pk=%s err=%v", pk, err)
	}
}

// Names the realization generated are not compared: a database whose
// constraints carry other names (here names PostgreSQL would not choose) shows
// no difference.
func TestRealPostgresGeneratedConstraintNamesAreNotComparedWithObservedNames(t *testing.T) {
	f := newRenameFixture(t, "", "", "legacy_label_key", "legacy_parent_fkey")
	diff, observation := f.observeAndDiff(t, f.provider, true)
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 || len(observation.ConstraintData) != 0 {
		t.Fatalf("diff=%s", renameChangeSummary(diff))
	}
}

func additionOf(blocked *postgres.ConstraintDataReadinessError, kind postgres.ConstraintRenameKind) *postgres.ConstraintAdditionReadiness {
	for i := range blocked.Readiness.Additions {
		if blocked.Readiness.Additions[i].Check.RenameKind == kind {
			return &blocked.Readiness.Additions[i]
		}
	}
	return nil
}

func TestRealPostgresConstraintRenameOntoUsedNameIsUnresolved(t *testing.T) {
	// "child_pkey" is the primary key's index, which the observed schema model
	// does not list; only the catalog read at observation time sees it. A
	// foreign key is not a relation, so a table's name ("parent") does not
	// block it.
	f := newRenameFixture(t, "child_pkey", "parent", "legacy_label_key", "legacy_parent_fkey")
	diff, observation := f.observeAndDiff(t, f.provider, true)
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil {
		t.Fatalf("statements=%v err=%v", statements, err)
	}
	unique := additionOf(blocked, postgres.ConstraintRenameUnique)
	if unique == nil || unique.Executable || unique.Reason != postgres.ConstraintRenameNameTaken || !strings.Contains(unique.Detail, "relation of kind i") {
		t.Fatalf("unique = %+v", unique)
	}
	if foreignKey := additionOf(blocked, postgres.ConstraintRenameForeignKey); foreignKey == nil || !foreignKey.Executable {
		t.Fatalf("foreign key = %+v", foreignKey)
	}
	if !f.relationExists(t, "legacy_label_key") {
		t.Fatal("nothing may change when the diff is refused")
	}
}

func TestRealPostgresConstraintRenameOntoAnotherConstraintOfTheTableIsUnresolved(t *testing.T) {
	// The foreign key is declared with the name the table's unique constraint
	// has now.
	f := newRenameFixture(t, "child_label_declared_uq", "legacy_label_key", "legacy_label_key", "legacy_parent_fkey")
	diff, observation := f.observeAndDiff(t, f.provider, true)
	_, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) {
		t.Fatalf("err=%v", err)
	}
	foreignKey := additionOf(blocked, postgres.ConstraintRenameForeignKey)
	if foreignKey == nil || foreignKey.Reason != postgres.ConstraintRenameNameTaken || !strings.Contains(foreignKey.Detail, "constraint of type u") {
		t.Fatalf("foreign key = %+v", foreignKey)
	}
}

func TestRealPostgresConstraintRenameWithoutObservationIsUnresolved(t *testing.T) {
	f := newRenameFixture(t, "child_label_declared_uq", "child_parent_declared_fk", "legacy_label_key", "legacy_parent_fkey")
	diff, observation := f.observeAndDiff(t, f.provider, false)
	_, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) {
		t.Fatalf("err=%v", err)
	}
	for _, addition := range blocked.Readiness.Additions {
		if addition.Reason != postgres.ConstraintDataUnobserved {
			t.Fatalf("addition = %+v", addition)
		}
	}
	if _, err := postgres.RenderMigrationSQL(diff); err == nil {
		t.Fatal("RenderMigrationSQL must not render a constraint rename")
	}
}

// A plan made before another tool renamed the constraint is refused at the next
// observation, and its statement fails on the database.
func TestRealPostgresStaleConstraintRenamePlanIsRefused(t *testing.T) {
	f := newRenameFixture(t, "child_label_declared_uq", "child_parent_declared_fk", "legacy_label_key", "legacy_parent_fkey")
	diff, observation := f.observeAndDiff(t, f.provider, true)
	planned, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`ALTER TABLE "public"."child" RENAME CONSTRAINT "legacy_label_key" TO "other_tool_key"`); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	fresh, err := f.provider.ObserveSchema(ctx, f.instance)
	if err != nil {
		t.Fatal(err)
	}
	if fresh, err = f.provider.ObserveConstraintData(ctx, fresh, diff); err != nil {
		t.Fatal(err)
	}
	_, err = postgres.RenderMigrationSQLWithObservation(diff, fresh)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || !strings.Contains(err.Error(), string(postgres.ConstraintRenameSourceMissing)) {
		t.Fatalf("err=%v", err)
	}
	failed := 0
	for _, statement := range planned {
		if _, err := f.db.Exec(statement); err != nil {
			failed++
		}
	}
	if failed == 0 {
		t.Fatal("the stale statement must fail on the database")
	}
}
