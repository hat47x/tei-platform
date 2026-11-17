package businessprocedure

import (
	"crypto/sha256"
	"encoding/hex"
)

// fingerprintOf returns a sha256 digest of the bytes, in the same shape the
// Source Index uses for an artifact.
func fingerprintOf(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
