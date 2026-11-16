package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// Plan hand-over: the entry gives the builder the plan the entry itself
// resolved.
//
// Without it, the builder is handed a plan file by the caller, and the entry's
// own in-process plan is only compared with what the builder reports
// (planCrossCheck). Two plans exist, made by two different programs, and the
// one that is built is not the one this entry derived. That is enough to say
// the two agree; it is not enough to say this process resolved the plan that
// was built. A later stage of a self-build needs the second statement: the
// plan must come out of the stage's own TEI process, not out of a resolver the
// earlier stage left behind.
//
// With -hand-over-plan the entry writes its resolved plan to a file only it
// names and passes `-plan <file>` to the child. -plan becomes a reserved child
// argument for that attempt.

const (
	planHandoverHanded       = "handed-over"
	planHandoverNotRequested = "not-requested"

	selfBuildPlanHandoverBoundary = "handed-over means the child was started with -plan pointing at a file this entry wrote from its own in-process resolution, and the file still had those bytes after the child exited. It does not show that the child read that file or built from it: that is what planCrossCheck (the plan fingerprint the child reports) and the artifact checks are for"
)

type selfBuildPlanHandover struct {
	State string `json:"state"`
	// PlanPath is the file the child was pointed at.
	PlanPath string `json:"planPath,omitempty"`
	// PlanBytesDigest is the digest of the bytes this entry wrote.
	PlanBytesDigest string `json:"planBytesDigest,omitempty"`
	// PlanFingerprint is the fingerprint of the plan this entry wrote.
	PlanFingerprint string `json:"planFingerprint,omitempty"`
	// UnchangedAfterChild reports whether the file still had the written bytes
	// once the child (and any rebuild) had exited.
	UnchangedAfterChild *bool  `json:"unchangedAfterChild,omitempty"`
	Reason              string `json:"reason,omitempty"`
	Boundary            string `json:"boundary"`
}

func planHandoverNotAsked() *selfBuildPlanHandover {
	return &selfBuildPlanHandover{
		State:    planHandoverNotRequested,
		Reason:   "-hand-over-plan was not given; the plan the child built from was supplied by the caller, and this entry only compared fingerprints",
		Boundary: selfBuildPlanHandoverBoundary,
	}
}

// writePlanForChild writes the in-process resolved plan to a fresh directory
// and returns the record and the directory to remove afterwards.
func writePlanForChild(receipt *PlanReceipt) (*selfBuildPlanHandover, string, error) {
	if receipt == nil || receipt.State != intentResolutionResolved || receipt.DerivedBy != derivedByEntryInProcess {
		return nil, "", errors.New("-hand-over-plan needs a plan this entry resolved in process (-build-catalog)")
	}
	var resolved buildintent.ResolvedBuildReceipt
	if err := json.Unmarshal([]byte(receipt.ReceiptJSON), &resolved); err != nil {
		return nil, "", fmt.Errorf("cannot decode the in-process plan receipt: %w", err)
	}
	fingerprint, err := plugin.BuildPlanFingerprint(resolved.Plan)
	if err != nil {
		return nil, "", fmt.Errorf("the in-process plan has no fingerprint: %w", err)
	}
	if fingerprint != receipt.PlanFingerprint {
		return nil, "", errors.New("the in-process plan does not have the fingerprint its receipt states")
	}
	raw, err := json.MarshalIndent(resolved.Plan, "", "  ")
	if err != nil {
		return nil, "", err
	}
	raw = append(raw, '\n')
	dir, err := os.MkdirTemp("", "tei-self-build-plan-")
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(path, raw, 0o400); err != nil {
		_ = os.RemoveAll(dir)
		return nil, "", err
	}
	return &selfBuildPlanHandover{
		State:           planHandoverHanded,
		PlanPath:        path,
		PlanBytesDigest: digestOfBytes(raw),
		PlanFingerprint: fingerprint,
		Boundary:        selfBuildPlanHandoverBoundary,
	}, dir, nil
}

// confirmPlanUnchanged re-reads the handed-over file after the child exited.
func confirmPlanUnchanged(record *selfBuildPlanHandover) {
	if record == nil || record.State != planHandoverHanded {
		return
	}
	raw, err := os.ReadFile(record.PlanPath)
	unchanged := err == nil && digestOfBytes(raw) == record.PlanBytesDigest
	record.UnchangedAfterChild = &unchanged
	if !unchanged {
		record.Reason = "the handed-over plan file did not have the written bytes after the child exited"
	}
}
