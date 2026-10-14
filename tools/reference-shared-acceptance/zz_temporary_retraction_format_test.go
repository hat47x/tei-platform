package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

// Temporary exact-diff diagnostic; remove before merge.
func TestZZTemporaryRetractionOperationGofmt(t *testing.T) {
	source, err := os.ReadFile("operation.go")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := format.Source(source)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(source, expected) {
		return
	}
	have := strings.Split(string(source), "\n")
	want := strings.Split(string(expected), "\n")
	for i := 0; i < len(have) && i < len(want); i++ {
		if have[i] != want[i] {
			t.Error(fmt.Sprintf("gofmt operation.go line %d: have=%q want=%q", i+1, have[i], want[i]))
		}
	}
}
