package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
)

// Candidate: alternative. This implementation parses decimals into scaled
// integers and rounds once at the end. It shares the contract but not the
// arithmetic representation of the template candidate.

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
	quantity, quantityScale, err := parseDecimal(*in.Quantity)
	if err != nil {
		emit(output{State: "error", Code: "DECIMAL_INVALID"})
		return
	}
	price, priceScale, err := parseDecimal(*in.UnitPrice)
	if err != nil {
		emit(output{State: "error", Code: "DECIMAL_INVALID"})
		return
	}
	product := new(big.Int).Mul(quantity, price)
	scale := quantityScale + priceScale
	cents, err := roundToScale(product, scale, 2)
	if err != nil {
		emit(output{State: "error", Code: err.Error()})
		return
	}
	absolute := new(big.Int).Abs(cents)
	integer := new(big.Int).Quo(absolute, big.NewInt(100))
	if len(integer.String()) > 6 {
		emit(output{State: "error", Code: "PRECISION_OVERFLOW"})
		return
	}
	fraction := new(big.Int).Rem(absolute, big.NewInt(100))
	fractionText := fraction.String()
	if len(fractionText) < 2 {
		fractionText = "0" + fractionText
	}
	sign := ""
	if cents.Sign() < 0 {
		sign = "-"
	}
	emit(output{State: "present", Amount: sign + integer.String() + "." + fractionText, Currency: "JPY"})
}

func parseDecimal(value string) (*big.Int, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, 0, fmt.Errorf("empty decimal")
	}
	sign := 1
	if strings.HasPrefix(value, "-") {
		sign = -1
		value = value[1:]
	} else if strings.HasPrefix(value, "+") {
		value = value[1:]
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return nil, 0, fmt.Errorf("invalid decimal")
	}
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if whole == "" && fraction == "" {
		return nil, 0, fmt.Errorf("invalid decimal")
	}
	digits := whole + fraction
	for _, r := range digits {
		if r < '0' || r > '9' {
			return nil, 0, fmt.Errorf("invalid decimal")
		}
	}
	if digits == "" {
		return nil, 0, fmt.Errorf("invalid decimal")
	}
	coefficient, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid decimal")
	}
	if sign < 0 {
		coefficient.Neg(coefficient)
	}
	return coefficient, len(fraction), nil
}

func roundToScale(coefficient *big.Int, from, to int) (*big.Int, error) {
	if from < 0 || to < 0 {
		return nil, fmt.Errorf("invalid scale")
	}
	if from <= to {
		factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(to-from)), nil)
		return new(big.Int).Mul(coefficient, factor), nil
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(from-to)), nil)
	quotient, remainder := new(big.Int).QuoRem(coefficient, divisor, new(big.Int))
	twice := new(big.Int).Abs(remainder)
	twice.Lsh(twice, 1)
	if twice.Cmp(divisor) >= 0 {
		if coefficient.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	return quotient, nil
}

func emit(out output) {
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		os.Exit(1)
	}
}
