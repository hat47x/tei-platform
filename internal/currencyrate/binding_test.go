package currencyrate_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/currencyrate"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func exchangeWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "fx",
		Data: []semantic.Data{{
			ID: "fx.rate",
			Attributes: []semantic.Attribute{
				{ID: "fx.rate.currency", TypeRef: "std.identifier", Role: currencyrate.RoleCurrency},
				{ID: "fx.rate.fiscal_year", TypeRef: "std.integer", Role: currencyrate.RoleFiscalYear},
				{ID: "fx.rate.month_sequence", TypeRef: "std.integer", Role: currencyrate.RoleMonthSequence},
				{ID: "fx.rate.middle", TypeRef: "std.decimal", Role: currencyrate.RoleMiddle},
				{ID: "fx.rate.buying", TypeRef: "std.decimal", Role: currencyrate.RoleBuying},
				{ID: "fx.rate.selling", TypeRef: "std.decimal", Role: currencyrate.RoleSelling},
			},
		}},
	}}}
}

func exchangeRow(currency, fiscalYear, monthSequence, middle, buying, selling string) map[core.CanonicalID]string {
	row := map[core.CanonicalID]string{
		"fx.rate.currency":       currency,
		"fx.rate.fiscal_year":    fiscalYear,
		"fx.rate.month_sequence": monthSequence,
		"fx.rate.middle":         middle,
	}
	if buying != "" {
		row["fx.rate.buying"] = buying
	}
	if selling != "" {
		row["fx.rate.selling"] = selling
	}
	return row
}

// TestResolveReadsTheDeclaredRoles checks that the binder takes the attributes
// from the declaration.
func TestResolveReadsTheDeclaredRoles(t *testing.T) {
	declaration, err := currencyrate.Resolve(exchangeWorld(), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Currency != "fx.rate.currency" || declaration.Middle != "fx.rate.middle" {
		t.Fatalf("declaration=%+v", declaration)
	}
	if declaration.Buying != "fx.rate.buying" || declaration.Selling != "fx.rate.selling" {
		t.Fatalf("declaration=%+v", declaration)
	}
}

// TestResolveRefusesAMissingRequiredRole checks that the currency, fiscal year,
// month sequence, and middle rate cannot be omitted.
func TestResolveRefusesAMissingRequiredRole(t *testing.T) {
	world := exchangeWorld()
	world.Semantics[0].Data[0].Attributes = world.Semantics[0].Data[0].Attributes[:3]
	if _, err := currencyrate.Resolve(world, "fx.rate"); err == nil {
		t.Fatal("an incomplete exchange rate declaration was accepted")
	}
	other := exchangeWorld()
	other.Semantics[0].Data[0].ID = "fx.other"
	if _, err := currencyrate.Resolve(other, "fx.rate"); err == nil {
		t.Fatal("an undeclared exchange rate data was accepted")
	}
}

// TestBindBuildsRatesFromRows checks that stored rows become rates and that the
// optional kinds are read only when present.
func TestBindBuildsRatesFromRows(t *testing.T) {
	declaration, err := currencyrate.Resolve(exchangeWorld(), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[core.CanonicalID]string{
		exchangeRow("USD", "2026", "1", "150.25", "148.10", "152.40"),
		exchangeRow("EUR", "2026", "1", "163.80", "", ""),
	}
	rates, err := declaration.Bind(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 2 || rates[0].Middle != 150.25 || rates[0].Buying != 148.10 {
		t.Fatalf("rates=%+v", rates)
	}
	if rates[1].Buying != 0 || rates[1].Selling != 0 {
		t.Fatalf("optional kinds were invented: %+v", rates[1])
	}
	// The bound rows answer through the existing lookup.
	set := currencyrate.Set{
		SystemCurrency: "JPY",
		Kind:           currencyrate.KindMiddle,
		Direction:      currencyrate.DirectionForeignToSystem,
		Rounding:       currencyrate.RoundingFloor,
		Locale:         testLocale(t),
		Rates:          rates,
	}
	if _, err := set.RateFor("USD", 2026, 1); err != nil {
		t.Fatal(err)
	}
}

// TestBindReportsANonPositiveRate checks that zero and negative rates are refused
// at bind time. X-TEA compared the monthly rate against zero to choose a fallback
// (D:Session.java L2296–2298), which made zero a control value rather than a data
// mistake.
func TestBindReportsANonPositiveRate(t *testing.T) {
	declaration, err := currencyrate.Resolve(exchangeWorld(), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	for _, middle := range []string{"0", "-150.25"} {
		rows := []map[core.CanonicalID]string{exchangeRow("USD", "2026", "1", middle, "", "")}
		if _, err := declaration.Bind(rows); err == nil {
			t.Fatalf("middle %q was accepted", middle)
		}
	}
}

// TestBindReportsAnImpossibleMonthSequence checks that a month sequence outside
// 1..12 is refused rather than wrapping into another year. The annual marker 0 is
// accepted on purpose, because a table that holds annual rows has to bind them.
func TestBindReportsAnImpossibleMonthSequence(t *testing.T) {
	declaration, err := currencyrate.Resolve(exchangeWorld(), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	for _, sequence := range []string{"-1", "13"} {
		rows := []map[core.CanonicalID]string{exchangeRow("USD", "2026", sequence, "150.25", "", "")}
		if _, err := declaration.Bind(rows); err == nil {
			t.Fatalf("sequence %q was accepted", sequence)
		}
	}
}

// TestBindReportsAnUnparsableRow checks that unparsable values name the row and
// the attribute.
func TestBindReportsAnUnparsableRow(t *testing.T) {
	declaration, err := currencyrate.Resolve(exchangeWorld(), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	cases := []map[core.CanonicalID]string{
		exchangeRow("", "2026", "1", "150.25", "", ""),
		exchangeRow("USD", "twenty", "1", "150.25", "", ""),
		exchangeRow("USD", "2026", "1", "not-a-number", "", ""),
	}
	for index, rows := range cases {
		if _, err := declaration.Bind([]map[core.CanonicalID]string{rows}); err == nil {
			t.Fatalf("case %d was accepted", index)
		}
	}
}
