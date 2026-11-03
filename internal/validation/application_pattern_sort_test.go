package validation

import (
	"testing"

	"tei/internal/application"
)

func TestValidateBrowseSorts(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Sorts = []application.Sort{{Attribute: "sales.order.status", Direction: "asc"}}
	for _, diagnostic := range validateApplicationPatterns(w) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateBrowseSortsRejectUnknownAttribute(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Sorts = []application.Sort{{Attribute: "sales.order.missing", Direction: "asc"}}
	if diagnosticByCode(validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_SORT_ATTRIBUTE") == nil {
		t.Fatal("unknown sort attribute was accepted")
	}
}

func TestValidateBrowseSortsRejectUnsupportedDirection(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Sorts = []application.Sort{{Attribute: "sales.order.status", Direction: "up"}}
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_SORT_DIRECTION_UNSUPPORTED") == nil {
		t.Fatal("unsupported sort direction was accepted")
	}
}

func TestValidateBrowseSortsRejectDuplicate(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Sorts = []application.Sort{
		{Attribute: "sales.order.status", Direction: "asc"},
		{Attribute: "sales.order.status", Direction: "desc"},
	}
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_SORT_DUPLICATE") == nil {
		t.Fatal("duplicate sort was accepted")
	}
}

func TestValidateBrowseSortsRejectOnNonBrowse(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Kind = application.PatternRecord
	w.Applications[0].Patterns[0].Sorts = []application.Sort{{Attribute: "sales.order.status", Direction: "asc"}}
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_SORTS_NOT_ALLOWED") == nil {
		t.Fatal("sorts on a record pattern were accepted")
	}
}
