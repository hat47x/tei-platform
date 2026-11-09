package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func filterWorld() ir.World {
	w := validApplicationPatternWorld()
	w.Semantics[0].Data[0].Attributes = []semantic.Attribute{
		{ID: "sales.order.status", TypeRef: "std.code"},
	}
	w.Applications[0].Patterns[0].Filters = []application.Filter{{
		Attribute: "sales.order.status",
		Operators: []string{"eq", "ne"},
	}}
	return w
}

func TestValidateBrowseFilters(t *testing.T) {
	for _, diagnostic := range validateApplicationPatterns(filterWorld()) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateBrowseFiltersRejectUnknownAttribute(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Filters[0].Attribute = "sales.order.missing"
	if diagnosticByCode(validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_FILTER_ATTRIBUTE") == nil {
		t.Fatal("unknown filter attribute was accepted")
	}
}

func TestValidateBrowseFiltersRejectUnsupportedOperator(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Filters[0].Operators = []string{"like"}
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_FILTER_OPERATOR_UNSUPPORTED") == nil {
		t.Fatal("unsupported filter operator was accepted")
	}
}

func TestValidateBrowseFiltersRejectDuplicate(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Filters = append(w.Applications[0].Patterns[0].Filters, w.Applications[0].Patterns[0].Filters[0])
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_FILTER_DUPLICATE") == nil {
		t.Fatal("duplicate filter was accepted")
	}
}

func TestValidateBrowseFiltersRejectOnNonBrowse(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Kind = application.PatternRecord
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_FILTERS_NOT_ALLOWED") == nil {
		t.Fatal("filters on a record pattern were accepted")
	}
}
