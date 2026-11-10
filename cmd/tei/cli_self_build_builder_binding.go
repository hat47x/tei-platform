package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// The plan names its builder by digest (plugin.BuildPlan.Builder.ArtifactDigest,
// resolved from the catalog), and the entry digests the file it starts (-child).
// Until this record existed the two were never compared: an attempt could derive
// a plan that declares builder D, start a file with bytes E, and still claim that
// the plan produced the artifact. A wrapper or proxy named as -child is the same
// case — the file started is not the builder the plan declares.
//
// This compares the two digests the entry already holds. Neither value comes from
// the child. It answers one question only: is the file this entry asked the OS to
// start, as read just before starting it, the builder the plan declares?
//
// What it cannot answer is stated in the boundary rather than left to a reader:
// the bytes are read and then started by path (not atomically), and whatever the
// started file itself launches is that builder's behaviour, not a second identity
// this entry observes.

const builderBindingBoundary = "the entry compared the digest of the file it started (-child, read from disk immediately before starting it) with the builder artifact digest its own plan declares; neither value is the child's statement. A match shows the named file had the declared builder's bytes when it was read. It does not show that the process image the OS loaded is those bytes (the file is read and then started by path, not atomically), and it says nothing about other executables the started file launches: a declared builder that delegates to another program is still one declared builder. It is not acceptance, runtime authority, or a trust judgement about the catalog or the builder"

const (
	builderBindingBound      = "bound"
	builderBindingMismatched = "mismatched"
	builderBindingNotRun     = "unperformed"
)

// selfBuildBuilderBinding records whether the file the entry started is the
// builder the entry's plan declares.
type selfBuildBuilderBinding struct {
	State string `json:"state"`
	// DeclaredDigest comes from the plan the entry holds (plan.builder.artifactDigest).
	DeclaredDigest string `json:"declaredBuilderDigest,omitempty"`
	// StartedFileDigest is the entry's own digest of the -child file.
	StartedFileDigest string `json:"startedFileDigest,omitempty"`
	// Matched is true only when both values are present and equal.
	Matched           bool   `json:"matched"`
	Reason            string `json:"reason,omitempty"`
	UnperformedReason string `json:"notPerformedReason,omitempty"`
	Boundary          string `json:"boundary"`
}

// bindBuilderToPlan compares the digest of the started file with the builder the
// receipt's plan declares. A missing value on either side is unperformed rather
// than agreement: an unrecorded builder is not the declared builder (SC0-18's rule).
func bindBuilderToPlan(receipt *PlanReceipt, startedFileDigest string) selfBuildBuilderBinding {
	record := selfBuildBuilderBinding{Boundary: builderBindingBoundary}
	if receipt == nil {
		record.State = builderBindingNotRun
		record.UnperformedReason = "no plan was derived, so no builder is declared for the started file to be compared with"
		return record
	}
	declared, ok := plannedBuilderArtifactDigest(receipt)
	if !ok || declared == "" {
		record.State = builderBindingNotRun
		record.UnperformedReason = "the plan does not record a builder artifact digest, so there is nothing to compare the started file with"
		return record
	}
	record.DeclaredDigest = declared
	record.StartedFileDigest = strings.TrimSpace(startedFileDigest)
	if record.StartedFileDigest == "" {
		record.State = builderBindingNotRun
		record.UnperformedReason = "the digest of the started file was not recorded, so it cannot be compared with the declared builder"
		return record
	}
	if !strings.EqualFold(record.StartedFileDigest, declared) {
		record.State = builderBindingMismatched
		record.Reason = fmt.Sprintf("the file this entry started (%s) is not the builder the plan declares (%s); a wrapper, a proxy or another build of the builder is a different file", record.StartedFileDigest, declared)
		return record
	}
	record.State = builderBindingBound
	record.Matched = true
	return record
}

// plannedBuilderArtifactDigest reads the builder digest out of the receipt's plan
// object, which is the same bytes the plan fingerprint was recomputed from.
func plannedBuilderArtifactDigest(receipt *PlanReceipt) (string, bool) {
	if receipt == nil || receipt.ReceiptJSON == "" {
		return "", false
	}
	var parsed struct {
		Plan struct {
			Builder struct {
				ArtifactDigest string `json:"artifactDigest"`
			} `json:"builder"`
		} `json:"plan"`
	}
	if err := json.Unmarshal([]byte(receipt.ReceiptJSON), &parsed); err != nil {
		return "", false
	}
	digest := strings.TrimSpace(parsed.Plan.Builder.ArtifactDigest)
	return digest, digest != ""
}
