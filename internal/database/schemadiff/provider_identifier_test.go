package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func TestIndexCollationWithOneUnresolvedSideDoesNotBecomeMigrationChange(t *testing.T) {
	desired, actual := identifierSnapshots(
		physical.IndexKey{Column: "name"},
		physical.IndexKey{Column: "name", Collation: "pg_catalog.C"},
	)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("collation uncertainty must not become a migration change: %#v", diff.Changes)
	}
	if len(diff.Unresolved) != 1 {
		t.Fatalf("unresolved=%#v", diff.Unresolved)
	}
	got := diff.Unresolved[0]
	if got.Kind != schemadiff.UnresolvedProviderIdentifier || got.FieldPath != "index.keys[0].collation" || got.DesiredValue != "" || got.ActualValue != "pg_catalog.C" {
		t.Fatalf("unresolved=%#v", got)
	}
}

func TestExplicitDesiredCollationAgainstCollapsedActualDefaultIsUnresolved(t *testing.T) {
	desired, actual := identifierSnapshots(
		physical.IndexKey{Column: "name", Collation: "pg_catalog.default"},
		physical.IndexKey{Column: "name"},
	)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 1 {
		t.Fatalf("diff=%#v", diff)
	}
	got := diff.Unresolved[0]
	if got.Kind != schemadiff.UnresolvedProviderIdentifier || got.DesiredValue != "pg_catalog.default" || got.ActualValue != "" {
		t.Fatalf("unresolved=%#v", got)
	}
}

func TestDifferentExplicitCollationsRemainStructuralDrift(t *testing.T) {
	desired, actual := identifierSnapshots(
		physical.IndexKey{Column: "name", Collation: "pg_catalog.C"},
		physical.IndexKey{Column: "name", Collation: "public.ja_x_icu"},
	)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Unresolved) != 0 || len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("diff=%#v", diff)
	}
}

func TestOmittedDesiredOperatorClassDiffersFromExplicitActualNonDefault(t *testing.T) {
	desired, actual := identifierSnapshots(
		physical.IndexKey{Column: "name"},
		physical.IndexKey{Column: "name", OperatorClass: "pg_catalog.varchar_pattern_ops"},
	)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Unresolved) != 0 || len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("diff=%#v", diff)
	}
}

func TestExplicitDesiredOperatorClassAgainstActualDefaultIsUnresolved(t *testing.T) {
	desired, actual := identifierSnapshots(
		physical.IndexKey{Column: "name", OperatorClass: "pg_catalog.varchar_ops"},
		physical.IndexKey{Column: "name"},
	)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 1 {
		t.Fatalf("diff=%#v", diff)
	}
	got := diff.Unresolved[0]
	if got.Kind != schemadiff.UnresolvedProviderIdentifier || got.FieldPath != "index.keys[0].operatorClass" || got.DesiredValue != "pg_catalog.varchar_ops" || got.ActualValue != "" {
		t.Fatalf("unresolved=%#v", got)
	}
}

func TestSameExplicitProviderIdentifiersConverge(t *testing.T) {
	desired, actual := identifierSnapshots(
		physical.IndexKey{Column: "name", Collation: "pg_catalog.C", OperatorClass: "pg_catalog.varchar_pattern_ops"},
		physical.IndexKey{Column: "name", Collation: "pg_catalog.C", OperatorClass: "pg_catalog.varchar_pattern_ops"},
	)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 {
		t.Fatalf("diff=%#v", diff)
	}
}

func identifierSnapshots(desiredKey, actualKey physical.IndexKey) (physical.SchemaSnapshot, physical.SchemaSnapshot) {
	database := core.MustCanonicalID("sales_core")
	desired := physical.SchemaSnapshot{
		Database: database,
		Tables: []physical.Table{{
			Schema: "public",
			Name:   "customers",
			Kind:   physical.TableKindTable,
			Indexes: []physical.Index{{
				Name:   "customers_name_idx",
				Method: "btree",
				Keys:   []physical.IndexKey{desiredKey},
			}},
		}},
	}
	actual := physical.SchemaSnapshot{
		Database: database,
		Tables: []physical.Table{{
			Schema: "public",
			Name:   "customers",
			Kind:   physical.TableKindTable,
			Indexes: []physical.Index{{
				Name:   "customers_name_idx",
				Method: "btree",
				Keys:   []physical.IndexKey{actualKey},
			}},
		}},
	}
	return desired.Normalize(), actual.Normalize()
}
