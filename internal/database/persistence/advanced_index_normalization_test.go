package persistence_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
)

func TestRealizeDesiredNormalizesPostgresBtreeDefaults(t *testing.T) {
	m := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("customer.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("customer.name"), Column: "name", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
		Indexes: []persistence.IndexMapping{
			{
				Name: "customers_name_asc_idx",
				Keys: []persistence.IndexKeyMapping{{
					Attribute:  core.MustCanonicalID("customer.name"),
					Collation:  "pg_catalog.default",
					Order:      persistence.IndexOrderAsc,
					NullsOrder: persistence.IndexNullsOrderLast,
				}},
			},
			{
				Name: "customers_name_desc_idx",
				Keys: []persistence.IndexKeyMapping{{
					Attribute:  core.MustCanonicalID("customer.name"),
					Order:      persistence.IndexOrderDesc,
					NullsOrder: persistence.IndexNullsOrderFirst,
				}},
			},
		},
	}

	snapshot, err := persistence.RealizeDesired([]persistence.Mapping{m}, "sales_core", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	table, ok := snapshot.Table("public.customers")
	if !ok || len(table.Indexes) != 2 {
		t.Fatalf("table=%#v", table)
	}

	ascending := table.Indexes[0].Keys[0]
	if ascending.Order != physical.IndexOrderDefault || ascending.NullsOrder != physical.IndexNullsOrderDefault || ascending.Collation != "" {
		t.Fatalf("ASC NULLS LAST/default collation were not normalized: %#v", ascending)
	}

	descending := table.Indexes[1].Keys[0]
	if descending.Order != physical.IndexOrderDesc || descending.NullsOrder != physical.IndexNullsOrderDefault {
		t.Fatalf("DESC NULLS FIRST was not normalized: %#v", descending)
	}
}
