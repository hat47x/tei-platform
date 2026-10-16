// Package quantityarith is a read-only, synthetic reference for Data Derivation
// D8 (#1705). It checks a small, explicit set of semantic annotations on
// arithmetic: unit and currency dimension, decimal precision, and rounding of a
// final value.
//
// It is not a Canonical model, a unit-conversion library, a currency registry, a
// generic expression engine, or an accepted API. Unit and currency tokens are
// opaque synthetic labels supplied by the caller. The pure rule and input state
// are reused from Data Derivation D1 (#1637) in internal/semantic/derivationref,
// so a missing or null input is never turned into numeric zero here.
package quantityarith

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"tei/internal/semantic/derivationref"
)

// Rounding modes. The reference does not treat them as interchangeable.
const (
	RoundHalfUp   = "HALF_UP"
	RoundHalfEven = "HALF_EVEN"
)

// Operation names.
const (
	OpMultiply = "multiply"
	OpAdd      = "add"
	OpCompare  = "compare"
	OpRound    = "round"
	OpStore    = "store"
)

// Result codes.
const (
	CodeOK                = "OK"
	CodeNullPropagated    = "NULL_PROPAGATED"
	CodeInputUnavailable  = "INPUT_UNAVAILABLE"
	CodeDimensionMismatch = "DIMENSION_MISMATCH"
	CodeCurrencyMismatch  = "CURRENCY_MISMATCH"
	CodePrecisionOverflow = "PRECISION_OVERFLOW"
	CodeUnknownType       = "UNKNOWN_TYPE"
	CodeInvalidDecimal    = "INVALID_DECIMAL"
	CodeInvalidRequest    = "INVALID_REQUEST"
	CodeUnsupported       = "UNSUPPORTED_OPERATION"
)

// TypeAnnotation is a synthetic semantic annotation for an operand type. Unit is
// a numerator unit token, Per is a denominator unit token, and Currency is a
// currency token. A token cancels against an equal token of the opposite side,
// which is what lets "個" multiplied by "JPY/個" leave only "JPY".
type TypeAnnotation struct {
	Unit      string `json:"unit,omitempty"`
	Per       string `json:"per,omitempty"`
	Currency  string `json:"currency,omitempty"`
	Precision *int   `json:"precision,omitempty"`
	Scale     *int   `json:"scale,omitempty"`
}

// Operand references a declared type and carries an input state.
type Operand struct {
	Type    string `json:"type"`
	State   string `json:"state"`
	Decimal string `json:"decimal,omitempty"`
}

// Operation is one arithmetic step.
type Operation struct {
	Op       string   `json:"op"`
	Left     Operand  `json:"left"`
	Right    *Operand `json:"right,omitempty"`
	Rounding string   `json:"rounding,omitempty"`
	Scale    *int     `json:"scale,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	Currencies []string                  `json:"currencies,omitempty"`
	Types      map[string]TypeAnnotation `json:"types"`
	Operation  Operation                 `json:"operation"`
}

// Report is the read-only result. No Source is written and no side effect is
// performed.
type Report struct {
	Op              string         `json:"op"`
	Accepted        bool           `json:"accepted"`
	Code            string         `json:"code"`
	Message         string         `json:"message,omitempty"`
	ResultState     string         `json:"resultState,omitempty"`
	ResultDecimal   string         `json:"resultDecimal,omitempty"`
	Dimension       map[string]int `json:"dimension,omitempty"`
	IsAmount        bool           `json:"isAmount"`
	Currency        string         `json:"currency,omitempty"`
	RoundingApplied string         `json:"roundingApplied,omitempty"`
	Scale           *int           `json:"scale,omitempty"`
}

// dimension is a token exponent vector. A token with a positive exponent is a
// numerator, a negative exponent a denominator, and zero exponents are dropped.
type dimension map[string]int

func (d dimension) add(o dimension) dimension {
	out := dimension{}
	for k, v := range d {
		out[k] += v
	}
	for k, v := range o {
		out[k] += v
	}
	for k, v := range out {
		if v == 0 {
			delete(out, k)
		}
	}
	return out
}

func (d dimension) key() string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s^%d", k, d[k]))
	}
	return strings.Join(parts, ",")
}

func (d dimension) currencyTokens(currencies map[string]bool) []string {
	var out []string
	for token, exp := range d {
		if exp != 0 && currencies[token] {
			out = append(out, token)
		}
	}
	sort.Strings(out)
	return out
}

// nonCurrencyPresent reports whether a token other than a declared currency has
// a non-zero exponent.
func (d dimension) nonCurrencyPresent(currencies map[string]bool) bool {
	for token, exp := range d {
		if exp == 0 || currencies[token] {
			continue
		}
		return true
	}
	return false
}

func (t TypeAnnotation) dimension() dimension {
	d := dimension{}
	if t.Unit != "" {
		d[t.Unit]++
	}
	if t.Per != "" {
		d[t.Per]--
	}
	if t.Currency != "" {
		d[t.Currency]++
	}
	for k, v := range d {
		if v == 0 {
			delete(d, k)
		}
	}
	return d
}

func (t TypeAnnotation) hasDecimalPolicy() bool {
	return t.Precision != nil && t.Scale != nil
}

type evaluator struct {
	input      Input
	currencies map[string]bool
}

// Evaluate checks one operation against the declared annotations and returns a
// structure. It never writes anything and never turns a missing or null input
// into zero.
func Evaluate(in Input) Report {
	e := evaluator{input: in, currencies: map[string]bool{}}
	for _, c := range in.Currencies {
		c = strings.TrimSpace(c)
		if c != "" {
			e.currencies[c] = true
		}
	}
	op := strings.TrimSpace(in.Operation.Op)
	switch op {
	case OpMultiply:
		return e.multiply()
	case OpAdd:
		return e.add()
	case OpCompare:
		return e.compare()
	case OpRound:
		return e.round()
	case OpStore:
		return e.store()
	default:
		return Report{Op: op, Accepted: false, Code: CodeUnsupported,
			Message: fmt.Sprintf("unsupported operation %q", in.Operation.Op)}
	}
}

func reject(op, code, msg string) Report {
	return Report{Op: op, Accepted: false, Code: code, Message: msg}
}

// inputState maps an operand to a D1 input state and parsed exact value.
func (e evaluator) inputState(operand Operand) (derivationref.InputState, *big.Rat, error) {
	state := derivationref.InputState(strings.TrimSpace(operand.State))
	switch state {
	case derivationref.Present,
		derivationref.Null,
		derivationref.Missing,
		derivationref.AbsentRelation,
		derivationref.MultipleRelations:
	default:
		return "", nil, fmt.Errorf("invalid state %q", operand.State)
	}
	value := strings.TrimSpace(operand.Decimal)
	if state != derivationref.Present {
		if value != "" {
			return "", nil, fmt.Errorf("%s operand must not carry a decimal", state)
		}
		return state, nil, nil
	}
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return "", nil, fmt.Errorf("invalid decimal %q", operand.Decimal)
	}
	return state, r, nil
}

// unavailable reports whether the state must stop the operation instead of
// producing a value. Only null becomes a null result.
func unavailable(state derivationref.InputState) bool {
	switch state {
	case derivationref.Missing, derivationref.AbsentRelation, derivationref.MultipleRelations:
		return true
	}
	return false
}

func (e evaluator) operandPair() (TypeAnnotation, TypeAnnotation, derivationref.InputState, *big.Rat, derivationref.InputState, *big.Rat, Report, bool) {
	op := strings.TrimSpace(e.input.Operation.Op)
	leftType, ok := e.input.Types[strings.TrimSpace(e.input.Operation.Left.Type)]
	if !ok {
		return TypeAnnotation{}, TypeAnnotation{}, "", nil, "", nil,
			reject(op, CodeUnknownType, "left type is not declared"), false
	}
	if e.input.Operation.Right == nil {
		return TypeAnnotation{}, TypeAnnotation{}, "", nil, "", nil,
			reject(op, CodeInvalidRequest, op+" needs a right operand"), false
	}
	rightType, ok := e.input.Types[strings.TrimSpace(e.input.Operation.Right.Type)]
	if !ok {
		return TypeAnnotation{}, TypeAnnotation{}, "", nil, "", nil,
			reject(op, CodeUnknownType, "right type is not declared"), false
	}
	leftState, leftValue, err := e.inputState(e.input.Operation.Left)
	if err != nil {
		return TypeAnnotation{}, TypeAnnotation{}, "", nil, "", nil,
			reject(op, CodeInvalidDecimal, err.Error()), false
	}
	rightState, rightValue, err := e.inputState(*e.input.Operation.Right)
	if err != nil {
		return TypeAnnotation{}, TypeAnnotation{}, "", nil, "", nil,
			reject(op, CodeInvalidDecimal, err.Error()), false
	}
	if unavailable(leftState) || unavailable(rightState) {
		return TypeAnnotation{}, TypeAnnotation{}, "", nil, "", nil,
			reject(op, CodeInputUnavailable, "an input is missing, not null"), false
	}
	if leftState == derivationref.Null || rightState == derivationref.Null {
		return TypeAnnotation{}, TypeAnnotation{}, "", nil, "", nil,
			Report{Op: op, Accepted: true, Code: CodeNullPropagated, ResultState: string(derivationref.Null)}, false
	}
	return leftType, rightType, leftState, leftValue, rightState, rightValue, Report{}, true
}

func (e evaluator) multiply() Report {
	op := OpMultiply
	leftType, rightType, _, _, _, _, failure, ok := e.operandPair()
	if !ok {
		return failure
	}
	dim := leftType.dimension().add(rightType.dimension())
	currencies := dim.currencyTokens(e.currencies)
	if len(currencies) > 1 {
		return reject(op, CodeCurrencyMismatch,
			"multiplying different currencies needs an explicit conversion rule")
	}
	if len(currencies) == 1 && dim.nonCurrencyPresent(e.currencies) {
		return reject(op, CodeDimensionMismatch,
			"currency with a leftover unit does not form a business amount")
	}
	product, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Present, Decimal: e.input.Operation.Left.Decimal},
		derivationref.Input{State: derivationref.Present, Decimal: e.input.Operation.Right.Decimal},
	)
	if err != nil {
		return reject(op, CodeInvalidDecimal, err.Error())
	}
	value, _ := new(big.Rat).SetString(product.Decimal)
	report := Report{Op: op, Dimension: map[string]int(dim)}
	return e.finish(report, value)
}

func (e evaluator) add() Report {
	op := OpAdd
	leftType, rightType, _, leftValue, _, rightValue, failure, ok := e.operandPair()
	if !ok {
		return failure
	}
	leftDim := leftType.dimension()
	rightDim := rightType.dimension()
	if leftDim.key() != rightDim.key() {
		code := CodeDimensionMismatch
		if len(leftDim.currencyTokens(e.currencies)) == 1 && len(rightDim.currencyTokens(e.currencies)) == 1 {
			code = CodeCurrencyMismatch
		}
		return reject(op, code, "operands have different dimensions")
	}
	report := Report{Op: op, Dimension: map[string]int(leftDim)}
	return e.finish(report, new(big.Rat).Add(leftValue, rightValue))
}

func (e evaluator) compare() Report {
	op := OpCompare
	leftType, rightType, _, leftValue, _, rightValue, failure, ok := e.operandPair()
	if !ok {
		return failure
	}
	leftDim := leftType.dimension()
	rightDim := rightType.dimension()
	if leftDim.key() != rightDim.key() {
		return reject(op, CodeDimensionMismatch, "operands have different dimensions")
	}
	report := Report{Op: op, Accepted: true, Code: CodeOK,
		Dimension:     map[string]int(leftDim),
		ResultDecimal: fmt.Sprintf("%d", leftValue.Cmp(rightValue))}
	return report
}

func (e evaluator) round() Report {
	op := OpRound
	left := e.input.Operation.Left
	typeAnnotation, ok := e.input.Types[strings.TrimSpace(left.Type)]
	if !ok {
		return reject(op, CodeUnknownType, "operand type is not declared")
	}
	if e.input.Operation.Scale == nil {
		return reject(op, CodeInvalidRequest, "round needs a scale")
	}
	state, value, err := e.inputState(left)
	if err != nil {
		return reject(op, CodeInvalidDecimal, err.Error())
	}
	if unavailable(state) {
		return reject(op, CodeInputUnavailable, "an input is missing, not null")
	}
	if state == derivationref.Null {
		return Report{Op: op, Accepted: true, Code: CodeNullPropagated, ResultState: string(derivationref.Null)}
	}
	rounded, err := roundRat(value, *e.input.Operation.Scale, e.input.Operation.Rounding)
	if err != nil {
		return reject(op, CodeInvalidRequest, err.Error())
	}
	report := Report{Op: op, Accepted: true, Code: CodeOK,
		ResultDecimal:   rounded,
		Dimension:       map[string]int(typeAnnotation.dimension()),
		RoundingApplied: normalizeRounding(e.input.Operation.Rounding),
		Scale:           e.input.Operation.Scale}
	report.markAmount(e.currencies)
	return report
}

func (e evaluator) store() Report {
	op := OpStore
	left := e.input.Operation.Left
	typeAnnotation, ok := e.input.Types[strings.TrimSpace(left.Type)]
	if !ok {
		return reject(op, CodeUnknownType, "operand type is not declared")
	}
	state, value, err := e.inputState(left)
	if err != nil {
		return reject(op, CodeInvalidDecimal, err.Error())
	}
	if unavailable(state) {
		return reject(op, CodeInputUnavailable, "an input is missing, not null")
	}
	if state == derivationref.Null {
		return Report{Op: op, Accepted: true, Code: CodeNullPropagated, ResultState: string(derivationref.Null)}
	}
	if typeAnnotation.hasDecimalPolicy() && !fitsDecimal(value, *typeAnnotation.Precision, *typeAnnotation.Scale) {
		return reject(op, CodePrecisionOverflow, "value does not fit the declared decimal")
	}
	report := Report{Op: op, Accepted: true, Code: CodeOK,
		ResultDecimal: formatExact(value),
		Dimension:     map[string]int(typeAnnotation.dimension())}
	report.markAmount(e.currencies)
	return report
}

// finish attaches the exact or rounded result and the currency flag.
func (e evaluator) finish(report Report, value *big.Rat) Report {
	report.Accepted = true
	report.Code = CodeOK
	if e.input.Operation.Scale != nil {
		rounded, err := roundRat(value, *e.input.Operation.Scale, e.input.Operation.Rounding)
		if err != nil {
			return reject(report.Op, CodeInvalidRequest, err.Error())
		}
		report.ResultDecimal = rounded
		report.RoundingApplied = normalizeRounding(e.input.Operation.Rounding)
		report.Scale = e.input.Operation.Scale
	} else {
		report.ResultDecimal = formatExact(value)
	}
	report.markAmount(e.currencies)
	return report
}

func (r *Report) markAmount(currencies map[string]bool) {
	dim := dimension(r.Dimension)
	cur := dim.currencyTokens(currencies)
	if len(cur) == 1 && !dim.nonCurrencyPresent(currencies) {
		r.IsAmount = true
		r.Currency = cur[0]
	}
}

func normalizeRounding(mode string) string {
	m := strings.TrimSpace(strings.ToUpper(mode))
	if m == "" {
		return RoundHalfUp
	}
	return m
}

// formatExact writes r exactly. A rational whose denominator is not a product of
// 2s and 5s does not terminate, so callers that need a value must declare a
// rounding policy first.
func formatExact(r *big.Rat) string {
	scale, terminates := exactScale(r)
	if !terminates {
		return ""
	}
	return r.FloatString(scale)
}

// exactScale returns the number of fractional digits needed to write r exactly.
func exactScale(r *big.Rat) (int, bool) {
	den := new(big.Int).Set(r.Denom())
	count := func(factor *big.Int) int {
		n := 0
		for {
			q, rem := new(big.Int).QuoRem(den, factor, new(big.Int))
			if rem.Sign() != 0 {
				break
			}
			den = q
			n++
		}
		return n
	}
	twos := count(big.NewInt(2))
	fives := count(big.NewInt(5))
	if den.Cmp(big.NewInt(1)) != 0 {
		return 0, false
	}
	if twos > fives {
		return twos, true
	}
	return fives, true
}

// fitsDecimal reports whether r is exactly representable as DECIMAL(p, s) with p
// total digits and s fractional digits, without truncation.
func fitsDecimal(r *big.Rat, p, s int) bool {
	if p <= 0 || s < 0 || s > p {
		return false
	}
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(s)), nil)
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(factor))
	if !scaled.IsInt() {
		return false
	}
	digits := len(new(big.Int).Abs(scaled.Num()).String())
	integerDigits := digits - s
	if integerDigits < 0 {
		integerDigits = 0
	}
	return integerDigits <= p-s
}

// roundRat rounds r to the given scale with an explicit mode and returns the
// decimal string. HALF_UP rounds a tie away from zero; HALF_EVEN rounds a tie to
// the even last digit. The two are not silently treated as the same.
func roundRat(r *big.Rat, scale int, mode string) (string, error) {
	mode = normalizeRounding(mode)
	if mode != RoundHalfUp && mode != RoundHalfEven {
		return "", fmt.Errorf("unsupported rounding %q", mode)
	}
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(factor))
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(scaled.Num(), scaled.Denom(), remainder)

	// |remainder| / denominator is the dropped fraction. Numerator sign carries
	// through QuoRem, so work with magnitudes and restore the sign at the end.
	sign := scaled.Sign()
	twiceRem := new(big.Int).Abs(remainder)
	twiceRem.Lsh(twiceRem, 1)
	cmp := twiceRem.Cmp(scaled.Denom())
	roundAway := false
	switch mode {
	case RoundHalfUp:
		roundAway = cmp >= 0
	case RoundHalfEven:
		if cmp > 0 {
			roundAway = true
		} else if cmp == 0 {
			roundAway = quotient.Bit(0) == 1
		}
	}
	if roundAway {
		if sign < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	return formatScaled(quotient, scale), nil
}

// formatScaled writes integer/10^scale as a decimal string with exactly scale
// fractional digits.
func formatScaled(integer *big.Int, scale int) string {
	if scale <= 0 {
		return integer.String()
	}
	negative := integer.Sign() < 0
	digits := new(big.Int).Abs(integer).String()
	if len(digits) <= scale {
		digits = strings.Repeat("0", scale-len(digits)+1) + digits
	}
	split := len(digits) - scale
	out := digits[:split] + "." + digits[split:]
	if negative {
		out = "-" + out
	}
	return out
}
