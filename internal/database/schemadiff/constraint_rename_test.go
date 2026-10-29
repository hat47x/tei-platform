package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

// #1803: a declared constraint name is part of the desired state; a name the
// realization generated for an unnamed definition is not compared.

func renameTable(unique []physical.UniqueConstraint, fks []physical.ForeignKey) physical.SchemaSnapshot {
	return physical.SchemaSnapshot{
		Database: core.MustCanonicalID("sales_core"),
		Tables: []physical.Table{{
			Schema: "public", Name: "customers", Kind: physical.TableKindTable,
			UniqueConstraints: unique, ForeignKeys: fks,
		}},
	}.Normalize()
}

func TestCompareDeclaredUniqueNameDifferenceIsRename(t *testing.T) {
	desired := renameTable([]physical.UniqueConstraint{{Name: "uq_email", Columns: []string{"email"}}}, nil)
	actual := renameTable([]physical.UniqueConstraint{{Name: "customers_email_key", Columns: []string{"email"}}}, nil)
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 1 {
		t.Fatalf("changes=%#v", diff.Changes)
	}
	c := diff.Changes[0]
	if c.Kind != schemadiff.RenameConstraint || c.BeforeName != "customers_email_key" || c.UniqueConstraint == nil || c.UniqueConstraint.Name != "uq_email" || c.Table != "public.customers" {
		t.Fatalf("change=%#v", c)
	}
}

func TestCompareDeclaredForeignKeyNameDifferenceIsRename(t *testing.T) {
	fk := func(name string, generated bool) []physical.ForeignKey {
		return []physical.ForeignKey{{Name: name, NameGenerated: generated, Columns: []string{"region"}, ReferencedSchema: "public", ReferencedTable: "regions", ReferencedColumns: []string{"code"}}}
	}
	diff, err := schemadiff.Compare(renameTable(nil, fk("fk_region", false)), renameTable(nil, fk("customers_region_fkey", false)))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.RenameConstraint || diff.Changes[0].ForeignKey == nil || diff.Changes[0].BeforeName != "customers_region_fkey" {
		t.Fatalf("changes=%#v", diff.Changes)
	}
	diff, err = schemadiff.Compare(renameTable(nil, fk("fk_region", true)), renameTable(nil, fk("customers_region_fkey1", false)))
	if err != nil || len(diff.Changes) != 0 {
		t.Fatalf("generated name must not be compared: %v %#v", err, diff.Changes)
	}
}

func TestCompareGeneratedAndEmptyNamesAreNotCompared(t *testing.T) {
	actual := renameTable([]physical.UniqueConstraint{{Name: "customers_email_key1", Columns: []string{"email"}}}, nil)
	for name, desired := range map[string]physical.UniqueConstraint{
		"generated": {Name: "customers_email_key", NameGenerated: true, Columns: []string{"email"}},
		"empty":     {Columns: []string{"email"}},
	} {
		diff, err := schemadiff.Compare(renameTable([]physical.UniqueConstraint{desired}, nil), actual)
		if err != nil || len(diff.Changes) != 0 {
			t.Fatalf("%s: %v %#v", name, err, diff.Changes)
		}
	}
}

func TestCompareEqualNamesProduceNoRename(t *testing.T) {
	both := []physical.UniqueConstraint{{Name: "uq_email", Columns: []string{"email"}}}
	diff, err := schemadiff.Compare(renameTable(both, nil), renameTable(both, nil))
	if err != nil || len(diff.Changes) != 0 {
		t.Fatalf("%v %#v", err, diff.Changes)
	}
}

// Equal names pair first, so the constraint that already has its declared name
// is left alone while the other one is renamed.
func TestCompareSameStructureConstraintsPairByNameFirst(t *testing.T) {
	desired := renameTable([]physical.UniqueConstraint{
		{Name: "a_first", Columns: []string{"email"}}, {Name: "b_second", Columns: []string{"email"}},
	}, nil)
	actual := renameTable([]physical.UniqueConstraint{
		{Name: "b_second", Columns: []string{"email"}}, {Name: "legacy", Columns: []string{"email"}},
	}, nil)
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.RenameConstraint || diff.Changes[0].BeforeName != "legacy" || diff.Changes[0].UniqueConstraint.Name != "a_first" {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}

func TestCompareRenameIsCaseSensitive(t *testing.T) {
	desired := renameTable([]physical.UniqueConstraint{{Name: "UQ_Email", Columns: []string{"email"}}}, nil)
	actual := renameTable([]physical.UniqueConstraint{{Name: "uq_email", Columns: []string{"email"}}}, nil)
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil || len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.RenameConstraint {
		t.Fatalf("%v %#v", err, diff.Changes)
	}
}

func TestComparePrimaryKeyHasNoName(t *testing.T) {
	d := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "t", PrimaryKey: []string{"id"}}}}
	diff, err := schemadiff.Compare(d, d)
	if err != nil || len(diff.Changes) != 0 {
		t.Fatalf("%v %#v", err, diff.Changes)
	}
}
