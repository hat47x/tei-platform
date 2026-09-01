package realization_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/realization"
	"tei/internal/environment"
)

func TestPolicyForSelectsBuiltInOfflinePolicies(t *testing.T) {
	pg, err := realization.PolicyFor(environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if pg.Provider() != environment.ProviderPostgres {
		t.Fatalf("postgres provider=%q", pg.Provider())
	}
	if got, err := pg.PhysicalType(persistence.StorageInt64); err != nil || got != "pg_catalog.int8" {
		t.Fatalf("postgres int64=%q err=%v", got, err)
	}
	defaults := pg.IndexDefaults()
	if defaults.Method != "btree" || defaults.UniqueNullTreatment != physical.UniqueNullTreatmentDistinct {
		t.Fatalf("postgres defaults=%#v", defaults)
	}

	ora, err := realization.PolicyFor(environment.ProviderOracle)
	if err != nil {
		t.Fatal(err)
	}
	if ora.Provider() != environment.ProviderOracle {
		t.Fatalf("oracle provider=%q", ora.Provider())
	}
	if got, err := ora.PhysicalType(persistence.StorageInt64); err != nil || got != "NUMBER(19)" {
		t.Fatalf("oracle int64=%q err=%v", got, err)
	}
}

func TestPolicyForRejectsProviderWithoutBuiltInPolicy(t *testing.T) {
	_, err := realization.PolicyFor(environment.ProviderGeneric)
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompositionRealizeDesiredKeepsProviderSpecificOutput(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("customer.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("customer.id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("customer.name"), Column: "name", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
		Indexes: []persistence.IndexMapping{{
			Name:   "customers_name_uidx",
			Unique: true,
			Keys: []persistence.IndexKeyMapping{{
				Attribute:  core.MustCanonicalID("customer.name"),
				Collation:  "pg_catalog.default",
				Order:      persistence.IndexOrderAsc,
				NullsOrder: persistence.IndexNullsOrderLast,
			}},
		}},
	}

	pg, err := realization.RealizeDesired([]persistence.Mapping{mapping}, "sales_core", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	pgTable, ok := pg.Table("public.customers")
	if !ok {
		t.Fatal("postgres customers table not found")
	}
	if got := pgTable.Columns[0].PhysicalType; got != "pg_catalog.int8" {
		t.Fatalf("postgres id type=%q", got)
	}
	if len(pgTable.Indexes) != 1 {
		t.Fatalf("postgres indexes=%#v", pgTable.Indexes)
	}
	pgIndex := pgTable.Indexes[0]
	if pgIndex.Method != "btree" || pgIndex.UniqueNullTreatment != physical.UniqueNullTreatmentDistinct {
		t.Fatalf("postgres index defaults=%#v", pgIndex)
	}
	if len(pgIndex.Keys) != 1 || pgIndex.Keys[0].Order != physical.IndexOrderDefault || pgIndex.Keys[0].NullsOrder != physical.IndexNullsOrderDefault {
		t.Fatalf("postgres normalized key=%#v", pgIndex.Keys)
	}
	if pgIndex.Keys[0].Collation != "pg_catalog.default" {
		t.Fatalf("explicit collation must survive offline realization: %#v", pgIndex.Keys)
	}

	oraMapping := mapping
	oraMapping.Indexes = nil
	ora, err := realization.RealizeDesired([]persistence.Mapping{oraMapping}, "sales_core", environment.ProviderOracle)
	if err != nil {
		t.Fatal(err)
	}
	oraTable, ok := ora.Table("public.customers")
	if !ok {
		t.Fatal("oracle customers table not found")
	}
	if got := oraTable.Columns[0].PhysicalType; got != "NUMBER(19)" {
		t.Fatalf("oracle id type=%q", got)
	}
}
