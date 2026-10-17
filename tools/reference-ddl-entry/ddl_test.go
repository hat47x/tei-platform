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
	m := persistence.Mapping{
		ID: core.CanonicalID(id), Database: "store", SemanticData: core.CanonicalID("s." + table), Table: table,
		Columns: []persistence.ColumnMapping{
			{Attribute: core.CanonicalID("s." + table + ".code"), Column: "code", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
	}
	if withPK {
		m.PrimaryKey = []core.CanonicalID{core.CanonicalID("s." + table + ".code")}
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

func TestProviderRefusalIsReportedPerTableNotHidden(t *testing.T) {
	w := ir.World{Persistence: []persistence.Mapping{mapping("m.a", "alpha", false), mapping("m.p", "pk_table", true)}}
	res, err := Render(w, "store", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tables) != 1 || !strings.HasSuffix(res.Tables[0].Name, "alpha") {
		t.Fatalf("renderable tables = %+v", res.Tables)
	}
	if len(res.Unresolved) != 1 || !strings.HasSuffix(res.Unresolved[0].Table, "pk_table") ||
		!strings.Contains(res.Unresolved[0].Reason, "primary key") {
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
