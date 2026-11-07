package taxrate_test

import (
	"errors"
	"testing"
	"time"

	"tei/internal/taxrate"
)

func testLocale(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load time zone: %v", err)
	}
	return location
}

func day(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", value, testLocale(t))
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

func standardSet(t *testing.T) taxrate.Set {
	t.Helper()
	return taxrate.Set{
		Rounding: taxrate.RoundingFloor,
		Locale:   testLocale(t),
		Rates: []taxrate.Rate{
			{AppliesFrom: day(t, "2019-10-01"), AppliesUntil: day(t, "2023-09-30"), Value: 0.10},
			{AppliesFrom: day(t, "2023-10-01"), AppliesUntil: day(t, "2026-09-30"), Value: 0.08},
			{AppliesFrom: day(t, "2026-10-01"), Value: 0.05},
		},
	}
}

// TestRateForUsesTheDeclaredSpan checks that a rate applies between its declared
// dates, including both ends, so a boundary is testable. X-TEA had no end date.
func TestRateForUsesTheDeclaredSpan(t *testing.T) {
	set := standardSet(t)
	cases := []struct {
		date string
		want float64
	}{
		{"2023-09-30", 0.10},
		{"2023-10-01", 0.08},
		{"2026-09-30", 0.08},
		{"2026-10-01", 0.05},
	}
	for _, tc := range cases {
		rate, err := set.RateFor(day(t, tc.date))
		if err != nil {
			t.Fatalf("%s: %v", tc.date, err)
		}
		if rate.Value != tc.want {
			t.Fatalf("%s rate=%v want=%v", tc.date, rate.Value, tc.want)
		}
	}
}

// TestRateForRefusesADateNoRowCovers checks that a missing rate is reported
// instead of a zero rate.
func TestRateForRefusesADateNoRowCovers(t *testing.T) {
	set := standardSet(t)
	if _, err := set.RateFor(day(t, "2019-09-30")); !errors.Is(err, taxrate.ErrInvalidRate) {
		t.Fatalf("err=%v", err)
	}
}

// TestRateForRefusesOverlappingRows checks that two rows covering the same date
// are reported, not silently resolved by order. X-TEA took the newest start date
// and never noticed the overlap.
func TestRateForRefusesOverlappingRows(t *testing.T) {
	set := taxrate.Set{
		Rounding: taxrate.RoundingFloor,
		Locale:   testLocale(t),
		Rates: []taxrate.Rate{
			{AppliesFrom: day(t, "2023-10-01"), AppliesUntil: day(t, "2026-09-30"), Value: 0.08},
			{AppliesFrom: day(t, "2025-01-01"), AppliesUntil: day(t, "2027-12-31"), Value: 0.10},
		},
	}
	if _, err := set.RateFor(day(t, "2025-06-01")); !errors.Is(err, taxrate.ErrInvalidRate) {
		t.Fatalf("err=%v", err)
	}
	if overlaps := set.Overlaps(); len(overlaps) != 1 {
		t.Fatalf("overlaps=%v", overlaps)
	}
}

// TestAmountRoundsByTheDeclaredRule checks that the rounding is the declaration's
// rule and not the sign of the amount. X-TEA floored a positive amount and
// ceilinged a negative one (D:Session.java L2330–2334), so a correction rounded
// the other way.
func TestAmountRoundsByTheDeclaredRule(t *testing.T) {
	set := standardSet(t)
	positive, err := set.Amount(day(t, "2024-05-01"), 1080)
	if err != nil {
		t.Fatal(err)
	}
	if positive != 86 {
		t.Fatalf("positive=%d", positive)
	}
	negative, err := set.Amount(day(t, "2024-05-01"), -1080)
	if err != nil {
		t.Fatal(err)
	}
	if negative != -87 {
		t.Fatalf("negative=%d", negative)
	}
}

// TestRoundNamesTheRule checks each rounding rule against the same value, so a
// declaration cannot leave the rule unnamed.
func TestRoundNamesTheRule(t *testing.T) {
	cases := []struct {
		rounding taxrate.Rounding
		value    float64
		want     int64
	}{
		{taxrate.RoundingFloor, 86.4, 86},
		{taxrate.RoundingCeiling, 86.4, 87},
		{taxrate.RoundingHalfUp, 86.5, 87},
		{taxrate.RoundingHalfEven, 86.5, 86},
		{taxrate.RoundingHalfEven, 87.5, 88},
	}
	for _, tc := range cases {
		set := taxrate.Set{Rounding: tc.rounding, Locale: testLocale(t)}
		got, err := set.Round(tc.value)
		if err != nil {
			t.Fatalf("%s: %v", tc.rounding, err)
		}
		if got != tc.want {
			t.Fatalf("%s(%v)=%d want=%d", tc.rounding, tc.value, got, tc.want)
		}
	}
}

// TestSetWithoutARoundingRuleIsRefused checks that silence is not a rounding
// rule.
func TestSetWithoutARoundingRuleIsRefused(t *testing.T) {
	set := taxrate.Set{Locale: testLocale(t)}
	if _, err := set.Round(10); !errors.Is(err, taxrate.ErrInvalidRate) {
		t.Fatalf("err=%v", err)
	}
	noZone := taxrate.Set{Rounding: taxrate.RoundingFloor}
	if _, err := noZone.Round(10); !errors.Is(err, taxrate.ErrInvalidRate) {
		t.Fatalf("err=%v", err)
	}
}

// TestRateForUsesTheCategory checks that a row for another category is not used.
func TestRateForUsesTheCategory(t *testing.T) {
	set := taxrate.Set{
		Category: "reduced",
		Rounding: taxrate.RoundingFloor,
		Locale:   testLocale(t),
		Rates: []taxrate.Rate{
			{Category: "standard", AppliesFrom: day(t, "2023-10-01"), Value: 0.08},
			{Category: "reduced", AppliesFrom: day(t, "2023-10-01"), Value: 0.04},
		},
	}
	if _, err := set.RateFor(day(t, "2024-05-01")); err != nil {
		t.Fatal(err)
	}
	// Two rows that name different categories do not overlap.
	if overlaps := set.Overlaps(); len(overlaps) != 0 {
		t.Fatalf("overlaps=%v", overlaps)
	}
}
