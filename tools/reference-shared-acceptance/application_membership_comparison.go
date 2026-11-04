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
	Schema                    string                                             `json:"schema"`
	ComparisonOnly            bool                                               `json:"comparisonOnly"`
	WorkspaceValidationPassed bool                                               `json:"workspaceValidationPassed"`
	StrictChanges             []changeanalysis.SubjectChange                     `json:"strictChanges"`
	SemanticChanges           []changeanalysis.SubjectChange                     `json:"semanticChanges"`
	ChangedCollections        []string                                           `json:"changedCollections"`
	SemanticImpact            []changeanalysis.ImpactEdge                        `json:"semanticImpact,omitempty"`
	SemanticImpactUnresolved  []changeanalysis.Unresolved                        `json:"semanticImpactUnresolved,omitempty"`
	Membership                *changeanalysis.ApplicationMembershipCandidateDiff `json:"membershipCandidates,omitempty"`
	ComparisonError           string                                             `json:"comparisonError,omitempty"`
	BeforeDiagnostics         []validation.Diagnostic                            `json:"beforeDiagnostics"`
	AfterDiagnostics          []validation.Diagnostic                            `json:"afterDiagnostics"`
	AuthorityLimits           []string                                           `json:"authorityLimits"`
}

func runApplicationMembershipComparison(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compare-application-membership", flag.ContinueOnError)
	flags.SetOutput(stderr)
	beforePath := flags.String("before", "", "変更前のWorkspace")
	afterPath := flags.String("after", "", "変更後のWorkspace")
	unvalidated := flags.Bool("include-unvalidated-candidates", false, "検証エラーを保持し、非成功のまま限定候補を表示")
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
	report.WorkspaceValidationPassed = !validation.HasErrors(report.BeforeDiagnostics) && !validation.HasErrors(report.AfterDiagnostics)
	if !report.WorkspaceValidationPassed {
		exit = 1
		report.AuthorityLimits = append(report.AuthorityLimits, "workspace-validation-errors-candidates-unverified")
	}
	if report.WorkspaceValidationPassed || *unvalidated {
		compare := func() error {
			semantic, err := changeanalysis.Diff(before.World, after.World)
			if err != nil {
				return err
			}
			strict, err := changeanalysis.DiffApplications(before.World, after.World)
			if err != nil {
				return err
			}
			candidate, err := changeanalysis.DiffApplicationMembershipCandidates(before.World, after.World)
			if err != nil {
				return err
			}
			collections, _, err := changeanalysis.CollectionDiff(before.World, after.World)
			if err != nil {
				return err
			}
			report.ChangedCollections = collections
			report.SemanticChanges, report.StrictChanges, report.Membership = semantic, strict, &candidate
			if len(semantic) != 0 {
				report.SemanticImpact, report.SemanticImpactUnresolved = changeanalysis.ImpactWithBefore(before.World, after.World, semantic)
			}
			return nil
		}
		if err := compare(); err != nil {
			if report.WorkspaceValidationPassed {
				fmt.Fprintln(stderr, err)
				return 2
			}
			report.ComparisonError = err.Error()
		}
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return exit
}
