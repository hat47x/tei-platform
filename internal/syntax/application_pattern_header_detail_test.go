package syntax_test

import (
	"strings"
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
		t.Fatalf("header-detail PatternのParseに失敗しました: %v", err)
	}
	if got.ID != "order_entry.order_lines" || got.Kind != application.PatternHeaderDetail {
		t.Fatalf("header-detail Patternのidentityが期待値と異なります: %#v", got)
	}
	if got.Relation != "sales.order.lines" {
		t.Fatalf("header-detail relation=%q", got.Relation)
	}
	if got.Data != "" || got.Query != "" || got.Capability != "" || got.Datasource != "" {
		t.Fatalf("Relationが持つ意味や実装情報がheader-detail Patternへ重複しています: %#v", got)
	}
}

func TestParseHeaderDetailApplicationPatternRequiresRelation(t *testing.T) {
	src := `pattern order_entry.order_lines header-detail {
}`

	_, err := syntax.NewParser(src).ParseApplicationPattern()
	if err == nil || !strings.Contains(err.Error(), "requires relation") {
		t.Fatalf("Relationなしのheader-detail Patternが受理されました: %v", err)
	}
}

func TestParseHeaderDetailApplicationPatternRejectsUnrelatedFields(t *testing.T) {
	cases := map[string]string{
		"data": `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
    data sales.order
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
				t.Fatalf("header-detail Patternに不要な%sが受理されました", name)
			}
		})
	}
}
