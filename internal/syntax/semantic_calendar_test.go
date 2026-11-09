package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

// TestSemanticCalendarParses checks that a calendar declaration records what it
// declares. The calendar is meaning: it does not decide how a period boundary
// is computed.
func TestSemanticCalendarParses(t *testing.T) {
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
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(namespace.Calendars) != 1 {
		t.Fatalf("calendars=%+v", namespace.Calendars)
	}
	calendar := namespace.Calendars[0]
	if calendar.ID != "sales.fiscal" {
		t.Fatalf("id=%q", calendar.ID)
	}
	if calendar.FiscalYearStartMonth != 4 {
		t.Fatalf("fiscal year start=%d", calendar.FiscalYearStartMonth)
	}
	if calendar.WeekStarts != "monday" {
		t.Fatalf("week starts=%q", calendar.WeekStarts)
	}
	if calendar.BusinessDayExcludes != "sales.holiday" {
		t.Fatalf("excludes=%q", calendar.BusinessDayExcludes)
	}
}

// TestSemanticPeriodCarriesCalendar checks that a period can name the calendar
// its boundary follows.
func TestSemanticPeriodCarriesCalendar(t *testing.T) {
	src := `namespace sales

data Holiday {
    attribute holiday_on type std.date
}

data Order {
    attribute order_date type std.date
}

calendar sales.fiscal {
    fiscal-year starts month 4
    week starts monday
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    calendar sales.fiscal
    states open closing closed
}
`
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(namespace.Periods) != 1 || namespace.Periods[0].Calendar != "sales.fiscal" {
		t.Fatalf("period calendar=%+v", namespace.Periods)
	}
}

// TestSemanticCalendarRejectsUnknownMember checks that an unsupported member is
// refused rather than ignored, so a misspelt rule is not silently dropped.
func TestSemanticCalendarRejectsUnknownMember(t *testing.T) {
	src := `namespace sales

calendar sales.fiscal {
    fiscal-year starts month 4
    week starts monday
    fiscal-year ends month 3
}
`
	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("an unsupported calendar member was accepted")
	}
}
