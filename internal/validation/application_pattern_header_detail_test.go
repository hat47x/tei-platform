package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateApplicationPatternsAcceptsRelationOnlyHeaderDetail(t *testing.T) {
	w := headerDetailPatternWorld()

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("正常なheader-detail PatternにDiagnosticが返りました: %#v", ds)
	}
}

func TestValidateApplicationPatternsRejectsUnknownHeaderDetailRelation(t *testing.T) {
	w := headerDetailPatternWorld()
	w.Applications[0].Patterns[0].Relation = "sales.order.missing"

	ds := validateApplicationPatterns(w)
	d := headerDetailDiagnosticByCode(ds, "UNKNOWN_APPLICATION_PATTERN_RELATION")
	if d == nil {
		t.Fatalf("未知RelationのDiagnosticがありません: %#v", ds)
	}
	assertHeaderDetailSubject(t, d)
}

func TestValidateApplicationPatternsRejectsOneRelationForHeaderDetail(t *testing.T) {
	w := headerDetailPatternWorld()
	w.Applications[0].Patterns[0].Relation = "sales.order.customer"

	ds := validateApplicationPatterns(w)
	d := headerDetailDiagnosticByCode(ds, "APPLICATION_PATTERN_RELATION_NOT_MANY")
	if d == nil {
		t.Fatalf("one Relationをheader-detailとして受理しました: %#v", ds)
	}
	assertHeaderDetailSubject(t, d)
}

func TestValidateApplicationPatternsRejectsDuplicatedHeaderDetailMeaning(t *testing.T) {
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
			w := headerDetailPatternWorld()
			tc.edit(&w.Applications[0].Patterns[0])

			ds := validateApplicationPatterns(w)
			d := headerDetailDiagnosticByCode(ds, tc.code)
			if d == nil {
				t.Fatalf("header-detailの不要な%s fieldを受理しました: %#v", tc.name, ds)
			}
			assertHeaderDetailSubject(t, d)
		})
	}
}

func headerDetailPatternWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{
				{
					ID: "sales.order",
					Relations: []semantic.Relation{
						{ID: "sales.order.customer", Target: "customer.customer", Cardinality: semantic.CardinalityOne},
						{ID: "sales.order.lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
					},
				},
				{ID: "sales.order_line"},
			},
		}},
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:       "order_entry.order_lines",
				Kind:     application.PatternHeaderDetail,
				Relation: "sales.order.lines",
			}},
		}},
	}
}

func headerDetailDiagnosticByCode(ds []Diagnostic, code string) *Diagnostic {
	for i := range ds {
		if ds[i].Code == code {
			return &ds[i]
		}
	}
	return nil
}

func assertHeaderDetailSubject(t *testing.T, d *Diagnostic) {
	t.Helper()
	if d.Subject == nil || d.Subject.Kind != "application-pattern" || d.Subject.ID != "order_entry.order_lines" {
		t.Fatalf("header-detail DiagnosticのSubjectが期待値と異なります: %#v", d.Subject)
	}
}
