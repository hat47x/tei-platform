package currencyrate_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/currencyrate"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func fallbackWorld(role bool) ir.World {
	attributes := []semantic.Attribute{
		{ID: "fx.rate.currency", TypeRef: "std.identifier", Role: currencyrate.RoleCurrency},
		{ID: "fx.rate.fiscal_year", TypeRef: "std.integer", Role: currencyrate.RoleFiscalYear},
		{ID: "fx.rate.month_sequence", TypeRef: "std.integer", Role: currencyrate.RoleMonthSequence},
		{ID: "fx.rate.middle", TypeRef: "std.decimal", Role: currencyrate.RoleMiddle},
	}
	if role {
		attributes = append(attributes, semantic.Attribute{ID: "fx.rate.fallback", TypeRef: "std.code", Role: currencyrate.RoleFallback})
	}
	return ir.World{Semantics: []semantic.Namespace{{
		ID:   "fx",
		Data: []semantic.Data{{ID: "fx.rate", Attributes: attributes}},
	}}}
}

func fallbackRows(rule string) []map[core.CanonicalID]string {
	rows := []map[core.CanonicalID]string{
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "1", "fx.rate.middle": "150.25"},
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "0", "fx.rate.middle": "149.00"},
	}
	if rule != "" {
		for _, row := range rows {
			row["fx.rate.fallback"] = rule
		}
	}
	return rows
}

// TestResolveFallbackReadsTheOptionalRole checks that the fallback role is
// optional and that its presence is reported.
func TestResolveFallbackReadsTheOptionalRole(t *testing.T) {
	declaration, err := currencyrate.ResolveFallback(fallbackWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Attribute != "fx.rate.fallback" {
		t.Fatalf("declaration=%+v", declaration)
	}
	absent, err := currencyrate.ResolveFallback(fallbackWorld(false), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	if absent.Attribute != "" {
		t.Fatalf("declaration=%+v", absent)
	}
	if _, err := currencyrate.ResolveFallback(fallbackWorld(false), "fx.missing"); err == nil {
		t.Fatal("an undeclared data was accepted")
	}
}

// TestFallbackRowsReadsTheRule checks that a declared rule is read, and that a Data
// without the role keeps the default.
func TestFallbackRowsReadsTheRule(t *testing.T) {
	declaration, err := currencyrate.ResolveFallback(fallbackWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	rule, err := declaration.FallbackRows(fallbackRows("annual"))
	if err != nil {
		t.Fatal(err)
	}
	if rule != currencyrate.FallbackAnnual {
		t.Fatalf("rule=%q", rule)
	}
	// A Data that does not declare the role keeps the default.
	absent, err := currencyrate.ResolveFallback(fallbackWorld(false), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	defaulted, err := absent.FallbackRows(fallbackRows(""))
	if err != nil {
		t.Fatal(err)
	}
	if defaulted != currencyrate.FallbackNone {
		t.Fatalf("defaulted=%q", defaulted)
	}
}

// TestFallbackRowsRejectsAnUnknownRule checks that a misspelt rule is refused,
// so a declaration cannot ask for a fallback the code does not implement.
func TestFallbackRowsRejectsAnUnknownRule(t *testing.T) {
	declaration, err := currencyrate.ResolveFallback(fallbackWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := declaration.FallbackRows(fallbackRows("monthly")); err == nil {
		t.Fatal("an unknown fallback was accepted")
	}
}

// TestFallbackRowsRejectsDisagreeingRows checks that one Data cannot carry two
// rules, because then the answer would depend on row order.
func TestFallbackRowsRejectsDisagreeingRows(t *testing.T) {
	declaration, err := currencyrate.ResolveFallback(fallbackWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	rows := fallbackRows("annual")
	rows[1]["fx.rate.fallback"] = "none"
	if _, err := declaration.FallbackRows(rows); err == nil {
		t.Fatal("disagreeing fallback rows were accepted")
	}
}

// TestFallbackRowsRequiresAValueWhenTheRoleIsDeclared checks that a declared role
// with an empty value is refused.
func TestFallbackRowsRequiresAValueWhenTheRoleIsDeclared(t *testing.T) {
	declaration, err := currencyrate.ResolveFallback(fallbackWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	rows := fallbackRows("")
	rows[0]["fx.rate.fallback"] = ""
	if _, err := declaration.FallbackRows(rows); err == nil {
		t.Fatal("an empty fallback value was accepted")
	}
}
