package currencyrate_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/currencyrate"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func granularityWorld(role bool) ir.World {
	attributes := []semantic.Attribute{
		{ID: "fx.rate.currency", TypeRef: "std.identifier", Role: currencyrate.RoleCurrency},
		{ID: "fx.rate.fiscal_year", TypeRef: "std.integer", Role: currencyrate.RoleFiscalYear},
		{ID: "fx.rate.month_sequence", TypeRef: "std.integer", Role: currencyrate.RoleMonthSequence},
		{ID: "fx.rate.middle", TypeRef: "std.decimal", Role: currencyrate.RoleMiddle},
	}
	if role {
		attributes = append(attributes, semantic.Attribute{ID: "fx.rate.granularity", TypeRef: "std.code", Role: currencyrate.RoleGranularity})
	}
	return ir.World{Semantics: []semantic.Namespace{{
		ID:   "fx",
		Data: []semantic.Data{{ID: "fx.rate", Attributes: attributes}},
	}}}
}

func granularityRows(value string) []map[core.CanonicalID]string {
	rows := []map[core.CanonicalID]string{
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "1", "fx.rate.middle": "150.25"},
		{"fx.rate.currency": "USD", "fx.rate.fiscal_year": "2026", "fx.rate.month_sequence": "2", "fx.rate.middle": "151.00"},
	}
	if value != "" {
		for _, row := range rows {
			row["fx.rate.granularity"] = value
		}
	}
	return rows
}

// TestResolveGranularityReadsTheOptionalRole checks that the granularity role is
// optional and that its presence is reported.
func TestResolveGranularityReadsTheOptionalRole(t *testing.T) {
	declaration, err := currencyrate.ResolveGranularity(granularityWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Attribute != "fx.rate.granularity" {
		t.Fatalf("declaration=%+v", declaration)
	}
	absent, err := currencyrate.ResolveGranularity(granularityWorld(false), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	if absent.Attribute != "" {
		t.Fatalf("declaration=%+v", absent)
	}
	if _, err := currencyrate.ResolveGranularity(granularityWorld(false), "fx.missing"); err == nil {
		t.Fatal("an undeclared data was accepted")
	}
}

// TestGranularityRowsReadsTheDeclaredValue checks that a declared granularity is
// read, and that a Data without the role reports none.
func TestGranularityRowsReadsTheDeclaredValue(t *testing.T) {
	declaration, err := currencyrate.ResolveGranularity(granularityWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	granularity, declared, err := declaration.GranularityRows(granularityRows("monthly"))
	if err != nil {
		t.Fatal(err)
	}
	if !declared || granularity != currencyrate.GranularityMonthly {
		t.Fatalf("granularity=%q declared=%v", granularity, declared)
	}
	absent, err := currencyrate.ResolveGranularity(granularityWorld(false), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	_, declared, err = absent.GranularityRows(granularityRows(""))
	if err != nil {
		t.Fatal(err)
	}
	if declared {
		t.Fatal("a data without the role reported a declaration")
	}
}

// TestGranularityRowsRejectsAnUnknownOrDisagreeingValue checks that a misspelt
// granularity and two values in one table are refused.
func TestGranularityRowsRejectsAnUnknownOrDisagreeingValue(t *testing.T) {
	declaration, err := currencyrate.ResolveGranularity(granularityWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := declaration.GranularityRows(granularityRows("weekly")); err == nil {
		t.Fatal("an unknown granularity was accepted")
	}
	rows := granularityRows("monthly")
	rows[1]["fx.rate.granularity"] = "annual"
	if _, _, err := declaration.GranularityRows(rows); err == nil {
		t.Fatal("disagreeing rows were accepted")
	}
	if _, _, err := declaration.GranularityRows(granularityRows("")); err == nil {
		t.Fatal("an empty value was accepted when the role is declared")
	}
}

// TestCheckDeclaredComparesTheDeclarationWithTheRows checks that a declaration the
// rows contradict is reported. This is the case neither side shows alone: a table
// declared annual whose rows use months 1..12 would be read two ways.
func TestCheckDeclaredComparesTheDeclarationWithTheRows(t *testing.T) {
	if err := currencyrate.CheckDeclared("", currencyrate.GranularityMonthly); err != nil {
		t.Fatalf("an absent declaration was refused: %v", err)
	}
	if err := currencyrate.CheckDeclared(currencyrate.GranularityMonthly, currencyrate.GranularityMonthly); err != nil {
		t.Fatalf("a matching declaration was refused: %v", err)
	}
	if err := currencyrate.CheckDeclared(currencyrate.GranularityAnnual, currencyrate.GranularityMonthly); err == nil {
		t.Fatal("a contradicting declaration was accepted")
	}
}

// TestResolveTableReconcilesTheDeclaration checks that ResolveTable compares the
// declared granularity with the one the rows imply, and refuses a table that
// contradicts itself.
func TestResolveTableReconcilesTheDeclaration(t *testing.T) {
	world := granularityWorld(true)
	if _, err := currencyrate.ResolveTable(world, "fx.rate", granularityRows("monthly")); err != nil {
		t.Fatalf("a matching declaration was refused: %v", err)
	}
	if _, err := currencyrate.ResolveTable(world, "fx.rate", granularityRows("annual")); err == nil {
		t.Fatal("a declaration the rows contradict was accepted")
	}
	// The granularity value lives in the rows, so an empty table has none: the role
	// names the column, not the value. An empty table is refused either way.
	if _, err := currencyrate.ResolveTable(world, "fx.rate", nil); err == nil {
		t.Fatal("an empty table was accepted")
	}
}

// TestDeclaredGranularityMakesAnEmptyTableReadable checks the gap the role closes: a
// table with no rows has no implied granularity, so only the declaration can say
// what it holds.
func TestDeclaredGranularityMakesAnEmptyTableReadable(t *testing.T) {
	declaration, err := currencyrate.ResolveGranularity(granularityWorld(true), "fx.rate")
	if err != nil {
		t.Fatal(err)
	}
	_, declared, err := declaration.GranularityRows(nil)
	if err != nil {
		t.Fatal(err)
	}
	if declared {
		t.Fatal("an empty table with the role reported a declaration from no rows")
	}
	// The declaration is still known from the role, which is what makes a table
	// describable before it holds rows.
	if declaration.Attribute == "" {
		t.Fatal("the role was lost for an empty table")
	}
}
