package ddlentry

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/environment"
	"tei/internal/ir"
)

func mapping(id, table string, withPK bool) persistence.Mapping {
	return mappingWith(id, table, withPK, false)
}

func mappingWith(id, table string, withPK, withUnique bool) persistence.Mapping {
	m := persistence.Mapping{
		ID: core.CanonicalID(id), Database: "store", SemanticData: core.CanonicalID("s." + table), Table: table,
		Columns: []persistence.ColumnMapping{
			{Attribute: core.CanonicalID("s." + table + ".code"), Column: "code", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
	}
	if withPK {
		m.PrimaryKey = []core.CanonicalID{core.CanonicalID("s." + table + ".code")}
	}
	if withUnique {
		m.UniqueConstraints = []persistence.UniqueConstraintMapping{{
			Name: table + "_code_key", Attributes: []core.CanonicalID{core.CanonicalID("s." + table + ".code")},
		}}
	}
	return m
}

func TestTableWithoutPrimaryKeyIsRenderedByTheProvider(t *testing.T) {
	w := ir.World{Persistence: []persistence.Mapping{mapping("m.b", "beta", false), mapping("m.a", "alpha", false)}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unresolved) != 0 || len(res.Tables) != 2 {
		t.Fatalf("result = %+v", res)
	}
	if res.Tables[0].Name >= res.Tables[1].Name {
		t.Fatalf("tables are not in name order: %+v", res.Tables)
	}
	joined := strings.Join(res.Tables[0].Statements, "\n")
	if !strings.Contains(joined, "CREATE TABLE") || !strings.Contains(joined, "ADD COLUMN") {
		t.Fatalf("statements = %s", joined)
	}
}

func TestPrimaryKeyIsRenderedAfterColumnsWithoutAName(t *testing.T) {
	w := ir.World{Persistence: []persistence.Mapping{mapping("m.p", "pk_table", true)}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unresolved) != 0 || len(res.Tables) != 1 {
		t.Fatalf("result = %+v", res)
	}
	st := res.Tables[0].Statements
	last := st[len(st)-1]
	if !strings.Contains(last, "ADD PRIMARY KEY") || strings.Contains(last, "CONSTRAINT") {
		t.Fatalf("statements = %v", st)
	}
}

func TestUniqueConstraintIsRenderedForANewTable(t *testing.T) {
	w := ir.World{Persistence: []persistence.Mapping{mappingWith("m.u", "uq_table", false, true)}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unresolved) != 0 || len(res.Tables) != 1 {
		t.Fatalf("result = %+v", res)
	}
	st := res.Tables[0].Statements
	if !strings.Contains(st[len(st)-1], "UNIQUE") {
		t.Fatalf("statements = %v", st)
	}
}

func TestProviderRefusalIsReportedPerTableNotHidden(t *testing.T) {
	// A nullable primary key column is refused by the provider; the other table
	// must still be returned and the refusal must carry the provider's reason.
	nullablePK := mapping("m.n", "nullable_pk", true)
	nullablePK.Columns[0].Nullability = persistence.Optional
	w := ir.World{Persistence: []persistence.Mapping{mapping("m.a", "alpha", false), nullablePK}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tables) != 1 || !strings.HasSuffix(res.Tables[0].Name, "alpha") {
		t.Fatalf("renderable tables = %+v", res.Tables)
	}
	if len(res.Unresolved) != 1 || !strings.HasSuffix(res.Unresolved[0].Table, "nullable_pk") ||
		!strings.Contains(res.Unresolved[0].Reason, "is nullable") {
		t.Fatalf("unresolved = %+v", res.Unresolved)
	}
}

func TestOnlyTheRequestedDatabaseAndKnownProvider(t *testing.T) {
	w := ir.World{Persistence: []persistence.Mapping{mapping("m.a", "alpha", false)}}
	if _, err := Render(w, "other", environment.ProviderPostgres); err == nil {
		t.Fatal("database without mappings accepted")
	}
	if _, err := Render(w, "store", environment.DatabaseProvider("unknown")); err == nil {
		t.Fatal("unknown provider accepted")
	}
}

func TestStatesThatItIsNotAMigration(t *testing.T) {
	w := ir.World{Persistence: []persistence.Mapping{mapping("m.a", "alpha", false)}}
	res, _ := Render(w, "store", environment.ProviderPostgres)
	if !strings.Contains(res.Note, "移行文ではありません") {
		t.Fatalf("note = %q", res.Note)
	}
}

func TestUnnamedUniqueConstraintGetsTheGeneratedName(t *testing.T) {
	m := mappingWith("m.u", "public.uq_table", false, true)
	m.UniqueConstraints[0].Name = ""
	res, err := Render(ir.World{Persistence: []persistence.Mapping{m}}, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unresolved) != 0 || len(res.Tables) != 1 {
		t.Fatalf("result = %+v", res)
	}
	st := res.Tables[0].Statements
	if want := `ALTER TABLE "public"."uq_table" ADD CONSTRAINT "uq_table_code_key" UNIQUE ("code");`; st[len(st)-1] != want {
		t.Fatalf("statements = %v", st)
	}
}

func TestGeneratedNameCollisionIsReportedPerTable(t *testing.T) {
	// x's unique constraint would be named x_code_key, which y declares; both
	// back an index in the same schema, so PostgreSQL would refuse one.
	x := mappingWith("m.x", "public.x", false, true)
	x.UniqueConstraints[0].Name = ""
	y := mappingWith("m.y", "public.y", false, true)
	y.UniqueConstraints[0].Name = "x_code_key"
	res, err := Render(ir.World{Persistence: []persistence.Mapping{x, y}}, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tables) != 1 || res.Tables[0].Name != "y" {
		t.Fatalf("renderable tables = %+v", res.Tables)
	}
	if len(res.Unresolved) != 1 || res.Unresolved[0].Table != "x" || !strings.Contains(res.Unresolved[0].Reason, "collision") {
		t.Fatalf("unresolved = %+v", res.Unresolved)
	}
	if len(res.Script) != 0 {
		t.Fatalf("a partial script must not be offered: %v", res.Script)
	}
}
