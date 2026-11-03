package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/changeanalysis"
)

// This report is deliberately separate from Preview and Review fingerprints.
// It cannot be used as a candidate, a Decision, or an acceptance delta.
type applicationMembershipComparison struct {
	Schema            string                                             `json:"schema"`
	ComparisonOnly    bool                                               `json:"comparisonOnly"`
	StrictChanges     []changeanalysis.SubjectChange                     `json:"strictChanges"`
	SemanticChanges   []changeanalysis.SubjectChange                     `json:"semanticChanges"`
	Membership        *changeanalysis.ApplicationMembershipCandidateDiff `json:"membershipCandidates,omitempty"`
	BeforeDiagnostics []validation.Diagnostic                            `json:"beforeDiagnostics"`
	AfterDiagnostics  []validation.Diagnostic                            `json:"afterDiagnostics"`
	AuthorityLimits   []string                                           `json:"authorityLimits"`
}

func runApplicationMembershipComparison(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compare-application-membership", flag.ContinueOnError)
	flags.SetOutput(stderr)
	beforePath := flags.String("before", "", "変更前のWorkspace")
	afterPath := flags.String("after", "", "変更後のWorkspace")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *beforePath == "" || *afterPath == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "compare-application-membership requires -before and -after; no positional arguments or output file")
		return 2
	}
	before, err := workspace.Load(*beforePath)
	if err != nil {
		fmt.Fprintln(stderr, "before workspace:", err)
		return 2
	}
	after, err := workspace.Load(*afterPath)
	if err != nil {
		fmt.Fprintln(stderr, "after workspace:", err)
		return 2
	}
	report := applicationMembershipComparison{
		Schema: "tei.reference.application-membership-comparison/v0", ComparisonOnly: true,
		BeforeDiagnostics: validation.Validate(before.World), AfterDiagnostics: validation.Validate(after.World),
		AuthorityLimits: []string{"not-an-acceptance-preview-or-decision", "non-semantic-and-non-application-collections-not-evaluated", "source-order-and-target-effects-not-evaluated", "dependency-coverage-not-verified", "workspace-snapshots-not-pinned-or-atomic"},
	}
	exit := 0
	if validation.HasErrors(report.BeforeDiagnostics) || validation.HasErrors(report.AfterDiagnostics) {
		exit = 1 // Do not emit candidate changes for invalid models.
	} else {
		report.SemanticChanges, err = changeanalysis.Diff(before.World, after.World)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		report.StrictChanges, err = changeanalysis.DiffApplications(before.World, after.World)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		candidate, err := changeanalysis.DiffApplicationMembershipCandidates(before.World, after.World)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		report.Membership = &candidate
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return exit
}
