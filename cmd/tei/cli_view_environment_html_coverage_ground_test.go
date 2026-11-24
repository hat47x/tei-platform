package main

import (
	"bytes"
	"strings"
	"testing"
)

// The page printed the coverage verdict alone, so a reader of the screen could
// not tell an undeclared binding from a Query the read could not confirm. The
// text form prints the ground (#5811); both renderings of one read must agree,
// so the page prints it too, and only when the read has one.
func TestDestinationHTMLPrintsTheGroundOfCoverage(t *testing.T) {
	t.Run("a partial read shows the reason it stopped at", func(t *testing.T) {
		read := destinationReadFixture()
		read.Coverage = "partial"
		read.CoverageReason = "the Contract Query has an extra condition"
		var out bytes.Buffer
		if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v", code, err)
		}
		if !strings.Contains(out.String(), "coverage reason: the Contract Query has an extra condition") {
			t.Fatalf("the page does not show the reason:\n%s", out.String())
		}
	})

	t.Run("a confirmed read shows the basis it matched", func(t *testing.T) {
		read := destinationReadFixture()
		read.Coverage = "complete"
		read.CoverageBasis = "the Query reads the mapped table by its parent"
		var out bytes.Buffer
		if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v", code, err)
		}
		if !strings.Contains(out.String(), "coverage basis: the Query reads the mapped table by its parent") {
			t.Fatalf("the page does not show the basis:\n%s", out.String())
		}
	})

	t.Run("neither is invented when neither was given", func(t *testing.T) {
		read := destinationReadFixture()
		read.CoverageReason = ""
		read.CoverageBasis = ""
		var out bytes.Buffer
		if code, err := writeDestinationReadHTML(&out, read); err != nil || code != 0 {
			t.Fatalf("code=%d err=%v", code, err)
		}
		page := out.String()
		if strings.Contains(page, "coverage reason:") || strings.Contains(page, "coverage basis:") {
			t.Fatalf("a read with no ground invented one:\n%s", page)
		}
	})
}
