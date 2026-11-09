package currencyrate_test

import (
	"errors"
	"testing"
	"time"

	"tei/internal/currencyrate"
)

func testLocale(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load time zone: %v", err)
	}
	return location
}

func standardSet(t *testing.T) currencyrate.Set {
	t.Helper()
	return currencyrate.Set{
		SystemCurrency: "JPY",
		Kind:           currencyrate.KindMiddle,
		Direction:      currencyrate.DirectionForeignToSystem,
		Rounding:       currencyrate.RoundingFloor,
		Locale:         testLocale(t),
		Rates: []currencyrate.Rate{
			{Currency: "USD", FiscalYear: 2026, MonthSequence: 1, Middle: 150.25, Buying: 148.10, Selling: 152.40},
			{Currency: "EUR", FiscalYear: 2026, MonthSequence: 1, Middle: 163.80, Buying: 161.50, Selling: 166.10},
		},
	}
}

// TestRateForSelectsTheDeclaredKind checks that the rate is the declared kind,
// not the string the caller typed. X-TEA treated anything other than "TTB" and
// "TTS" as the middle rate (D:Session.java L2265–2277).
func TestRateForSelectsTheDeclaredKind(t *testing.T) {
	set := standardSet(t)
	middle, err := set.RateFor("USD", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if middle != 150.25 {
		t.Fatalf("middle=%v", middle)
	}
	set.Kind = currencyrate.KindBuying
	buying, err := set.RateFor("USD", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if buying != 148.10 {
		t.Fatalf("buying=%v", buying)
	}
	set.Kind = currencyrate.KindSelling
	selling, err := set.RateFor("USD", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if selling != 152.40 {
		t.Fatalf("selling=%v", selling)
	}
}

// TestSystemCurrencyConvertsAtOne checks the one shortcut that is still correct:
// the system currency has no other row that could apply.
func TestSystemCurrencyConvertsAtOne(t *testing.T) {
	set := standardSet(t)
	rate, err := set.RateFor("JPY", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 1 {
		t.Fatalf("rate=%v", rate)
	}
}

// TestRateForRefusesAMissingRow checks that a missing row is reported, not
// silently replaced by an annual fallback. X-TEA fell back when a monthly rate
// read as zero (L2296–2298), so "no row" and "a zero rate" were the same value.
func TestRateForRefusesAMissingRow(t *testing.T) {
	set := standardSet(t)
	if _, err := set.RateFor("USD", 2026, 2); !errors.Is(err, currencyrate.ErrInvalidRate) {
		t.Fatalf("err=%v", err)
	}
}

// TestRateForRefusesTwoRowsForOnePeriod checks that an ambiguous period is
// reported instead of resolved by order.
func TestRateForRefusesTwoRowsForOnePeriod(t *testing.T) {
	set := standardSet(t)
	set.Rates = append(set.Rates, currencyrate.Rate{Currency: "USD", FiscalYear: 2026, MonthSequence: 1, Middle: 151.00})
	if _, err := set.RateFor("USD", 2026, 1); !errors.Is(err, currencyrate.ErrInvalidRate) {
		t.Fatalf("err=%v", err)
	}
}

// TestConvertUsesTheDeclaredDirection checks both directions, so the caller does
// not decide the direction by how it uses the number.
func TestConvertUsesTheDeclaredDirection(t *testing.T) {
	set := standardSet(t)
	foreign, err := set.Convert(100, "USD", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if foreign != 15025 {
		t.Fatalf("foreign-to-system=%d", foreign)
	}
	set.Direction = currencyrate.DirectionSystemToForeign
	system, err := set.Convert(15025, "USD", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if system != 100 {
		t.Fatalf("system-to-foreign=%d", system)
	}
}

// TestConvertRefusesAZeroRate checks that a zero rate is an error rather than a
// silent fallback.
func TestConvertRefusesAZeroRate(t *testing.T) {
	set := standardSet(t)
	set.Rates = []currencyrate.Rate{{Currency: "USD", FiscalYear: 2026, MonthSequence: 1, Middle: 0}}
	for _, kind := range []currencyrate.Kind{currencyrate.KindMiddle, currencyrate.KindBuying, currencyrate.KindSelling} {
		set.Kind = kind
		if _, err := set.Convert(100, "USD", 2026, 1); !errors.Is(err, currencyrate.ErrInvalidRate) {
			t.Fatalf("%s err=%v", kind, err)
		}
	}
}

// TestSetWithoutADeclaredRuleIsRefused checks that silence is not a rule, for
// each of the four declarations the conversion needs.
func TestSetWithoutADeclaredRuleIsRefused(t *testing.T) {
	cases := []currencyrate.Set{
		{Kind: currencyrate.KindMiddle, Direction: currencyrate.DirectionForeignToSystem, Rounding: currencyrate.RoundingFloor, Locale: testLocale(t)},
		{SystemCurrency: "JPY", Direction: currencyrate.DirectionForeignToSystem, Rounding: currencyrate.RoundingFloor, Locale: testLocale(t)},
		{SystemCurrency: "JPY", Kind: currencyrate.KindMiddle, Rounding: currencyrate.RoundingFloor, Locale: testLocale(t)},
		{SystemCurrency: "JPY", Kind: currencyrate.KindMiddle, Direction: currencyrate.DirectionForeignToSystem, Locale: testLocale(t)},
		{SystemCurrency: "JPY", Kind: currencyrate.KindMiddle, Direction: currencyrate.DirectionForeignToSystem, Rounding: currencyrate.RoundingFloor},
	}
	for index, set := range cases {
		if _, err := set.Round(1); !errors.Is(err, currencyrate.ErrInvalidRate) {
			t.Fatalf("case %d err=%v", index, err)
		}
	}
}

// TestMissingPeriodsNamesTheGaps checks that a report can list the periods with
// no row instead of falling back.
func TestMissingPeriodsNamesTheGaps(t *testing.T) {
	set := standardSet(t)
	set.Rates = []currencyrate.Rate{
		{Currency: "USD", FiscalYear: 2026, MonthSequence: 1, Middle: 150.25},
		{Currency: "USD", FiscalYear: 2026, MonthSequence: 3, Middle: 151.00},
	}
	missing := set.MissingPeriods("USD", 2026, 1, 2026, 3)
	if len(missing) != 1 || missing[0].MonthSequence != 2 {
		t.Fatalf("missing=%v", missing)
	}
	if missing := set.MissingPeriods("USD", 2026, 1, 2026, 1); len(missing) != 0 {
		t.Fatalf("missing=%v", missing)
	}
}
