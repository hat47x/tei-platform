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
