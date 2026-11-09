package taxrate_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/taxrate"
)

func taxRateWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "tax",
		Data: []semantic.Data{{
			ID: "tax.rate",
			Attributes: []semantic.Attribute{
				{ID: "tax.rate.applies_from", TypeRef: "std.date", Role: taxrate.RoleAppliesFrom},
				{ID: "tax.rate.applies_until", TypeRef: "std.date", Role: taxrate.RoleAppliesUntil},
				{ID: "tax.rate.category", TypeRef: "tax.category", Role: taxrate.RoleCategory},
				{ID: "tax.rate.value", TypeRef: "std.decimal", Role: taxrate.RoleValue},
				{ID: "tax.rate.rounding", TypeRef: "std.code", Role: taxrate.RoleRounding},
			},
		}},
	}}}
}

func taxRateRow(t *testing.T, from, until, category, value, rounding string) map[core.CanonicalID]string {
	t.Helper()
	row := map[core.CanonicalID]string{
		"tax.rate.applies_from": from,
		"tax.rate.value":        value,
	}
	if until != "" {
		row["tax.rate.applies_until"] = until
	}
	if category != "" {
		row["tax.rate.category"] = category
	}
	if rounding != "" {
		row["tax.rate.rounding"] = rounding
	}
	return row
}

// TestResolveReadsTheDeclaredRoles checks that the binder takes the attributes
// from the declaration, not from column names.
func TestResolveReadsTheDeclaredRoles(t *testing.T) {
	declaration, err := taxrate.Resolve(taxRateWorld(), "tax.rate")
	if err != nil {
		t.Fatal(err)
	}
	if declaration.AppliesFrom != "tax.rate.applies_from" ||
		declaration.AppliesUntil != "tax.rate.applies_until" ||
		declaration.Category != "tax.rate.category" ||
		declaration.Value != "tax.rate.value" ||
		declaration.Rounding != "tax.rate.rounding" {
		t.Fatalf("declaration=%+v", declaration)
	}
}

// TestResolveRefusesAnIncompleteDeclaration checks that a rate Data missing a
// required role is reported with the role name, so the fix is the declaration.
func TestResolveRefusesAnIncompleteDeclaration(t *testing.T) {
	world := taxRateWorld()
	world.Semantics[0].Data[0].Attributes = world.Semantics[0].Data[0].Attributes[:2]
	if _, err := taxrate.Resolve(world, "tax.rate"); err == nil {
		t.Fatal("an incomplete rate declaration was accepted")
	}
	other := taxRateWorld()
	other.Semantics[0].Data[0].ID = "tax.other"
	if _, err := taxrate.Resolve(other, "tax.rate"); err == nil {
		t.Fatal("an undeclared rate data was accepted")
	}
}

// TestBindBuildsRatesFromRows checks that stored rows become rates with the
// declared span, and that the row's own rounding is kept.
func TestBindBuildsRatesFromRows(t *testing.T) {
	declaration, err := taxrate.Resolve(taxRateWorld(), "tax.rate")
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[core.CanonicalID]string{
		taxRateRow(t, "2023-10-01", "2026-09-30", "standard", "0.08", ""),
		taxRateRow(t, "2026-10-01", "", "standard", "0.05", "half-up"),
	}
	rates, err := declaration.Bind(rows, testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 2 {
		t.Fatalf("rates=%+v", rates)
	}
	if !rates[1].AppliesUntil.IsZero() {
		t.Fatalf("open end=%v", rates[1].AppliesUntil)
	}
	if rates[1].Rounding != taxrate.RoundingHalfUp {
		t.Fatalf("row rounding=%q", rates[1].Rounding)
	}
	set := declaration.Set(rates, taxrate.RoundingFloor, "standard", testLocale(t))
	if set.Rates[0].Rounding != taxrate.RoundingFloor {
		t.Fatalf("default rounding=%q", set.Rates[0].Rounding)
	}
	if set.Rates[1].RateRounding(taxrate.RoundingFloor) != taxrate.RoundingHalfUp {
		t.Fatalf("row rounding was overridden")
	}
	// The bound set answers the declared span.
	rate, err := set.RateFor(day(t, "2026-10-01"))
	if err != nil {
		t.Fatal(err)
	}
	if rate.Value != 0.05 {
		t.Fatalf("value=%v", rate.Value)
	}
}

// TestBindReportsAnUnparsableRow checks that a bad value names the row and the
// attribute instead of being skipped.
func TestBindReportsAnUnparsableRow(t *testing.T) {
	declaration, err := taxrate.Resolve(taxRateWorld(), "tax.rate")
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[core.CanonicalID]string{
		taxRateRow(t, "2023-10-01", "", "", "not-a-number", ""),
	}
	if _, err := declaration.Bind(rows, testLocale(t)); err == nil {
		t.Fatal("an unparsable rate was accepted")
	}
	badDate := []map[core.CanonicalID]string{
		taxRateRow(t, "01/10/2023", "", "", "0.08", ""),
	}
	if _, err := declaration.Bind(badDate, testLocale(t)); err == nil {
		t.Fatal("an unparsable date was accepted")
	}
}

// TestBindReportsAnUnknownRounding checks that a row naming a rounding this
// package does not implement is refused, not treated as the default.
func TestBindReportsAnUnknownRounding(t *testing.T) {
	declaration, err := taxrate.Resolve(taxRateWorld(), "tax.rate")
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[core.CanonicalID]string{
		taxRateRow(t, "2023-10-01", "", "", "0.08", "half-away-from-zero"),
	}
	if _, err := declaration.Bind(rows, testLocale(t)); err == nil {
		t.Fatal("an unknown rounding was accepted")
	}
}

// TestBindRequiresALocale checks that a bind without a zone is refused, so the
// dates are not read in the host's zone.
func TestBindRequiresALocale(t *testing.T) {
	declaration, err := taxrate.Resolve(taxRateWorld(), "tax.rate")
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[core.CanonicalID]string{taxRateRow(t, "2023-10-01", "", "", "0.08", "")}
	if _, err := declaration.Bind(rows, nil); err == nil {
		t.Fatal("a bind without a zone was accepted")
	}
}
