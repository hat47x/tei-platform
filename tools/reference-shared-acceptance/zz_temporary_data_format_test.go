package main

import (
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

func TestZZTemporaryDataCompositionFormat(t *testing.T) {
	b, err := os.ReadFile("data_business_rules_test.go")
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := format.Source(b)
	if err != nil {
		t.Fatal(err)
	}
	have := strings.Split(string(b), "\n")
	want := strings.Split(string(formatted), "\n")
	for i := 0; i < len(have) && i < len(want); i++ {
		if have[i] != want[i] {
			t.Error(fmt.Sprintf("gofmt difference line %d: have=%q want=%q", i+1, have[i], want[i]))
		}
	}
}
