package validation

import (
	"testing"

	"tei/internal/application"
)

func TestValidateHeaderDetailPageBreak(t *testing.T) {
	w := headerDetailPatternWorld()
	w.Applications[0].Patterns[0].PageBreak = &application.PageBreak{Rows: 20, RepeatHeader: true}
	for _, diagnostic := range validateApplicationPatterns(w) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateHeaderDetailPageBreakRejectsOutOfRange(t *testing.T) {
	for _, rows := range []int{0, 1001} {
		w := headerDetailPatternWorld()
		w.Applications[0].Patterns[0].PageBreak = &application.PageBreak{Rows: rows}
		if headerDetailDiagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_PAGE_BREAK_INVALID") == nil {
			t.Fatalf("page-break rows %d was accepted", rows)
		}
	}
}

func TestValidateHeaderDetailPageBreakRejectsOnNonHeaderDetail(t *testing.T) {
	w := headerDetailPatternWorld()
	w.Applications[0].Patterns[0].Kind = application.PatternRecord
	w.Applications[0].Patterns[0].PageBreak = &application.PageBreak{Rows: 20}
	if headerDetailDiagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_PAGE_BREAK_NOT_ALLOWED") == nil {
		t.Fatal("page-break on a record pattern was accepted")
	}
}
