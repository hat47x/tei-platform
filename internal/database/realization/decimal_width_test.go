package realization

import (
	"strings"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/providers/oracle"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func widthPtr(v int) *int { return &v }

// Both providers render the declared width, and neither invents one when the
// declaration has none. The expected strings are each provider's documented
// decimal type: PostgreSQL numeric(p,s), Oracle NUMBER(p,s).
func TestProviderDecimalTypeFollowsTheDeclaredWidth(t *testing.T) {
	pg, _ := (postgres.RealizationPolicy{}).DecimalType(12, 3)
	if pg != "pg_catalog.numeric(12,3)" {
		t.Fatalf("postgres=%q", pg)
	}
	ora, _ := (oracle.RealizationPolicy{}).DecimalType(12, 3)
	if ora != "NUMBER(12,3)" {
		t.Fatalf("oracle=%q", ora)
	}
	if got, err := (postgres.RealizationPolicy{}).PhysicalType(persistence.StorageDecimal); err != nil || got != "pg_catalog.numeric" {
		t.Fatalf("bare decimal changed: %q err=%v", got, err)
	}
}

// Realizing a mapping with a declared width reaches the physical column, and
// the DDL renderer emits it. This is the path `tei desired-schema -format sql`
// takes, so the test asserts the emitted statement rather than only the string.
func TestDeclaredDecimalWidthReachesTheRenderedDDL(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           "sales_core.order_line",
		Database:     "sales_core",
		SemanticData: "sales.order_line",
		Table:        "public.order_lines",
		Columns: []persistence.ColumnMapping{{
			Attribute: "sales.order_line.quantity", Column: "quantity",
			Storage: persistence.StorageDecimal, Nullability: persistence.Required,
			Precision: widthPtr(12), Scale: widthPtr(3),
		}},
	}
	snapshot, err := persistence.RealizeDesired([]persistence.Mapping{mapping}, "sales_core", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatalf("realize: %v", err)
	}
	if got := snapshot.Tables[0].Columns[0].PhysicalType; got != "pg_catalog.numeric(12,3)" {
		t.Fatalf("physical type=%q", got)
	}

	// Compare against an empty schema of the same database: every declared
	// table is then a create, which is the DDL `tei desired-schema -format sql`
	// renders.
	empty := physical.SchemaSnapshot{Database: snapshot.Database}
	diff, err := schemadiff.Compare(snapshot.Normalize(), empty.Normalize())
	if err != nil {
		t.Fatal(err)
	}
	statements, err := migration.Render(environment.ProviderPostgres, diff)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(statements, "\n")
	if !strings.Contains(joined, "pg_catalog.numeric(12,3)") {
		t.Fatalf("rendered DDL lost the width:\n%s", joined)
	}
}
