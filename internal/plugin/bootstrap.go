package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const BootstrapConvergenceFormatVersion = 1

type BootstrapStageName string

const (
	BootstrapStage1 BootstrapStageName = "stage1"
	BootstrapStage2 BootstrapStageName = "stage2"
)

// BootstrapStageEvidence records one reproducible builder-construction stage.
// Builder is the exact builder that executed the stage. Output is the exact
// builder artifact produced by the stage. SourceClosureDigest must describe the
// same authoritative source closure across stage1 and stage2.
type BootstrapSeedEvidence struct {
	Builder       BuilderIdentity `json:"builder"`
	TrustBoundary string          `json:"trustBoundary"`
}

type BootstrapStageEvidence struct {
	Stage               BootstrapStageName `json:"stage"`
	Builder             BuilderIdentity    `json:"builder"`
	SourceClosureDigest string             `json:"sourceClosureDigest"`
	// BuildInputFingerprint identifies the semantic inputs that must remain
	// comparable across bootstrap stages. Exact executing builder / interpreter
	// identities are stage-specific and belong in Builder, BuildPlanFingerprint,
	// and the Reference execution Evidence rather than this common fingerprint.
	BuildInputFingerprint string          `json:"buildInputFingerprint"`
	BuildPlanFingerprint  string          `json:"buildPlanFingerprint"`
	Output                BuilderIdentity `json:"output"`
}

// BootstrapConvergenceEvidence is read-only Evidence for SB-3. It does not
// execute a builder and does not grant Acceptance or Runtime Authority.
type BootstrapConvergenceEvidence struct {
	FormatVersion int                    `json:"formatVersion"`
	Stage0Seed    BootstrapSeedEvidence  `json:"stage0Seed"`
	Stage1        BootstrapStageEvidence `json:"stage1"`
	Stage2        BootstrapStageEvidence `json:"stage2"`
}

type BootstrapConvergenceResult struct {
	EvidenceFingerprint  string `json:"evidenceFingerprint"`
	Converged            bool   `json:"converged"`
	Stage1ArtifactDigest string `json:"stage1ArtifactDigest"`
	Stage2ArtifactDigest string `json:"stage2ArtifactDigest"`
}

func (e BootstrapConvergenceEvidence) Validate() error {
	if e.FormatVersion != BootstrapConvergenceFormatVersion {
		return fmt.Errorf("unsupported bootstrap convergence format %d", e.FormatVersion)
	}
	if err := e.Stage0Seed.Builder.Validate(); err != nil {
		return fmt.Errorf("stage0 seed: %w", err)
	}
	if strings.TrimSpace(e.Stage0Seed.TrustBoundary) == "" {
		return fmt.Errorf("stage0 seed trust boundary is required")
	}
	if err := validateBootstrapStage(e.Stage1, BootstrapStage1); err != nil {
		return err
	}
	if err := validateBootstrapStage(e.Stage2, BootstrapStage2); err != nil {
		return err
	}
	if e.Stage1.Builder != e.Stage0Seed.Builder {
		return fmt.Errorf("stage1 builder does not match exact stage0 seed")
	}
	if e.Stage2.Builder != e.Stage1.Output {
		return fmt.Errorf("stage2 builder does not match exact stage1 output")
	}
	if e.Stage1.SourceClosureDigest != e.Stage2.SourceClosureDigest {
		return fmt.Errorf("stage1 and stage2 source closures differ")
	}
	if e.Stage1.BuildInputFingerprint != e.Stage2.BuildInputFingerprint {
		return fmt.Errorf("stage1 and stage2 build inputs differ beyond builder identity")
	}
	if e.Stage1.Output.ID != e.Stage2.Output.ID {
		return fmt.Errorf("stage1 and stage2 output builder ids differ")
	}
	if e.Stage1.Output.Version != e.Stage2.Output.Version {
		return fmt.Errorf("stage1 and stage2 output builder versions differ")
	}
	if e.Stage1.Output.ContractVersion != e.Stage2.Output.ContractVersion {
		return fmt.Errorf("stage1 and stage2 output builder contracts differ")
	}
	return nil
}

func BootstrapConvergenceFingerprint(e BootstrapConvergenceEvidence) (string, error) {
	if err := e.Validate(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func EvaluateBootstrapConvergence(e BootstrapConvergenceEvidence) (BootstrapConvergenceResult, error) {
	fingerprint, err := BootstrapConvergenceFingerprint(e)
	if err != nil {
		return BootstrapConvergenceResult{}, err
	}
	return BootstrapConvergenceResult{
		EvidenceFingerprint:  fingerprint,
		Converged:            e.Stage1.Output.ArtifactDigest == e.Stage2.Output.ArtifactDigest,
		Stage1ArtifactDigest: e.Stage1.Output.ArtifactDigest,
		Stage2ArtifactDigest: e.Stage2.Output.ArtifactDigest,
	}, nil
}

func validateBootstrapStage(stage BootstrapStageEvidence, expected BootstrapStageName) error {
	if stage.Stage != expected {
		return fmt.Errorf("bootstrap stage must be %s, got %s", expected, stage.Stage)
	}
	if err := stage.Builder.Validate(); err != nil {
		return fmt.Errorf("%s builder: %w", expected, err)
	}
	if err := stage.Output.Validate(); err != nil {
		return fmt.Errorf("%s output: %w", expected, err)
	}
	if err := validateBuildDigest(string(expected)+" source closure digest", stage.SourceClosureDigest); err != nil {
		return err
	}
	if err := validateHexFingerprint(string(expected)+" build input fingerprint", stage.BuildInputFingerprint); err != nil {
		return err
	}
	if err := validateHexFingerprint(string(expected)+" build plan fingerprint", stage.BuildPlanFingerprint); err != nil {
		return err
	}
	return nil
}
