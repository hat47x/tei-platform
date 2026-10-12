package webcomposition

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

func TestTemporaryExactQuoteFormattingDiagnostic(t *testing.T) {
	src, err := os.ReadFile("canonical_quote_authorization_test.go")
	if err != nil {
		t.Fatal(err)
	}
	dst, err := format.Source(src)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(src, dst) {
		t.Fatal("no formatting difference")
	}
	before, after := strings.Split(string(src), "\n"), strings.Split(string(dst), "\n")
	var differences []string
	for i := 0; i < len(before) && i < len(after); i++ {
		if before[i] != after[i] {
			differences = append(differences, fmt.Sprintf("%d: BEFORE: %q AFTER: %q", i+1, before[i], after[i]))
		}
	}
	t.Fatal(strings.Join(differences, "\n"))
}
