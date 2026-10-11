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
	worldBytes, err := json.Marshal(world)
	if err != nil {
		return "", "", fmt.Errorf("serialize normalized World: %w", err)
	}
	sourceBytes, err := json.Marshal(idx)
	if err != nil {
		return "", "", fmt.Errorf("serialize normalized Source Index: %w", err)
	}
	return sourceindex.Fingerprint(worldBytes), sourceindex.Fingerprint(sourceBytes), nil
}

// workspaceTraceFingerprint binds a modelref trace query, the entire Canonical
// World and Source Index to a continuation token. It never grants write rights.
func workspaceTraceFingerprint(worldFP, sourceFP, graphFP string) string {
	return sourceindex.Fingerprint([]byte(
		"tei.operations.trace.snapshot/v1\n" +
			worldFP + "\n" + sourceFP + "\n" + graphFP,
	))
}
