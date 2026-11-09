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
