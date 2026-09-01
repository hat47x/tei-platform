package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func uniqueSchema(constraints ...physical.UniqueConstraint) physical.SchemaSnapshot {
	return physical.SchemaSnapshot{
		Database: core.MustCanonicalID("sales_core"),
		Tables: []physical.Table{{
			Schema:            "public",
			Name:              "customers",
			Kind:              physical.TableKindTable,
			UniqueConstraints: constraints,
		}},
	}.Normalize()
}

func TestCompareUniqueConstraintIgnoresNameOnlyDifference(t *testing.T) {
	desired := uniqueSchema(physical.UniqueConstraint{Name: "customers_email_key", Columns: []string{"email"}})
	actual := uniqueSchema(physical.UniqueConstraint{Name: "generated_unique_42", Columns: []string{"email"}})

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("name-only difference produced drift: %#v", diff.Changes)
	}
}

func TestCompareUniqueConstraintStructuralDifferenceDropsThenAdds(t *testing.T) {
	desired := uniqueSchema(physical.UniqueConstraint{Name: "customers_identity_key", Columns: []string{"tenant_id", "external_id"}})
	actual := uniqueSchema(physical.UniqueConstraint{Name: "customers_identity_key", Columns: []string{"external_id", "tenant_id"}})

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 2 {
		t.Fatalf("changes=%#v", diff.Changes)
	}
	if diff.Changes[0].Kind != schemadiff.DropUniqueConstraint || diff.Changes[1].Kind != schemadiff.AddUniqueConstraint {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}

func TestCompareUniqueConstraintPreservesMultiplicity(t *testing.T) {
	constraintA := physical.UniqueConstraint{Name: "customers_email_key_a", Columns: []string{"email"}}
	constraintB := physical.UniqueConstraint{Name: "customers_email_key_b", Columns: []string{"email"}}
	desired := uniqueSchema(constraintA, constraintB)
	actual := uniqueSchema(constraintA)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.AddUniqueConstraint || diff.Changes[0].UniqueConstraint == nil {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}
