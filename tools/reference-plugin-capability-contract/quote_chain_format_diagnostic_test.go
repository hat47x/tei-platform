package main

import (
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

// Transient diagnostic: discarded after exact gofmt differences are fixed.
func TestTemporaryQuoteChainGofmtDifferences(t *testing.T) {
	before, err := os.ReadFile("order_quote_runtime_chain_test.go")
	if err != nil {
		t.Fatal(err)
	}
	after, err := format.Source(before)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Split(string(before), "\n")
	formatted := strings.Split(string(after), "\n")
	var differences []string
	for i := 0; i < len(source) && i < len(formatted); i++ {
		if source[i] != formatted[i] {
			differences = append(differences, fmt.Sprintf("L%d: BEFORE: %q AFTER: %q", i+1, source[i], formatted[i]))
		}
	}
	if len(differences) > 0 {
		t.Fatal(strings.Join(differences, "\n"))
	}
}
