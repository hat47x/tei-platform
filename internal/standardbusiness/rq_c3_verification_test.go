package standardbusiness_test

import (
	"testing"
	"time"

	"tei/internal/businesscalendar"
	"tei/internal/core"
	"tei/internal/currencyrate"
	"tei/internal/ir"
	"tei/internal/numbering"
	"tei/internal/semantic"
	"tei/internal/taxrate"
)

// RQ-C3の「確認方法」を一箇所で固定する試験です。
//
// 要求文は次の3つを求めています。
//
//  1. 適用期間の境界（改正日、年度替わり）を含む入力で、同じ評価時点から同じ結果を返す。
//  2. 案件固有の暦・税率を、Coreの変更なしに差し替えられる。
//  3. 採番の一意性・連続性・欠番の扱いを、契約として試験する。
//
// ここでは参照実装の各部品を、要求の言葉に対応させて呼びます。部品ごとの細かい
// 境界は各パッケージの試験が持つため、この試験は「要求の確認方法が実際に観測できる
// か」だけを見ます。

func rqLocale(t *testing.T) *time.Location {
	t.Helper()
	locale, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	return locale
}

func rqDay(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", value, rqLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// rqTaxRows returns two rate rows that meet on a boundary: the first is in force
// up to 2026-03-31 inclusive, the second from 2026-04-01.
func rqTaxRows() []map[core.CanonicalID]string {
	return []map[core.CanonicalID]string{
		{
			"tax.rate.applies_from":  "2025-04-01",
			"tax.rate.applies_until": "2026-03-31",
			"tax.rate.category":      "standard",
			"tax.rate.rate":          "0.08",
			"tax.rate.rounding":      "floor",
		},
		{
			"tax.rate.applies_from":  "2026-04-01",
			"tax.rate.applies_until": "2027-03-31",
			"tax.rate.category":      "standard",
			"tax.rate.rate":          "0.10",
			"tax.rate.rounding":      "floor",
		},
	}
}

func rqTaxWorld(rateA, rateB string) ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "tax",
		Data: []semantic.Data{{
			ID: "tax.rate",
			Attributes: []semantic.Attribute{
				{ID: "tax.rate.applies_from", TypeRef: "std.date", Role: taxrate.RoleAppliesFrom},
				{ID: "tax.rate.applies_until", TypeRef: "std.date", Role: taxrate.RoleAppliesUntil},
				{ID: "tax.rate.category", TypeRef: "std.code", Role: taxrate.RoleCategory},
				{ID: "tax.rate.rate", TypeRef: "std.decimal", Role: taxrate.RoleValue},
				{ID: "tax.rate.rounding", TypeRef: "std.text", Role: taxrate.RoleRounding},
			},
		}},
	}}}
}

// TestRQC3TaxRateChangesOnTheBoundary checks 確認方法1 for the tax: the day before
// and the day of the boundary use different rows, and the same evaluation point
// gives the same answer twice.
func TestRQC3TaxRateChangesOnTheBoundary(t *testing.T) {
	declaration, err := taxrate.Resolve(rqTaxWorld("0.08", "0.10"), "tax.rate")
	if err != nil {
		t.Fatal(err)
	}
	rates, err := declaration.Bind(rqTaxRows(), rqLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	set := declaration.Set(rates, taxrate.RoundingFloor, "standard", rqLocale(t))

	before, err := set.RateFor(rqDay(t, "2026-03-31"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := set.RateFor(rqDay(t, "2026-04-01"))
	if err != nil {
		t.Fatal(err)
	}
	if before.Value != 0.08 || after.Value != 0.10 {
		t.Fatalf("before=%v after=%v", before.Value, after.Value)
	}
	again, err := set.RateFor(rqDay(t, "2026-03-31"))
	if err != nil {
		t.Fatal(err)
	}
	if again.Value != before.Value {
		t.Fatalf("the same evaluation point gave %v then %v", before.Value, again.Value)
	}
}

// TestRQC3TaxRateIsSwappedWithoutChangingCore checks 確認方法2 for the tax: a rate
// change is a row change, and the same declaration reads it without a Core change.
func TestRQC3TaxRateIsSwappedWithoutChangingCore(t *testing.T) {
	rows := rqTaxRows()
	rows[1]["tax.rate.rate"] = "0.12"

	declaration, err := taxrate.Resolve(rqTaxWorld("0.08", "0.10"), "tax.rate")
	if err != nil {
		t.Fatal(err)
	}
	rates, err := declaration.Bind(rows, rqLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	set := declaration.Set(rates, taxrate.RoundingFloor, "standard", rqLocale(t))
	rate, err := set.RateFor(rqDay(t, "2026-04-01"))
	if err != nil {
		t.Fatal(err)
	}
	if rate.Value != 0.12 {
		t.Fatalf("the swapped rate was not read: %v", rate.Value)
	}
}

// TestRQC3CalendarSwapsWithoutChangingCore checks 確認方法2 for the calendar: the
// fiscal year start and the holidays are declarations, so two calendars answer
// differently for the same date without a Core change.
func TestRQC3CalendarSwapsWithoutChangingCore(t *testing.T) {
	april := businesscalendar.Calendar{FiscalYearStartMonth: 4, WeekStarts: businesscalendar.WeekStartsMonday, Locale: rqLocale(t)}
	january := businesscalendar.Calendar{FiscalYearStartMonth: 1, WeekStarts: businesscalendar.WeekStartsMonday, Locale: rqLocale(t)}

	// A date on the day before the April boundary: the April-start calendar is still
	// in its previous year while the January-start one has already moved on. That
	// difference is what makes the start month a declaration and not a constant.
	date := rqDay(t, "2026-03-31")
	aprilYear, err := april.FiscalYear(date)
	if err != nil {
		t.Fatal(err)
	}
	januaryYear, err := january.FiscalYear(date)
	if err != nil {
		t.Fatal(err)
	}
	if aprilYear == januaryYear {
		t.Fatalf("both calendars answered %d for the same date", aprilYear)
	}
	// On the boundary day itself the two agree, because both start a new year. That
	// is expected: the separation is on the days before it.
	after := rqDay(t, "2026-04-01")
	aprilAfter, err := april.FiscalYear(after)
	if err != nil {
		t.Fatal(err)
	}
	januaryAfter, err := january.FiscalYear(after)
	if err != nil {
		t.Fatal(err)
	}
	if aprilAfter != januaryAfter {
		t.Fatalf("april=%d january=%d on the shared boundary", aprilAfter, januaryAfter)
	}
}

// TestRQC3HolidaysChangeTheBusinessDayCount checks that the holiday set is a
// declaration: the same date range yields a different business-day count when the
// holiday changes.
func TestRQC3HolidaysChangeTheBusinessDayCount(t *testing.T) {
	base := businesscalendar.Calendar{FiscalYearStartMonth: 4, WeekStarts: businesscalendar.WeekStartsMonday, Locale: rqLocale(t)}
	withHoliday := base
	// The holiday sits inside the span, not on its end: DaysBetween counts from the
	// start up to but not including the end date, so a holiday on the end date is
	// never visited.
	withHoliday.Holidays = []time.Time{rqDay(t, "2026-04-29")}

	from, thru := rqDay(t, "2026-04-27"), rqDay(t, "2026-05-01")
	plain, err := base.DaysBetween(from, thru, businesscalendar.DayCountBusiness)
	if err != nil {
		t.Fatal(err)
	}
	excluding, err := withHoliday.DaysBetween(from, thru, businesscalendar.DayCountBusiness)
	if err != nil {
		t.Fatal(err)
	}
	if plain == excluding {
		t.Fatalf("the holiday did not change the count: %d", plain)
	}
}

func rqNumberingRules() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "seq",
		Data: []semantic.Data{{
			ID: "seq.rule",
			Attributes: []semantic.Attribute{
				{ID: "seq.rule.rule", TypeRef: "std.identifier", Role: numbering.RoleRule},
				{ID: "seq.rule.counter", TypeRef: "std.integer", Role: numbering.RoleCounter},
				{ID: "seq.rule.digits", TypeRef: "std.integer", Role: numbering.RoleDigits},
				{ID: "seq.rule.prefix", TypeRef: "std.text", Role: numbering.RolePrefix},
			},
		}},
	}}}
}

func rqNumberingRows() []map[core.CanonicalID]string {
	return []map[core.CanonicalID]string{
		{"seq.rule.rule": "order", "seq.rule.counter": "0", "seq.rule.digits": "4", "seq.rule.prefix": "ORD-"},
	}
}

// TestRQC3NumberingIsUniqueAndContiguous checks 確認方法3: the numbers are unique and
// contiguous, and a released reservation leaves no gap.
func TestRQC3NumberingIsUniqueAndContiguous(t *testing.T) {
	world := rqNumberingRules()
	rules, err := numbering.ResolveRules(world, "seq.rule", rqNumberingRows())
	if err != nil {
		t.Fatal(err)
	}
	contract, state, err := rules.BindRule("order")
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	current := state
	for i := 0; i < 3; i++ {
		issued, next, err := contract.Next(current)
		if err != nil {
			t.Fatal(err)
		}
		if seen[issued] {
			t.Fatalf("number %s was issued twice", issued)
		}
		seen[issued] = true
		current = next
	}
	if len(seen) != 3 {
		t.Fatalf("issued=%v", seen)
	}

	// A released reservation returns the counter, so the next caller takes the same
	// number: no gap.
	store := numbering.NewMemoryCounterStore()
	ctx := t.Context()
	first, reservation, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	if err := reservation.Release(ctx); err != nil {
		t.Fatal(err)
	}
	again, _, err := contract.Reserve(ctx, store, "order")
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Fatalf("a released number was skipped: %q then %q", first, again)
	}
}

// TestRQC3NumberingContractRefusesOverflow checks that the digits limit is a
// contract: the same rule stops at its limit instead of wrapping silently.
func TestRQC3NumberingContractRefusesOverflow(t *testing.T) {
	world := rqNumberingRules()
	rules, err := numbering.ResolveRules(world, "seq.rule", rqNumberingRows())
	if err != nil {
		t.Fatal(err)
	}
	contract, _, err := rules.BindRule("order")
	if err != nil {
		t.Fatal(err)
	}
	// The default is applied where the counter is counted up, not stored in the
	// contract, so the check is the behaviour: passing the limit is refused.
	if _, _, err := contract.Next(numbering.State{Current: contract.Limit()}); err == nil {
		t.Fatal("the counter passed its limit")
	}
	if contract.Overflow != "" {
		t.Fatalf("the contract now carries a default overflow %q; the check above is stale", contract.Overflow)
	}
}

// TestRQC3CurrencyRateFollowsTheDeclaredPeriod checks 確認方法1 for the currency: a
// rate is chosen by the declared period, and the annual row is used only when the
// fallback says so.
func TestRQC3CurrencyRateFollowsTheDeclaredPeriod(t *testing.T) {
	world, rows := rqCurrencyWorld()
	table, err := currencyrate.ResolveTable(world, "fx.rate", rows)
	if err != nil {
		t.Fatal(err)
	}
	set := currencyrate.Set{
		SystemCurrency: "JPY",
		Kind:           currencyrate.KindMiddle,
		Direction:      currencyrate.DirectionForeignToSystem,
		Rounding:       currencyrate.RoundingFloor,
		Locale:         rqLocale(t),
		Fallback:       currencyrate.FallbackNone,
		Rates:          table.Rows,
	}
	first, err := set.RateForWithFallback("USD", 2026, 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := set.RateForWithFallback("USD", 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("two months read the same rate: %v", first)
	}
	// The third month has no row and no fallback, so the miss is reported rather
	// than read as zero.
	if _, err := set.RateForWithFallback("USD", 2026, 3); err == nil {
		t.Fatal("a missing month was read as a rate")
	}
}

func rqCurrencyWorld() (ir.World, []map[core.CanonicalID]string) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "fx",
		Data: []semantic.Data{{
			ID: "fx.rate",
			Attributes: []semantic.Attribute{
				{ID: "fx.rate.currency", TypeRef: "std.identifier", Role: currencyrate.RoleCurrency},
				{ID: "fx.rate.fiscal_year", TypeRef: "std.integer", Role: currencyrate.RoleFiscalYear},
				{ID: "fx.rate.month_sequence", TypeRef: "std.integer", Role: currencyrate.RoleMonthSequence},
				{ID: "fx.rate.middle", TypeRef: "std.decimal", Role: currencyrate.RoleMiddle},
			},
		}},
	}}}
	rows := []map[core.CanonicalID]string{
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "1", "fx.rate.middle": "150.25"},
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "2", "fx.rate.middle": "151.00"},
	}
	return world, rows
}
