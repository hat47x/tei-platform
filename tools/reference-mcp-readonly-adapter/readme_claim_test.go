package main

import (
	"os"
	"strings"
	"testing"
)

// The adapter README describes which code each entrance uses for a missing
// required argument. When #3733 changed the CLI from INVALID_SUBJECT to
// INVALID_ARGUMENT, the test was updated but this prose was not, so the
// document claimed a code the CLI no longer returns. Pin the claim to the
// behaviour the parity test asserts.
func TestREADMEArgumentCodeClaimMatchesTheParityTest(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	text := string(readme)

	// The CLI side of the claim must name INVALID_ARGUMENT, not the old code.
	claim := "必須引数の欠落が**両方拒否**されること"
	idx := strings.Index(text, claim)
	if idx < 0 {
		t.Fatalf("README no longer states the missing-argument claim; update this test with the wording")
	}
	end := idx + 400
	if end > len(text) {
		end = len(text)
	}
	region := text[idx:end]
	if !strings.Contains(region, "`INVALID_ARGUMENT`") {
		t.Fatalf("README missing-argument claim does not name INVALID_ARGUMENT: %q", region)
	}
	if strings.Contains(region, "CLI では `INVALID_SUBJECT`") {
		t.Fatalf("README still says the CLI answers INVALID_SUBJECT for a missing argument")
	}

	// The parity test is the executable form of the same claim.
	parity, err := os.ReadFile("inspect_trace_parity_test.go")
	if err != nil {
		t.Fatalf("read parity test: %v", err)
	}
	if !strings.Contains(string(parity), `failure != "INVALID_ARGUMENT"`) {
		t.Fatalf("the parity test no longer asserts INVALID_ARGUMENT for the CLI")
	}
}
