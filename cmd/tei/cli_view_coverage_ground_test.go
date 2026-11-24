package main

import (
	"bytes"
	"strings"
	"testing"
)

// Coverage is a verdict. The text form printed the verdict alone, so a reader
// of it could not tell an undeclared binding from a Query the read could not
// confirm, although the same read's JSON carried the reason. The reason and the
// basis are now printed with the verdict, and only when there is one.
func TestTextFormPrintsTheGroundOfCoverage(t *testing.T) {
	t.Run("a partial read prints the reason it stopped at", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := writeDestinationReadText(&out, destinationRead{
			Application: "a", Pattern: "p", Environment: "e", Relation: "r",
			Coverage:       "partial",
			CoverageReason: "the Contract Query has an extra condition",
		}); err != nil || code != 0 {
			t.Fatalf("text rendering failed: code=%d err=%v", code, err)
		}
		if !strings.Contains(out.String(), "coverageReason the Contract Query has an extra condition\n") {
			t.Fatalf("the reason is not on its own line:\n%s", out.String())
		}
	})

	t.Run("a confirmed read prints the basis it matched", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := writeDestinationReadText(&out, destinationRead{
			Application: "a", Pattern: "p", Environment: "e", Relation: "r",
			Coverage:      "complete",
			CoverageBasis: "the Query reads the mapped table by its parent",
		}); err != nil || code != 0 {
			t.Fatalf("text rendering failed: code=%d err=%v", code, err)
		}
		if !strings.Contains(out.String(), "coverageBasis the Query reads the mapped table by its parent\n") {
			t.Fatalf("the basis is not on its own line:\n%s", out.String())
		}
	})

	t.Run("neither is invented when neither was given", func(t *testing.T) {
		var out bytes.Buffer
		if code, err := writeDestinationReadText(&out, destinationRead{
			Application: "a", Pattern: "p", Environment: "e", Relation: "r",
			Coverage: "partial",
		}); err != nil || code != 0 {
			t.Fatalf("text rendering failed: code=%d err=%v", code, err)
		}
		text := out.String()
		if strings.Contains(text, "coverageReason") || strings.Contains(text, "coverageBasis") {
			t.Fatalf("a read with no ground invented one:\n%s", text)
		}
	})
}
