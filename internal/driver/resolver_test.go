package driver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestReferenceOrderEntryPatternsResolveWithoutDriverSpecificMeaning(t *testing.T) {
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)

	t.Run("browseはQueryを正本としてsales Datasourceへ接続する", func(t *testing.T) {
		resolved := mustResolvePattern(t, resolver, "order_entry", "order_entry.recent_orders")
		if resolved.Pattern.Kind != application.PatternBrowse {
			t.Fatalf("kind=%q", resolved.Pattern.Kind)
		}
		if resolved.Data == nil || resolved.Data.ID != "sales.order" {
			t.Fatalf("data=%#v", resolved.Data)
		}
		if resolved.Query == nil || resolved.Query.ID != "order_entry.recent_orders" {
			t.Fatalf("query=%#v", resolved.Query)
		}
		if resolved.Datasource == nil ||
			resolved.Datasource.Alias != "sales" ||
			resolved.Datasource.TargetDatabase != "sales_core" ||
			resolved.Datasource.Access != database.AccessReadWrite {
			t.Fatalf("datasource=%#v", resolved.Datasource)
		}
		if resolved.Relation != nil || resolved.Capability != nil {
			t.Fatalf("browseに不要な意味が解決されています: relation=%#v capability=%#v", resolved.Relation, resolved.Capability)
		}
	})

	t.Run("recordはDataだけを解決してDatasourceを推測しない", func(t *testing.T) {
		resolved := mustResolvePattern(t, resolver, "order_entry", "order_entry.order_record")
		if resolved.Pattern.Kind != application.PatternRecord {
			t.Fatalf("kind=%q", resolved.Pattern.Kind)
		}
		if resolved.Data == nil || resolved.Data.ID != "sales.order" {
			t.Fatalf("data=%#v", resolved.Data)
		}
		if resolved.Datasource != nil || resolved.Query != nil || resolved.Relation != nil || resolved.Capability != nil {
			t.Fatalf("recordへCanonicalにない実現情報を補っています: %#v", resolved)
		}
	})

	t.Run("referenceはRelationと明示されたmaster Datasourceを解決する", func(t *testing.T) {
		resolved := mustResolvePattern(t, resolver, "order_entry", "order_entry.customer_reference")
		if resolved.Pattern.Kind != application.PatternReference {
			t.Fatalf("kind=%q", resolved.Pattern.Kind)
		}
		if resolved.Relation == nil ||
			resolved.Relation.Relation.ID != "sales.order.customer" ||
			resolved.Relation.Source.ID != "sales.order" ||
			resolved.Relation.Target.ID != "customer.customer" {
			t.Fatalf("relation=%#v", resolved.Relation)
		}
		if resolved.Datasource == nil ||
			resolved.Datasource.Alias != "master" ||
			resolved.Datasource.TargetDatabase != "master_data" ||
			resolved.Datasource.Access != database.AccessRead {
			t.Fatalf("datasource=%#v", resolved.Datasource)
		}
		if resolved.Data != nil || resolved.Query != nil || resolved.Capability != nil {
			t.Fatalf("referenceへ不要な意味が解決されています: %#v", resolved)
		}
	})

	t.Run("header-detailはmany Relationだけを解決して保存先を推測しない", func(t *testing.T) {
		resolved := mustResolvePattern(t, resolver, "order_entry", "order_entry.order_lines")
		if resolved.Pattern.Kind != application.PatternHeaderDetail {
			t.Fatalf("kind=%q", resolved.Pattern.Kind)
		}
		if resolved.Relation == nil ||
			resolved.Relation.Relation.ID != "sales.order.lines" ||
			resolved.Relation.Relation.Cardinality != semantic.CardinalityMany ||
			resolved.Relation.Source.ID != "sales.order" ||
			resolved.Relation.Target.ID != "sales.order_line" {
			t.Fatalf("relation=%#v", resolved.Relation)
		}
		if resolved.Datasource != nil || resolved.Data != nil || resolved.Query != nil || resolved.Capability != nil {
			t.Fatalf("header-detailへCanonicalにない実現情報を補っています: %#v", resolved)
		}
	})

	t.Run("commandはCapabilityまでを解決してBinding実行を先回りしない", func(t *testing.T) {
		resolved := mustResolvePattern(t, resolver, "order_entry", "order_entry.submit_order")
		if resolved.Pattern.Kind != application.PatternCommand {
			t.Fatalf("kind=%q", resolved.Pattern.Kind)
		}
		if resolved.Capability == nil || resolved.Capability.ID != "sales.order.submit" || resolved.Capability.Output != "sales.order" {
			t.Fatalf("capability=%#v", resolved.Capability)
		}
		if resolved.Datasource != nil || resolved.Data != nil || resolved.Query != nil || resolved.Relation != nil {
			t.Fatalf("commandへCapability以外の実現情報を補っています: %#v", resolved)
		}
	})
}

func TestResolverDoesNotSilentlyRepairInvalidCanonicalReferences(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{
			{
				ID: "app_a",
				Databases: []application.DatabaseRequirement{
					{Alias: "read_only", TargetDatabase: "db_a", Access: database.AccessRead},
					{Alias: "write_only", TargetDatabase: "db_b", Access: database.AccessWrite},
				},
				Patterns: []application.Pattern{
					{ID: "app_a.browse", Kind: application.PatternBrowse, Data: "sales.order", Query: "app_b.query"},
					{ID: "app_a.reference", Kind: application.PatternReference, Relation: "sales.order.customer", Datasource: "write_only"},
				},
			},
		},
		Queries: []query.EmbeddedSQL{
			{ID: "app_b.query", Application: "app_b", Datasource: "read_only"},
		},
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.order",
						Relations: []semantic.Relation{
							{ID: "sales.order.customer", Target: "customer.customer", Cardinality: semantic.CardinalityOne},
						},
					},
				},
			},
			{ID: "customer", Data: []semantic.Data{{ID: "customer.customer"}}},
		},
	}
	resolver := driver.NewResolver(world)

	if _, err := resolver.ResolvePattern("missing", "app_a.browse"); err == nil {
		t.Fatal("存在しないApplicationを暗黙に補っています")
	}
	if _, err := resolver.ResolvePattern("app_a", "missing"); err == nil {
		t.Fatal("存在しないPatternを暗黙に補っています")
	}
	if _, err := resolver.ResolvePattern("app_a", "app_a.browse"); err == nil {
		t.Fatal("別ApplicationのQueryを暗黙に受理しています")
	}
	if _, err := resolver.ResolvePattern("app_a", "app_a.reference"); err == nil {
		t.Fatal("write-only Datasourceを読み取り用途として暗黙に受理しています")
	}
}

func referenceWorld(t *testing.T) ir.World {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceを読み込めません: %v", err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("Reference WorkspaceにValidation Errorがあります: %s: %s", diagnostic.Code, diagnostic.Message)
		}
	}
	return loaded.World
}

func mustResolvePattern(t *testing.T, resolver driver.Resolver, applicationID, patternID core.CanonicalID) driver.PatternResolution {
	t.Helper()
	resolved, err := resolver.ResolvePattern(applicationID, patternID)
	if err != nil {
		t.Fatalf("%s / %sを解決できません: %v", applicationID, patternID, err)
	}
	return resolved
}
