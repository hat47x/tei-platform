package currencyrate_test

import (
	"testing"

	"tei/internal/currencyrate"
)

func fallbackSet(fallback currencyrate.Fallback) currencyrate.Set {
	return currencyrate.Set{
		SystemCurrency: "JPY",
		Kind:           currencyrate.KindMiddle,
		Direction:      currencyrate.DirectionForeignToSystem,
		Rounding:       currencyrate.RoundingFloor,
		Locale:         nil, // filled by the caller
		Fallback:       fallback,
		Rates: []currencyrate.Rate{
			{Currency: "USD", FiscalYear: 2026, MonthSequence: 1, Middle: 150.25},
			{Currency: "USD", FiscalYear: 2026, MonthSequence: currencyrate.AnnualMonthSequence, Middle: 149.00},
		},
	}
}

// TestFallbackDefaultIsNone checks that a set without a declared fallback does
// not substitute another period. X-TEA fell back whenever the monthly rate read
// as zero (D:Session.java L2296–2298), so a caller got a rate it did not ask for.
func TestFallbackDefaultIsNone(t *testing.T) {
	set := fallbackSet("")
	set.Locale = testLocale(t)
	if _, err := set.RateForWithFallback("USD", 2026, 2); err == nil {
		t.Fatal("a missing monthly rate was filled without a declared fallback")
	}
}

// TestFallbackAnnualUsesTheAnnualRow checks that a declared fallback reads the
// annual row, and that the monthly row wins when it exists.
func TestFallbackAnnualUsesTheAnnualRow(t *testing.T) {
	set := fallbackSet(currencyrate.FallbackAnnual)
	set.Locale = testLocale(t)
	rate, err := set.RateForWithFallback("USD", 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 149.00 {
		t.Fatalf("rate=%v", rate)
	}
	monthly, err := set.RateForWithFallback("USD", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	if monthly != 150.25 {
		t.Fatalf("monthly=%v", monthly)
	}
}

// TestFallbackAnnualReportsAMissingAnnualRow checks that a fallback which cannot
// answer is reported rather than returning zero.
func TestFallbackAnnualReportsAMissingAnnualRow(t *testing.T) {
	set := fallbackSet(currencyrate.FallbackAnnual)
	set.Locale = testLocale(t)
	set.Rates = []currencyrate.Rate{{Currency: "USD", FiscalYear: 2026, MonthSequence: 1, Middle: 150.25}}
	if _, err := set.RateForWithFallback("USD", 2026, 2); err == nil {
		t.Fatal("a fallback without an annual row was accepted")
	}
}

// TestFallbackRejectsAnUnknownName checks that a misspelt fallback is refused.
func TestFallbackRejectsAnUnknownName(t *testing.T) {
	set := fallbackSet("monthly")
	set.Locale = testLocale(t)
	if _, err := set.RateForWithFallback("USD", 2026, 1); err == nil {
		t.Fatal("an unknown fallback was accepted")
	}
}

// TestFallbackKeepsTheSystemCurrencyAtOne checks that the system currency is not
// affected by the fallback.
func TestFallbackKeepsTheSystemCurrencyAtOne(t *testing.T) {
	set := fallbackSet(currencyrate.FallbackAnnual)
	set.Locale = testLocale(t)
	rate, err := set.RateForWithFallback("JPY", 2026, 7)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 1 {
		t.Fatalf("rate=%v", rate)
	}
}
