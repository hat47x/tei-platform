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

// fkMapping declares app.<table> with a primary key on code and, for each
// referenced table, a <ref>_code column with a foreign key to it.
func fkMapping(table string, refs ...string) persistence.Mapping {
	attr := func(tbl, col string) core.CanonicalID { return core.CanonicalID("s." + tbl + "." + col) }
	m := persistence.Mapping{
		ID: core.CanonicalID("m." + table), Database: "store", SemanticData: core.CanonicalID("s." + table), Table: "app." + table,
		Columns: []persistence.ColumnMapping{
			{Attribute: attr(table, "code"), Column: "code", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
		PrimaryKey: []core.CanonicalID{attr(table, "code")},
	}
	for _, ref := range refs {
		m.Columns = append(m.Columns, persistence.ColumnMapping{
			Attribute: attr(table, ref+"_code"), Column: ref + "_code", Storage: persistence.StorageVarchar, Nullability: persistence.Required,
		})
		m.ForeignKeys = append(m.ForeignKeys, persistence.ForeignKeyMapping{
			Attributes: []core.CanonicalID{attr(table, ref+"_code")}, ReferencedMapping: core.CanonicalID("m." + ref),
			ReferencedAttributes: []core.CanonicalID{attr(ref, "code")},
		})
	}
	return m
}

func tableNames(tables []Table) []string {
	var names []string
	for _, t := range tables {
		names = append(names, t.Name)
	}
	return names
}

func TestTablesComeAfterTheTablesTheirForeignKeysReference(t *testing.T) {
	// alpha references beta, which comes later in name order; gamma references alpha.
	// Running the tables in output order must create each referenced table first.
	w := ir.World{Persistence: []persistence.Mapping{fkMapping("alpha", "beta"), fkMapping("beta"), fkMapping("gamma", "alpha"), fkMapping("delta")}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unresolved) != 0 {
		t.Fatalf("unresolved = %+v", res.Unresolved)
	}
	pos := map[string]int{}
	for i, name := range tableNames(res.Tables) {
		pos[name[strings.LastIndex(name, ".")+1:]] = i
	}
	if !(pos["beta"] < pos["alpha"] && pos["alpha"] < pos["gamma"]) {
		t.Fatalf("referenced tables are not created first: %v", tableNames(res.Tables))
	}
	if pos["beta"] != 0 || pos["alpha"] != 1 || pos["delta"] != 2 {
		t.Fatalf("ready tables do not keep name order: %v", tableNames(res.Tables))
	}
}

func TestForeignKeyCycleIsReportedInsteadOfEmittedInAFailingOrder(t *testing.T) {
	w := ir.World{Persistence: []persistence.Mapping{fkMapping("alpha", "beta"), fkMapping("beta", "alpha"), fkMapping("gamma", "alpha"), fkMapping("delta")}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tables) != 1 || !strings.HasSuffix(res.Tables[0].Name, "delta") {
		t.Fatalf("renderable tables = %v", tableNames(res.Tables))
	}
	if len(res.Unresolved) != 3 {
		t.Fatalf("unresolved = %+v", res.Unresolved)
	}
	for _, u := range res.Unresolved {
		if !strings.Contains(u.Reason, "循環") {
			t.Fatalf("unresolved = %+v", res.Unresolved)
		}
	}
}

func TestTableReferencingARefusedTableIsNotEmitted(t *testing.T) {
	// beta's primary key column is nullable, so the provider refuses beta;
	// alpha references beta and would fail at run time, so it is reported too.
	beta := fkMapping("beta")
	beta.Columns[0].Nullability = persistence.Optional
	w := ir.World{Persistence: []persistence.Mapping{fkMapping("alpha", "beta"), beta, fkMapping("delta")}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tables) != 1 || !strings.HasSuffix(res.Tables[0].Name, "delta") {
		t.Fatalf("renderable tables = %v", tableNames(res.Tables))
	}
	var alphaReason string
	for _, u := range res.Unresolved {
		if strings.HasSuffix(u.Table, "alpha") {
			alphaReason = u.Reason
		}
	}
	if len(res.Unresolved) != 2 || !strings.Contains(alphaReason, "参照先の表") {
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
