package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const selfBuildMismatchSchema = "tei.reference.self-build-reproducibility-failure/v0"

// Self-build output comparisons are distinct from build execution failures.
// A mismatch is reportable evidence but never an accepted artifact or lock.
type selfBuildMismatchEvidence struct {
	Schema               string   `json:"schema"`
	Stage                string   `json:"stage"`
	Outcome              string   `json:"outcome"`
	FirstArtifactDigest  string   `json:"firstArtifactDigest"`
	SecondArtifactDigest string   `json:"secondArtifactDigest"`
	Reproducible         bool     `json:"reproducible"`
	ArtifactDigest       string   `json:"artifactDigest"`
	ArtifactAccepted     bool     `json:"artifactAccepted"`
	ArtifactExportAllowed bool    `json:"artifactExportAllowed"`
	ArtifactLockAllowed  bool     `json:"artifactLockAllowed"`
	AuthorityLimits      []string `json:"authorityLimits"`
}

type selfBuildMismatchError struct {
	Evidence selfBuildMismatchEvidence
}

func (e *selfBuildMismatchError) Error() string {
	return fmt.Sprintf("%s is not reproducible: %s != %s",
		e.Evidence.Stage, e.Evidence.FirstArtifactDigest, e.Evidence.SecondArtifactDigest)
}

func nonReproducibleSelfBuild(stage, first, second string) error {
	return &selfBuildMismatchError{Evidence: selfBuildMismatchEvidence{
		Schema:               selfBuildMismatchSchema,
		Stage:                stage,
		Outcome:              "reproducibility-failure",
		FirstArtifactDigest:  first,
		SecondArtifactDigest: second,
		Reproducible:         false,
		ArtifactDigest:       "",
		ArtifactAccepted:     false,
		ArtifactExportAllowed: false,
		ArtifactLockAllowed:  false,
		AuthorityLimits: []string{
			"mismatch-evidence-is-not-build-success",
			"mismatch-evidence-does-not-produce-artifact-lock",
			"mismatch-evidence-does-not-grant-artifact-export",
		},
	}}
}

// Only completed byte comparisons produce public mismatch evidence.
// Ordinary build errors remain stderr-only and fail with a nonzero exit.
func emitSelfBuildMismatchEvidence(out io.Writer, err error) (bool, error) {
	var mismatch *selfBuildMismatchError
	if !errors.As(err, &mismatch) {
		return false, nil
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(mismatch.Evidence); err != nil {
		return true, fmt.Errorf("write non-reproducibility Evidence: %w", err)
	}
	return true, nil
}
