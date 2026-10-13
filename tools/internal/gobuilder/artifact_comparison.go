package gobuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ArtifactComparison records observations about two existing Go build outputs.
// Its CandidateDigest is NOT a published artifact identity, lock, or approval.
// Missing, aliased, or unreadable outputs are execution failures, not evidence
// of two completed yet divergent builds.
type ArtifactComparison struct {
	FirstDigest     string
	SecondDigest    string
	Reproducible    bool
	CandidateDigest string
}

// CompareArtifactFiles hashes two separate completed outputs before deciding
// whether they match. The same file (including a hard-link alias) cannot
// establish independent build attempts even if its bytes match. The caller
// retains responsibility for Evidence, acceptance, locks, and publication.
func CompareArtifactFiles(firstPath, secondPath string) (ArtifactComparison, error) {
	first, firstInfo, err := digestArtifactFile(firstPath)
	if err != nil {
		return ArtifactComparison{}, fmt.Errorf("hash first Go build artifact: %w", err)
	}
	second, secondInfo, err := digestArtifactFile(secondPath)
	if err != nil {
		return ArtifactComparison{FirstDigest: first}, fmt.Errorf("hash second Go build artifact: %w", err)
	}
	if os.SameFile(firstInfo, secondInfo) {
		return ArtifactComparison{FirstDigest: first, SecondDigest: second},
			fmt.Errorf("Go build artifacts are the same file, not two independent outputs")
	}
	reproducible, candidate := ResolveReproducibility(first, second)
	return ArtifactComparison{
		FirstDigest:     first,
		SecondDigest:    second,
		Reproducible:    reproducible,
		CandidateDigest: candidate,
	}, nil
}

// digestArtifactFile requires a direct regular-file output. Lstat rejects
// symbolic links; comparing entry identity with the opened handle also
// rejects a path changed between the initial check and opening the file.
// This is not a substitute for OS-level process or file-system isolation.
func digestArtifactFile(path string) (string, os.FileInfo, error) {
	entry, err := os.Lstat(path)
	if err != nil {
		return "", nil, err
	}
	if !entry.Mode().IsRegular() {
		return "", nil, fmt.Errorf("Go build artifact is not a direct regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(entry, info) {
		return "", nil, fmt.Errorf("Go build artifact changed identity before reading: %s", path)
	}
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", nil, err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), info, nil
}
