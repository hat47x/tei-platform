package persistence_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/providers/oracle"
	"tei/internal/database/providers/postgres"
)

func namingMappings(childUnique []persistence.UniqueConstraintMapping, childFKs []persistence.ForeignKeyMapping) []persistence.Mapping {
	col := func(table, name string) persistence.ColumnMapping {
		return persistence.ColumnMapping{Attribute: core.CanonicalID("s." + table + "." + name), Column: name, Storage: persistence.StorageInt64, Nullability: persistence.Required}
	}
	parent := persistence.Mapping{
		ID: "m.parent", Database: "store", SemanticData: "s.parent", Table: "public.parent",
		Columns:    []persistence.ColumnMapping{col("parent", "a"), col("parent", "b")},
		PrimaryKey: []core.CanonicalID{"s.parent.a", "s.parent.b"},
	}
	child := persistence.Mapping{
		ID: "m.child", Database: "store", SemanticData: "s.child", Table: "public.child",
		Columns:           []persistence.ColumnMapping{col("child", "id"), col("child", "pa"), col("child", "pb")},
		PrimaryKey:        []core.CanonicalID{"s.child.id"},
		UniqueConstraints: childUnique,
		ForeignKeys:       childFKs,
	}
	return []persistence.Mapping{child, parent}
}

func TestRealizeDesiredNamesUnnamedConstraintsWithPostgresPolicy(t *testing.T) {
	mappings := namingMappings(
		[]persistence.UniqueConstraintMapping{
			{Attributes: []core.CanonicalID{"s.child.pb", "s.child.pa"}},
			{Name: "child_declared_uq", Attributes: []core.CanonicalID{"s.child.pa"}},
		},
		[]persistence.ForeignKeyMapping{{
			Attributes: []core.CanonicalID{"s.child.pa", "s.child.pb"}, ReferencedMapping: "m.parent",
			ReferencedAttributes: []core.CanonicalID{"s.parent.a", "s.parent.b"},
		}},
	)
	snapshot, err := persistence.RealizeDesired(mappings, "store", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	child, _ := snapshot.Table("public.child")
	var names []string
	for _, uc := range child.UniqueConstraints {
		names = append(names, uc.Name)
	}
	if got := strings.Join(names, ","); got != "child_declared_uq,child_pb_pa_key" {
		t.Fatalf("unique names = %s", got)
	}
	if len(child.ForeignKeys) != 1 || child.ForeignKeys[0].Name != "child_pa_pb_fkey" {
		t.Fatalf("foreign keys = %#v", child.ForeignKeys)
	}
	if len(child.PrimaryKey) != 1 {
		t.Fatalf("primary key = %#v", child.PrimaryKey)
	}
}

func TestRealizeDesiredReportsGeneratedNameCollisionPerTable(t *testing.T) {
	// The generated foreign key name equals a declared unique constraint name
	// on the same table: PostgreSQL would refuse it.
	mappings := namingMappings(
		[]persistence.UniqueConstraintMapping{{Name: "child_pa_pb_fkey", Attributes: []core.CanonicalID{"s.child.pb"}}},
		[]persistence.ForeignKeyMapping{{
			Attributes: []core.CanonicalID{"s.child.pa", "s.child.pb"}, ReferencedMapping: "m.parent",
			ReferencedAttributes: []core.CanonicalID{"s.parent.a", "s.parent.b"},
		}},
	)
	_, err := persistence.RealizeDesired(mappings, "store", postgres.RealizationPolicy{})
	var naming *persistence.ConstraintNamingError
	if !errors.As(err, &naming) || len(naming.Unresolved) != 1 || naming.Unresolved[0].Table != "public.child" {
		t.Fatalf("err=%v", err)
	}
	snapshot, unresolved, err := persistence.RealizeDesiredWithUnresolved(mappings, "store", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 1 || unresolved[0].Table != "public.child" || !strings.Contains(unresolved[0].Reason, `"child_pa_pb_fkey"`) {
		t.Fatalf("unresolved = %#v", unresolved)
	}
	child, _ := snapshot.Table("public.child")
	if child.ForeignKeys[0].Name != "" || child.UniqueConstraints[0].Name != "child_pa_pb_fkey" {
		t.Fatalf("generated name must be cleared, declared kept: %#v %#v", child.ForeignKeys, child.UniqueConstraints)
	}
	if _, ok := snapshot.Table("public.parent"); !ok {
		t.Fatal("other tables must stay in the snapshot")
	}
}

func TestRealizeDesiredLeavesNamesToProvidersWithoutNamingPolicy(t *testing.T) {
	mappings := namingMappings([]persistence.UniqueConstraintMapping{{Attributes: []core.CanonicalID{"s.child.pb"}}}, nil)
	snapshot, err := persistence.RealizeDesired(mappings, "store", oracle.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	child, _ := snapshot.Table("public.child")
	if child.UniqueConstraints[0].Name != "" {
		t.Fatalf("oracle policy has no naming rule yet: %#v", child.UniqueConstraints)
	}
}

// #1803: only a name the realization generated is marked as generated; the
// diff compares the declared ones.
func TestRealizeDesiredMarksGeneratedNamesOnly(t *testing.T) {
	mappings := namingMappings(
		[]persistence.UniqueConstraintMapping{
			{Attributes: []core.CanonicalID{"s.child.pb"}},
			{Name: "child_declared_uq", Attributes: []core.CanonicalID{"s.child.pa"}},
		},
		[]persistence.ForeignKeyMapping{
			{Attributes: []core.CanonicalID{"s.child.pa", "s.child.pb"}, ReferencedMapping: "m.parent", ReferencedAttributes: []core.CanonicalID{"s.parent.a", "s.parent.b"}},
		},
	)
	snapshot, err := persistence.RealizeDesired(mappings, "store", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	child, _ := snapshot.Table("public.child")
	for _, uc := range child.UniqueConstraints {
		if want := uc.Name != "child_declared_uq"; uc.NameGenerated != want {
			t.Fatalf("unique %q NameGenerated=%v", uc.Name, uc.NameGenerated)
		}
	}
	if len(child.ForeignKeys) != 1 || !child.ForeignKeys[0].NameGenerated {
		t.Fatalf("foreign keys=%#v", child.ForeignKeys)
	}
}
