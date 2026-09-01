package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func TestAdvancedIndexAuthoringConvergesThroughDesiredAndDiff(t *testing.T) {
	mapping := persistence.Mapping{
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
			Name:   "customers_name_advanced_idx",
			Method: "btree",
			Keys: []persistence.IndexKeyMapping{{
				Attribute:     core.MustCanonicalID("customer.name"),
				Collation:     "pg_catalog.C",
				OperatorClass: "pg_catalog.varchar_pattern_ops",
				Order:         persistence.IndexOrderDesc,
				NullsOrder:    persistence.IndexNullsOrderLast,
			}},
			IncludedAttributes: []core.CanonicalID{core.MustCanonicalID("customer.external_id")},
			Predicate: &persistence.ProviderExpressionMapping{
				Provider: environment.ProviderPostgres,
				Dialect:  "postgresql",
				Source:   "profile IS NOT NULL",
			},
		}},
	}

	desired, err := persistence.RealizeDesired([]persistence.Mapping{mapping}, "sales_core", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	actual := desired.Normalize()
	for tableIndex := range actual.Tables {
		for index := range actual.Tables[tableIndex].Indexes {
			actual.Tables[tableIndex].Indexes[index].State = &physical.IndexState{Valid: true, Ready: true, Live: true}
		}
	}

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("same advanced intent did not converge: %#v", diff.Changes)
	}

	actual.Tables[0].Indexes[0].IncludedColumns = nil
	diff, err = schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("advanced structural drift=%#v", diff.Changes)
	}
}
