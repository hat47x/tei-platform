package main

import (
	"path/filepath"
	"testing"
	"time"

	"tei/internal/businesscalendar"
	"tei/internal/core"
	"tei/internal/currencyrate"
	"tei/internal/numbering"
	"tei/internal/taxrate"
	"tei/internal/usersetting"
	"tei/internal/workspace"
)

// TestXTEAExampleDeclaresStandardBusinessRoles checks that the example declares
// the standard business data by role, and that each binder reads it. A binder
// that cannot find its roles means the declaration and the example have drifted
// apart.
func TestXTEAExampleDeclaresStandardBusinessRoles(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "..", "examples", "xtea-lowcode"))
	if err != nil {
		t.Fatal(err)
	}
	locale, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	tax, err := taxrate.Resolve(loaded.World, "master.tax_rate")
	if err != nil {
		t.Fatalf("tax rate declaration: %v", err)
	}
	if tax.Value != "master.tax_rate.rate" || tax.AppliesFrom != "master.tax_rate.applies_from" {
		t.Fatalf("tax declaration=%+v", tax)
	}

	exchange, err := currencyrate.Resolve(loaded.World, "master.exchange_rate")
	if err != nil {
		t.Fatalf("exchange rate declaration: %v", err)
	}
	if exchange.Middle != "master.exchange_rate.middle" || exchange.MonthSequence != "master.exchange_rate.month_sequence" {
		t.Fatalf("exchange declaration=%+v", exchange)
	}

	sequence, err := numbering.Resolve(loaded.World, "master.numbering_rule")
	if err != nil {
		t.Fatalf("numbering declaration: %v", err)
	}
	if sequence.Counter != "master.numbering_rule.counter" || sequence.Digits != "master.numbering_rule.digits" {
		t.Fatalf("numbering declaration=%+v", sequence)
	}

	// The example declares a setting table with the user role, so a caller can
	// change a declared key and read the user's own row back.
	settingDeclaration, err := usersetting.Resolve(loaded.World, "master.setting")
	if err != nil {
		t.Fatalf("setting declaration: %v", err)
	}
	if settingDeclaration.Key != "master.setting.key" || settingDeclaration.User != "master.setting.user" {
		t.Fatalf("setting declaration=%+v", settingDeclaration)
	}
	settingRows, err := settingDeclaration.Bind([]map[core.CanonicalID]string{
		{"master.setting.key": "page_size", "master.setting.value": "50", "master.setting.user": ""},
	})
	if err != nil {
		t.Fatalf("setting bind: %v", err)
	}
	settingTable, err := usersetting.NewTable(settingDeclaration, settingRows)
	if err != nil {
		t.Fatalf("setting table: %v", err)
	}
	updated, err := settingTable.Change(usersetting.Change{Key: "page_size", User: "alice", Value: "200"})
	if err != nil {
		t.Fatalf("setting change: %v", err)
	}
	settingIndex, err := usersetting.NewSettings(true, updated)
	if err != nil {
		t.Fatalf("setting index: %v", err)
	}
	if value, err := settingIndex.Value("page_size", "alice"); err != nil || value != "200" {
		t.Fatalf("alice setting=%q err=%v", value, err)
	}
	if value, err := settingIndex.Value("page_size", "bob"); err != nil || value != "50" {
		t.Fatalf("shared setting=%q err=%v", value, err)
	}
	emptyRows, err := settingDeclaration.Bind(nil)
	if err != nil || len(emptyRows) != 0 {
		t.Fatalf("empty setting rows were refused: %v", err)
	}

	calendar, err := businesscalendar.Resolve(loaded.World, "sales.fiscal")
	if err != nil {
		t.Fatalf("calendar declaration: %v", err)
	}
	holiday, err := calendar.BindHolidays(loaded.World)
	if err != nil {
		t.Fatalf("holiday binding: %v", err)
	}
	if holiday.DataID != "master.holiday" || holiday.Date != "master.holiday.holiday_on" {
		t.Fatalf("holiday binding=%+v", holiday)
	}

	// The binders turn example rows into values, which is what a Host would do
	// with stored data.
	rates, err := tax.Bind([]map[core.CanonicalID]string{
		{
			"master.tax_rate.applies_from":  "2026-04-01",
			"master.tax_rate.applies_until": "2027-03-31",
			"master.tax_rate.category":      "standard",
			"master.tax_rate.rate":          "0.10",
			"master.tax_rate.rounding":      "floor",
		},
	}, locale)
	if err != nil {
		t.Fatalf("tax bind: %v", err)
	}
	if len(rates) != 1 || rates[0].Value != 0.10 {
		t.Fatalf("rates=%+v", rates)
	}

	exchangeRates, err := exchange.Bind([]map[core.CanonicalID]string{
		{
			"master.exchange_rate.currency":       "USD",
			"master.exchange_rate.fiscal_year":    "2027",
			"master.exchange_rate.month_sequence": "1",
			"master.exchange_rate.middle":         "150.25",
			"master.exchange_rate.buying":         "148.10",
			"master.exchange_rate.selling":        "152.40",
		},
	})
	if err != nil {
		t.Fatalf("exchange bind: %v", err)
	}
	if len(exchangeRates) != 1 || exchangeRates[0].Middle != 150.25 {
		t.Fatalf("exchange rates=%+v", exchangeRates)
	}

	// The example declares the rule identity, so one Data holds several rules
	// and a caller names the one it issues from.
	binding, err := sequence.BindRule(loaded.World)
	if err != nil {
		t.Fatalf("rule binding: %v", err)
	}
	if binding.Attribute != "master.numbering_rule.rule" {
		t.Fatalf("rule binding=%+v", binding)
	}
	rows := []map[core.CanonicalID]string{
		{
			"master.numbering_rule.rule":     "order",
			"master.numbering_rule.counter":  "41",
			"master.numbering_rule.digits":   "5",
			"master.numbering_rule.prefix":   "ORD-",
			"master.numbering_rule.check":    "T",
			"master.numbering_rule.overflow": "stop",
		},
		{
			"master.numbering_rule.rule":     "invoice",
			"master.numbering_rule.counter":  "7",
			"master.numbering_rule.digits":   "3",
			"master.numbering_rule.prefix":   "INV-",
			"master.numbering_rule.check":    "F",
			"master.numbering_rule.overflow": "stop",
		},
	}
	rules, err := numbering.ResolveRules(loaded.World, "master.numbering_rule", rows)
	if err != nil {
		t.Fatalf("resolve rules: %v", err)
	}
	contract, state, err := rules.BindRule("order")
	if err != nil {
		t.Fatalf("numbering bind: %v", err)
	}
	value, next, err := contract.Next(state)
	if err != nil {
		t.Fatalf("numbering next: %v", err)
	}
	if len(value) != len("ORD-")+5+1 || next.Current != 42 {
		t.Fatalf("value=%q next=%+v", value, next)
	}

	holidays, err := holiday.HolidayRows([]map[core.CanonicalID]string{
		{"master.holiday.holiday_on": "2026-05-02"},
	}, locale)
	if err != nil {
		t.Fatalf("holiday rows: %v", err)
	}
	built, err := calendar.Build(holidays, locale)
	if err != nil {
		t.Fatalf("calendar build: %v", err)
	}
	fiscalYear, err := built.FiscalYear(time.Date(2026, 3, 31, 0, 0, 0, 0, locale))
	if err != nil {
		t.Fatalf("fiscal year: %v", err)
	}
	if fiscalYear != 2026 {
		t.Fatalf("fiscal year=%d", fiscalYear)
	}
}
