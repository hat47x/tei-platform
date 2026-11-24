//go:build !tei_repair_only

package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/semantic/validationeval"
)

// A relation-wide answer is worth what its coverage is worth, and the page said
// nothing about it. Measured at 8f7c0d4d5 against a real PostgreSQL: the page
// showed `sales.order.lines.line_number_unique satisfied` with no coverage line
// at all, so a reader could not tell a confirmed answer from one that only
// looked satisfied. The verdict and its ground are now part of the read result.
func TestRelationCoverageRequiresAProofForThisTarget(t *testing.T) {
	target := realization.ContractQueryTarget{Database: "d", Contract: "c", Query: "q"}
	execution := driver.HeaderDetailContractQueryExecution{
		Resolution: driver.ResolvedHeaderDetailContractQuery{
			Binding: realization.HeaderDetailContractQueryBinding{Target: target},
		},
	}
	for _, tc := range []struct {
		name    string
		proof   any
		verdict validationeval.Coverage
		basis   string
		reason  string
	}{
		{
			name:    "a completed proof completes the read",
			proof:   webRelationCoverageProof{Target: target, Complete: true, Basis: "the Query reads the mapped table by its parent"},
			verdict: validationeval.CoverageComplete,
			basis:   "the Query reads the mapped table by its parent",
		},
		{
			name:    "an incomplete proof stays partial and keeps its reason",
			proof:   webRelationCoverageProof{Target: target, Complete: false, Reason: "the Query has an extra condition"},
			verdict: validationeval.CoveragePartial,
			reason:  "the Query has an extra condition",
		},
		{
			name:    "no proof stays partial and explains nothing",
			proof:   nil,
			verdict: validationeval.CoveragePartial,
		},
		{
			name: "a proof for another target does not count",
			proof: webRelationCoverageProof{
				Complete: true,
				Basis:    "some other query",
				Target:   realization.ContractQueryTarget{Database: "d", Contract: "c", Query: "other"},
			},
			verdict: validationeval.CoveragePartial,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			execution.Result.Payload = tc.proof
			coverage, reported := relationCoverage(execution)
			if coverage != tc.verdict {
				t.Fatalf("coverage=%q, want %q", coverage, tc.verdict)
			}
			if reported.Verdict != string(tc.verdict) {
				t.Fatalf("reported verdict=%q, want %q", reported.Verdict, tc.verdict)
			}
			if reported.Basis != tc.basis || reported.Reason != tc.reason {
				t.Fatalf("ground=%+v, want basis=%q reason=%q", reported, tc.basis, tc.reason)
			}
		})
	}
}

// The page must print the verdict beside the relation-wide answers, print the
// ground when the read has one, and not invent one when it has not.
func TestWebPagePrintsCoverageBesideRelationAnswers(t *testing.T) {
	for _, tc := range []struct {
		name     string
		coverage readCoverage
		want     []string
		absent   []string
	}{
		{
			name:     "complete with basis",
			coverage: readCoverage{Verdict: "complete", Basis: "the Query reads the mapped table by its parent"},
			want:     []string{"coverage: <code>complete</code>", "coverage basis: the Query reads the mapped table by its parent"},
		},
		{
			name:     "partial with reason",
			coverage: readCoverage{Verdict: "partial", Reason: "the Query has an extra condition"},
			want:     []string{"coverage: <code>partial</code>", "coverage reason: the Query has an extra condition"},
		},
		{
			name:     "partial with no ground",
			coverage: readCoverage{Verdict: "partial"},
			want:     []string{"coverage: <code>partial</code>"},
			absent:   []string{"coverage reason:", "coverage basis:"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := template.New("custom-submit").Parse(customSubmitHTML)
			if err != nil {
				t.Fatalf("parsing the page template failed: %v", err)
			}
			var body bytes.Buffer
			if err := page.Execute(&body, customSubmitPageData{
				Projection: webTargetProjection{},
				Result: &commandScenarioResult{Read: &readScenarioResult{
					RowAnswers:      []rowAnswerProjection{},
					RelationAnswers: []string{"sales.order.lines.line_number_unique satisfied"},
					Coverage:        tc.coverage,
				}},
			}); err != nil {
				t.Fatalf("rendering failed: %v", err)
			}
			rendered := body.String()
			for _, want := range tc.want {
				if !strings.Contains(rendered, want) {
					t.Fatalf("the page does not print %q:\n%s", want, rendered)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(rendered, absent) {
					t.Fatalf("the page invented %q:\n%s", absent, rendered)
				}
			}
		})
	}
}
