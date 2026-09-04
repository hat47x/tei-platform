package applicationdriver_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationdriver"
	"tei/internal/core"
	"tei/internal/ir"
)

func TestAssemblePatternScenarioRejectsAmbiguousOrUnknownIdentity(t *testing.T) {
	validPattern := application.Pattern{
		ID:   "sample.browse",
		Kind: application.PatternBrowse,
		Data: "sales.order",
	}

	tests := []struct {
		name        string
		world       ir.World
		application core.CanonicalID
		patterns    []core.CanonicalID
		wantError   string
	}{
		{
			name:        "Applicationが存在しない",
			world:       ir.World{},
			application: "sample",
			patterns:    []core.CanonicalID{"sample.browse"},
			wantError:   "Canonical Worldに見つかりません",
		},
		{
			name: "Applicationが重複している",
			world: ir.World{Applications: []application.Application{
				{ID: "sample"},
				{ID: "sample"},
			}},
			application: "sample",
			patterns:    []core.CanonicalID{"sample.browse"},
			wantError:   "一意に解決できません",
		},
		{
			name: "Patternが指定されていない",
			world: ir.World{Applications: []application.Application{
				{ID: "sample", Patterns: []application.Pattern{validPattern}},
			}},
			application: "sample",
			wantError:   "Patternが指定されていません",
		},
		{
			name: "Patternが存在しない",
			world: ir.World{Applications: []application.Application{
				{ID: "sample", Patterns: []application.Pattern{validPattern}},
			}},
			application: "sample",
			patterns:    []core.CanonicalID{"sample.missing"},
			wantError:   "Pattern sample.missingが見つかりません",
		},
		{
			name: "別ApplicationのPatternを指定している",
			world: ir.World{Applications: []application.Application{
				{ID: "sample", Patterns: []application.Pattern{validPattern}},
				{ID: "other", Patterns: []application.Pattern{{ID: "other.record", Kind: application.PatternRecord}}},
			}},
			application: "sample",
			patterns:    []core.CanonicalID{"other.record"},
			wantError:   "application sampleにPattern other.recordが見つかりません",
		},
		{
			name: "Pattern定義が重複している",
			world: ir.World{Applications: []application.Application{
				{ID: "sample", Patterns: []application.Pattern{validPattern, validPattern}},
			}},
			application: "sample",
			patterns:    []core.CanonicalID{"sample.browse"},
			wantError:   "一意に解決できません",
		},
		{
			name: "Scenario入力でPatternを重複指定している",
			world: ir.World{Applications: []application.Application{
				{ID: "sample", Patterns: []application.Pattern{validPattern}},
			}},
			application: "sample",
			patterns:    []core.CanonicalID{"sample.browse", "sample.browse"},
			wantError:   "重複指定されています",
		},
		{
			name: "Pattern Kindが未対応",
			world: ir.World{Applications: []application.Application{
				{ID: "sample", Patterns: []application.Pattern{{ID: "sample.future", Kind: application.PatternKind("future")}}},
			}},
			application: "sample",
			patterns:    []core.CanonicalID{"sample.future"},
			wantError:   "未対応のkind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := applicationdriver.AssemblePatternScenario(tt.world, tt.application, tt.patterns)
			if err == nil {
				t.Fatal("errorを期待しました")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error=%q want substring=%q", err, tt.wantError)
			}
		})
	}
}

func TestAssemblePatternScenarioPreservesExplicitPatternOrder(t *testing.T) {
	world := ir.World{Applications: []application.Application{
		{
			ID: "sample",
			Patterns: []application.Pattern{
				{ID: "sample.browse", Kind: application.PatternBrowse},
				{ID: "sample.record", Kind: application.PatternRecord},
			},
		},
	}}

	plan, err := applicationdriver.AssemblePatternScenario(
		world,
		"sample",
		[]core.CanonicalID{"sample.record", "sample.browse"},
	)
	if err != nil {
		t.Fatalf("Scenario assemblyに失敗しました: %v", err)
	}
	if len(plan.Patterns) != 2 {
		t.Fatalf("patterns=%#v", plan.Patterns)
	}
	if plan.Patterns[0].ID != "sample.record" || plan.Patterns[1].ID != "sample.browse" {
		t.Fatalf("明示したScenario順序が保持されていません: %#v", plan.Patterns)
	}
}
