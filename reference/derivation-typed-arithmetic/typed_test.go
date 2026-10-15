package typedarithmetic

import (
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/derivationref"
	"tei/internal/stdlib"
)

func fixture() ([]semantic.Type, Operand, Operand, AmountTarget) {
	types := append([]semantic.Type(nil), stdlib.SemanticNamespace().Types...)
	types = append(types,
		semantic.Type{ID: core.MustCanonicalID("sales.quantity_type"), Kind: semantic.TypeAlias, Underlying: core.MustCanonicalID("std.decimal")},
		semantic.Type{ID: core.MustCanonicalID("sales.price_type"), Kind: semantic.TypeAlias, Underlying: core.MustCanonicalID("std.money")},
	)
	quantity := Operand{
		Attribute: semantic.Attribute{ID: core.MustCanonicalID("sales.item.quantity"), TypeRef: core.MustCanonicalID("sales.quantity_type")},
		Value:     derivationref.Input{State: derivationref.Present, Decimal: "2"},
		Unit:      "piece",
	}
	price := Operand{
		Attribute: semantic.Attribute{ID: core.MustCanonicalID("sales.item.unit_price"), TypeRef: core.MustCanonicalID("sales.price_type")},
		Value:     derivationref.Input{State: derivationref.Present, Decimal: "300"},
		PerUnit:   "piece",
		Currency:  "JPY",
	}
	target := AmountTarget{
		Attribute: semantic.Attribute{ID: core.MustCanonicalID("sales.item.amount"), TypeRef: core.MustCanonicalID("std.money")},
		Currency:  "JPY",
	}
	return types, quantity, price, target
}

func TestMultiplyAmountExactAndFinalRounding(t *testing.T) {
	types, qty, price, target := fixture()
	spec := DecimalSpec{Precision: 8, Scale: 2, Mode: HalfUp}
	for _, tc := range []struct{ quantity, unitPrice, want string }{
		{"2", "300", "600.00"},
		{"0", "300", "0.00"},
		{"1.25", "0.4", "0.50"},
		{"1", "1.005", "1.01"},
		{"-1", "1.005", "-1.01"},
	} {
		qty.Value.Decimal, price.Value.Decimal = tc.quantity, tc.unitPrice
		got, err := MultiplyAmount(types, qty, price, target, spec)
		if err != nil || got.State != derivationref.Present || got.Decimal != tc.want {
			t.Fatalf("%s * %s = %+v, err=%v; want %s", tc.quantity, tc.unitPrice, got, err, tc.want)
		}
	}
	spec.Mode = HalfEven
	qty.Value.Decimal, price.Value.Decimal = "1", "1.005"
	got, err := MultiplyAmount(types, qty, price, target, spec)
	if err != nil || got.Decimal != "1.00" {
		t.Fatalf("half even: %+v %v", got, err)
	}
}

func TestRoundExactTieNegativeCarryAndPrecision(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		mode RoundingMode
		want string
	}{
		{"1.005", HalfUp, "1.01"},
		{"1.005", HalfEven, "1.00"},
		{"1.015", HalfEven, "1.02"},
		{"-1.005", HalfUp, "-1.01"},
		{"-1.005", HalfEven, "-1.00"},
		{"-1.015", HalfEven, "-1.02"},
		{"0.005", HalfEven, "0.00"},
		{"999.995", HalfUp, "1000.00"},
	} {
		input := derivationref.Input{State: derivationref.Present, Decimal: tc.raw}
		out, err := RoundExact(input, DecimalSpec{Precision: 8, Scale: 2, Mode: tc.mode})
		if err != nil || out.Decimal != tc.want {
			t.Fatalf("%s %s => %+v err=%v; want %s", tc.raw, tc.mode, out, err, tc.want)
		}
	}
	for _, tc := range []struct {
		raw string
		p, s int
	}{
		{"1999.98", 5, 2}, // 6 digits in scaled integer, cannot fit
		{"999.995", 5, 2}, // carry to 1000.00 cannot fit
		{"0.01", 2, 3},    // invalid scale
	} {
		_, err := RoundExact(derivationref.Input{State: derivationref.Present, Decimal: tc.raw},
			DecimalSpec{Precision: tc.p, Scale: tc.s, Mode: HalfUp})
		if !errors.Is(err, ErrPrecision) {
			t.Fatalf("%+v: want precision rejection, got %v", tc, err)
		}
	}
	for _, raw := range []string{"1/3", "NaN", "1e2", "1.2.3", "", "+"} {
		_, err := RoundExact(derivationref.Input{State: derivationref.Present, Decimal: raw},
			DecimalSpec{Precision: 8, Scale: 2, Mode: HalfEven})
		if !errors.Is(err, derivationref.ErrOperator) {
			t.Fatalf("non-decimal %q accepted: %v", raw, err)
		}
	}
	if _, err := RoundExact(derivationref.Input{State: derivationref.Present, Decimal: "1"},
		DecimalSpec{Precision: 8, Scale: 2}); !errors.Is(err, ErrPrecision) {
		t.Fatalf("missing rounding contract: %v", err)
	}
}

func TestDimensionCurrencyAndTypeFailClosed(t *testing.T) {
	types, qty, price, target := fixture()
	spec := DecimalSpec{Precision: 8, Scale: 2, Mode: HalfEven}
	checkReject := func(label string, q, p Operand, t AmountTarget, ts []semantic.Type) {
		t.Helper()
		_, err := MultiplyAmount(ts, q, p, t, spec)
		if !errors.Is(err, ErrMeaning) {
			t.Errorf("%s: expected meaning rejection, got %v", label, err)
		}
	}
	q := qty
	q.Unit = "kg"
	checkReject("kg times JPY/piece", q, price, target, types)
	p := price
	p.Currency = "USD"
	checkReject("USD to JPY without FX", qty, p, target, types)
	p = price
	p.PerUnit = ""
	checkReject("untyped price denominator", qty, p, target, types)
	p = price
	p.Attribute.TypeRef = core.MustCanonicalID("std.text")
	checkReject("text price", qty, p, target, types)
	q = qty
	q.Attribute.TypeRef = core.MustCanonicalID("sales.unresolved_type")
	checkReject("unknown quantity", q, price, target, types)
	if err := RequireSameCurrency("JPY", "USD"); !errors.Is(err, ErrMeaning) {
		t.Fatalf("100 JPY + 2 USD was not rejected: %v", err)
	}
	if err := RequireSameCurrency("JPY", "JPY"); err != nil {
		t.Fatal(err)
	}
}

func TestNullMissingAndRelationFailureNotZero(t *testing.T) {
	types, qty, price, target := fixture()
	spec := DecimalSpec{Precision: 8, Scale: 2, Mode: HalfEven}
	qty.Value = derivationref.Input{State: derivationref.Null}
	result, err := MultiplyAmount(types, qty, price, target, spec)
	if err != nil || result.State != derivationref.Null || result.Decimal != "" {
		t.Fatalf("null conflated with zero: %+v %v", result, err)
	}
	for _, state := range []derivationref.InputState{derivationref.Missing, derivationref.AbsentRelation, derivationref.MultipleRelations} {
		qty.Value = derivationref.Input{State: state}
		result, err := MultiplyAmount(types, qty, price, target, spec)
		if !errors.Is(err, derivationref.ErrInput) || result.State == derivationref.Present || result.State == derivationref.Null {
			t.Fatalf("state %q silently coerced: %+v %v", state, result, err)
		}
	}
}

func TestResolvePrimitiveRejectsCyclesDuplicatesAndUnknowns(t *testing.T) {
	types, qty, _, _ := fixture()
	got, err := ResolvePrimitive(types, qty.Attribute)
	if err != nil || got != core.MustCanonicalID("std.decimal") {
		t.Fatalf("alias did not resolve: %s %v", got, err)
	}
	for _, malformed := range [][]semantic.Type{
		{{ID: core.MustCanonicalID("sales.quantity_type"), Kind: semantic.TypeAlias, Underlying: core.MustCanonicalID("sales.quantity_type")}},
		{{ID: core.MustCanonicalID("sales.quantity_type"), Kind: semantic.TypeAlias, Underlying: core.MustCanonicalID("sales.bad")}},
		{{ID: core.MustCanonicalID("sales.quantity_type"), Kind: semantic.TypeAlias}},
		{{ID: core.MustCanonicalID("sales.quantity_type"), Kind: semantic.TypePrimitive}, {ID: core.MustCanonicalID("sales.quantity_type"), Kind: semantic.TypePrimitive}},
	} {
		if _, err := ResolvePrimitive(malformed, qty.Attribute); !errors.Is(err, ErrMeaning) {
			t.Fatalf("bad alias passed: %+v err=%v", malformed, err)
		}
	}
}
