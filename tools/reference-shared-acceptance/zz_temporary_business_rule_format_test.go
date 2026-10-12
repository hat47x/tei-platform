package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Temporary: removed before merging after exact gofmt differences are fixed.
func TestZZTemporaryBusinessRuleFormatDiagnosis(t *testing.T) {
	files := []string{
		"business_rule_add_test.go",
		filepath.Join("..", "internal", "changeanalysis", "business_rules.go"),
		filepath.Join("..", "internal", "changeanalysis", "business_rules_test.go"),
	}
	for _, path := range files {
		input, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		formatted, err := format.Source(input)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(input, formatted) {
			continue
		}
		have, want := strings.Split(string(input), "\n"), strings.Split(string(formatted), "\n")
		for i := 0; i < len(have) && i < len(want); i++ {
			if have[i] != want[i] {
				t.Error(fmt.Sprintf("gofmt %s line %d: have %q want %q", path, i+1, have[i], want[i]))
			}
		}
	}
}
