package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateApplicationPatternsAcceptsCapabilityOnlyCommand(t *testing.T) {
	w := validCommandPatternWorld()

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("valid command pattern diagnostics=%#v", ds)
	}
}

func TestValidateApplicationPatternsReportsCommandProblemsOnPatternSubject(t *testing.T) {
	cases := []struct {
		name string
		code string
		edit func(*application.Pattern)
	}{
		{name: "missing capability", code: "APPLICATION_PATTERN_CAPABILITY_REQUIRED", edit: func(p *application.Pattern) { p.Capability = "" }},
		{name: "unknown capability", code: "UNKNOWN_APPLICATION_PATTERN_CAPABILITY", edit: func(p *application.Pattern) { p.Capability = "sales.order.missing" }},
		{name: "data duplicates capability contract", code: "APPLICATION_PATTERN_DATA_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Data = "sales.order" }},
		{name: "relation is not part of command meaning", code: "APPLICATION_PATTERN_RELATION_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Relation = "sales.order.customer" }},
		{name: "query is not part of command meaning", code: "APPLICATION_PATTERN_QUERY_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Query = "order_entry.recent_orders" }},
		{name: "datasource belongs to implementation boundary", code: "APPLICATION_PATTERN_DATASOURCE_NOT_ALLOWED", edit: func(p *application.Pattern) { p.Datasource = "sales" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := validCommandPatternWorld()
			tc.edit(&w.Applications[0].Patterns[0])

			ds := validateApplicationPatterns(w)
			d := diagnosticByCode(ds, tc.code)
			if d == nil {
				t.Fatalf("missing diagnostic %s: %#v", tc.code, ds)
			}
			if d.Subject == nil || d.Subject.Kind != "application-pattern" || d.Subject.ID != "order_entry.submit_order" {
				t.Fatalf("command diagnostic subject=%#v", d.Subject)
			}
		})
	}
}

func TestValidateApplicationPatternsRejectsCapabilityOnOtherKinds(t *testing.T) {
	cases := []application.Pattern{
		{
			ID:         "order_entry.recent_orders",
			Kind:       application.PatternBrowse,
			Data:       "sales.order",
			Query:      "order_entry.recent_orders",
			Capability: "sales.order.submit",
		},
		{
			ID:         "order_entry.order_record",
			Kind:       application.PatternRecord,
			Data:       "sales.order",
			Capability: "sales.order.submit",
		},
		{
			ID:         "order_entry.customer_reference",
			Kind:       application.PatternReference,
			Relation:   "sales.order.customer",
			Datasource: "master",
			Capability: "sales.order.submit",
		},
	}

	for _, pattern := range cases {
		t.Run(string(pattern.Kind), func(t *testing.T) {
			w := validApplicationPatternWorld()
			w.Applications[0].Patterns = []application.Pattern{pattern}
			if d := diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_CAPABILITY_NOT_ALLOWED"); d == nil {
				t.Fatalf("%s Pattern accepted Capability unexpectedly", pattern.Kind)
			}
		})
	}
}

func TestValidateIncludesCommandCapabilityDiagnosticOnce(t *testing.T) {
	w := validCommandPatternWorld()
	w.Applications[0].Patterns[0].Capability = "sales.order.missing"

	diagnostics := Validate(w)
	matches := diagnosticsByCode(diagnostics, "UNKNOWN_APPLICATION_PATTERN_CAPABILITY")
	if len(matches) != 1 {
		t.Fatalf("command Capability Diagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
	if matches[0].Subject == nil || matches[0].Subject.Kind != "application-pattern" || matches[0].Subject.ID != "order_entry.submit_order" {
		t.Fatalf("command DiagnosticのPattern Subjectが失われました: %#v", matches[0].Subject)
	}
}

func TestCommandPatternDoesNotRedefineCapabilityInputOrOutput(t *testing.T) {
	w := validCommandPatternWorld()
	capability := w.Semantics[0].Capabilities[0]
	pattern := w.Applications[0].Patterns[0]

	if len(capability.Inputs) != 1 || capability.Inputs[0].Name != "order" || capability.Inputs[0].Type != "sales.order" {
		t.Fatalf("Capability input contract=%#v", capability.Inputs)
	}
	if capability.Output != "sales.order" {
		t.Fatalf("Capability output=%q", capability.Output)
	}
	if pattern.Data != "" || pattern.Relation != "" || pattern.Query != "" || pattern.Datasource != "" {
		t.Fatalf("command PatternがCapability契約を重複保持しています: %#v", pattern)
	}
}

func validCommandPatternWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
			}},
			Capabilities: []semantic.Capability{{
				ID: "sales.order.submit",
				Inputs: []semantic.Parameter{{
					Name: "order",
					Type: "sales.order",
				}},
				Output: "sales.order",
			}},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Patterns: []application.Pattern{{
				ID:         "order_entry.submit_order",
				Kind:       application.PatternCommand,
				Capability: "sales.order.submit",
			}},
		}},
	}
}
