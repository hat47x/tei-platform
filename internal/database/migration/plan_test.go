package migration_test

import (
	"strings"
	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"testing"
)

func TestPlanKeepsDatabaseLifecycleIndependentAndRendersPostgres(t *testing.T) {
	c := physical.Column{Name: "external_id", PhysicalType: "pg_catalog.varchar", Nullable: true}
	d := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.AddColumn, Table: "public.orders", Column: &c}}}
	m, err := migration.Plan("20260827_001", core.MustCanonicalID("sales_core"), "fp-1", d, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if m.Database != core.MustCanonicalID("sales_core") {
		t.Fatal("wrong database")
	}
	if len(m.Realizations) != 1 || len(m.Realizations[0].Statements) != 1 {
		t.Fatalf("unexpected realization: %#v", m.Realizations)
	}
	if !strings.Contains(m.Realizations[0].Statements[0], `ADD COLUMN "external_id"`) {
		t.Fatalf("unexpected sql: %s", m.Realizations[0].Statements[0])
	}
	if m.Steps[len(m.Steps)-1].Kind != migration.StepVerify {
		t.Fatal("migration must end with verification")
	}
}
