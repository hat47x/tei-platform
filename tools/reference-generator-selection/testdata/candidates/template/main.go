package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
)

// Candidate: template. This file is the rendered form of a small generator
// template for the line-amount contract. It uses exact rational arithmetic and
// half-up rounding at the final result boundary.

type input struct {
	Quantity  *string `json:"quantity"`
	UnitPrice *string `json:"unitPrice"`
	Currency  string  `json:"currency"`
	Unit      string  `json:"unit"`
	PerUnit   string  `json:"perUnit"`
}

type output struct {
	State    string `json:"state"`
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
	Code     string `json:"code,omitempty"`
}

func main() {
	var in input
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		emit(output{State: "error", Code: "INPUT_INVALID"})
		return
	}
	if in.Quantity == nil || in.UnitPrice == nil {
		emit(output{State: "missing"})
		return
	}
	if in.Currency != "JPY" {
		emit(output{State: "error", Code: "CURRENCY_MISMATCH"})
		return
	}
	if in.Unit != in.PerUnit {
		emit(output{State: "error", Code: "UNIT_MISMATCH"})
		return
	}
	quantity, ok := new(big.Rat).SetString(*in.Quantity)
	if !ok {
		emit(output{State: "error", Code: "DECIMAL_INVALID"})
		return
	}
	price, ok := new(big.Rat).SetString(*in.UnitPrice)
	if !ok {
		emit(output{State: "error", Code: "DECIMAL_INVALID"})
		return
	}
	amount, err := roundHalfUp(new(big.Rat).Mul(quantity, price), 2)
	if err != nil {
		emit(output{State: "error", Code: err.Error()})
		return
	}
	emit(output{State: "present", Amount: amount, Currency: "JPY"})
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
	if len(integer.String()) > 6 {
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

func emit(out output) {
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		os.Exit(1)
	}
}
