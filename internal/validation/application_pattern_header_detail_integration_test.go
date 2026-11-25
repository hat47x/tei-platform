package validation

import (
	"strings"
	"testing"

	"tei/internal/realization"
)

// A relation-only header-detail Pattern is a valid declaration: naming a
// Relation is all it has to do to state the shape.
//
// This asks the Pattern validator alone. Whether an Application can read the
// Pattern is a different question, because a Pattern states structure and a
// binding realizes it: a Pattern with no binding is reported by
// TestValidateRefusesAHeaderDetailPatternWithoutABinding, which is the check
// that keeps validate and a read from disagreeing.
func TestValidateAcceptsRelationOnlyHeaderDetailPattern(t *testing.T) {
	diagnostics := validateApplicationPatterns(headerDetailPatternWorld())
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

// A header-detail Pattern an Application declares but no binding realizes can
// never be read: the read refuses it with "the Contract Query binding is
// missing". Validate reports it as a warning, so the missing binding is not
// silent while the two products that declare a Pattern before its realization
// keep their acceptance.
func TestValidateRefusesAHeaderDetailPatternWithoutABinding(t *testing.T) {
	world := headerDetailPatternWorld()

	var found bool
	for _, diagnostic := range Validate(world) {
		if diagnostic.Code != "HEADER_DETAIL_PATTERN_BINDING_REQUIRED" {
			continue
		}
		found = true
		if diagnostic.Subject == nil || diagnostic.Subject.Kind != "application-pattern" ||
			diagnostic.Subject.ID != "order_entry.order_lines" {
			t.Fatalf("the finding does not name the Pattern: %#v", diagnostic.Subject)
		}
		if !strings.Contains(diagnostic.Message, "order_entry") {
			t.Fatalf("the finding does not name the Application: %q", diagnostic.Message)
		}
	}
	if !found {
		t.Fatal("a header-detail Pattern with no binding was reported quietly")
	}
}

// A binding that realizes the Pattern removes the finding, so the check is
// about the binding and not about the Pattern's shape.
func TestValidateAcceptsAHeaderDetailPatternWithABinding(t *testing.T) {
	world := headerDetailPatternWorld()
	world.HeaderDetailContractQueryBindings = []realization.HeaderDetailContractQueryBinding{{
		Application: "order_entry",
		Pattern:     "order_entry.order_lines",
		Datasource:  "sales",
		Target: realization.ContractQueryTarget{
			Database: "sales_core", Contract: "sales_core.order_line", Query: "find_by_order",
		},
		Inputs: []realization.HeaderDetailContractQueryInputBinding{
			{SourceAttribute: "sales.order.order_id", Input: "order_id"},
		},
	}}

	for _, diagnostic := range Validate(world) {
		if diagnostic.Code == "HEADER_DETAIL_PATTERN_BINDING_REQUIRED" {
			t.Fatalf("a Pattern with a binding was refused: %s", diagnostic.Message)
		}
	}
}
