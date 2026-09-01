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

func TestValidateApplicationPatternsReportsReferenceProblemsOnPatternSubject(t *testing.T) {
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
			edit: func(w *ir.World) {
				w.Applications[0].Databases = append(w.Applications[0].Databases, application.DatabaseRequirement{Alias: "master"})
				w.Applications[0].Patterns[0].Datasource = "master"
			},
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
		Semantics: []semantic.Namespace{{
			ID:   "sales",
			Data: []semantic.Data{{ID: "sales.order"}},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Databases: []application.DatabaseRequirement{{
				Alias:  "sales",
				Access: database.AccessReadWrite,
			}},
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
