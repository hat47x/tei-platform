// Package operationdriftcheck is a read-only reference for the W6 question in
// plan/design/OPERATION_SURFACE_DESIGN_2026-11.md: can a stored operation
// surface be held to the meaning it was derived from?
//
// It compares only the two fingerprint values the surface already carries
// (WorldFingerprint and ModelRevision) against a caller-supplied expectation.
// It does not re-derive the surface, does not re-read the Workspace, does not
// write anything, and does not decide whether such a check should be on by
// default. A mismatch is reported as STALE_OPERATION_SURFACE.
package operationdriftcheck

import (
	"crypto/sha256"
	"fmt"

	"tei/internal/operations"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.operation-drift-check/v0"
	// Authority states what the report is allowed to claim.
	Authority = "derived-read-only"

	// StaleOperationSurface is the diagnostic code for a mismatch. It reuses
	// the vocabulary of the design's 10章 and is deliberately distinct from
	// the existing STALE_SNAPSHOT, which covers continued trace pages.
	StaleOperationSurface = "STALE_OPERATION_SURFACE"
)

// rulePolicy is covered by the fingerprint. Changing what is compared without
// changing this text must fail review.
const rulePolicy = `operation-drift-check/v0
compared: OperationSurface.WorldFingerprint and OperationSurface.ModelRevision
not compared: OperationSurface.Basis (it describes the persistence mapping, not
              the whole meaning, so an attribute-only change leaves it equal)
not checked: the surface's own rule fingerprint, its operations, its gaps
on mismatch: STALE_OPERATION_SURFACE, and the caller must not proceed
`

// Expectation is what a caller stored from an earlier surface. Empty values
// mean "this caller does not check that value"; they are not treated as
// agreement.
type Expectation struct {
	WorldFingerprint string
	ModelRevision    string
}

// Finding is one compared value.
type Finding struct {
	Value    string `json:"value"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Match    bool   `json:"match"`
	// Reason explains a mismatch in one line. It is empty when Match is true.
	Reason string `json:"reason,omitempty"`
}

// Report is the whole comparison.
type Report struct {
	Schema          string    `json:"schema"`
	Authority       string    `json:"authority"`
	RuleFingerprint string    `json:"ruleFingerprint"`
	Application     string    `json:"application"`
	Status          string    `json:"status"`
	Findings        []Finding `json:"findings"`
	// Diagnostic is set only when Status is stale. A caller that only reads
	// the code can stop without parsing the findings.
	Diagnostic string `json:"diagnostic,omitempty"`
	// Notes record what this comparison does not establish.
	Notes []string `json:"notes"`
}

// Status values.
const (
	StatusCurrent = "current"
	StatusStale   = "stale"
	// StatusUnchecked means the caller supplied nothing to compare against.
	// It is not a pass: a surface with no expectation was never checked.
	StatusUnchecked = "unchecked"
)

// Check compares the surface's fingerprints to the expectation. It never
// derives a new surface, so it cannot detect that the World changed while the
// caller held this surface; it detects that the surface and the expectation
// disagree.
func Check(surface operations.OperationSurface, expected Expectation) Report {
	report := Report{
		Schema:          Schema,
		Authority:       Authority,
		RuleFingerprint: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))),
		Application:     surface.Application,
		Findings:        []Finding{},
		Notes: []string{
			"a match means the two fingerprints agree, not that the current meaning was re-read",
			"Basis is not compared: an attribute-only change leaves it equal",
			"no Workspace is read and nothing is written",
			"whether this check runs by default is undecided (design 15章 未決4)",
		},
	}

	compared := false
	if expected.WorldFingerprint != "" {
		compared = true
		report.Findings = append(report.Findings, compare(
			"worldFingerprint", expected.WorldFingerprint, surface.WorldFingerprint,
			"the surface was derived from a different World"))
	}
	if expected.ModelRevision != "" {
		compared = true
		report.Findings = append(report.Findings, compare(
			"modelRevision", expected.ModelRevision, surface.ModelRevision,
			"the management Source set changed since the surface was derived"))
	}
	if !compared {
		report.Status = StatusUnchecked
		return report
	}

	report.Status = StatusCurrent
	for _, finding := range report.Findings {
		if !finding.Match {
			report.Status = StatusStale
			report.Diagnostic = StaleOperationSurface
			return report
		}
	}
	return report
}

func compare(value, expected, actual, reason string) Finding {
	f := Finding{Value: value, Expected: expected, Actual: actual, Match: expected == actual}
	if !f.Match {
		f.Reason = reason
	}
	return f
}

// Precondition is the one line a caller can reuse in an error. It is empty
// when the surface is current or unchecked.
func (r Report) Precondition() string {
	if r.Status != StatusStale {
		return ""
	}
	return r.Diagnostic + ": " + r.Findings[0].Value + " changed since the surface was derived"
}
