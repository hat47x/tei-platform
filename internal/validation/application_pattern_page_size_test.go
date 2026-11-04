package validation

import (
	"testing"

	"tei/internal/application"
)

func TestValidateBrowsePageSize(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].PageSize = 20
	for _, diagnostic := range validateApplicationPatterns(w) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateBrowsePageSizeRejectsOutOfRange(t *testing.T) {
	for _, size := range []int{1001, -1} {
		w := filterWorld()
		w.Applications[0].Patterns[0].PageSize = size
		if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_PAGE_SIZE_INVALID") == nil {
			t.Fatalf("page-size %d was accepted", size)
		}
	}
}

func TestValidateBrowsePageSizeRejectsOnNonBrowse(t *testing.T) {
	w := filterWorld()
	w.Applications[0].Patterns[0].Kind = application.PatternRecord
	w.Applications[0].Patterns[0].PageSize = 20
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_PAGE_SIZE_NOT_ALLOWED") == nil {
		t.Fatal("page-size on a record pattern was accepted")
	}
}
