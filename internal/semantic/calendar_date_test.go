package semantic

import "testing"

func TestParseCalendarDateReadsOnlyAWholeRealDay(t *testing.T) {
	accepted := map[string]CalendarDate{
		"2026-04-01": {2026, 4, 1},
		"2024-02-29": {2024, 2, 29}, // a leap day
		"1999-12-31": {1999, 12, 31},
		"0001-01-01": {1, 1, 1},
	}
	for text, want := range accepted {
		got, err := ParseCalendarDate(text)
		if err != nil || got != want || got.String() != text || got.IsZero() {
			t.Fatalf("%q: got=%+v err=%v", text, got, err)
		}
	}
	refused := []string{
		"", "2026-4-1", "2026/04/01", "20260401", "2026-04-01 ", " 2026-04-01",
		"2026-04-01T00:00:00Z", "2026-04-01T09:00", "2026-02-30", "2025-02-29", "2026-13-01",
		"2026-00-10", "2026-04-00", "2026-04-31", "26-04-01", "2026-04-1", "２０２６-04-01",
		"0000-01-01", "2026-04", "today", "-2026-04-1",
	}
	for _, text := range refused {
		if got, err := ParseCalendarDate(text); err == nil || !got.IsZero() {
			t.Fatalf("%q was read as %+v", text, got)
		}
	}
}

func TestCalendarDateOrder(t *testing.T) {
	// Written in calendar order. Each is before every later one.
	ordered := []CalendarDate{{2025, 12, 31}, {2026, 1, 1}, {2026, 1, 2}, {2026, 2, 1}, {2026, 10, 9}, {2026, 10, 10}, {2027, 1, 1}}
	for i, left := range ordered {
		for j, right := range ordered {
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if got := left.Compare(right); got != want {
				t.Fatalf("%s compared with %s = %d, want %d", left, right, got, want)
			}
		}
	}
	if !(CalendarDate{}).IsZero() {
		t.Fatal("the zero value must not be a date")
	}
}
