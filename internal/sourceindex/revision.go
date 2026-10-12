package sourceindex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// Revision identifies the exact managed source inventory by path and bytes.
// It is neither semantic equivalence nor acceptance/runtime authority. Callers
// must build the index from the actual source snapshot they execute.
func Revision(index Index) string {
	artifacts := append([]Artifact(nil), index.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	hash := sha256.New()
	for _, artifact := range artifacts {
		fmt.Fprintf(hash, "%s\x00%s\n", artifact.Path, artifact.Fingerprint)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}
