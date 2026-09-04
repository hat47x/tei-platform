package syntax_test

import (
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/syntax"
)

func TestParseRelationStorageKeepsRelationAndSourceAttributeIdentity(t *testing.T) {
	src := `relation-storage sales_core.order_line {
    relation sales.order.lines
    component sales.order.order_id -> order_id storage int64 required
}`

	got, err := syntax.NewParser(src).ParseRelationStorage()
	if err != nil {
		t.Fatalf("Relation Storageをparseできません: %v", err)
	}
	if got.Mapping != "sales_core.order_line" {
		t.Fatalf("合流先Persistence Mappingが失われています: %s", got.Mapping)
	}
	if got.Storage.Relation != "sales.order.lines" {
		t.Fatalf("Semantic Relation identityが失われています: %s", got.Storage.Relation)
	}
	if len(got.Storage.Components) != 1 {
		t.Fatalf("component数が不正です: %#v", got.Storage.Components)
	}
	component := got.Storage.Components[0]
	if component.SourceAttribute != "sales.order.order_id" {
		t.Fatalf("Relation source attribute identityが失われています: %s", component.SourceAttribute)
	}
	if component.Column != "order_id" || component.Storage != persistence.StorageInt64 || component.Nullability != persistence.Required {
		t.Fatalf("物理support column契約が変わっています: %#v", component)
	}
}

func TestParseRelationStoragePreservesCompositeComponentOrder(t *testing.T) {
	src := `relation-storage sales_core.order_line {
    relation sales.order.lines
    component sales.order.tenant_id -> tenant_id storage int64 required
    component sales.order.order_id -> order_id storage int64 required
}`

	got, err := syntax.NewParser(src).ParseRelationStorage()
	if err != nil {
		t.Fatalf("複合Relation Storageをparseできません: %v", err)
	}
	if len(got.Storage.Components) != 2 {
		t.Fatalf("component数が不正です: %#v", got.Storage.Components)
	}
	if got.Storage.Components[0].SourceAttribute != "sales.order.tenant_id" || got.Storage.Components[1].SourceAttribute != "sales.order.order_id" {
		t.Fatalf("複合source identityの順序が変わっています: %#v", got.Storage.Components)
	}
}

func TestParseRelationStorageRequiresExplicitRelation(t *testing.T) {
	src := `relation-storage sales_core.order_line {
    component sales.order.order_id -> order_id storage int64 required
}`

	if _, err := syntax.NewParser(src).ParseRelationStorage(); err == nil {
		t.Fatal("Semantic Relationを省略したRelation Storageを受理しています")
	}
}

func TestParseRelationStorageRequiresExplicitComponent(t *testing.T) {
	src := `relation-storage sales_core.order_line {
    relation sales.order.lines
}`

	if _, err := syntax.NewParser(src).ParseRelationStorage(); err == nil {
		t.Fatal("物理support columnを持たないRelation Storageを受理しています")
	}
}

func TestParseRelationStorageRejectsDuplicateRelationDeclaration(t *testing.T) {
	src := `relation-storage sales_core.order_line {
    relation sales.order.lines
    relation sales.order.lines
    component sales.order.order_id -> order_id storage int64 required
}`

	if _, err := syntax.NewParser(src).ParseRelationStorage(); err == nil {
		t.Fatal("relationを複数宣言したRelation Storageを受理しています")
	}
}
