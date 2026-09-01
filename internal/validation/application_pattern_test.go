package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

func TestValidateApplicationPatternsAcceptsBrowseUsingQueryDatasource(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns[0].Datasource = ""

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("valid browse pattern diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsAcceptsDataOnlyRecord(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns = append(w.Applications[0].Patterns, application.Pattern{
		ID:   "order_entry.order_record",
		Kind: application.PatternRecord,
		Data: "sales.order",
	})

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("valid record pattern diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsAcceptsReferenceUsingRelationAndDatasource(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns = []application.Pattern{{
		ID:         "order_entry.customer_reference",
		Kind:       application.PatternReference,
		Relation:   "sales.order.customer",
		Datasource: "master",
	}}

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("valid reference pattern diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsBrowseProblemsOnPatternSubject(t *testing.T) {
	cases := []struct {
		name string
		code string
		edit func(*ir.World)
	}{
		{
			name: "unknown data",
			code: "UNKNOWN_APPLICATION_PATTERN_DATA",
			edit: func(w *ir.World) { w.Applications[0].Patterns[0].Data = "sales.missing" },
		},
		{
			name: "relation is not part of browse meaning",
			code: "APPLICATION_PATTERN_RELATION_NOT_ALLOWED",
			edit: func(w *ir.World) { w.Applications[0].Patterns[0].Relation = "sales.order.customer" },
		},
		{
			name: "unknown query",
			code: "UNKNOWN_APPLICATION_PATTERN_QUERY",
			edit: func(w *ir.World) { w.Applications[0].Patterns[0].Query = "order_entry.missing" },
		},
		{
			name: "query owned by another application",
			code: "APPLICATION_PATTERN_QUERY_APPLICATION_MISMATCH",
			edit: func(w *ir.World) {
				w.Applications = append(w.Applications, application.Application{ID: "other_app"})
				w.Queries[0].Application = "other_app"
			},
		},
		{
			name: "unknown explicit datasource",
			code: "UNKNOWN_APPLICATION_PATTERN_DATASOURCE",
			edit: func(w *ir.World) { w.Applications[0].Patterns[0].Datasource = "missing" },
		},
		{
			name: "explicit datasource conflicts with query",
			code: "APPLICATION_PATTERN_DATASOURCE_QUERY_MISMATCH",
			edit: func(w *ir.World) { w.Applications[0].Patterns[0].Datasource = "master" },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := validApplicationPatternWorld()
			tc.edit(&w)
			ds := validateApplicationPatterns(w)
			d := diagnosticByCode(ds, tc.code)
			if d == nil {
				t.Fatalf("missing diagnostic %s: %#v", tc.code, ds)
			}
			if d.Subject == nil || d.Subject.Kind != "application-pattern" || d.Subject.ID != "order_entry.recent_orders" {
				t.Fatalf("diagnostic subject=%#v", d.Subject)
			}
		})
	}
}

func TestValidateApplicationPatternsReportsRecordProblemsOnPatternSubject(t *testing.T) {
	cases := []struct {
		name string
		code string
		edit func(*application.Pattern)
	}{
		{name: "missing data", code: "APPLICATION_PATTERN_DATA_REQUIRED", edit: func(p *application.Pattern) { p.Data = "" }},
		{name: "unknown data", code: "UNKNOWN_APPLICATION_PATTERN_DATA", edit: func(p *application.Pattern) { p.Data = "sales.missing" }},
		{name: "relation is not part of record meaning", code: "APPLICATION_PATTERN_RELATION_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Relation = "sales.order.customer" }},
		{name: "query is not part of record meaning", code: "APPLICATION_PATTERN_QUERY_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Query = "order_entry.recent_orders" }},
		{name: "datasource is not part of record meaning", code: "APPLICATION_PATTERN_DATASOURCE_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Datasource = "sales" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := validApplicationPatternWorld()
			record := application.Pattern{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
			}
			tc.edit(&record)
			w.Applications[0].Patterns = []application.Pattern{record}

			ds := validateApplicationPatterns(w)
			d := diagnosticByCode(ds, tc.code)
			if d == nil {
				t.Fatalf("missing diagnostic %s: %#v", tc.code, ds)
			}
			if d.Subject == nil || d.Subject.Kind != "application-pattern" || d.Subject.ID != "order_entry.order_record" {
				t.Fatalf("diagnostic subject=%#v", d.Subject)
			}
		})
	}
}

func TestValidateApplicationPatternsReportsReferencePatternProblemsOnPatternSubject(t *testing.T) {
	cases := []struct {
		name string
		code string
		edit func(*application.Pattern)
	}{
		{name: "missing relation", code: "APPLICATION_PATTERN_RELATION_REQUIRED", edit: func(p *application.Pattern) { p.Relation = "" }},
		{name: "unknown relation", code: "UNKNOWN_APPLICATION_PATTERN_RELATION", edit: func(p *application.Pattern) { p.Relation = "sales.order.missing" }},
		{name: "missing datasource", code: "APPLICATION_PATTERN_DATASOURCE_REQUIRED", edit: func(p *application.Pattern) { p.Datasource = "" }},
		{name: "unknown datasource", code: "UNKNOWN_APPLICATION_PATTERN_DATASOURCE", edit: func(p *application.Pattern) { p.Datasource = "missing" }},
		{name: "data duplicates relation meaning", code: "APPLICATION_PATTERN_DATA_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Data = "sales.order" }},
		{name: "query is not part of reference meaning", code: "APPLICATION_PATTERN_QUERY_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Query = "order_entry.recent_orders" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := validApplicationPatternWorld()
			reference := application.Pattern{
				ID:         "order_entry.customer_reference",
				Kind:       application.PatternReference,
				Relation:   "sales.order.customer",
				Datasource: "master",
			}
			tc.edit(&reference)
			w.Applications[0].Patterns = []application.Pattern{reference}

			ds := validateApplicationPatterns(w)
			d := diagnosticByCode(ds, tc.code)
			if d == nil {
				t.Fatalf("missing diagnostic %s: %#v", tc.code, ds)
			}
			if d.Subject == nil || d.Subject.Kind != "application-pattern" || d.Subject.ID != "order_entry.customer_reference" {
				t.Fatalf("diagnostic subject=%#v", d.Subject)
			}
		})
	}
}

func TestApplicationPatternRelationIndexPreservesSourceAndTarget(t *testing.T) {
	w := validApplicationPatternWorld()
	var got applicationPatternRelationRef
	found := false
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, relation := range data.Relations {
				if relation.ID == "sales.order.customer" {
					got = applicationPatternRelationRef{Source: data.ID, Target: relation.Target}
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("sales.order.customer relation is missing from fixture")
	}
	if got.Source != "sales.order" || got.Target != "customer.customer" {
		t.Fatalf("relation source/target=%#v", got)
	}
}

func TestValidateApplicationPatternsRejectsDuplicateCanonicalIdentity(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications = append(w.Applications, application.Application{
		ID: "reporting",
		Patterns: []application.Pattern{{
			ID:    "order_entry.recent_orders",
			Kind:  application.PatternBrowse,
			Data:  "sales.order",
			Query: "order_entry.recent_orders",
		}},
	})

	ds := validateApplicationPatterns(w)
	if d := diagnosticByCode(ds, "DUPLICATE_APPLICATION_PATTERN"); d == nil {
		t.Fatalf("duplicate Pattern Canonical ID was accepted: %#v", ds)
	}
}

func TestValidateApplicationPatternsRechecksParserRequiredFieldsForCanonicalWorld(t *testing.T) {
	for _, tc := range []struct {
		name string
		code string
		edit func(*application.Pattern)
	}{
		{name: "data", code: "APPLICATION_PATTERN_DATA_REQUIRED", edit: func(p *application.Pattern) { p.Data = "" }},
		{name: "query", code: "APPLICATION_PATTERN_QUERY_REQUIRED", edit: func(p *application.Pattern) { p.Query = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := validApplicationPatternWorld()
			tc.edit(&w.Applications[0].Patterns[0])
			if d := diagnosticByCode(validateApplicationPatterns(w), tc.code); d == nil {
				t.Fatalf("missing diagnostic %s", tc.code)
			}
		})
	}
}

func validApplicationPatternWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{{
					ID: "sales.order",
					Relations: []semantic.Relation{{
						ID:          "sales.order.customer",
						Target:      "customer.customer",
						Cardinality: semantic.CardinalityOne,
					}},
				}},
			},
			{
				ID:   "customer",
				Data: []semantic.Data{{ID: "customer.customer"}},
			},
		},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales", "customer"},
			Databases: []application.DatabaseRequirement{
				{Alias: "sales", Access: database.AccessReadWrite},
				{Alias: "master", Access: database.AccessRead},
			},
			Patterns: []application.Pattern{{
				ID:         "order_entry.recent_orders",
				Kind:       application.PatternBrowse,
				Data:       "sales.order",
				Query:      "order_entry.recent_orders",
				Datasource: "sales",
			}},
		}},
		Queries: []query.EmbeddedSQL{{
			ID:          "order_entry.recent_orders",
			Application: "order_entry",
			Datasource:  "sales",
		}},
	}
}

func diagnosticByCode(ds []Diagnostic, code string) *Diagnostic {
	for i := range ds {
		if ds[i].Code == code {
			return &ds[i]
		}
	}
	return nil
}
