package ddlentry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/database/persistence"
	dbprovider "tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

// These tests run the rendered script on a real PostgreSQL and check that the
// observed schema converges on the declared one. They are skipped unless
// TEI_TEST_POSTGRES_DSN is set; each test uses its own throwaway database.
func realDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEI_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEI_TEST_POSTGRES_DSN is not set")
	}
	return dsn
}

func throwawayDB(t *testing.T, adminDSN string) *sql.DB {
	t.Helper()
	admin, err := postgresdb.OpenSQL(adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	name := fmt.Sprintf("tei_ddl_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() { admin.Exec(`DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`) })
	u, err := url.Parse(adminDSN)
	if err != nil {
		t.Fatal("invalid DSN")
	}
	u.Path = "/" + name
	db, err := postgresdb.OpenSQL(u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// converge executes the whole-database script and returns the observed schema
// difference against the declared one.
func converge(t *testing.T, db *sql.DB, world ir.World, databaseID string) (Result, schemadiff.Diff) {
	t.Helper()
	return convergeWith(t, db, world, databaseID, func(s string) string { return s })
}

// convergeWith is converge with each statement rewritten before execution, to
// reproduce DDL written by an earlier version of the renderer.
func convergeWith(t *testing.T, db *sql.DB, world ir.World, databaseID string, rewrite func(string) string) (Result, schemadiff.Diff) {
	t.Helper()
	res, err := Render(world, databaseID, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unresolved) != 0 || len(res.Script) == 0 {
		raw, _ := json.MarshalIndent(res, "", "  ")
		t.Fatalf("nothing to execute:\n%s", raw)
	}
	ctx := context.Background()
	var mappings []persistence.Mapping
	for _, m := range world.Persistence {
		if string(m.Database) == databaseID {
			mappings = append(mappings, m)
		}
	}
	desired, err := realization.RealizeDesired(mappings, databaseID, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	provider := postgres.Provider{
		Catalog: postgres.SQLCatalogReader{DB: db},
		Now:     func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) },
	}
	instance := environment.DatabaseInstance{
		ID: core.MustCanonicalID("ddl_entry.postgres.local"), Definition: core.CanonicalID(databaseID),
		Provider: environment.ProviderPostgres,
	}
	compare := func() schemadiff.Diff {
		actual, err := dbprovider.Introspect(ctx, provider, instance)
		if err != nil {
			t.Fatal(err)
		}
		diff, err := schemadiff.Compare(desired, actual)
		if err != nil {
			t.Fatal(err)
		}
		return diff
	}
	// The comparison must see the difference before the script runs; otherwise
	// an empty result after it would prove nothing.
	if before := compare(); len(before.Changes) == 0 {
		t.Fatal("comparison found no difference on an empty database")
	}
	for _, statement := range res.Script {
		statement = rewrite(statement)
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}
	return res, compare()
}

func constraintNames(t *testing.T, db *sql.DB, contype string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT conname FROM pg_catalog.pg_constraint WHERE contype = $1 AND connamespace = 'public'::regnamespace ORDER BY conname`, contype)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	// Byte order, as Go compares; the database collation may order differently.
	sort.Strings(out)
	return out
}

func TestRealPostgresReferenceSalesCoreConverges(t *testing.T) {
	dsn := realDSN(t)
	loaded, err := workspace.Load("../../reference/workspace")
	if err != nil {
		t.Fatal(err)
	}
	db := throwawayDB(t, dsn)
	_, diff := converge(t, db, loaded.World, "sales_core")
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 {
		raw, _ := json.MarshalIndent(diff, "", "  ")
		t.Fatalf("observed schema did not converge:\n%s", raw)
	}
	// The primary key is unnamed in the rendered SQL; observe what PostgreSQL chose.
	got := strings.Join(constraintNames(t, db, "p"), ",")
	if got != "order_lines_pkey,orders_pkey" {
		t.Fatalf("observed primary key names = %s", got)
	}
}

func TestRealPostgresUniqueAndForeignKeyConverge(t *testing.T) {
	dsn := realDSN(t)
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
			{Name: "child_label_key", Attributes: []core.CanonicalID{attr("child", "label")}},
			{Attributes: []core.CanonicalID{attr("child", "parent_code"), attr("child", "label")}},
		},
		ForeignKeys: []persistence.ForeignKeyMapping{{
			Name: "child_parent_fk", Attributes: []core.CanonicalID{attr("child", "parent_code")},
			ReferencedMapping: "m.parent", ReferencedAttributes: []core.CanonicalID{attr("parent", "code")},
		}},
	}
	// "child" sorts before "parent": the foreign key must still be created last.
	world := ir.World{Persistence: []persistence.Mapping{child, parent}}
	db := throwawayDB(t, dsn)
	res, diff := converge(t, db, world, "store")
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 {
		raw, _ := json.MarshalIndent(diff, "", "  ")
		t.Fatalf("observed schema did not converge:\n%s\nscript:\n%s", raw, strings.Join(res.Script, "\n"))
	}
	if last := res.Script[len(res.Script)-1]; !strings.Contains(last, "FOREIGN KEY") {
		t.Fatalf("foreign key is not last: %s", last)
	}
	if got := strings.Join(constraintNames(t, db, "f"), ","); got != "child_parent_fk" {
		t.Fatalf("observed foreign key names = %s", got)
	}
	unique := strings.Join(constraintNames(t, db, "u"), ",")
	if !strings.Contains(unique, "child_label_key") || len(constraintNames(t, db, "u")) != 2 {
		t.Fatalf("observed unique constraints = %s", unique)
	}
}

// namingWorld declares unique constraints and foreign keys without names:
// multi-column in a non-sorted declared order, and a multibyte table name.
func namingWorld(parentTable, childTable string, columnSuffix string) (ir.World, string) {
	col := func(table, name string) persistence.ColumnMapping {
		return persistence.ColumnMapping{
			Attribute: core.CanonicalID("s." + table + "." + name), Column: name,
			Storage: persistence.StorageVarchar, Nullability: persistence.Required,
		}
	}
	attr := func(table, name string) core.CanonicalID { return core.CanonicalID("s." + table + "." + name) }
	a, b := "a"+columnSuffix, "b"+columnSuffix
	parent := persistence.Mapping{
		ID: "m.parent", Database: "store", SemanticData: "s.parent", Table: "public." + parentTable,
		Columns:    []persistence.ColumnMapping{col("parent", a), col("parent", b)},
		PrimaryKey: []core.CanonicalID{attr("parent", a), attr("parent", b)},
	}
	child := persistence.Mapping{
		ID: "m.child", Database: "store", SemanticData: "s.child", Table: "public." + childTable,
		Columns:    []persistence.ColumnMapping{col("child", "id"), col("child", "p"+a), col("child", "p"+b), col("child", "ラベル")},
		PrimaryKey: []core.CanonicalID{attr("child", "id")},
		UniqueConstraints: []persistence.UniqueConstraintMapping{
			{Attributes: []core.CanonicalID{attr("child", "p"+b), attr("child", "p"+a)}},
			{Attributes: []core.CanonicalID{attr("child", "ラベル")}},
		},
		ForeignKeys: []persistence.ForeignKeyMapping{{
			Attributes:        []core.CanonicalID{attr("child", "p"+a), attr("child", "p"+b)},
			ReferencedMapping: "m.parent", ReferencedAttributes: []core.CanonicalID{attr("parent", a), attr("parent", b)},
		}},
	}
	return ir.World{Persistence: []persistence.Mapping{child, parent}}, "store"
}

// desiredConstraintNames lists the names realization decided, by kind.
func desiredConstraintNames(t *testing.T, world ir.World, databaseID string) (unique, foreign []string) {
	t.Helper()
	desired, err := realization.RealizeDesired(world.Persistence, databaseID, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range desired.Tables {
		for _, uc := range table.UniqueConstraints {
			unique = append(unique, uc.Name)
		}
		for _, fk := range table.ForeignKeys {
			foreign = append(foreign, fk.Name)
		}
	}
	sort.Strings(unique)
	sort.Strings(foreign)
	return unique, foreign
}

var namedConstraint = regexp.MustCompile(`ADD CONSTRAINT "[^"]+" (UNIQUE|FOREIGN KEY)`)

func TestRealPostgresGeneratedNamesMatchDatabasesCreatedWithoutNames(t *testing.T) {
	dsn := realDSN(t)
	world, databaseID := namingWorld("親表", "子表", "")
	wantUnique, wantForeign := desiredConstraintNames(t, world, databaseID)
	if strings.Join(wantUnique, ",") != "子表_pb_pa_key,子表_ラベル_key" || strings.Join(wantForeign, ",") != "子表_pa_pb_fkey" {
		t.Fatalf("desired names unique=%v foreign=%v", wantUnique, wantForeign)
	}
	for _, variant := range []struct {
		name    string
		rewrite func(string) string
	}{
		{"named DDL (this renderer)", func(s string) string { return s }},
		{"unnamed DDL (earlier renderer)", func(s string) string { return namedConstraint.ReplaceAllString(s, "ADD $1") }},
	} {
		t.Run(variant.name, func(t *testing.T) {
			db := throwawayDB(t, dsn)
			res, diff := convergeWith(t, db, world, databaseID, variant.rewrite)
			if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 {
				raw, _ := json.MarshalIndent(diff, "", "  ")
				t.Fatalf("observed schema did not converge:\n%s\nscript:\n%s", raw, strings.Join(res.Script, "\n"))
			}
			// The diff compares structure only; the names are checked directly.
			if got := constraintNames(t, db, "u"); strings.Join(got, ",") != strings.Join(wantUnique, ",") {
				t.Fatalf("observed unique names = %v, desired %v", got, wantUnique)
			}
			if got := constraintNames(t, db, "f"); strings.Join(got, ",") != strings.Join(wantForeign, ",") {
				t.Fatalf("observed foreign key names = %v, desired %v", got, wantForeign)
			}
		})
	}
}

func TestRealPostgresLongGeneratedNamesAreCreatedAsDecided(t *testing.T) {
	dsn := realDSN(t)
	// The generated names exceed 63 bytes and are shortened with a digest; an
	// unnamed DDL would get PostgreSQL's truncated names instead (not checked).
	world, databaseID := namingWorld("parent_table_with_a_long_name", "child_table_with_a_rather_long_name", "_column_with_a_long_name")
	wantUnique, wantForeign := desiredConstraintNames(t, world, databaseID)
	for _, name := range append(append([]string(nil), wantUnique...), wantForeign...) {
		if len(name) > 63 {
			t.Fatalf("desired name %q is %d bytes", name, len(name))
		}
	}
	db := throwawayDB(t, dsn)
	res, diff := converge(t, db, world, databaseID)
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 {
		raw, _ := json.MarshalIndent(diff, "", "  ")
		t.Fatalf("observed schema did not converge:\n%s\nscript:\n%s", raw, strings.Join(res.Script, "\n"))
	}
	if got := constraintNames(t, db, "u"); strings.Join(got, ",") != strings.Join(wantUnique, ",") {
		t.Fatalf("observed unique names = %v, desired %v", got, wantUnique)
	}
	if got := constraintNames(t, db, "f"); strings.Join(got, ",") != strings.Join(wantForeign, ",") {
		t.Fatalf("observed foreign key names = %v, desired %v", got, wantForeign)
	}
}
