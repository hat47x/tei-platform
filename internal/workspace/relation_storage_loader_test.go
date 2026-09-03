package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
)

func TestLoadRelationStorageAuthoringsMergesIntoExplicitPersistenceMapping(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "databases", "sales-core", "relation-storage", "order-line.tei")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	src := `relation-storage sales_core.order_line {
    relation sales.order.lines
    component sales.order.order_id -> order_id storage int64 required
}`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	r := Result{
		Sources: map[string]SourceLocation{},
	}
	r.World.Persistence = []persistence.Mapping{
		{ID: core.MustCanonicalID("sales_core.order")},
		{ID: core.MustCanonicalID("sales_core.order_line")},
	}

	if err := loadRelationStorageAuthorings(root, &r); err != nil {
		t.Fatalf("Relation StorageをPersistence Mappingへ合流できません: %v", err)
	}
	if len(r.World.Persistence[0].RelationStorage) != 0 {
		t.Fatalf("別のPersistence Mappingを変更しています: %#v", r.World.Persistence[0].RelationStorage)
	}
	if len(r.World.Persistence[1].RelationStorage) != 1 {
		t.Fatalf("Relation Storageが合流されていません: %#v", r.World.Persistence[1].RelationStorage)
	}
	storage := r.World.Persistence[1].RelationStorage[0]
	if storage.Relation != "sales.order.lines" || len(storage.Components) != 1 || storage.Components[0].Column != "order_id" {
		t.Fatalf("Relation Storageのidentityまたは物理契約が変わっています: %#v", storage)
	}

	key := "persistence-relation-storage:sales_core.order_line/sales.order.lines"
	loc, ok := r.Sources[key]
	if !ok {
		t.Fatalf("Relation StorageのSource Index keyがありません: %#v", r.Sources)
	}
	if loc.File != "databases/sales-core/relation-storage/order-line.tei" {
		t.Fatalf("Relation Storageのauthoring sourceが追跡できません: %#v", loc)
	}
	if len(r.SourceContributions) != 1 || r.SourceContributions[0].Subject.Kind != "persistence-relation-storage" || r.SourceContributions[0].Subject.ID != "sales_core.order_line/sales.order.lines" {
		t.Fatalf("Source contributionが不正です: %#v", r.SourceContributions)
	}
}

func TestLoadRelationStorageAuthoringsRejectsUnknownPersistenceMapping(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "databases", "sales-core", "relation-storage", "order-line.tei")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	src := `relation-storage sales_core.order_line {
    relation sales.order.lines
    component sales.order.order_id -> order_id storage int64 required
}`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	r := Result{Sources: map[string]SourceLocation{}}
	r.World.Persistence = []persistence.Mapping{{ID: core.MustCanonicalID("sales_core.order")}}

	err := loadRelationStorageAuthorings(root, &r)
	if err == nil {
		t.Fatal("存在しないPersistence MappingへのRelation Storageを受理しています")
	}
	if !strings.Contains(err.Error(), "sales_core.order_line") {
		t.Fatalf("診断から合流先identityを追跡できません: %v", err)
	}
}

func TestLoadRelationStorageAuthoringsRejectsDuplicateSourceIdentity(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "databases", "sales-core", "relation-storage")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `relation-storage sales_core.order_line {
    relation sales.order.lines
    component sales.order.order_id -> order_id storage int64 required
}`
	for _, name := range []string{"a.tei", "b.tei"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	r := Result{Sources: map[string]SourceLocation{}}
	r.World.Persistence = []persistence.Mapping{{ID: core.MustCanonicalID("sales_core.order_line")}}

	if err := loadRelationStorageAuthorings(root, &r); err == nil {
		t.Fatal("同一Mapping / Relation identityのauthoring source重複を受理しています")
	}
}
