package quantityarith

import "testing"

func intPtr(v int) *int { return &v }

// The fixtures follow Data Derivation D8 (#1705). Token names are synthetic
// labels, not accepted DSL or canonical type names.

func TestMultiplyCountByRateFormsAmount(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY", "USD"},
		Types: map[string]TypeAnnotation{
			"個":     {Unit: "個"},
			"JPY/個": {Currency: "JPY", Per: "個"},
		},
		Operation: Operation{
			Op:    OpMultiply,
			Left:  Operand{Type: "個", State: "present", Decimal: "2"},
			Right: &Operand{Type: "JPY/個", State: "present", Decimal: "300"},
		},
	}
	report := Evaluate(in)
	if !report.Accepted || report.Code != CodeOK {
		t.Fatalf("accepted=%v code=%s message=%s", report.Accepted, report.Code, report.Message)
	}
	if report.ResultDecimal != "600" {
		t.Fatalf("decimal=%q want 600", report.ResultDecimal)
	}
	if !report.IsAmount || report.Currency != "JPY" {
		t.Fatalf("amount=%v currency=%q want JPY amount", report.IsAmount, report.Currency)
	}
}

func TestMultiplyZeroIsAmountNotNull(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types: map[string]TypeAnnotation{
			"個":     {Unit: "個"},
			"JPY/個": {Currency: "JPY", Per: "個"},
		},
		Operation: Operation{
			Op:    OpMultiply,
			Left:  Operand{Type: "個", State: "present", Decimal: "0"},
			Right: &Operand{Type: "JPY/個", State: "present", Decimal: "300"},
		},
	}
	report := Evaluate(in)
	if !report.Accepted || report.ResultDecimal != "0" {
		t.Fatalf("accepted=%v decimal=%q code=%s", report.Accepted, report.ResultDecimal, report.Code)
	}
	if !report.IsAmount {
		t.Fatalf("zero must stay an amount, got %+v", report)
	}
}

func TestMultiplyFractionalProductStaysExact(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types: map[string]TypeAnnotation{
			"個":     {Unit: "個"},
			"JPY/個": {Currency: "JPY", Per: "個"},
		},
		Operation: Operation{
			Op:    OpMultiply,
			Left:  Operand{Type: "個", State: "present", Decimal: "1.25"},
			Right: &Operand{Type: "JPY/個", State: "present", Decimal: "0.4"},
		},
	}
	report := Evaluate(in)
	if !report.Accepted || report.ResultDecimal != "0.5" {
		t.Fatalf("accepted=%v decimal=%q code=%s", report.Accepted, report.ResultDecimal, report.Code)
	}
}

func TestMultiplyUnitMismatchIsRejected(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types: map[string]TypeAnnotation{
			"kg":    {Unit: "kg"},
			"JPY/個": {Currency: "JPY", Per: "個"},
		},
		Operation: Operation{
			Op:    OpMultiply,
			Left:  Operand{Type: "kg", State: "present", Decimal: "1"},
			Right: &Operand{Type: "JPY/個", State: "present", Decimal: "300"},
		},
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeDimensionMismatch {
		t.Fatalf("accepted=%v code=%s want DIMENSION_MISMATCH", report.Accepted, report.Code)
	}
}

func TestAddDifferentCurrenciesIsRejected(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY", "USD"},
		Types: map[string]TypeAnnotation{
			"JPY": {Currency: "JPY"},
			"USD": {Currency: "USD"},
		},
		Operation: Operation{
			Op:    OpAdd,
			Left:  Operand{Type: "JPY", State: "present", Decimal: "100"},
			Right: &Operand{Type: "USD", State: "present", Decimal: "2"},
		},
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeCurrencyMismatch {
		t.Fatalf("accepted=%v code=%s want CURRENCY_MISMATCH", report.Accepted, report.Code)
	}
}

func TestAddSameDimensionIsAccepted(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types: map[string]TypeAnnotation{
			"JPY": {Currency: "JPY"},
		},
		Operation: Operation{
			Op:    OpAdd,
			Left:  Operand{Type: "JPY", State: "present", Decimal: "100"},
			Right: &Operand{Type: "JPY", State: "present", Decimal: "2"},
		},
	}
	report := Evaluate(in)
	if !report.Accepted || report.ResultDecimal != "102" || !report.IsAmount {
		t.Fatalf("report=%+v", report)
	}
}

func TestRoundingModesAreDistinct(t *testing.T) {
	base := func(mode string) Input {
		return Input{
			Currencies: []string{"JPY"},
			Types:      map[string]TypeAnnotation{"JPY": {Currency: "JPY"}},
			Operation: Operation{
				Op:       OpRound,
				Left:     Operand{Type: "JPY", State: "present", Decimal: "1.005"},
				Rounding: mode,
				Scale:    intPtr(2),
			},
		}
	}
	halfUp := Evaluate(base(RoundHalfUp))
	halfEven := Evaluate(base(RoundHalfEven))
	if !halfUp.Accepted || halfUp.ResultDecimal != "1.01" {
		t.Fatalf("HALF_UP=%+v want 1.01", halfUp)
	}
	if !halfEven.Accepted || halfEven.ResultDecimal != "1.00" {
		t.Fatalf("HALF_EVEN=%+v want 1.00", halfEven)
	}
}

func TestRoundingHalfEvenRoundsTieToEven(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types:      map[string]TypeAnnotation{"JPY": {Currency: "JPY"}},
		Operation: Operation{
			Op:       OpRound,
			Left:     Operand{Type: "JPY", State: "present", Decimal: "1.015"},
			Rounding: RoundHalfEven,
			Scale:    intPtr(2),
		},
	}
	report := Evaluate(in)
	if !report.Accepted || report.ResultDecimal != "1.02" {
		t.Fatalf("report=%+v want 1.02", report)
	}
}

func TestStorePrecisionOverflowIsRejected(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types: map[string]TypeAnnotation{
			"JPY5_2": {Currency: "JPY", Precision: intPtr(5), Scale: intPtr(2)},
		},
		Operation: Operation{
			Op:   OpStore,
			Left: Operand{Type: "JPY5_2", State: "present", Decimal: "1999.98"},
		},
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodePrecisionOverflow {
		t.Fatalf("accepted=%v code=%s want PRECISION_OVERFLOW", report.Accepted, report.Code)
	}
}

func TestStoreFittingValueIsAccepted(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types: map[string]TypeAnnotation{
			"JPY5_2": {Currency: "JPY", Precision: intPtr(5), Scale: intPtr(2)},
		},
		Operation: Operation{
			Op:   OpStore,
			Left: Operand{Type: "JPY5_2", State: "present", Decimal: "999.99"},
		},
	}
	report := Evaluate(in)
	if !report.Accepted || report.ResultDecimal != "999.99" {
		t.Fatalf("report=%+v", report)
	}
}

func TestNullPropagatesButMissingFails(t *testing.T) {
	types := map[string]TypeAnnotation{
		"個":     {Unit: "個"},
		"JPY/個": {Currency: "JPY", Per: "個"},
	}
	nullInput := Input{
		Currencies: []string{"JPY"},
		Types:      types,
		Operation: Operation{
			Op:    OpMultiply,
			Left:  Operand{Type: "個", State: "null"},
			Right: &Operand{Type: "JPY/個", State: "present", Decimal: "300"},
		},
	}
	report := Evaluate(nullInput)
	if !report.Accepted || report.Code != CodeNullPropagated || report.ResultState != "null" {
		t.Fatalf("null report=%+v", report)
	}
	if report.ResultDecimal != "" {
		t.Fatalf("null must not become zero, decimal=%q", report.ResultDecimal)
	}

	missingInput := nullInput
	missingInput.Operation.Left = Operand{Type: "個", State: "missing"}
	report = Evaluate(missingInput)
	if report.Accepted || report.Code != CodeInputUnavailable {
		t.Fatalf("missing report=%+v want INPUT_UNAVAILABLE", report)
	}
}

func TestUnknownTypeIsRejected(t *testing.T) {
	in := Input{
		Types: map[string]TypeAnnotation{"個": {Unit: "個"}},
		Operation: Operation{
			Op:    OpMultiply,
			Left:  Operand{Type: "個", State: "present", Decimal: "1"},
			Right: &Operand{Type: "nope", State: "present", Decimal: "1"},
		},
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeUnknownType {
		t.Fatalf("report=%+v want UNKNOWN_TYPE", report)
	}
}

func TestUnsupportedOperationIsRejected(t *testing.T) {
	report := Evaluate(Input{Operation: Operation{Op: "divide"}})
	if report.Accepted || report.Code != CodeUnsupported {
		t.Fatalf("report=%+v want UNSUPPORTED_OPERATION", report)
	}
}
