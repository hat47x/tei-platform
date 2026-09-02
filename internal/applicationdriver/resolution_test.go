package applicationdriver_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationdriver"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/semantic"
)

func TestResolveScenarioResourcesRejectsUnknownAmbiguousOrConflictingResources(t *testing.T) {
	browse := application.Pattern{ID: "sample.browse", Kind: application.PatternBrowse, Query: "sample.query"}
	reference := application.Pattern{ID: "sample.reference", Kind: application.PatternReference, Relation: "sales.order.customer", Datasource: "master"}
	headerDetail := application.Pattern{ID: "sample.lines", Kind: application.PatternHeaderDetail, Relation: "sales.order.lines"}
	command := application.Pattern{ID: "sample.submit", Kind: application.PatternCommand, Capability: "sales.order.submit"}

	baseApp := application.Application{
		ID: "sample",
		Databases: []application.DatabaseRequirement{
			{Alias: "sales", TargetDatabase: "sales_core"},
			{Alias: "master", TargetDatabase: "master_data"},
		},
	}
	baseSemantics := []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{
			ID: "sales.order",
			Relations: []semantic.Relation{
				{ID: "sales.order.customer", Target: "customer.customer", Cardinality: semantic.CardinalityOne},
				{ID: "sales.order.lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
			},
		}},
		Capabilities: []semantic.Capability{{ID: "sales.order.submit"}},
	}}
	baseQueries := []query.EmbeddedSQL{{ID: "sample.query", Application: "sample", Datasource: "sales"}}
	baseBindings := []realization.CustomGoBinding{{Application: "sample", Capability: "sales.order.submit"}}

	tests := []struct {
		name      string
		world     ir.World
		pattern   application.Pattern
		wantError string
	}{
		{
			name:      "Queryが存在しない",
			world:     ir.World{Applications: []application.Application{baseApp}},
			pattern:   browse,
			wantError: "query sample.queryがCanonical Worldに見つかりません",
		},
		{
			name: "Queryが重複している",
			world: ir.World{
				Applications: []application.Application{baseApp},
				Queries:      []query.EmbeddedSQL{baseQueries[0], baseQueries[0]},
			},
			pattern:   browse,
			wantError: "一意に解決できません",
		},
		{
			name: "Queryが別Applicationに属する",
			world: ir.World{
				Applications: []application.Application{baseApp},
				Queries:      []query.EmbeddedSQL{{ID: "sample.query", Application: "other", Datasource: "sales"}},
			},
			pattern:   browse,
			wantError: "application otherに属しており",
		},
		{
			name: "PatternとQueryのDatasourceが矛盾する",
			world: ir.World{
				Applications: []application.Application{baseApp},
				Queries:      baseQueries,
			},
			pattern: application.Pattern{
				ID:         browse.ID,
				Kind:       browse.Kind,
				Query:      browse.Query,
				Datasource: "master",
			},
			wantError: "query sample.queryのdatasource salesと一致しません",
		},
		{
			name: "Datasourceが存在しない",
			world: ir.World{
				Applications: []application.Application{{ID: "sample"}},
				Queries:      baseQueries,
			},
			pattern:   browse,
			wantError: "datasource salesが見つかりません",
		},
		{
			name: "Datasourceが重複している",
			world: ir.World{
				Applications: []application.Application{{
					ID: "sample",
					Databases: []application.DatabaseRequirement{
						{Alias: "sales", TargetDatabase: "sales_core"},
						{Alias: "sales", TargetDatabase: "sales_shadow"},
					},
				}},
				Queries: baseQueries,
			},
			pattern:   browse,
			wantError: "一意に解決できません",
		},
		{
			name:      "Relationが存在しない",
			world:     ir.World{Applications: []application.Application{baseApp}},
			pattern:   reference,
			wantError: "semantic relation sales.order.customerがCanonical Worldに見つかりません",
		},
		{
			name: "Relationが重複している",
			world: ir.World{
				Applications: []application.Application{baseApp},
				Semantics: []semantic.Namespace{
					baseSemantics[0],
					{ID: "duplicate", Data: []semantic.Data{{ID: "other", Relations: []semantic.Relation{{ID: "sales.order.customer", Target: "customer.customer", Cardinality: semantic.CardinalityOne}}}}},
				},
			},
			pattern:   reference,
			wantError: "一意に解決できません",
		},
		{
			name:      "header-detailのRelationが存在しない",
			world:     ir.World{Applications: []application.Application{baseApp}},
			pattern:   headerDetail,
			wantError: "semantic relation sales.order.linesがCanonical Worldに見つかりません",
		},
		{
			name:      "Capabilityが存在しない",
			world:     ir.World{Applications: []application.Application{baseApp}},
			pattern:   command,
			wantError: "semantic capability sales.order.submitがCanonical Worldに見つかりません",
		},
		{
			name: "Capabilityが重複している",
			world: ir.World{
				Applications: []application.Application{baseApp},
				Semantics: []semantic.Namespace{
					baseSemantics[0],
					{ID: "duplicate", Capabilities: []semantic.Capability{{ID: "sales.order.submit"}}},
				},
				Bindings: baseBindings,
			},
			pattern:   command,
			wantError: "一意に解決できません",
		},
		{
			name: "Capability Bindingが存在しない",
			world: ir.World{
				Applications: []application.Application{baseApp},
				Semantics:    baseSemantics,
			},
			pattern:   command,
			wantError: "custom Go binding sample/sales.order.submit is not defined",
		},
		{
			name: "Capability Bindingが重複している",
			world: ir.World{
				Applications: []application.Application{baseApp},
				Semantics:    baseSemantics,
				Bindings:     []realization.CustomGoBinding{baseBindings[0], baseBindings[0]},
			},
			pattern:   command,
			wantError: "is ambiguous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := applicationdriver.ScenarioPlan{
				Application: tt.world.Applications[0],
				Patterns:    []application.Pattern{tt.pattern},
			}
			_, err := applicationdriver.ResolveScenarioResources(tt.world, plan)
			if err == nil {
				t.Fatal("errorを期待しました")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error=%q want substring=%q", err, tt.wantError)
			}
		})
	}
}

func TestResolveScenarioResourcesDoesNotInventDependenciesForRecord(t *testing.T) {
	pattern := application.Pattern{ID: "sample.record", Kind: application.PatternRecord, Data: core.CanonicalID("sales.order")}
	plan := applicationdriver.ScenarioPlan{
		Application: application.Application{ID: "sample"},
		Patterns:    []application.Pattern{pattern},
	}

	resolved, err := applicationdriver.ResolveScenarioResources(ir.World{}, plan)
	if err != nil {
		t.Fatalf("record Patternの解決に失敗しました: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved=%#v", resolved)
	}
	step := resolved[0]
	if step.Query != nil || step.Datasource != nil || step.Relation != nil || step.Capability != nil || step.Binding != nil {
		t.Fatalf("record Patternへ未定義の依存が追加されています: %#v", step)
	}
}
