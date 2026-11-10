package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateSemanticCalendars validates the declared business calendar: the
// fiscal year start month, the weekday the week starts, and the data that holds
// the exceptions. It does not compute a date or decide a period boundary.
func validateSemanticCalendars(w ir.World) []Diagnostic {
	data := map[core.CanonicalID]bool{}
	for _, namespace := range w.Semantics {
		for _, element := range namespace.Data {
			data[element.ID] = true
		}
	}
	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, calendar := range namespace.Calendars {
			subj := subject.Subject{Kind: "semantic-calendar", ID: string(calendar.ID)}
			if calendar.FiscalYearStartMonth == 0 {
				out = append(out, errFor(subj, "CALENDAR_FISCAL_YEAR_START_REQUIRED", "calendar %s does not declare the fiscal year start month", calendar.ID))
			} else if calendar.FiscalYearStartMonth < 1 || calendar.FiscalYearStartMonth > 12 {
				out = append(out, errFor(subj, "CALENDAR_FISCAL_YEAR_START_RANGE", "calendar %s fiscal year starts at month %d, which is outside 1..12", calendar.ID, calendar.FiscalYearStartMonth))
			}
			switch calendar.WeekStarts {
			case "monday", "sunday":
			case "":
				out = append(out, errFor(subj, "CALENDAR_WEEK_START_REQUIRED", "calendar %s does not declare which weekday starts the week", calendar.ID))
			default:
				out = append(out, errFor(subj, "CALENDAR_WEEK_START_UNSUPPORTED", "calendar %s declares unsupported week start %q", calendar.ID, calendar.WeekStarts))
			}
			if calendar.BusinessDayExcludes != "" && !data[calendar.BusinessDayExcludes] {
				out = append(out, errFor(subj, "CALENDAR_UNKNOWN_EXCLUDES_DATA", "calendar %s excludes business days using unknown data %s", calendar.ID, calendar.BusinessDayExcludes))
			}
		}
	}
	return out
}

// validatePeriodCalendars checks that a period's declared calendar exists and
// belongs to the same semantic namespace as the period. It does not decide how
// the calendar derives the period key.
func validatePeriodCalendars(w ir.World) []Diagnostic {
	calendars := map[core.CanonicalID]core.CanonicalID{}
	for _, namespace := range w.Semantics {
		for _, calendar := range namespace.Calendars {
			calendars[calendar.ID] = namespace.ID
		}
	}
	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, period := range namespace.Periods {
			if period.Calendar == "" {
				continue
			}
			subj := subject.Subject{Kind: "semantic-period", ID: string(period.ID)}
			owner, ok := calendars[period.Calendar]
			if !ok {
				out = append(out, errFor(subj, "PERIOD_UNKNOWN_CALENDAR", "period %s refers to unknown calendar %s", period.ID, period.Calendar))
				continue
			}
			if owner != namespace.ID {
				out = append(out, errFor(subj, "PERIOD_CALENDAR_NAMESPACE_MISMATCH", "period %s refers to calendar %s from another namespace %s", period.ID, period.Calendar, owner))
			}
		}
		// A closing day splits a month, so it needs a month to split: the period
		// must also name the calendar that decides the month.
		for _, period := range namespace.Periods {
			if period.ClosingDay == 0 {
				continue
			}
			subj := subject.Subject{Kind: "semantic-period", ID: string(period.ID)}
			if period.Calendar == "" {
				out = append(out, errFor(subj, "PERIOD_CLOSING_REQUIRES_CALENDAR", "period %s declares closing day %d but no calendar", period.ID, period.ClosingDay))
			}
			if period.ClosingDay < 1 || period.ClosingDay > 28 {
				out = append(out, errFor(subj, "PERIOD_CLOSING_DAY_RANGE", "period %s declares closing day %d, which is outside 1..28", period.ID, period.ClosingDay))
			}
		}
	}
	return out
}
