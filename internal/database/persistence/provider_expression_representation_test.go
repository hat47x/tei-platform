package persistence_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/environment"
)

func TestDesiredProviderExpressionIsAuthored(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("customer.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("customer.name"), Column: "name", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("customer.profile"), Column: "profile", Storage: persistence.StorageJSON, Nullability: persistence.Optional},
		},
		Indexes: []persistence.IndexMapping{{
			Name: "customers_lower_name_idx",
			Keys: []persistence.IndexKeyMapping{{
				Expression: &persistence.ProviderExpressionMapping{
					Provider: environment.ProviderPostgres,
					Dialect:  "postgresql",
					Source:   " lower(name) ",
				},
			}},
			Predicate: &persistence.ProviderExpressionMapping{
				Provider: environment.ProviderPostgres,
				Dialect:  "postgresql",
				Source:   " profile IS NOT NULL ",
			},
		}},
	}

	snapshot, err := persistence.RealizeDesired([]persistence.Mapping{mapping}, "sales_core", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	table, ok := snapshot.Table("public.customers")
	if !ok || len(table.Indexes) != 1 {
		t.Fatalf("table=%#v", table)
	}
	index := table.Indexes[0]
	assertAuthoredExpression(t, index.Keys[0].Expression, "lower(name)")
	assertAuthoredExpression(t, index.Predicate, "profile IS NOT NULL")
}

func assertAuthoredExpression(t *testing.T, expression *physical.ProviderExpression, source string) {
	t.Helper()
	if expression == nil {
		t.Fatal("expression is nil")
	}
	if expression.Provider != "postgres" || expression.Dialect != "postgresql" || expression.Source != source {
		t.Fatalf("expression=%#v", expression)
	}
	if expression.Representation != physical.ProviderExpressionAuthored {
		t.Fatalf("representation=%q", expression.Representation)
	}
	if expression.Normalization != nil {
		t.Fatalf("authored expression unexpectedly has normalization provenance: %#v", expression.Normalization)
	}
}
