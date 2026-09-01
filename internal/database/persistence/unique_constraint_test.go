package persistence_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
)

func TestRealizeDesiredKeepsUniqueConstraintSeparateFromUniqueIndex(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           core.MustCanonicalID("sales.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("sales.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("sales.customer.id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("sales.customer.email"), Column: "email", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("sales.customer.external_id"), Column: "external_id", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
		UniqueConstraints: []persistence.UniqueConstraintMapping{{
			Name:       "customers_email_key",
			Attributes: []core.CanonicalID{core.MustCanonicalID("sales.customer.email")},
		}},
		Indexes: []persistence.IndexMapping{{
			Name:       "customers_external_id_idx",
			Attributes: []core.CanonicalID{core.MustCanonicalID("sales.customer.external_id")},
			Unique:     true,
		}},
	}

	snapshot, err := persistence.RealizeDesired([]persistence.Mapping{mapping}, "sales_core", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	table, ok := snapshot.Table("public.customers")
	if !ok {
		t.Fatal("public.customers not found")
	}
	if len(table.UniqueConstraints) != 1 {
		t.Fatalf("unique constraints=%#v", table.UniqueConstraints)
	}
	constraint := table.UniqueConstraints[0]
	if constraint.Name != "customers_email_key" || len(constraint.Columns) != 1 || constraint.Columns[0] != "email" {
		t.Fatalf("constraint=%#v", constraint)
	}
	if len(table.Indexes) != 1 || table.Indexes[0].Name != "customers_external_id_idx" || !table.Indexes[0].Unique {
		t.Fatalf("indexes=%#v", table.Indexes)
	}
}

func TestRealizeDesiredRejectsEmptyUniqueConstraint(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           core.MustCanonicalID("sales.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("sales.customer"),
		Table:        "customers",
		Columns: []persistence.ColumnMapping{{
			Attribute: core.MustCanonicalID("sales.customer.id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required,
		}},
		UniqueConstraints: []persistence.UniqueConstraintMapping{{Name: "customers_empty_key"}},
	}

	_, err := persistence.RealizeDesired([]persistence.Mapping{mapping}, "sales_core", postgres.RealizationPolicy{})
	if err == nil || !strings.Contains(err.Error(), "has no attributes") {
		t.Fatalf("error=%v", err)
	}
}

func TestRealizeDesiredRejectsUnknownUniqueConstraintAttribute(t *testing.T) {
	mapping := persistence.Mapping{
		ID:           core.MustCanonicalID("sales.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("sales.customer"),
		Table:        "customers",
		Columns: []persistence.ColumnMapping{{
			Attribute: core.MustCanonicalID("sales.customer.id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required,
		}},
		UniqueConstraints: []persistence.UniqueConstraintMapping{{
			Name:       "customers_email_key",
			Attributes: []core.CanonicalID{core.MustCanonicalID("sales.customer.email")},
		}},
	}

	_, err := persistence.RealizeDesired([]persistence.Mapping{mapping}, "sales_core", postgres.RealizationPolicy{})
	if err == nil || !strings.Contains(err.Error(), "has no column mapping") {
		t.Fatalf("error=%v", err)
	}
}
