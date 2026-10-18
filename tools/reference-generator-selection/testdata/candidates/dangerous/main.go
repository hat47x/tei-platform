package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
)

// Candidate: dangerous. It deliberately uses float64, maps missing inputs to
// zero, ignores currency and unit mismatches, and skips precision checks. The
// selection harness must reject it from the fixed fixtures.

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
	quantity := 0.0
	if in.Quantity != nil {
		parsed, err := strconv.ParseFloat(*in.Quantity, 64)
		if err != nil {
			emit(output{State: "error", Code: "DECIMAL_INVALID"})
			return
		}
		quantity = parsed
	}
	price := 0.0
	if in.UnitPrice != nil {
		parsed, err := strconv.ParseFloat(*in.UnitPrice, 64)
		if err != nil {
			emit(output{State: "error", Code: "DECIMAL_INVALID"})
			return
		}
		price = parsed
	}
	amount := math.Round(quantity*price*100) / 100
	emit(output{State: "present", Amount: fmt.Sprintf("%.2f", amount), Currency: "JPY"})
}

func emit(out output) {
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		os.Exit(1)
	}
}
