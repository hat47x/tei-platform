package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/query"
)

func filterParameterWorld() ir.World {
	w := filterWorld()
	w.Queries[0].Inputs = []query.Input{{Name: "status", Type: "std.code"}}
	w.Applications[0].Patterns[0].Filters[0].Parameter = "status"
	return w
}

func TestValidateBrowseFilterParameter(t *testing.T) {
	for _, diagnostic := range validateApplicationPatterns(filterParameterWorld()) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateBrowseFilterParameterRejectsUnknownInput(t *testing.T) {
	w := filterParameterWorld()
	w.Applications[0].Patterns[0].Filters[0].Parameter = "missing"
	if diagnosticByCode(validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_FILTER_PARAMETER") == nil {
		t.Fatal("unknown filter parameter was accepted")
	}
}

func TestValidateBrowseFilterParameterRejectsTypeMismatch(t *testing.T) {
	w := filterParameterWorld()
	w.Queries[0].Inputs[0].Type = "std.text"
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_FILTER_PARAMETER_TYPE_MISMATCH") == nil {
		t.Fatal("filter parameter type mismatch was accepted")
	}
}

func queryParameterWorld() ir.World {
	w := filterParameterWorld()
	w.Queries[0].Inputs = append(w.Queries[0].Inputs,
		query.Input{Name: "status_operator", Type: "std.code"},
		query.Input{Name: "page_size", Type: "std.integer"},
		query.Input{Name: "page_offset", Type: "std.integer"})
	p := &w.Applications[0].Patterns[0]
	p.Filters[0].OperatorParameter = "status_operator"
	p.PageSize = 20
	p.PageSizeParameter = "page_size"
	p.PageOffsetParameter = "page_offset"
	return w
}

func TestValidateBrowseQueryParameters(t *testing.T) {
	for _, diagnostic := range validateApplicationPatterns(queryParameterWorld()) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateBrowseQueryParametersRejects(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   string
		mutate func(*ir.World)
	}{
		{"unknown operator input", "UNKNOWN_APPLICATION_PATTERN_FILTER_OPERATOR_PARAMETER", func(w *ir.World) { w.Applications[0].Patterns[0].Filters[0].OperatorParameter = "nope" }},
		{"operator input type", "APPLICATION_PATTERN_FILTER_OPERATOR_PARAMETER_TYPE_MISMATCH", func(w *ir.World) { w.Queries[0].Inputs[1].Type = "std.text" }},
		{"unknown page input", "UNKNOWN_APPLICATION_PATTERN_PAGE_PARAMETER", func(w *ir.World) { w.Applications[0].Patterns[0].PageOffsetParameter = "nope" }},
		{"page input type", "APPLICATION_PATTERN_PAGE_PARAMETER_TYPE_MISMATCH", func(w *ir.World) { w.Queries[0].Inputs[2].Type = "std.text" }},
		{"page parameter without page-size", "APPLICATION_PATTERN_PAGE_PARAMETER_REQUIRES_PAGE_SIZE", func(w *ir.World) { w.Applications[0].Patterns[0].PageSize = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := queryParameterWorld()
			tc.mutate(&w)
			if diagnosticByCode(validateApplicationPatterns(w), tc.code) == nil {
				t.Fatalf("%s was not reported", tc.code)
			}
		})
	}
}
