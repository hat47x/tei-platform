package syntax_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/syntax"
)

func TestParseHeaderDetailApplicationPatternUsesRelationOnly(t *testing.T) {
	src := `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("header-detail Patternのparseに失敗しました: %v", err)
	}
	if got.ID != "order_entry.order_lines" || got.Kind != application.PatternHeaderDetail {
		t.Fatalf("header-detail Patternが期待値と異なります: %#v", got)
	}
	if got.Relation != "sales.order.lines" {
		t.Fatalf("header-detail Relation=%q", got.Relation)
	}
	if got.Data != "" || got.Query != "" || got.Capability != "" || got.Datasource != "" {
		t.Fatalf("header-detail PatternへRelation以外の意味が入り込んでいます: %#v", got)
	}
}

func TestParseHeaderDetailApplicationPatternRequiresRelation(t *testing.T) {
	src := `pattern order_entry.order_lines header-detail {
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("Relationのないheader-detail Patternを受理しました")
	}
}

func TestParseHeaderDetailApplicationPatternRejectsOtherFields(t *testing.T) {
	cases := map[string]string{
		"data": `pattern order_entry.order_lines header-detail {
    data sales.order
    relation sales.order.lines
}`,
		"query": `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
    query order_entry.recent_orders
}`,
		"capability": `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
    capability sales.order.submit
}`,
		"datasource": `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
    datasource sales
}`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
				t.Fatalf("header-detail Patternが不要な%s fieldを受理しました", name)
			}
		})
	}
}
