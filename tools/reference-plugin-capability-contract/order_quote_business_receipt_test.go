package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

// verifyQuoteReceiptForTest is a consumer-side, TEST-ONLY independent oracle
// for the unaccepted quote candidate. It is not a production policy engine,
// a provider conformance certificate or a substitute for an accepted model.
// The separate arithmetic recomputation is intentional: Shape only verifies
// structure and the provider's own subtotal must not certify itself.
func verifyQuoteReceiptForTest(request, result []byte) error {
	requestShape, err := quoteCandidateShapeForReceipt("request-v0.json")
	if err != nil {
		return err
	}
	resultShape, err := quoteCandidateShapeForReceipt("result-v0.json")
	if err != nil {
		return err
	}
	if err := validateInvocationPayload(requestShape, request); err != nil {
		return fmt.Errorf("request shape: %w", err)
	}
	if err := validateInvocationPayload(resultShape, result); err != nil {
		return fmt.Errorf("result shape: %w", err)
	}
	var input struct {
		Currency string `json:"currency"`
		Lines    []struct {
			SKU            string `json:"sku"`
			Quantity       int64  `json:"quantity"`
			UnitPriceMinor int64  `json:"unitPriceMinor"`
		} `json:"lines"`
	}
	var output struct {
		Currency      string `json:"currency"`
		LineCount     int    `json:"lineCount"`
		SubtotalMinor int64  `json:"subtotalMinor"`
	}
	if err := json.Unmarshal(request, &input); err != nil {
		return fmt.Errorf("invalid quote request: %w", err)
	}
	if err := json.Unmarshal(result, &output); err != nil {
		return fmt.Errorf("invalid quote result: %w", err)
	}
	if input.Currency != "JPY" || len(input.Lines) < 1 || len(input.Lines) > 50 {
		return errors.New("unsupported quote business input")
	}
	sum := big.NewInt(0)
	for _, line := range input.Lines {
		if strings.TrimSpace(line.SKU) == "" || len(line.SKU) > 64 ||
			line.Quantity < 1 || line.Quantity > 100 ||
			line.UnitPriceMinor < 0 || line.UnitPriceMinor > 1_000_000_000 {
			return errors.New("invalid quote business line")
		}
		product := new(big.Int).Mul(big.NewInt(line.Quantity), big.NewInt(line.UnitPriceMinor))
		sum.Add(sum, product)
	}
	if sum.Cmp(big.NewInt(1_000_000_000_000)) > 0 {
		return errors.New("quote business total exceeds ceiling")
	}
	if output.Currency != input.Currency ||
		output.LineCount != len(input.Lines) ||
		output.SubtotalMinor < 0 ||
		big.NewInt(output.SubtotalMinor).Cmp(sum) != 0 {
		return errors.New("quote semantic receipt mismatch")
	}
	return nil
}

func quoteCandidateShapeForReceipt(name string) (InvocationShapeArtifact, error) {
	// Use the exact same candidate file, not a second handwritten JSON schema.
	path := "../../reference/candidates/order-quote/shapes/" + name
	raw, err := os.ReadFile(path)
	if err != nil {
		return InvocationShapeArtifact{}, err
	}
	var artifact InvocationShapeArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return InvocationShapeArtifact{}, err
	}
	if err := validateInvocationShapeArtifact(artifact); err != nil {
		return InvocationShapeArtifact{}, err
	}
	return artifact, nil
}

func TestQuoteCandidateBusinessReceiptRejectsStructurallyValidFalseClaims(t *testing.T) {
	const request = `{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250},{"sku":"BOOK","quantity":2,"unitPriceMinor":400}]}`
	cases := []struct {
		name    string
		request string
		result  string
		wantOK  bool
	}{
		{"real subtotal", request, `{"currency":"JPY","lineCount":2,"subtotalMinor":1550}`, true},
		{"incorrect arithmetic", request, `{"currency":"JPY","lineCount":2,"subtotalMinor":1}`, false},
		{"incorrect line count", request, `{"currency":"JPY","lineCount":1,"subtotalMinor":1550}`, false},
		{"incorrect currency", request, `{"currency":"USD","lineCount":2,"subtotalMinor":1550}`, false},
		{"negative claimed subtotal", request, `{"currency":"JPY","lineCount":2,"subtotalMinor":-1550}`, false},
		{"numeric string result", request, `{"currency":"JPY","lineCount":2,"subtotalMinor":"1550"}`, false},
		{"zero-priced line", `{"currency":"JPY","lines":[{"sku":"FREE","quantity":1,"unitPriceMinor":0}]}`, `{"currency":"JPY","lineCount":1,"subtotalMinor":0}`, true},
		{"fractional quantity", `{"currency":"JPY","lines":[{"sku":"P","quantity":1.5,"unitPriceMinor":1}]}`, `{"currency":"JPY","lineCount":1,"subtotalMinor":1}`, false},
		{"negative quantity", `{"currency":"JPY","lines":[{"sku":"P","quantity":-1,"unitPriceMinor":1}]}`, `{"currency":"JPY","lineCount":1,"subtotalMinor":-1}`, false},
		{"unsupported currency", `{"currency":"USD","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1}]}`, `{"currency":"USD","lineCount":1,"subtotalMinor":1}`, false},
		{"over ceiling", `{"currency":"JPY","lines":[{"sku":"A","quantity":100,"unitPriceMinor":1000000000},{"sku":"B","quantity":100,"unitPriceMinor":1000000000},{"sku":"C","quantity":100,"unitPriceMinor":1000000000},{"sku":"D","quantity":100,"unitPriceMinor":1000000000},{"sku":"E","quantity":100,"unitPriceMinor":1000000000},{"sku":"F","quantity":100,"unitPriceMinor":1000000000},{"sku":"G","quantity":100,"unitPriceMinor":1000000000},{"sku":"H","quantity":100,"unitPriceMinor":1000000000},{"sku":"I","quantity":100,"unitPriceMinor":1000000000},{"sku":"J","quantity":100,"unitPriceMinor":1000000000},{"sku":"K","quantity":100,"unitPriceMinor":1000000000}]}`, `{"currency":"JPY","lineCount":11,"subtotalMinor":1100000000000}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyQuoteReceiptForTest([]byte(tc.request), []byte(tc.result))
			if (err == nil) != tc.wantOK {
				t.Fatalf("accepted=%t want=%t error=%v", err == nil, tc.wantOK, err)
			}
		})
	}
}
