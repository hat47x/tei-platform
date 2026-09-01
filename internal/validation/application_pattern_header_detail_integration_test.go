package validation

import "testing"

func TestValidateAcceptsRelationOnlyHeaderDetailPattern(t *testing.T) {
	diagnostics := Validate(headerDetailPatternWorld())
	for _, diagnostic := range diagnostics {
		if diagnostic.Subject != nil && diagnostic.Subject.Kind == "application-pattern" && diagnostic.Subject.ID == "order_entry.order_lines" {
			t.Fatalf("正常なheader-detail Patternに不要なDiagnosticが返りました: %#v", diagnostic)
		}
	}
}

func TestValidateReportsHeaderDetailCardinalityMismatchOnce(t *testing.T) {
	w := headerDetailPatternWorld()
	w.Applications[0].Patterns[0].Relation = "sales.order.customer"

	diagnostics := Validate(w)
	var matches []Diagnostic
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "APPLICATION_PATTERN_RELATION_NOT_MANY" {
			matches = append(matches, diagnostic)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("header-detail cardinality Diagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
	assertHeaderDetailSubject(t, &matches[0])
}
