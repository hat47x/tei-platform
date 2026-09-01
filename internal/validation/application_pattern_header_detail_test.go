package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateApplicationPatternsAcceptsRelationOnlyHeaderDetail(t *testing.T) {
	w := validHeaderDetailPatternWorld()

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("正常なheader-detail PatternにDiagnosticが返されました: %#v", ds)
	}
}

func TestValidateApplicationPatternsRequiresManyRelationForHeaderDetail(t *testing.T) {
	w := validHeaderDetailPatternWorld()
	w.Applications[0].Patterns[0].Relation = "sales.order.customer"

	ds := validateApplicationPatterns(w)
	d := diagnosticByCode(ds, "APPLICATION_PATTERN_RELATION_NOT_MANY")
	if d == nil {
		t.Fatalf("one Relationをheader-detailとして受理しました: %#v", ds)
	}
	if d.Subject == nil || d.Subject.Kind != "application-pattern" || d.Subject.ID != "order_entry.order_lines" {
		t.Fatalf("header-detail DiagnosticのPattern Subjectが失われました: %#v", d.Subject)
	}
}

func TestValidateApplicationPatternsReportsUnknownHeaderDetailRelation(t *testing.T) {
	w := validHeaderDetailPatternWorld()
	w.Applications[0].Patterns[0].Relation = "sales.order.missing"

	ds := validateApplicationPatterns(w)
	if d := diagnosticByCode(ds, "UNKNOWN_APPLICATION_PATTERN_RELATION"); d == nil {
		t.Fatalf("未知Relationをheader-detailとして受理しました: %#v", ds)
	}
}

func TestValidateApplicationPatternsRejectsUnrelatedHeaderDetailFields(t *testing.T) {
	cases := []struct {
		name string
		code string
		edit func(*application.Pattern)
	}{
		{name: "data", code: "APPLICATION_PATTERN_DATA_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Data = "sales.order" }},
		{name: "query", code: "APPLICATION_PATTERN_QUERY_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Query = "order_entry.recent_orders" }},
		{name: "capability", code: "APPLICATION_PATTERN_CAPABILITY_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Capability = "sales.order.submit" }},
		{name: "datasource", code: "APPLICATION_PATTERN_DATASOURCE_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Datasource = "sales" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := validHeaderDetailPatternWorld()
			tc.edit(&w.Applications[0].Patterns[0])
			if d := diagnosticByCode(validateApplicationPatterns(w), tc.code); d == nil {
				t.Fatalf("header-detail Patternに不要な%sが受理されました", tc.name)
			}
		})
	}
}

func TestValidateIncludesHeaderDetailCardinalityDiagnosticOnce(t *testing.T) {
	w := validHeaderDetailPatternWorld()
	w.Applications[0].Patterns[0].Relation = "sales.order.customer"

	diagnostics := Validate(w)
	matches := diagnosticsByCode(diagnostics, "APPLICATION_PATTERN_RELATION_NOT_MANY")
	if len(matches) != 1 {
		t.Fatalf("header-detail cardinality Diagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
	if matches[0].Subject == nil || matches[0].Subject.Kind != "application-pattern" || matches[0].Subject.ID != "order_entry.order_lines" {
		t.Fatalf("共通Validateでheader-detailのPattern Subjectが失われました: %#v", matches[0].Subject)
	}
}

func validHeaderDetailPatternWorld() ir.World {
	w := validApplicationPatternWorld()

	w.Semantics[0].Data[0].Relations = append(w.Semantics[0].Data[0].Relations, semantic.Relation{
		ID:          "sales.order.lines",
		Symbol:      "lines",
		Target:      "sales.order_line",
		Cardinality: semantic.CardinalityMany,
	})
	w.Semantics[0].Data = append(w.Semantics[0].Data, semantic.Data{
		ID:     "sales.order_line",
		Symbol: "OrderLine",
	})
	w.Applications[0].Patterns = []application.Pattern{{
		ID:       "order_entry.order_lines",
		Kind:     application.PatternHeaderDetail,
		Relation: "sales.order.lines",
	}}

	return w
}
