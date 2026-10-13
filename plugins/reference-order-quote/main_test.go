package main

import "testing"

func TestQuoteCalculationRejectsAmbiguousAndIncompleteJSON(t *testing.T) {
	const valid = `{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250},{"sku":"BOOK","quantity":2,"unitPriceMinor":400}]}`
	cases := []struct {
		name  string
		input string
		ok    bool
	}{
		{name: "valid two lines", input: valid, ok: true},
		{name: "empty list", input: `{"currency":"JPY","lines":[]}`},
		{name: "duplicate currency", input: `{"currency":"JPY","currency":"JPY","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1}]}`},
		{name: "duplicate line quantity", input: `{"currency":"JPY","lines":[{"sku":"P","quantity":1,"quantity":2,"unitPriceMinor":1}]}`},
		{name: "case folded currency", input: `{"Currency":"JPY","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1}]}`},
		{name: "case folded price", input: `{"currency":"JPY","lines":[{"sku":"P","quantity":1,"UnitPriceMinor":1}]}`},
		{name: "missing price", input: `{"currency":"JPY","lines":[{"sku":"P","quantity":1}]}`},
		{name: "missing SKU", input: `{"currency":"JPY","lines":[{"quantity":1,"unitPriceMinor":1}]}`},
		{name: "unknown principal", input: `{"currency":"JPY","principalId":"admin","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1}]}`},
		{name: "unknown line field", input: `{"currency":"JPY","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1,"grant":true}]}`},
		{name: "null price", input: `{"currency":"JPY","lines":[{"sku":"P","quantity":1,"unitPriceMinor":null}]}`},
		{name: "fractional quantity", input: `{"currency":"JPY","lines":[{"sku":"P","quantity":1.5,"unitPriceMinor":1}]}`},
		{name: "negative quantity", input: `{"currency":"JPY","lines":[{"sku":"P","quantity":-1,"unitPriceMinor":1}]}`},
		{name: "invalid currency", input: `{"currency":"USD","lines":[{"sku":"P","quantity":1,"unitPriceMinor":1}]}`},
		{name: "array instead of object", input: `[{"currency":"JPY"}]`},
		{name: "null lines", input: `{"currency":"JPY","lines":null}`},
		{name: "trailing JSON", input: valid + `{}`},
		{name: "zero price allowed explicitly", input: `{"currency":"JPY","lines":[{"sku":"FREE","quantity":1,"unitPriceMinor":0}]}`, ok: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := calculate([]byte(tc.input))
			if (err == nil) != tc.ok {
				t.Fatalf("accepted=%t want=%t result=%+v err=%v", err == nil, tc.ok, result, err)
			}
			if tc.name == "valid two lines" &&
				(result.Currency != "JPY" || result.LineCount != 2 || result.SubtotalMinor != 1550) {
				t.Fatalf("business quote calculation incorrect: %+v", result)
			}
		})
	}
}
