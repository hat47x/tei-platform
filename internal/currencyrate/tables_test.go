package currencyrate_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/currencyrate"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func twoDataWorld() ir.World {
	attributes := []semantic.Attribute{
		{ID: "fx.rate.currency", TypeRef: "std.identifier", Role: currencyrate.RoleCurrency},
		{ID: "fx.rate.fiscal_year", TypeRef: "std.integer", Role: currencyrate.RoleFiscalYear},
		{ID: "fx.rate.month_sequence", TypeRef: "std.integer", Role: currencyrate.RoleMonthSequence},
		{ID: "fx.rate.middle", TypeRef: "std.decimal", Role: currencyrate.RoleMiddle},
	}
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "fx",
		Data: []semantic.Data{
			{ID: "fx.monthly", Attributes: attributes},
			{ID: "fx.annual", Attributes: attributes},
		},
	}}}
}

func monthlyRows() []map[core.CanonicalID]string {
	return []map[core.CanonicalID]string{
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "1", "fx.rate.middle": "150.25"},
	}
}

func annualRows() []map[core.CanonicalID]string {
	return []map[core.CanonicalID]string{
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "0", "fx.rate.middle": "149.00"},
	}
}

func twoDataPair(t *testing.T, fallback currencyrate.Fallback) currencyrate.Pair {
	t.Helper()
	world := twoDataWorld()
	monthly, err := currencyrate.ResolveTable(world, "fx.monthly", monthlyRows())
	if err != nil {
		t.Fatal(err)
	}
	annual, err := currencyrate.ResolveTable(world, "fx.annual", annualRows())
	if err != nil {
		t.Fatal(err)
	}
	pair, err := currencyrate.NewPair(monthly, annual, fallback, "JPY",
		currencyrate.KindMiddle, currencyrate.DirectionForeignToSystem, currencyrate.RoundingFloor, testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	return pair
}

// TestResolveTableReadsTheGranularity checks that a table's granularity comes from
// its rows, so a monthly table and an annual table are told apart by what they
// store rather than by a name.
func TestResolveTableReadsTheGranularity(t *testing.T) {
	world := twoDataWorld()
	monthly, err := currencyrate.ResolveTable(world, "fx.monthly", monthlyRows())
	if err != nil {
		t.Fatal(err)
	}
	if monthly.Granularity != currencyrate.GranularityMonthly {
		t.Fatalf("granularity=%s", monthly.Granularity)
	}
	annual, err := currencyrate.ResolveTable(world, "fx.annual", annualRows())
	if err != nil {
		t.Fatal(err)
	}
	if annual.Granularity != currencyrate.GranularityAnnual {
		t.Fatalf("granularity=%s", annual.Granularity)
	}
}

// TestResolveTableRefusesMixedRows checks that one table cannot hold both periods,
// because a year would then refer to a row the caller has to guess.
func TestResolveTableRefusesMixedRows(t *testing.T) {
	rows := append(monthlyRows(), annualRows()...)
	if _, err := currencyrate.ResolveTable(twoDataWorld(), "fx.monthly", rows); err == nil {
		t.Fatal("a table mixing annual and monthly rows was accepted")
	}
	if _, err := currencyrate.ResolveTable(twoDataWorld(), "fx.monthly", nil); err == nil {
		t.Fatal("an empty table was accepted")
	}
}

// TestNewPairRefusesTheWrongOrder checks that the monthly and annual tables cannot be
// swapped: the fallback means "a missing month uses the year".
func TestNewPairRefusesTheWrongOrder(t *testing.T) {
	world := twoDataWorld()
	monthly, err := currencyrate.ResolveTable(world, "fx.monthly", monthlyRows())
	if err != nil {
		t.Fatal(err)
	}
	annual, err := currencyrate.ResolveTable(world, "fx.annual", annualRows())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := currencyrate.NewPair(annual, monthly, currencyrate.FallbackAnnual, "JPY",
		currencyrate.KindMiddle, currencyrate.DirectionForeignToSystem, currencyrate.RoundingFloor, testLocale(t)); err == nil {
		t.Fatal("the tables were accepted in the wrong order")
	}
}

// TestPairRateForUsesTheMonthlyRowFirst checks that a monthly row wins and a missing
// month uses the annual row when the fallback asks for it.
func TestPairRateForUsesTheMonthlyRowFirst(t *testing.T) {
	pair := twoDataPair(t, currencyrate.FallbackAnnual)
	monthly, err := pair.RateFor("USD", 2026, 1)
	if err != nil || monthly != 150.25 {
		t.Fatalf("monthly=%v err=%v", monthly, err)
	}
	annual, err := pair.RateFor("USD", 2026, 2)
	if err != nil || annual != 149.00 {
		t.Fatalf("annual=%v err=%v", annual, err)
	}
}

// TestPairRateForReportsAMissingMonthWithoutFallback checks that a pair declared
// without a fallback reports the missing month instead of reading a zero rate, which
// is what X-TEA's zero comparison could not tell apart
// (D:Session.java L2296–2298).
func TestPairRateForReportsAMissingMonthWithoutFallback(t *testing.T) {
	pair := twoDataPair(t, currencyrate.FallbackNone)
	if _, err := pair.RateFor("USD", 2026, 2); err == nil {
		t.Fatal("a missing month was filled without a declared fallback")
	}
}

// TestPairRateForKeepsTheSystemCurrencyAtOne checks that the system currency needs no
// row in either table.
func TestPairRateForKeepsTheSystemCurrencyAtOne(t *testing.T) {
	pair := twoDataPair(t, currencyrate.FallbackNone)
	rate, err := pair.RateFor("JPY", 2026, 5)
	if err != nil || rate != 1 {
		t.Fatalf("rate=%v err=%v", rate, err)
	}
}

// TestPairRateForReportsAMissingAnnualRow checks that a fallback which cannot answer
// is reported rather than returning zero.
func TestPairRateForReportsAMissingAnnualRow(t *testing.T) {
	world := twoDataWorld()
	monthly, err := currencyrate.ResolveTable(world, "fx.monthly", monthlyRows())
	if err != nil {
		t.Fatal(err)
	}
	annual, err := currencyrate.ResolveTable(world, "fx.annual", []map[core.CanonicalID]string{
		{"fx.rate.currency": "EUR", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "0", "fx.rate.middle": "162.00"},
	})
	if err != nil {
		t.Fatal(err)
	}
	pair, err := currencyrate.NewPair(monthly, annual, currencyrate.FallbackAnnual, "JPY",
		currencyrate.KindMiddle, currencyrate.DirectionForeignToSystem, currencyrate.RoundingFloor, testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pair.RateFor("USD", 2026, 2); err == nil {
		t.Fatal("a missing annual row was filled")
	}
}
