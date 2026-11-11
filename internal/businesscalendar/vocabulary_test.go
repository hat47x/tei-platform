package businesscalendar

import "testing"

// WeekStarts decides which day a calendar week begins on. A reader of a derived
// schedule branches on it, so the set is part of this package's contract (#4032).
func TestWeekStartsValuesAreCovered(t *testing.T) {
	values := []WeekStarts{WeekStartsSunday, WeekStartsMonday}
	want := []string{"sunday", "monday"}
	if len(values) != len(want) {
		t.Fatalf("a week-start value was dropped: %d values", len(values))
	}
	for i, value := range values {
		if string(value) != want[i] {
			t.Fatalf("values[%d]=%q want %q", i, value, want[i])
		}
	}
}
