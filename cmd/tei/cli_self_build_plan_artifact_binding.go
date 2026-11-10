package main

import (
	"encoding/json"
	"strings"

	"tei/internal/plugin"
)

// This file records §4.8's remaining observation (c): the link between the
// plan a builder reported and the artifact it produced.
//
// The plan cross-check (§4.8 item 3) compares two plan fingerprints. It says
// nothing about which artifact came out. This file asks the entry to form an
// exact plugin.BuildArtifactLock from the resolver's plan identity plus the
// builder's reported plan fingerprint and artifact digest, validate it, and
// check that the artifact digest matches the bytes on disk.
//
// What it adds: the plan fingerprint and the artifact digest are recorded as
// one validated binding, and the artifact digest is independently confirmed
// against the file. What it does NOT add: the builder's reading is still its
// own report, and the declaration this entry held is still not provably the
// input the builder used. l2bClaimed stays false, and the boundary says why.

const (
	planArtifactBound       = "bound"
	planArtifactMismatched  = "mismatched"
	planArtifactUnperformed = "unperformed"
)

// planArtifactBinding is §4.8(c): one validated plan-to-artifact binding.
type planArtifactBinding struct {
	State string `json:"state"`
	// EntryPlanFingerprint is the fingerprint this entry derived itself from the
	// plan inside the receipt. It is the basis the binding uses when the builder
	// reports no fingerprint of its own, so one side of the binding no longer
	// depends only on what a child said.
	EntryPlanFingerprint string `json:"entryPlanFingerprint,omitempty"`
	// PlanSideSource says whose fingerprint the lock's plan side carries:
	// "builder-report" when the child reported its own, "entry-derivation" when
	// the entry fell back to its own reading because the child was silent.
	PlanSideSource       string `json:"planSideSource,omitempty"`
	PlanFingerprint      string `json:"planFingerprint,omitempty"`
	ArtifactDigest       string `json:"artifactDigest,omitempty"`
	ArtifactDigestOnDisk string `json:"artifactDigestOnDisk,omitempty"`
	LockFingerprint      string `json:"lockFingerprint,omitempty"`
	Matched              bool   `json:"matched"`
	L2BClaimed           bool   `json:"l2bClaimed"`
	Boundary             string `json:"boundary"`
	UnperformedReason    string `json:"unperformedReason,omitempty"`
	FailureCause         string `json:"failureCause,omitempty"`
	// §4.11: when the child forms its own lock, the entry compares it with the
	// lock it derived. Two independently formed locks that agree mean both
	// readings describe the same binding.
	ChildLockChecked     bool   `json:"childLockChecked"`
	ChildLockValidated   bool   `json:"childLockValidated"`
	ChildLockMatched     bool   `json:"childLockMatched"`
	ChildLockFingerprint string `json:"childLockFingerprint,omitempty"`
}

const planArtifactBoundary = "the entry formed one validated BuildArtifactLock from the resolver's plan identity and the builder child's reported plan fingerprint and artifact digest, and the artifact digest matched the bytes on disk; both bound values are the builder child's self-report, so this is not proof that the builder derived the plan from the declaration this entry held, and not acceptance or runtime authority"

// bindPlanToArtifact forms and validates the lock. It runs only after both
// children finished, so a failure here is a recorded finding, never a
// precondition that could have stopped the build.
func bindPlanToArtifact(receipt *PlanReceipt, entryPlanFingerprint, builderPlanFingerprint, builderArtifactDigest, artifactOnDiskDigest string, childLock *childSelfLock) planArtifactBinding {
	record := planArtifactBinding{
		L2BClaimed: false,
		Boundary:   planArtifactBoundary,
	}
	if receipt == nil {
		record.State = planArtifactUnperformed
		record.UnperformedReason = "no resolver receipt was produced, so there is no plan identity to bind"
		return record
	}
	record.EntryPlanFingerprint = entryPlanFingerprint
	// The plan side of the lock prefers the builder's own report, because that is
	// what the artifact was built from. When the builder is silent, the entry's
	// own derivation is used instead: it is a reading this process made from the
	// receipt, not a value it was told. Which one was used is recorded, so the
	// weaker basis is never mistaken for the stronger one.
	planSide := strings.TrimSpace(builderPlanFingerprint)
	planSideSource := "builder-report"
	if planSide == "" {
		planSide = strings.TrimSpace(entryPlanFingerprint)
		planSideSource = "entry-derivation"
	}
	record.PlanSideSource = planSideSource
	if planSide == "" {
		record.State = planArtifactUnperformed
		record.UnperformedReason = "neither the builder child nor the entry produced a plan fingerprint, so the artifact cannot be bound to a plan"
		return record
	}
	if strings.TrimSpace(builderArtifactDigest) == "" {
		record.State = planArtifactUnperformed
		record.UnperformedReason = "the builder child did not report the artifact digest it produced, so there is nothing to bind"
		return record
	}
	if receipt.PlanFingerprint != "" && planSideSource == "builder-report" && planSide != receipt.PlanFingerprint {
		record.State = planArtifactMismatched
		record.PlanFingerprint = planSide
		record.ArtifactDigest = builderArtifactDigest
		record.ArtifactDigestOnDisk = artifactOnDiskDigest
		record.FailureCause = "the builder child reports a different plan fingerprint than the resolver, so the artifact would be bound to the wrong plan"
		return record
	}
	identity, ok := receiptPlanIdentity(receipt)
	if !ok {
		record.State = planArtifactUnperformed
		record.UnperformedReason = "the receipt does not carry a plan object, so the lock's plan identity cannot be formed"
		return record
	}

	lock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{{
			Plugin:          identity.Plugin,
			OutputKind:      identity.OutputKind,
			Target:          identity.Target,
			Profile:         identity.Profile,
			PlanFingerprint: planSide,
			ArtifactDigest:  builderArtifactDigest,
		}},
	}
	// Validate the lock itself. A digest that is not in the build digest form,
	// or a missing identity field, is a refusal rather than a silent pass.
	if err := lock.Validate(); err != nil {
		record.State = planArtifactUnperformed
		record.PlanFingerprint = planSide
		record.ArtifactDigest = builderArtifactDigest
		record.UnperformedReason = "the plan-to-artifact lock does not validate: " + err.Error()
		return record
	}
	fingerprint, err := plugin.BuildArtifactLockFingerprint(lock)
	if err != nil {
		record.State = planArtifactUnperformed
		record.PlanFingerprint = planSide
		record.ArtifactDigest = builderArtifactDigest
		record.UnperformedReason = "the lock fingerprint could not be derived: " + err.Error()
		return record
	}
	record.PlanFingerprint = planSide
	record.ArtifactDigest = builderArtifactDigest
	record.ArtifactDigestOnDisk = artifactOnDiskDigest
	record.LockFingerprint = fingerprint

	// §4.11: when the child formed and validated its own lock, compare it with
	// the one the entry derived. The entry's identity comes from the receipt's
	// plan; the child's comes from the plan file it loaded. Two independently
	// formed locks that agree mean both readings name the same identity.
	if childLock != nil {
		record.ChildLockFingerprint = childLock.LockFingerprint
		record.ChildLockValidated = childLock.Validated
		record.ChildLockChecked = true
		record.ChildLockMatched = childLock.LockFingerprint != "" && childLock.LockFingerprint == fingerprint
		if !record.ChildLockMatched {
			record.State = planArtifactMismatched
			record.FailureCause = "the child's own lock fingerprint differs from the lock this entry formed, so the two readings do not describe the same binding"
			return record
		}
	}

	// The artifact must be the bytes on disk. Without this the lock would bind
	// a plan to a digest the child merely asserts.
	if strings.TrimSpace(artifactOnDiskDigest) == "" {
		record.State = planArtifactUnperformed
		record.UnperformedReason = "the artifact bytes could not be read, so the lock's artifact side is only a claim"
		return record
	}
	if builderArtifactDigest != artifactOnDiskDigest {
		record.State = planArtifactMismatched
		record.FailureCause = "the child's artifact digest differs from the bytes on disk, so the lock would bind a plan to a different artifact"
		return record
	}
	record.State = planArtifactBound
	record.Matched = true
	return record
}

// receiptPlanIdentity extracts the identity fields a BuildArtifactLock needs.
// It reads the receipt's plan object rather than any summary field, so the lock
// is formed from the same bytes the resolver reported.
type receiptPlanIdentityFields struct {
	Plugin     plugin.Identity
	OutputKind plugin.BuildOutputKind
	Target     string
	Profile    string
}

func receiptPlanIdentity(receipt *PlanReceipt) (receiptPlanIdentityFields, bool) {
	if receipt == nil {
		return receiptPlanIdentityFields{}, false
	}
	var parsed struct {
		Plan json.RawMessage `json:"plan"`
	}
	if receipt.ReceiptJSON == "" {
		return receiptPlanIdentityFields{}, false
	}
	if err := json.Unmarshal([]byte(receipt.ReceiptJSON), &parsed); err != nil {
		return receiptPlanIdentityFields{}, false
	}
	if len(parsed.Plan) == 0 {
		return receiptPlanIdentityFields{}, false
	}
	var plan plugin.BuildPlan
	if err := json.Unmarshal(parsed.Plan, &plan); err != nil {
		return receiptPlanIdentityFields{}, false
	}
	return receiptPlanIdentityFields{
		Plugin:     plan.Plugin,
		OutputKind: plan.OutputKind,
		Target:     plan.Target,
		Profile:    plan.Profile,
	}, true
}

// planArtifactMismatchCause states the mismatch in the record's own words.
func planArtifactMismatchCause(record planArtifactBinding) string {
	return record.FailureCause
}
