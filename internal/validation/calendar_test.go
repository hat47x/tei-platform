package validation_test

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// calendarDiagnostics validates one calendar declaration and returns the codes
// the calendar or period rules produced.
func calendarDiagnostics(t *testing.T, src string) []string {
	t.Helper()
	namespace := parseSemantic(t, src)
	diagnostics := validation.Validate(ir.World{Semantics: []semantic.Namespace{namespace}})
	var codes []string
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Code, "CALENDAR_") || strings.HasPrefix(diagnostic.Code, "PERIOD_CALENDAR") ||
			diagnostic.Code == "PERIOD_UNKNOWN_CALENDAR" {
			codes = append(codes, diagnostic.Code)
		}
	}
	return codes
}

// TestValidateCalendarAcceptsACompleteDeclaration checks that a well-formed
// calendar produces no calendar diagnostic, so the rules do not fire on correct
// meaning.
func TestValidateCalendarAcceptsACompleteDeclaration(t *testing.T) {
	src := `namespace sales

data Holiday {
    attribute holiday_on type std.date
}

calendar sales.fiscal {
    fiscal-year starts month 4
    week starts monday
    business-day excludes sales.holiday
}
`
	if codes := calendarDiagnostics(t, src); len(codes) != 0 {
		t.Fatalf("codes=%v", codes)
	}
}

// TestValidateCalendarRequiresTheFiscalYearStart checks that a calendar without
// a start month is reported: the boundary cannot be derived from silence.
func TestValidateCalendarRequiresTheFiscalYearStart(t *testing.T) {
	src := `namespace sales

calendar sales.fiscal {
    week starts monday
}
`
	codes := calendarDiagnostics(t, src)
	if !containsCode(codes, "CALENDAR_FISCAL_YEAR_START_REQUIRED") {
		t.Fatalf("codes=%v", codes)
	}
}

// TestValidateCalendarRejectsAnImpossibleStartMonth checks that a month outside
// 1..12 is refused.
func TestValidateCalendarRejectsAnImpossibleStartMonth(t *testing.T) {
	src := `namespace sales

calendar sales.fiscal {
    fiscal-year starts month 13
    week starts monday
}
`
	codes := calendarDiagnostics(t, src)
	if !containsCode(codes, "CALENDAR_FISCAL_YEAR_START_RANGE") {
		t.Fatalf("codes=%v", codes)
	}
}

// TestValidateCalendarRequiresTheWeekStart checks that a calendar without a week
// start is reported instead of assuming Monday.
func TestValidateCalendarRequiresTheWeekStart(t *testing.T) {
	src := `namespace sales

calendar sales.fiscal {
    fiscal-year starts month 4
}
`
	codes := calendarDiagnostics(t, src)
	if !containsCode(codes, "CALENDAR_WEEK_START_REQUIRED") {
		t.Fatalf("codes=%v", codes)
	}
}

// TestValidateCalendarRejectsAnUnknownWeekStart checks that an unsupported
// weekday is refused rather than guessed.
func TestValidateCalendarRejectsAnUnknownWeekStart(t *testing.T) {
	src := `namespace sales

calendar sales.fiscal {
    fiscal-year starts month 4
    week starts wednesday
}
`
	codes := calendarDiagnostics(t, src)
	if !containsCode(codes, "CALENDAR_WEEK_START_UNSUPPORTED") {
		t.Fatalf("codes=%v", codes)
	}
}

// TestValidateCalendarRejectsAnUnknownHolidayData checks that the exclusions
// must name declared data.
func TestValidateCalendarRejectsAnUnknownHolidayData(t *testing.T) {
	src := `namespace sales

calendar sales.fiscal {
    fiscal-year starts month 4
    week starts monday
    business-day excludes sales.nonexistent
}
`
	codes := calendarDiagnostics(t, src)
	if !containsCode(codes, "CALENDAR_UNKNOWN_EXCLUDES_DATA") {
		t.Fatalf("codes=%v", codes)
	}
}

// TestValidatePeriodRejectsAnUnknownCalendar checks that a period naming a
// calendar that does not exist is reported.
func TestValidatePeriodRejectsAnUnknownCalendar(t *testing.T) {
	src := `namespace sales

data Order {
    attribute order_date type std.date
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    calendar sales.missing
    states open closing closed
}
`
	codes := calendarDiagnostics(t, src)
	if !containsCode(codes, "PERIOD_UNKNOWN_CALENDAR") {
		t.Fatalf("codes=%v", codes)
	}
}

// TestValidatePeriodRejectsACalendarFromAnotherNamespace checks that a period
// does not take a calendar from a different namespace, which would let a
// boundary rule cross an ownership line silently.
func TestValidatePeriodRejectsACalendarFromAnotherNamespace(t *testing.T) {
	world := parseSemantic(t, `namespace sales

data Order {
    attribute order_date type std.date
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    calendar company.fiscal
    states open closing closed
}
`)
	company := parseSemantic(t, `namespace company

calendar company.fiscal {
    fiscal-year starts month 4
    week starts monday
}
`)
	diagnostics := validation.Validate(ir.World{Semantics: []semantic.Namespace{world, company}})
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "PERIOD_CALENDAR_NAMESPACE_MISMATCH" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func containsCode(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}
