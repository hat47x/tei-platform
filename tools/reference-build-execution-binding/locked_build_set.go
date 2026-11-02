package main

import (
	"fmt"
	"strings"

	"tei/internal/plugin"
)

const lockedBuildArtifactSetBindingFormatVersion = 1

// LockedBuildArtifactSetBinding connects one existing single-artifact
// plugin.LockedBuild to one member role of a multi-member ArtifactSet.
//
// It does not change plugin.LockedBuild, add a multi-artifact field to Core,
// or grant Build Acceptance / Runtime Authority. The LockedBuild remains the
// exact single-artifact lock; this Reference records which member of the set
// is the artifact that lock already fixed.
type LockedBuildArtifactSetBinding struct {
	FormatVersion                   int                `json:"formatVersion"`
	LockedBuild                     plugin.LockedBuild `json:"lockedBuild"`
	ArtifactSetFingerprint          string             `json:"artifactSetFingerprint"`
	BuildExecutionResultFingerprint string             `json:"buildExecutionResultFingerprint"`
	OutputRole                      string             `json:"outputRole"`
	MemberDigest                    string             `json:"memberDigest"`
}

func (b LockedBuildArtifactSetBinding) Validate() error {
	if b.FormatVersion != lockedBuildArtifactSetBindingFormatVersion {
		return fmt.Errorf("unsupported LockedBuild Artifact Set binding format %d", b.FormatVersion)
	}
	if strings.TrimSpace(string(b.LockedBuild.Plugin.ID)) == "" ||
		strings.TrimSpace(b.LockedBuild.Plugin.Version) == "" {
		return fmt.Errorf("locked build Plugin identity is incomplete")
	}
	if strings.TrimSpace(string(b.LockedBuild.OutputKind)) == "" ||
		strings.TrimSpace(b.LockedBuild.Target) == "" ||
		strings.TrimSpace(b.LockedBuild.PlanFingerprint) == "" {
		return fmt.Errorf("locked build coordinates are incomplete")
	}
	if err := validateDigest("locked build artifact", b.LockedBuild.ArtifactDigest); err != nil {
		return err
	}
	if strings.TrimSpace(b.OutputRole) == "" {
		return fmt.Errorf("bound Artifact Set member role is required")
	}
	if err := validateDigest("bound Artifact Set member", b.MemberDigest); err != nil {
		return err
	}
	if normalizeDigest(b.MemberDigest) != normalizeDigest(b.LockedBuild.ArtifactDigest) {
		return fmt.Errorf("bound member digest does not match the LockedBuild artifact")
	}
	if err := validateFingerprint("Artifact Set fingerprint", b.ArtifactSetFingerprint); err != nil {
		return err
	}
	if err := validateFingerprint("Build Execution Result fingerprint", b.BuildExecutionResultFingerprint); err != nil {
		return err
	}
	return nil
}

// bindLockedBuildToArtifactSet verifies that the supplied execution result
// binds the supplied Artifact Set, that the requested role identifies exactly
// one member, and that the member's digest equals the already fixed
// LockedBuild artifact digest. The expected digest is never recalculated from
// the set or rewritten into the lock.
func bindLockedBuildToArtifactSet(
	build plugin.LockedBuild,
	result BuildExecutionResult,
	set ArtifactSet,
	role string,
) (LockedBuildArtifactSetBinding, error) {
	if err := result.Validate(); err != nil {
		return LockedBuildArtifactSetBinding{}, fmt.Errorf("build execution result: %w", err)
	}
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		return LockedBuildArtifactSetBinding{}, err
	}
	if normalizeFingerprint(result.ArtifactSetFingerprint) != normalizeFingerprint(setFingerprint) {
		return LockedBuildArtifactSetBinding{}, fmt.Errorf("build execution result does not bind the supplied Artifact Set")
	}
	resultFingerprint, err := buildExecutionResultFingerprint(result)
	if err != nil {
		return LockedBuildArtifactSetBinding{}, err
	}
	if strings.TrimSpace(role) == "" {
		return LockedBuildArtifactSetBinding{}, fmt.Errorf("bound Artifact Set member role is required")
	}
	var member *ArtifactMember
	for i := range set.Members {
		if set.Members[i].Role != role {
			continue
		}
		if member != nil {
			return LockedBuildArtifactSetBinding{}, fmt.Errorf("Artifact Set role %s is declared more than once", role)
		}
		member = &set.Members[i]
	}
	if member == nil {
		return LockedBuildArtifactSetBinding{}, fmt.Errorf("Artifact Set has no member with role %s", role)
	}
	if normalizeDigest(member.Digest) != normalizeDigest(build.ArtifactDigest) {
		return LockedBuildArtifactSetBinding{}, fmt.Errorf(
			"Artifact Set member %s digest does not match the LockedBuild artifact", role)
	}
	binding := LockedBuildArtifactSetBinding{
		FormatVersion:                   lockedBuildArtifactSetBindingFormatVersion,
		LockedBuild:                     build,
		ArtifactSetFingerprint:          setFingerprint,
		BuildExecutionResultFingerprint: resultFingerprint,
		OutputRole:                      role,
		MemberDigest:                    normalizeDigest(member.Digest),
	}
	if err := binding.Validate(); err != nil {
		return LockedBuildArtifactSetBinding{}, err
	}
	return binding, nil
}

func lockedBuildArtifactSetBindingFingerprint(binding LockedBuildArtifactSetBinding) (string, error) {
	if err := binding.Validate(); err != nil {
		return "", err
	}
	normalized := binding
	normalized.LockedBuild.ArtifactDigest = normalizeDigest(binding.LockedBuild.ArtifactDigest)
	normalized.MemberDigest = normalizeDigest(binding.MemberDigest)
	normalized.ArtifactSetFingerprint = normalizeFingerprint(binding.ArtifactSetFingerprint)
	normalized.BuildExecutionResultFingerprint = normalizeFingerprint(binding.BuildExecutionResultFingerprint)
	return fingerprintJSON(normalized)
}
