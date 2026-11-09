package monthendclose

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func testCalendar(t *testing.T) Calendar {
	t.Helper()
	c, err := ReadCalendar(filepath.Join("testdata", "calendar-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func testRule(t *testing.T) CloseRule {
	t.Helper()
	r, err := ReadRule(filepath.Join("testdata", "close-rule.json"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// syntheticPayables have nominal due dates around the September close.
func syntheticPayables() []Payable {
	return []Payable{
		{"P-001", "2026-10-01"}, // Thursday; becomes a temporary holiday in v2
		{"P-002", "2026-10-03"}, // Saturday
		{"P-003", "2026-10-04"}, // Sunday
		{"P-004", "2026-09-23"}, // autumn equinox, after two holidays and a weekend
		{"P-005", "2026-10-02"},
		{"P-006", "2026-10-31"}, // Saturday
		{"P-007", "2026-10-01"},
	}
}

const tempHoliday = "2026-10-01"

func TestComputeDecisionAndRevisionImpact(t *testing.T) {
	v1 := testCalendar(t)
	rule := testRule(t)
	d1, err := ComputeDecision(v1, rule, "2026-09", mustTime(t, "2026-09-28T09:00:00+09:00"), syntheticPayables())
	if err != nil {
		t.Fatal(err)
	}
	if d1.CloseDate != "2026-09-30" || d1.CutoffAt != "2026-09-30T17:00:00+09:00" {
		t.Fatalf("close date %s cutoff %s", d1.CloseDate, d1.CutoffAt)
	}
	if at, _ := d1.JobAt(JobClose); at.Format(time.RFC3339) != "2026-10-01T00:00:00+09:00" {
		t.Fatalf("close job %s", at)
	}
	due := map[string]string{}
	for _, x := range d1.DueDates {
		due[x.PayableID] = x.Due
	}
	if due["P-001"] != "2026-10-01" || due["P-002"] != "2026-10-02" || due["P-004"] != "2026-09-18" || due["P-006"] != "2026-10-30" {
		t.Fatalf("v1 due dates %v", due)
	}
	// The revision is recorded the day before the close date.
	v2, err := v1.Revise([]Holiday{{tempHoliday, "臨時休業"}}, "2026-09-29T10:00:00+09:00")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ComputeDecision(v2, rule, "2026-09", mustTime(t, "2026-09-28T09:00:00+09:00"), syntheticPayables()); !errors.Is(err, ErrCalendarNotYetKnown) {
		t.Fatalf("v2 used before it was recorded: %v", err)
	}
	d2, err := ComputeDecision(v2, rule, "2026-09", mustTime(t, "2026-09-29T12:00:00+09:00"), syntheticPayables())
	if err != nil {
		t.Fatal(err)
	}
	im := DiffDecisions(d1, d2, v1, v2)
	if len(im.JobShifts) != 1 || im.JobShifts[0].Name != JobClose || im.JobShifts[0].To != "2026-10-02T00:00:00+09:00" {
		t.Fatalf("job shifts %+v", im.JobShifts)
	}
	if len(im.DueShifts) != 2 {
		t.Fatalf("due shifts %+v", im.DueShifts)
	}
	for _, s := range im.DueShifts {
		if s.From != "2026-10-01" || s.To != "2026-09-30" || !s.DueOnCloseDate {
			t.Fatalf("due shift %+v", s)
		}
	}
	if im.CloseDateFrom != im.CloseDateTo || len(im.AddedHolidays) != 1 || im.AddedHolidays[0] != tempHoliday {
		t.Fatalf("impact %+v", im)
	}
	if d1.Digest == d2.Digest || d1.CalendarDigest == d2.CalendarDigest {
		t.Fatal("revision did not change the digests")
	}
}

func TestNthBusinessDayRule(t *testing.T) {
	v1 := testCalendar(t)
	rule := testRule(t)
	rule.CloseDay, rule.N = "nth-business-day", 3
	d, err := ComputeDecision(v1, rule, "2026-10", mustTime(t, "2026-09-28T09:00:00+09:00"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.CloseDate != "2026-10-05" {
		t.Fatalf("3rd business day of October v1: %s", d.CloseDate)
	}
	v2, _ := v1.Revise([]Holiday{{tempHoliday, "臨時休業"}}, "2026-09-29T10:00:00+09:00")
	d2, err := ComputeDecision(v2, rule, "2026-10", mustTime(t, "2026-09-29T12:00:00+09:00"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if d2.CloseDate != "2026-10-06" {
		t.Fatalf("3rd business day of October v2: %s", d2.CloseDate)
	}
}

func TestDecisionTamperDetected(t *testing.T) {
	d, err := ComputeDecision(testCalendar(t), testRule(t), "2026-09", mustTime(t, "2026-09-28T09:00:00+09:00"), syntheticPayables())
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Verify(); err != nil {
		t.Fatal(err)
	}
	d.CloseDate = "2026-09-29"
	if err := d.Verify(); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("tampered decision accepted: %v", err)
	}
}

func TestCalendarVersionImmutable(t *testing.T) {
	c := testCalendar(t)
	c.Holidays = append(c.Holidays, Holiday{"2026-12-31", "x"})
	if err := ValidateCalendar(c); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("edited calendar version accepted: %v", err)
	}
}

// The business date is the local date of acceptance in the decision's
// timezone, whatever the zone of the instant passed in.
func TestBusinessDateBoundary(t *testing.T) {
	d, err := ComputeDecision(testCalendar(t), testRule(t), "2026-09", mustTime(t, "2026-09-28T09:00:00+09:00"), nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ at, month string }{
		{"2026-09-30T23:59:59+09:00", "2026-09"},
		{"2026-09-30T14:59:59Z", "2026-09"},
		{"2026-09-30T15:00:00Z", "2026-10"}, // 00:00 JST on 1 October; a UTC date would say September
		{"2026-08-31T15:00:00Z", "2026-09"}, // 00:00 JST on 1 September; a UTC date would say August
		{"2026-08-31T23:59:59+09:00", "2026-08"},
	}
	for _, c := range cases {
		_, m, err := d.BusinessDate(mustTime(t, c.at).UTC())
		if err != nil || m != c.month {
			t.Fatalf("%s: month %s err %v, want %s", c.at, m, err, c.month)
		}
	}
}
