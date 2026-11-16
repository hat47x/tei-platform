package operations

import (
	"encoding/json"
	"fmt"

	"tei/internal/ir"
	"tei/internal/sourceindex"
)

// workspaceProjectionFingerprints identifies the Canonical World and the
// complete normalized Source Index as separate read-only observations.
// File moves may preserve the World fingerprint but change Source references.
// These hashes are not Shared Acceptance revisions or transactional locks.
func workspaceProjectionFingerprints(world ir.World, idx sourceindex.Index) (worldFP, sourceFP string, err error) {
	return WorkspaceProjectionFingerprints(world, idx)
}

// WorkspaceProjectionFingerprints exposes the two read-only observations that
// the JSON projections report as `worldFingerprint` and
// `sourceIndexFingerprint`. A new command must use this function rather than
// compute a third version for the same Workspace (#3345).
//
// Neither value is a Shared Acceptance revision, a lock, or proof that the
// Workspace is valid. A file move may preserve the World while changing the
// Source Index.
func WorkspaceProjectionFingerprints(world ir.World, idx sourceindex.Index) (worldFP, sourceFP string, err error) {
	worldFP, err = worldFingerprint(world)
	if err != nil {
		return "", "", err
	}
	sourceFP, err = sourceIndexFingerprint(idx)
	if err != nil {
		return "", "", err
	}
	return worldFP, sourceFP, nil
}

// worldFingerprint and sourceIndexFingerprint are the two halves of
// workspaceProjectionFingerprints. They only read their argument, so callers
// that hold both inputs may compute them concurrently.
func worldFingerprint(world ir.World) (string, error) {
	worldBytes, err := json.Marshal(world)
	if err != nil {
		return "", fmt.Errorf("serialize normalized World: %w", err)
	}
	return sourceindex.Fingerprint(worldBytes), nil
}

func sourceIndexFingerprint(idx sourceindex.Index) (string, error) {
	sourceBytes, err := json.Marshal(idx)
	if err != nil {
		return "", fmt.Errorf("serialize normalized Source Index: %w", err)
	}
	return sourceindex.Fingerprint(sourceBytes), nil
}

// workspaceTraceFingerprint binds a modelref trace query, the entire Canonical
// World and Source Index to a continuation token. It never grants write rights.
func workspaceTraceFingerprint(worldFP, sourceFP, graphFP string) string {
	return sourceindex.Fingerprint([]byte(
		"tei.operations.trace.snapshot/v1\n" +
			worldFP + "\n" + sourceFP + "\n" + graphFP,
	))
}

// InspectionSnapshotFingerprint binds the Canonical World and the Source Index
// of one inspect read to a single token. It uses its own domain-separation
// prefix rather than reusing trace's, so an inspect token is never mistaken for
// a trace continuation token and a caller cannot pass one where the other is
// required. It never grants write rights.
func InspectionSnapshotFingerprint(worldFP, sourceFP string) string {
	return sourceindex.Fingerprint([]byte(
		"tei.operations.inspect.snapshot/v1\n" +
			worldFP + "\n" + sourceFP + "\n",
	))
}

// CheckInspectionSnapshot compares the snapshot fingerprint an inspect read
// produced with the one the caller held from an earlier read of the same
// subject. An empty expectation is not a check: a caller that supplied no
// fingerprint never claimed the Workspace was unchanged, so there is nothing to
// refute. A mismatch is reported as STALE_SNAPSHOT, the code trace already uses
// for the same question, and means only that the World or the Source Index
// changed between the two reads. It grants neither acceptance nor write
// authority, and it is not a lock: nothing prevents the Workspace from changing
// again after this check passed.
func CheckInspectionSnapshot(worldFP, sourceFP, expected string) error {
	if expected == "" {
		return nil
	}
	if InspectionSnapshotFingerprint(worldFP, sourceFP) != expected {
		return &InspectionError{
			Code:    InspectionStaleSnapshot,
			Message: "the World or Source Index changed since this snapshot fingerprint was read",
		}
	}
	return nil
}
