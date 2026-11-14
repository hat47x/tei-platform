package main

import "strings"

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

const toolchainObservationBoundary = "the builder reported the Go toolchain it ran and this entry re-read the executable at that path; matched means the file there has the bytes the builder reported, not that the builder ran it. The plan does not name a toolchain, so nothing here is checked against a declaration, and two attempts with the same plan and different toolchains are told apart only by this record. It is not one of the five conditions, and it is not trust in the toolchain"

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
	// NamedByPlan is always false today: it is here so a reader does not have to
	// know the plan's shape to learn that the toolchain is outside it.
	NamedByPlan bool `json:"namedByPlan"`
	// Reason says why the state is not matched.
	Reason   string `json:"reason,omitempty"`
	Boundary string `json:"boundary"`
}

// observeChildToolchain compares the builder's statement with the file on disk.
func observeChildToolchain(reported *selfBuildChildToolchain) *selfBuildToolchainObservation {
	record := &selfBuildToolchainObservation{
		State:    toolchainObservationUnreported,
		Reported: reported,
		Boundary: toolchainObservationBoundary,
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
