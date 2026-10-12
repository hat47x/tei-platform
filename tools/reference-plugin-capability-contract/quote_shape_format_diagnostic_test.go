package main

import (
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

// Temporary diagnostic; deleted before PR integration.
func TestTemporaryQuoteShapeFormatterDiff(t *testing.T) {
	before, err := os.ReadFile("order_quote_candidate_shape_test.go")
	if err != nil {
		t.Fatal(err)
	}
	after, err := format.Source(before)
	if err != nil {
		t.Fatal(err)
	}
	var difference []string
	source, formatted := strings.Split(string(before), "\n"), strings.Split(string(after), "\n")
	for i := 0; i < len(source) && i < len(formatted); i++ {
		if source[i] != formatted[i] {
			difference = append(difference, fmt.Sprintf("L%d before=%q after=%q", i+1, source[i], formatted[i]))
		}
	}
	if len(difference) != 0 {
		t.Fatal("exact gofmt differences: " + strings.Join(difference, "\n"))
	}
}
