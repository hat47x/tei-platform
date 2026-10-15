package main

import (
	"fmt"
	"os"

	"tei/internal/plugin"
)

// publishLockedBuilderOutput is the narrow bridge from a live Builder output
// file to the existing exact-digest CAS publication contract. The expected
// digest comes only from the caller's previously fixed LockedBuild; it is
// never replaced by a digest calculated from the output file.
//
// This does not run a Builder, select a Lock, or grant Host/Trust authority.
// The caller owns the output file's lifetime through this synchronous call.
func (s localArtifactStore) publishLockedBuilderOutput(
	locked plugin.LockedBuild, outputPath string,
) (artifactSetPublication, error) {
	info, err := os.Lstat(outputPath)
	if err != nil {
		return artifactSetPublication{}, fmt.Errorf("inspect locked Builder output: %w", err)
	}
	if !info.Mode().IsRegular() {
		return artifactSetPublication{}, fmt.Errorf("locked Builder output must be a regular file")
	}
	if info.Size() > s.maxBytes {
		return artifactSetPublication{}, fmt.Errorf("locked Builder output exceeds store maxBytes")
	}
	value, err := os.ReadFile(outputPath)
	if err != nil {
		return artifactSetPublication{}, fmt.Errorf("read locked Builder output: %w", err)
	}
	return s.publishExactMembers(
		[]artifactMemberRequest{{Role: "executable", Digest: locked.ArtifactDigest}},
		map[string][]byte{"executable": value},
	)
}
