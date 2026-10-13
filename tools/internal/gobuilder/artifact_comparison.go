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
// Missing and unreadable outputs are execution failures, not evidence of two
// completed yet divergent builds.
type ArtifactComparison struct {
	FirstDigest     string
	SecondDigest    string
	Reproducible    bool
	CandidateDigest string
}

// CompareArtifactFiles hashes both completed outputs before deciding whether
// they match. The caller retains responsibility for subject-specific Evidence,
// acceptance policy, build locks, and verified publication.
func CompareArtifactFiles(firstPath, secondPath string) (ArtifactComparison, error) {
	first, err := digestArtifactFile(firstPath)
	if err != nil {
		return ArtifactComparison{}, fmt.Errorf("hash first Go build artifact: %w", err)
	}
	second, err := digestArtifactFile(secondPath)
	if err != nil {
		return ArtifactComparison{FirstDigest: first}, fmt.Errorf("hash second Go build artifact: %w", err)
	}
	reproducible, candidate := ResolveReproducibility(first, second)
	return ArtifactComparison{
		FirstDigest:     first,
		SecondDigest:    second,
		Reproducible:    reproducible,
		CandidateDigest: candidate,
	}, nil
}

func digestArtifactFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Go build artifact is not a regular file: %s", path)
	}
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}
