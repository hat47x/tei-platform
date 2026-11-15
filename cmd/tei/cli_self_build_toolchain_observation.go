package main

import (
	"encoding/json"
	"strings"
)

// This file records which Go toolchain a build used (SB-C0 §6, toolchain as an
// input the plan does not name).
//
// The plan this entry derives names sources, a builder and a recipe. It does not
// name the toolchain: the builder takes it as -go, a child argument this entry
// forwards without reading. The same plan built with two toolchains produces two
// different artifacts, and every one of the five conditions holds for both
// (plan/evidence/TEI_GO_SELF_BUILD_SEED_REACH_2026-11-13.md). So "the artifact is
// that plan's output" is true only relative to a toolchain the plan is silent on.
//
// Changing the plan is a contract decision. What this file adds is the record: the
// builder says which toolchain it ran, and the entry re-reads that executable so
// the statement is not taken on the builder's word alone. It does not change
// l2b.claimed.

const (
	toolchainObservationMatched    = "matched"
	toolchainObservationMismatched = "mismatched"
	toolchainObservationUnreported = "unreported"
)

const toolchainObservationBoundary = "the builder reported the Go toolchain it ran and this entry re-read the executable at that path; matched means the file there has the bytes the builder reported, not that the builder ran it. When the plan names a toolchain (go.toolchain.closureDigest), the builder's reported closure digest is compared with it: this entry re-reads the executable, not the whole toolchain, so the closure comparison rests on the builder's hashing. When the plan names none, nothing is checked against a declaration and two attempts with the same plan and different toolchains are told apart only by this record. It is not trust in the toolchain"

// selfBuildChildToolchain is the builder's own statement, copied from its report.
type selfBuildChildToolchain struct {
	GoExecutable       string `json:"goExecutable,omitempty"`
	GoExecutableDigest string `json:"goExecutableDigest,omitempty"`
	GoRoot             string `json:"goRoot,omitempty"`
	ClosureDigest      string `json:"closureDigest,omitempty"`
	NotObserved        string `json:"notObserved,omitempty"`
}

// selfBuildToolchainObservation puts the builder's statement beside what the
// entry read itself.
type selfBuildToolchainObservation struct {
	State string `json:"state"`
	// Reported is the builder's statement. It is nil when the builder said
	// nothing, which is what a builder that does not use Go does.
	Reported *selfBuildChildToolchain `json:"reported,omitempty"`
	// EntryGoExecutableDigest is this entry's own reading of the file at the
	// reported path.
	EntryGoExecutableDigest string `json:"entryGoExecutableDigest,omitempty"`
	// NamedByPlan is true when the plan carries go.toolchain.closureDigest. A plan
	// that does not leaves the toolchain outside what the plan describes.
	NamedByPlan bool `json:"namedByPlan"`
	// PlannedClosureDigest is the value the plan names; MatchesPlan is true only
	// when the builder reported that closure digest. PlanReason says why not.
	PlannedClosureDigest string `json:"plannedClosureDigest,omitempty"`
	MatchesPlan          bool   `json:"matchesPlan"`
	PlanReason           string `json:"planReason,omitempty"`
	// Reason says why the state is not matched.
	Reason   string `json:"reason,omitempty"`
	Boundary string `json:"boundary"`
}

// observeChildToolchain compares the builder's statement with the file on disk
// and, when the plan names a toolchain, with the plan.
func observeChildToolchain(reported *selfBuildChildToolchain, planned string, planNames bool) *selfBuildToolchainObservation {
	record := &selfBuildToolchainObservation{
		State:    toolchainObservationUnreported,
		Reported: reported,
		Boundary: toolchainObservationBoundary,
	}
	if planNames {
		record.NamedByPlan = true
		record.PlannedClosureDigest = planned
		switch {
		case reported == nil || strings.TrimSpace(reported.ClosureDigest) == "":
			record.PlanReason = "the plan names Go toolchain " + planned + ", but the builder did not report the toolchain it ran"
		case reported.ClosureDigest != planned:
			record.PlanReason = "the plan names Go toolchain " + planned + ", but the builder reported " + reported.ClosureDigest
		default:
			record.MatchesPlan = true
		}
	}
	switch {
	case reported == nil:
		record.Reason = "the builder did not report a toolchain, so which one produced the artifact is not recorded"
		return record
	case strings.TrimSpace(reported.GoExecutable) == "" || strings.TrimSpace(reported.GoExecutableDigest) == "":
		record.Reason = "the builder reported no Go executable digest" + reasonSuffix(reported.NotObserved)
		return record
	}
	digest, err := digestOfFile(reported.GoExecutable)
	if err != nil {
		record.Reason = "this entry could not read the Go executable the builder reported: " + err.Error()
		return record
	}
	record.EntryGoExecutableDigest = digest
	if digest != reported.GoExecutableDigest {
		record.State = toolchainObservationMismatched
		record.Reason = "the Go executable at the reported path does not have the digest the builder reported"
		return record
	}
	record.State = toolchainObservationMatched
	return record
}

// plannedPlanParameter reads one parameter out of the receipt's plan object, the
// same bytes the plan fingerprint was recomputed from.
func plannedPlanParameter(receipt *PlanReceipt, name string) (string, bool) {
	if receipt == nil || receipt.ReceiptJSON == "" {
		return "", false
	}
	var parsed struct {
		Plan struct {
			Parameters []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"parameters"`
		} `json:"plan"`
	}
	if err := json.Unmarshal([]byte(receipt.ReceiptJSON), &parsed); err != nil {
		return "", false
	}
	for _, param := range parsed.Plan.Parameters {
		if param.Name == name {
			return param.Value, true
		}
	}
	return "", false
}
