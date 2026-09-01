package validation

import (
	"testing"

	"tei/internal/application"
)

func TestValidateIncludesApplicationPatternReferenceDiagnosticsOnce(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns[0].Data = "sales.missing"

	diagnostics := Validate(w)
	matches := diagnosticsByCode(diagnostics, "UNKNOWN_APPLICATION_PATTERN_DATA")
	if len(matches) != 1 {
		t.Fatalf("共通Validateに期待するPattern Diagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
	if matches[0].Subject == nil || matches[0].Subject.Kind != "application-pattern" || matches[0].Subject.ID != "order_entry.recent_orders" {
		t.Fatalf("共通ValidateでPattern Subjectが失われました: %#v", matches[0].Subject)
	}
}

func TestValidateIncludesDuplicateApplicationPatternDiagnosticOnce(t *testing.T) {
	w := validApplicationPatternWorld()
	duplicate := w.Applications[0]
	duplicate.ID = "reporting"
	w.Applications = append(w.Applications, duplicate)

	diagnostics := Validate(w)
	matches := diagnosticsByCode(diagnostics, "DUPLICATE_APPLICATION_PATTERN")
	if len(matches) != 1 {
		t.Fatalf("Pattern重複Diagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
}

func TestValidateDoesNotAddPatternDiagnosticsForValidBrowseReference(t *testing.T) {
	diagnostics := Validate(validApplicationPatternWorld())
	for _, diagnostic := range diagnostics {
		if isApplicationPatternDiagnostic(diagnostic.Code) {
			t.Fatalf("正常なbrowse Patternに不要なDiagnosticが追加されました: %#v", diagnostic)
		}
	}
}

func TestValidateAcceptsDataOnlyRecordPattern(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns = []application.Pattern{{
		ID:   "order_entry.order_record",
		Kind: application.PatternRecord,
		Data: "sales.order",
	}}

	diagnostics := Validate(w)
	for _, diagnostic := range diagnostics {
		if isApplicationPatternDiagnostic(diagnostic.Code) {
			t.Fatalf("Dataだけを持つ正常なrecord Patternに不要なDiagnosticが追加されました: %#v", diagnostic)
		}
	}
}

func TestValidateAcceptsRelationAndDatasourceReferencePattern(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns = []application.Pattern{{
		ID:         "order_entry.customer_reference",
		Kind:       application.PatternReference,
		Relation:   "sales.order.customer",
		Datasource: "master",
	}}

	diagnostics := Validate(w)
	for _, diagnostic := range diagnostics {
		if isApplicationPatternDiagnostic(diagnostic.Code) {
			t.Fatalf("RelationとDatasourceを持つ正常なreference Patternに不要なDiagnosticが追加されました: %#v", diagnostic)
		}
	}
}

func TestValidateAcceptsCapabilityOnlyCommandPattern(t *testing.T) {
	diagnostics := Validate(validCommandPatternWorld())
	for _, diagnostic := range diagnostics {
		if isApplicationPatternDiagnostic(diagnostic.Code) {
			t.Fatalf("Capabilityだけを持つ正常なcommand Patternに不要なDiagnosticが追加されました: %#v", diagnostic)
		}
	}
}

func TestValidateRejectsRecordQueryOnce(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns = []application.Pattern{{
		ID:    "order_entry.order_record",
		Kind:  application.PatternRecord,
		Data:  "sales.order",
		Query: "order_entry.recent_orders",
	}}

	diagnostics := Validate(w)
	matches := diagnosticsByCode(diagnostics, "APPLICATION_PATTERN_QUERY_NOT_ALLOWED")
	if len(matches) != 1 {
		t.Fatalf("recordのQuery禁止Diagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
	if matches[0].Subject == nil || matches[0].Subject.ID != "order_entry.order_record" {
		t.Fatalf("record DiagnosticのPattern Subjectが失われました: %#v", matches[0].Subject)
	}
}

func TestValidateRejectsUnknownReferenceRelationOnce(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Patterns = []application.Pattern{{
		ID:         "order_entry.customer_reference",
		Kind:       application.PatternReference,
		Relation:   "sales.order.missing",
		Datasource: "master",
	}}

	diagnostics := Validate(w)
	matches := diagnosticsByCode(diagnostics, "UNKNOWN_APPLICATION_PATTERN_RELATION")
	if len(matches) != 1 {
		t.Fatalf("referenceの未知Relation Diagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
	if matches[0].Subject == nil || matches[0].Subject.Kind != "application-pattern" || matches[0].Subject.ID != "order_entry.customer_reference" {
		t.Fatalf("reference DiagnosticのPattern Subjectが失われました: %#v", matches[0].Subject)
	}
}

func diagnosticsByCode(diagnostics []Diagnostic, code string) []Diagnostic {
	var matches []Diagnostic
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			matches = append(matches, diagnostic)
		}
	}
	return matches
}

func isApplicationPatternDiagnostic(code string) bool {
	switch code {
	case "DUPLICATE_APPLICATION_PATTERN",
		"INVALID_APPLICATION_PATTERN_KIND",
		"APPLICATION_PATTERN_DATA_REQUIRED",
		"UNKNOWN_APPLICATION_PATTERN_DATA",
		"APPLICATION_PATTERN_DATA_NOT_ALLOWED",
		"APPLICATION_PATTERN_RELATION_REQUIRED",
		"UNKNOWN_APPLICATION_PATTERN_RELATION",
		"APPLICATION_PATTERN_RELATION_NOT_ALLOWED",
		"APPLICATION_PATTERN_QUERY_REQUIRED",
		"UNKNOWN_APPLICATION_PATTERN_QUERY",
		"APPLICATION_PATTERN_QUERY_APPLICATION_MISMATCH",
		"APPLICATION_PATTERN_QUERY_NOT_ALLOWED",
		"APPLICATION_PATTERN_CAPABILITY_REQUIRED",
		"UNKNOWN_APPLICATION_PATTERN_CAPABILITY",
		"APPLICATION_PATTERN_CAPABILITY_NOT_ALLOWED",
		"APPLICATION_PATTERN_DATASOURCE_REQUIRED",
		"UNKNOWN_APPLICATION_PATTERN_DATASOURCE",
		"APPLICATION_PATTERN_DATASOURCE_QUERY_MISMATCH",
		"APPLICATION_PATTERN_DATASOURCE_NOT_READABLE",
		"APPLICATION_PATTERN_DATASOURCE_NOT_ALLOWED":
		return true
	default:
		return false
	}
}
