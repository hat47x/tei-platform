package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// §4.36 makes the builder refuse to build from a definition the plan does not
// name. That refusal is the builder's own behaviour, so a builder that does not
// implement it would still pass. This adds the same check on the entry side: the
// child reports the digest of the file it read, and the entry compares it with
// the digest its own plan records.
//
// It does not prove the child read that file — the digest is still the child's
// statement. What it removes is the case where the two disagree and nobody
// notices, which is what §4.36's builder-side check alone would leave open for a
// child that skips it.

const definitionBindingBoundary = "the entry compared the definition digest the builder reported with the digest its own plan records for the definition source; this checks the builder's statement against the plan, not the act of reading, and it is not L2-b, acceptance, or runtime authority"

const (
	definitionBindingBound  = "bound"
	definitionBindingFailed = "mismatched"
	definitionBindingNotRun = "unperformed"
)

// selfBuildDefinitionBinding records whether the builder's reported definition is
// the one the entry's plan names.
type selfBuildDefinitionBinding struct {
	State string `json:"state"`
	// PlannedDigest comes from the plan the entry holds, not from the child.
	PlannedDigest string `json:"plannedDigest,omitempty"`
	// ReportedDigest is the child's statement about the file it read.
	ReportedDigest string `json:"reportedDigest,omitempty"`
	// Matched is true only when both values are present and equal.
	Matched           bool   `json:"matched"`
	Reason            string `json:"reason,omitempty"`
	UnperformedReason string `json:"notPerformedReason,omitempty"`
	Boundary          string `json:"boundary"`
}

// bindDefinitionToPlan compares the reported definition digest with the one the
// receipt's plan records. A missing value on either side is unperformed rather
// than agreement: an unrecorded input is not a matching input (SC0-18's rule).
func bindDefinitionToPlan(receipt *PlanReceipt, reported string) selfBuildDefinitionBinding {
	record := selfBuildDefinitionBinding{Boundary: definitionBindingBoundary}
	if receipt == nil {
		record.State = definitionBindingNotRun
		record.UnperformedReason = "no resolver receipt was produced, so there is no plan whose definition the builder's input could be compared with"
		return record
	}
	planned, ok := plannedDefinitionSourceDigest(receipt)
	if !ok || planned == "" {
		record.State = definitionBindingNotRun
		record.UnperformedReason = "the plan does not record a digest for its definition source, so there is nothing to compare the builder's input with"
		return record
	}
	record.PlannedDigest = planned
	record.ReportedDigest = strings.TrimSpace(reported)
	if record.ReportedDigest == "" {
		record.State = definitionBindingNotRun
		record.UnperformedReason = "the builder did not report the digest of the definition it read, so its input cannot be compared with the plan"
		return record
	}
	if record.ReportedDigest != planned {
		record.State = definitionBindingFailed
		record.Reason = fmt.Sprintf("the builder read a definition (%s) that is not the one the plan names (%s); the artifact was not built from the planned input", record.ReportedDigest, planned)
		return record
	}
	record.State = definitionBindingBound
	record.Matched = true
	return record
}

// plannedDefinitionSourceDigest reads the digest out of the receipt's plan object,
// which is the same bytes the resolver reported.
func plannedDefinitionSourceDigest(receipt *PlanReceipt) (string, bool) {
	if receipt == nil || receipt.ReceiptJSON == "" {
		return "", false
	}
	var parsed struct {
		Plan struct {
			Sources []struct {
				Kind   string `json:"kind"`
				Digest string `json:"digest"`
			} `json:"sources"`
		} `json:"plan"`
	}
	if err := json.Unmarshal([]byte(receipt.ReceiptJSON), &parsed); err != nil {
		return "", false
	}
	for _, source := range parsed.Plan.Sources {
		if source.Kind == "plugin-definition" && source.Digest != "" {
			return source.Digest, true
		}
	}
	return "", false
}
