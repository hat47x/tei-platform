package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Fixture meta generator for the sales.line-amount contract. It reads the
// contract and emits a standalone candidate source. This is a recorded fixture,
// not an AI model call. TEI builds and runs it once (lineage depth 1).

type contract struct {
	Rounding struct {
		Precision int
		Scale     int
	} `json:"rounding"`
}

func main() {
	path := flag.String("contract", "", "contract JSON")
	flag.Parse()
	raw, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var c contract
	if err := json.Unmarshal(raw, &c); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	scale := c.Rounding.Scale
	scalePow := 1
	for i := 0; i < scale; i++ {
		scalePow *= 10
	}
	integerDigits := c.Rounding.Precision - scale
	source := strings.NewReplacer(
		"{{SCALE}}", fmt.Sprint(scale),
		"{{SCALE_POW}}", fmt.Sprint(scalePow),
		"{{INTEGER_DIGITS}}", fmt.Sprint(integerDigits),
	).Replace(candidateTemplate)
	fmt.Print(source)
}

const candidateTemplate = `package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
)

type input struct {
	Quantity  *string
	UnitPrice *string
	Currency  string
	Unit      string
	PerUnit   string
}

func main() {
	var in input
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		emit(map[string]any{"state": "error", "code": "INPUT_INVALID"})
		return
	}
	if in.Quantity == nil || in.UnitPrice == nil {
		emit(map[string]any{"state": "missing"})
		return
	}
	if in.Currency != "JPY" {
		emit(map[string]any{"state": "error", "code": "CURRENCY_MISMATCH"})
		return
	}
	if in.Unit != in.PerUnit {
		emit(map[string]any{"state": "error", "code": "UNIT_MISMATCH"})
		return
	}
	quantity, ok := new(big.Rat).SetString(*in.Quantity)
	if !ok {
		emit(map[string]any{"state": "error", "code": "DECIMAL_INVALID"})
		return
	}
	price, ok := new(big.Rat).SetString(*in.UnitPrice)
	if !ok {
		emit(map[string]any{"state": "error", "code": "DECIMAL_INVALID"})
		return
	}
	amount, err := roundHalfUp(new(big.Rat).Mul(quantity, price), {{SCALE}})
	if err != nil {
		emit(map[string]any{"state": "error", "code": err.Error()})
		return
	}
	emit(map[string]any{"state": "present", "amount": amount, "currency": "JPY"})
}

func roundHalfUp(value *big.Rat, scale int) (string, error) {
	scaled := new(big.Rat).Mul(value, new(big.Rat).SetInt(pow10(scale)))
	num, den := scaled.Num(), scaled.Denom()
	quotient, remainder := new(big.Int).QuoRem(num, den, new(big.Int))
	twice := new(big.Int).Abs(remainder)
	twice.Lsh(twice, 1)
	if twice.Cmp(den) >= 0 {
		if num.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	absolute := new(big.Int).Abs(quotient)
	integer := new(big.Int).Quo(absolute, pow10(scale))
	if len(integer.String()) > {{INTEGER_DIGITS}} {
		return "", fmt.Errorf("PRECISION_OVERFLOW")
	}
	fraction := new(big.Int).Rem(absolute, pow10(scale))
	fractionText := fraction.String()
	for len(fractionText) < scale {
		fractionText = "0" + fractionText
	}
	sign := ""
	if quotient.Sign() < 0 {
		sign = "-"
	}
	return fmt.Sprintf("%s%s.%s", sign, integer.String(), fractionText), nil
}

func pow10(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

func emit(value map[string]any) {
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
		os.Exit(1)
	}
}
`
