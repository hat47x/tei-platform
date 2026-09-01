package validation

import "testing"

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
		"APPLICATION_PATTERN_QUERY_REQUIRED",
		"UNKNOWN_APPLICATION_PATTERN_QUERY",
		"APPLICATION_PATTERN_QUERY_APPLICATION_MISMATCH",
		"UNKNOWN_APPLICATION_PATTERN_DATASOURCE",
		"APPLICATION_PATTERN_DATASOURCE_QUERY_MISMATCH":
		return true
	default:
		return false
	}
}
