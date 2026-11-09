// Package typedarithmetic is a synthetic D8 reference for semantic unit,
// currency and exact decimal rounding. It does not define a Canonical type
// registry, expression language or a production Operator implementation.
package typedarithmetic

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/derivationref"
)

var ErrMeaning = errors.New("typed arithmetic meaning mismatch")
var ErrPrecision = errors.New("decimal precision or rounding invalid")

type RoundingMode string

const (
	HalfUp   RoundingMode = "half-up"
	HalfEven RoundingMode = "half-even"
)

// DecimalSpec is an explicit fixture-level output constraint. It is not a
// global money/decimal policy or an accepted Semantic type declaration.
type DecimalSpec struct {
	Precision int
	Scale     int
	Mode      RoundingMode
}

// Operand binds a D1 input to an existing Semantic Attribute TypeRef. Unit,
// PerUnit and Currency are explicit test annotations, not inferred from names.
type Operand struct {
	Attribute semantic.Attribute
	Value     derivationref.Input
	Unit      string
	PerUnit   string
	Currency  string
}

type AmountTarget struct {
	Attribute semantic.Attribute
	Currency  string
}

// ResolvePrimitive follows *existing* Semantic aliases without copying the
// Semantic type registry. Absent, cyclic and unsupported references fail shut.
func ResolvePrimitive(types []semantic.Type, attr semantic.Attribute) (core.CanonicalID, error) {
	if attr.ID == "" || attr.TypeRef == "" {
		return "", fmt.Errorf("%w: missing typed attribute identity", ErrMeaning)
	}
	index := make(map[core.CanonicalID]semantic.Type, len(types))
	for _, typ := range types {
		if typ.ID == "" {
			return "", fmt.Errorf("%w: empty type identity", ErrMeaning)
		}
		if _, duplicate := index[typ.ID]; duplicate {
			return "", fmt.Errorf("%w: duplicate semantic type %s", ErrMeaning, typ.ID)
		}
		index[typ.ID] = typ
	}
	id := attr.TypeRef
	visited := map[core.CanonicalID]bool{}
	for {
		if visited[id] {
			return "", fmt.Errorf("%w: cyclic type alias", ErrMeaning)
		}
		visited[id] = true
		typ, ok := index[id]
		if !ok {
			return "", fmt.Errorf("%w: unknown type %s", ErrMeaning, id)
		}
		switch typ.Kind {
		case semantic.TypePrimitive:
			if typ.Underlying != "" {
				return "", fmt.Errorf("%w: primitive with underlying type", ErrMeaning)
			}
			return typ.ID, nil
		case semantic.TypeAlias:
			if typ.Underlying == "" {
				return "", fmt.Errorf("%w: alias missing underlying type", ErrMeaning)
			}
			id = typ.Underlying
		default:
			return "", fmt.Errorf("%w: unknown semantic type kind", ErrMeaning)
		}
	}
}

func requireType(types []semantic.Type, attr semantic.Attribute, allowed ...string) error {
	id, err := ResolvePrimitive(types, attr)
	if err != nil {
		return err
	}
	for _, candidate := range allowed {
		if string(id) == candidate {
			return nil
		}
	}
	return fmt.Errorf("%w: unsupported numeric primitive %s", ErrMeaning, id)
}

func validateDecimalSpec(spec DecimalSpec) error {
	if spec.Precision < 1 || spec.Scale < 0 || spec.Scale > spec.Precision {
		return fmt.Errorf("%w: invalid decimal precision/scale", ErrPrecision)
	}
	if spec.Mode != HalfUp && spec.Mode != HalfEven {
		return fmt.Errorf("%w: explicit rounding mode required", ErrPrecision)
	}
	return nil
}

// RoundExact quantizes a finite decimal at the final result boundary. It never
// uses float64 and checks the *rounded* result against DECIMAL(precision,scale).
// Null is preserved; missing inputs are not quietly converted to zero.
func RoundExact(input derivationref.Input, spec DecimalSpec) (derivationref.Input, error) {
	if err := validateDecimalSpec(spec); err != nil {
		return derivationref.Input{}, err
	}
	if input.State == derivationref.Null {
		return derivationref.Input{State: derivationref.Null}, nil
	}
	if input.State != derivationref.Present {
		return derivationref.Input{}, fmt.Errorf("%w: %s", derivationref.ErrInput, input.State)
	}
	if !validFiniteDecimal(input.Decimal) {
		return derivationref.Input{}, fmt.Errorf("%w: nondecimal input", derivationref.ErrOperator)
	}
	number, ok := new(big.Rat).SetString(input.Decimal)
	if !ok {
		return derivationref.Input{}, fmt.Errorf("%w: decimal parse failed", derivationref.ErrOperator)
	}
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(spec.Scale)), nil)
	absNumerator := new(big.Int).Mul(new(big.Int).Abs(number.Num()), multiplier)
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(absNumerator, number.Denom(), remainder)
	doubleRem := new(big.Int).Mul(remainder, big.NewInt(2))
	cmp := doubleRem.Cmp(number.Denom())
	if cmp > 0 || (cmp == 0 && (spec.Mode == HalfUp || quotient.Bit(0) == 1)) {
		quotient.Add(quotient, big.NewInt(1))
	}
	if len(quotient.String()) > spec.Precision {
		return derivationref.Input{}, fmt.Errorf("%w: result exceeds declared precision", ErrPrecision)
	}
	digits := quotient.String()
	if spec.Scale > 0 {
		if len(digits) <= spec.Scale {
			digits = strings.Repeat("0", spec.Scale-len(digits)+1) + digits
		}
		cut := len(digits) - spec.Scale
		digits = digits[:cut] + "." + digits[cut:]
	}
	if number.Sign() < 0 && quotient.Sign() != 0 {
		digits = "-" + digits
	}
	return derivationref.Input{State: derivationref.Present, Decimal: digits}, nil
}

func validFiniteDecimal(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	digits, dots := 0, 0
	for _, ch := range s {
		if ch == '.' {
			dots++
			if dots > 1 {
				return false
			}
		} else if ch >= '0' && ch <= '9' {
			digits++
		} else {
			return false
		}
	}
	return digits > 0
}

// MultiplyAmount checks the semantic types and explicit measurement
// annotations before delegating *exact multiplication* to the existing D1
// reference. It then rounds only once at the declared result boundary.
func MultiplyAmount(types []semantic.Type, quantity, unitPrice Operand, target AmountTarget, spec DecimalSpec) (derivationref.Input, error) {
	if err := validateDecimalSpec(spec); err != nil {
		return derivationref.Input{}, err
	}
	if err := requireType(types, quantity.Attribute, "std.integer", "std.decimal"); err != nil {
		return derivationref.Input{}, err
	}
	if err := requireType(types, unitPrice.Attribute, "std.money"); err != nil {
		return derivationref.Input{}, err
	}
	if err := requireType(types, target.Attribute, "std.money"); err != nil {
		return derivationref.Input{}, err
	}
	if quantity.Unit == "" || unitPrice.PerUnit != quantity.Unit || quantity.PerUnit != "" || quantity.Currency != "" ||
		unitPrice.Unit != "" || unitPrice.Currency == "" || target.Currency == "" || unitPrice.Currency != target.Currency {
		return derivationref.Input{}, fmt.Errorf("%w: incompatible unit or currency", ErrMeaning)
	}
	exact, err := derivationref.Multiply(quantity.Value, unitPrice.Value)
	if err != nil {
		return derivationref.Input{}, err
	}
	return RoundExact(exact, spec)
}

// RequireSameCurrency is a candidate *check*, not an additive Operator or a
// currency conversion capability. Conversion must use an explicit rule.
func RequireSameCurrency(a, b string) error {
	if a == "" || b == "" || a != b {
		return fmt.Errorf("%w: currency conversion not declared", ErrMeaning)
	}
	return nil
}
