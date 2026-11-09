package numbering_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/numbering"
	"tei/internal/semantic"
)

func numberingWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "seq",
		Data: []semantic.Data{{
			ID: "seq.number",
			Attributes: []semantic.Attribute{
				{ID: "seq.number.counter", TypeRef: "std.integer", Role: numbering.RoleCounter},
				{ID: "seq.number.digits", TypeRef: "std.integer", Role: numbering.RoleDigits},
				{ID: "seq.number.prefix", TypeRef: "std.text", Role: numbering.RolePrefix},
				{ID: "seq.number.check", TypeRef: "std.code", Role: numbering.RoleCheck},
				{ID: "seq.number.overflow", TypeRef: "std.code", Role: numbering.RoleOverflow},
			},
		}},
	}}}
}

func numberingRow(counter, digits, prefix, check, overflow string) map[core.CanonicalID]string {
	row := map[core.CanonicalID]string{
		"seq.number.counter": counter,
		"seq.number.digits":  digits,
	}
	if prefix != "" {
		row["seq.number.prefix"] = prefix
	}
	if check != "" {
		row["seq.number.check"] = check
	}
	if overflow != "" {
		row["seq.number.overflow"] = overflow
	}
	return row
}

// TestResolveReadsTheDeclaredRoles checks that the binder takes the attributes
// from the declaration.
func TestResolveReadsTheDeclaredRoles(t *testing.T) {
	declaration, err := numbering.Resolve(numberingWorld(), "seq.number")
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Counter != "seq.number.counter" || declaration.Digits != "seq.number.digits" {
		t.Fatalf("declaration=%+v", declaration)
	}
	if declaration.Prefix != "seq.number.prefix" {
		t.Fatalf("prefix role lost: %+v", declaration)
	}
}

// TestResolveRefusesAMissingRequiredRole checks that the counter and the digit
// width cannot be omitted.
func TestResolveRefusesAMissingRequiredRole(t *testing.T) {
	world := numberingWorld()
	world.Semantics[0].Data[0].Attributes = world.Semantics[0].Data[0].Attributes[:1]
	if _, err := numbering.Resolve(world, "seq.number"); err == nil {
		t.Fatal("a numbering declaration without a digit width was accepted")
	}
	other := numberingWorld()
	other.Semantics[0].Data[0].ID = "seq.other"
	if _, err := numbering.Resolve(other, "seq.number"); err == nil {
		t.Fatal("an undeclared numbering data was accepted")
	}
}

// TestBindBuildsTheContractAndState checks that one row becomes a contract and
// the counter it holds, and that the contract issues the expected number.
func TestBindBuildsTheContractAndState(t *testing.T) {
	declaration, err := numbering.Resolve(numberingWorld(), "seq.number")
	if err != nil {
		t.Fatal(err)
	}
	contract, state, err := declaration.Bind(numberingRow("41", "5", "ORD-", "T", ""))
	if err != nil {
		t.Fatal(err)
	}
	if contract.Digits != 5 || contract.Prefix != "ORD-" || contract.CheckDigit != numbering.CheckDigitModulus10Weighted {
		t.Fatalf("contract=%+v", contract)
	}
	if state.Current != 41 {
		t.Fatalf("state=%+v", state)
	}
	value, next, err := contract.Next(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != len("ORD-")+5+1 {
		t.Fatalf("value=%q", value)
	}
	if next.Current != 42 {
		t.Fatalf("next=%+v", next)
	}
}

// TestBindDefaultsAreNotInventedFromEmptyColumns checks that an empty optional
// column keeps the contract default: no prefix, no check digit, stop at the limit.
func TestBindDefaultsAreNotInventedFromEmptyColumns(t *testing.T) {
	declaration, err := numbering.Resolve(numberingWorld(), "seq.number")
	if err != nil {
		t.Fatal(err)
	}
	contract, _, err := declaration.Bind(numberingRow("9", "1", "", "F", ""))
	if err != nil {
		t.Fatal(err)
	}
	if contract.Prefix != "" || contract.CheckDigit != numbering.CheckDigitNone {
		t.Fatalf("contract=%+v", contract)
	}
	if contract.Overflow != "" {
		t.Fatalf("overflow was invented: %q", contract.Overflow)
	}
	if _, _, err := contract.Next(numbering.State{Current: 9}); err == nil {
		t.Fatal("the default overflow did not stop at the limit")
	}
}

// TestBindReadsTheOverflowRule checks that a declared wrap is honoured.
func TestBindReadsTheOverflowRule(t *testing.T) {
	declaration, err := numbering.Resolve(numberingWorld(), "seq.number")
	if err != nil {
		t.Fatal(err)
	}
	contract, state, err := declaration.Bind(numberingRow("9", "1", "", "", "wrap"))
	if err != nil {
		t.Fatal(err)
	}
	value, next, err := contract.Next(state)
	if err != nil {
		t.Fatal(err)
	}
	if value != "1" || next.Current != 1 {
		t.Fatalf("value=%q next=%+v", value, next)
	}
}

// TestBindReportsUnparsableValues checks that bad rows name the attribute.
func TestBindReportsUnparsableValues(t *testing.T) {
	declaration, err := numbering.Resolve(numberingWorld(), "seq.number")
	if err != nil {
		t.Fatal(err)
	}
	cases := []map[core.CanonicalID]string{
		numberingRow("41", "", "", "", ""),
		numberingRow("forty-one", "5", "", "", ""),
		numberingRow("41", "0", "", "", ""),
		numberingRow("41", "5", "", "maybe", ""),
		numberingRow("-1", "5", "", "", ""),
	}
	for index, row := range cases {
		if _, _, err := declaration.Bind(row); err == nil {
			t.Fatalf("case %d was accepted", index)
		}
	}
}
