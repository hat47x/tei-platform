package persistence_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/environment"
)

func TestRealizeDesiredAdvancedIndex(t *testing.T) {
	m := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("customer.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("customer.name"), Column: "name", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("customer.external_id"), Column: "external_id", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("customer.profile"), Column: "profile", Storage: persistence.StorageJSON, Nullability: persistence.Optional},
		},
		Indexes: []persistence.IndexMapping{{
			Name:   "customers_name_advanced_uidx",
			Unique: true,
			Method: "btree",
			Keys: []persistence.IndexKeyMapping{
				{
					Attribute:     core.MustCanonicalID("customer.name"),
					Collation:     "pg_catalog.C",
					OperatorClass: "pg_catalog.varchar_pattern_ops",
					Order:         persistence.IndexOrderDesc,
					NullsOrder:    persistence.IndexNullsOrderLast,
				},
				{
					Expression: &persistence.ProviderExpressionMapping{
						Provider: environment.ProviderPostgres,
						Dialect:  "postgresql",
						Source:   " lower(name) ",
					},
				},
			},
			IncludedAttributes: []core.CanonicalID{core.MustCanonicalID("customer.external_id")},
			Predicate: &persistence.ProviderExpressionMapping{
				Provider: environment.ProviderPostgres,
				Dialect:  "postgresql",
				Source:   " profile IS NOT NULL ",
			},
			UniqueNullTreatment: persistence.UniqueNullTreatmentNotDistinct,
		}},
	}

	snapshot, err := persistence.RealizeDesired([]persistence.Mapping{m}, "sales_core", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	table, ok := snapshot.Table("public.customers")
	if !ok || len(table.Indexes) != 1 {
		t.Fatalf("table=%#v", table)
	}
	index := table.Indexes[0]
	if index.Name != "customers_name_advanced_uidx" || !index.Unique || index.Method != "btree" {
		t.Fatalf("index=%#v", index)
	}
	if index.UniqueNullTreatment != physical.UniqueNullTreatmentNotDistinct || index.State != nil {
		t.Fatalf("unique/state=%#v", index)
	}
	if len(index.Keys) != 2 {
		t.Fatalf("keys=%#v", index.Keys)
	}
	first := index.Keys[0]
	if first.Column != "name" || first.Expression != nil || first.Collation != "pg_catalog.C" || first.OperatorClass != "pg_catalog.varchar_pattern_ops" || first.Order != physical.IndexOrderDesc || first.NullsOrder != physical.IndexNullsOrderLast {
		t.Fatalf("first key=%#v", first)
	}
	second := index.Keys[1]
	if second.Column != "" || second.Expression == nil || second.Expression.Provider != "postgres" || second.Expression.Dialect != "postgresql" || second.Expression.Source != "lower(name)" {
		t.Fatalf("second key=%#v", second)
	}
	if second.Order != physical.IndexOrderDefault || second.NullsOrder != physical.IndexNullsOrderDefault {
		t.Fatalf("second defaults=%#v", second)
	}
	if len(index.IncludedColumns) != 1 || index.IncludedColumns[0] != "external_id" {
		t.Fatalf("include=%#v", index.IncludedColumns)
	}
	if index.Predicate == nil || index.Predicate.Source != "profile IS NOT NULL" || index.Predicate.Provider != "postgres" || index.Predicate.Dialect != "postgresql" {
		t.Fatalf("predicate=%#v", index.Predicate)
	}
}

func TestRealizeDesiredAdvancedIndexRejectsInvalidIntent(t *testing.T) {
	base := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("customer.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("customer.name"), Column: "name", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
	}

	tests := []struct {
		name      string
		index     persistence.IndexMapping
		wantError string
	}{
		{
			name: "mixed simple and advanced",
			index: persistence.IndexMapping{
				Name:       "mixed_idx",
				Attributes: []core.CanonicalID{core.MustCanonicalID("customer.name")},
				Keys:       []persistence.IndexKeyMapping{{Attribute: core.MustCanonicalID("customer.name")}},
			},
			wantError: "mixes simple attributes and advanced keys",
		},
		{
			name:      "no key",
			index:     persistence.IndexMapping{Name: "empty_idx"},
			wantError: "has no keys",
		},
		{
			name: "provider mismatch",
			index: persistence.IndexMapping{
				Name: "provider_idx",
				Keys: []persistence.IndexKeyMapping{{Expression: &persistence.ProviderExpressionMapping{Provider: environment.ProviderOracle, Dialect: "oracle", Source: "lower(name)"}}},
			},
			wantError: "does not match desired provider",
		},
		{
			name: "unknown attribute",
			index: persistence.IndexMapping{
				Name: "unknown_idx",
				Keys: []persistence.IndexKeyMapping{{Attribute: core.MustCanonicalID("customer.missing")}},
			},
			wantError: "has no column mapping",
		},
		{
			name: "non unique null treatment",
			index: persistence.IndexMapping{
				Name:                "bad_nulls_idx",
				Keys:                []persistence.IndexKeyMapping{{Attribute: core.MustCanonicalID("customer.name")}},
				UniqueNullTreatment: persistence.UniqueNullTreatmentNotDistinct,
			},
			wantError: "cannot declare NULL treatment",
		},
		{
			name: "invalid key shape",
			index: persistence.IndexMapping{
				Name: "bad_key_idx",
				Keys: []persistence.IndexKeyMapping{{
					Attribute:   core.MustCanonicalID("customer.name"),
					Expression: &persistence.ProviderExpressionMapping{Provider: environment.ProviderPostgres, Dialect: "postgresql", Source: "lower(name)"},
				}},
			},
			wantError: "exactly one attribute or expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := base
			m.Indexes = []persistence.IndexMapping{tt.index}
			_, err := persistence.RealizeDesired([]persistence.Mapping{m}, "sales_core", environment.ProviderPostgres)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("err=%v want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestRealizeDesiredSimpleIndexCompatibility(t *testing.T) {
	m := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("customer.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("customer.name"), Column: "name", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
		Indexes: []persistence.IndexMapping{{Name: "customers_name_uidx", Attributes: []core.CanonicalID{core.MustCanonicalID("customer.name")}, Unique: true}},
	}

	snapshot, err := persistence.RealizeDesired([]persistence.Mapping{m}, "sales_core", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	table, ok := snapshot.Table("public.customers")
	if !ok || len(table.Indexes) != 1 {
		t.Fatalf("table=%#v", table)
	}
	index := table.Indexes[0]
	if index.Method != "btree" || index.UniqueNullTreatment != physical.UniqueNullTreatmentDistinct || len(index.Keys) != 1 || index.Keys[0].Column != "name" {
		t.Fatalf("simple index changed meaning: %#v", index)
	}
}
