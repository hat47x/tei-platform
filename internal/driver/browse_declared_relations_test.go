package driver_test

import (
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/driver"
)

func TestDeclaredBrowseRelationsComesFromThePersistenceMappingOnly(t *testing.T) {
	world := referenceWorld(t)
	got, err := driver.NewResolver(world).DeclaredBrowseRelations(browseApp, browsePat)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != driver.DeclaredRelationsResolved || got.Data != "sales.order" || got.Database != "sales_core" {
		t.Fatalf("%+v", got)
	}
	if len(got.Tables) != 1 || got.Tables[0] != "public.orders" {
		t.Fatalf("tables=%v", got.Tables)
	}
	if len(got.PrimaryKey) != 1 || got.PrimaryKey[0] != "sales.order.order_id" {
		t.Fatalf("primary key=%v", got.PrimaryKey)
	}
}

func TestDeclaredBrowseRelationsIsNeverGuessedWhenTheMappingIsMissingOrAmbiguous(t *testing.T) {
	world := referenceWorld(t)
	var kept, dropped []persistence.Mapping
	for _, m := range world.Persistence {
		if m.SemanticData == "sales.order" && m.Database == "sales_core" {
			dropped = append(dropped, m)
			continue
		}
		kept = append(kept, m)
	}
	if len(dropped) != 1 {
		t.Fatalf("premise: the Reference Workspace declares one sales.order mapping, got %d", len(dropped))
	}

	none := world
	none.Persistence = kept
	got, err := driver.NewResolver(none).DeclaredBrowseRelations(browseApp, browsePat)
	if err != nil || got.Status != driver.DeclaredRelationsNone || len(got.Tables) != 0 {
		t.Fatalf("a missing mapping must yield no tables: %+v %v", got, err)
	}

	twin := dropped[0]
	twin.ID = "sales_core.order_other"
	twin.Table = "public.orders_archive"
	ambiguous := world
	ambiguous.Persistence = append(append([]persistence.Mapping(nil), world.Persistence...), twin)
	got, err = driver.NewResolver(ambiguous).DeclaredBrowseRelations(browseApp, browsePat)
	if err != nil || got.Status != driver.DeclaredRelationsAmbiguous || len(got.Tables) != 0 {
		t.Fatalf("an ambiguous mapping must choose nothing: %+v %v", got, err)
	}
}

func TestDeclaredBrowseRelationsRefusesAPatternWithoutData(t *testing.T) {
	if _, err := driver.NewResolver(referenceWorld(t)).DeclaredBrowseRelations(browseApp, "order_entry.submit_order"); err == nil {
		t.Fatal("a command Pattern has no declared browse relations")
	}
}
