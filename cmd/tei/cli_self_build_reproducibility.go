package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// This file adds the observation that turns "the plan and the artifact digest are
// bound" into "the artifact is that plan's output".
//
// SB-C0 §4.8(c) asks whether the artifact that exists is the one the plan
// produced. A validated lock over (planFingerprint, artifactDigest) does not
// answer that: it ties two values together, it does not show that running the
// builder on that plan yields those bytes. Reproducing the build does.
//
// If running the builder on plan P twice, in separate state, yields the same
// bytes D, and the artifact on disk has digest D, then the artifact is P's output
// — modulo the determinism having been observed rather than proven, and modulo
// digest collisions. That scope is stated in the boundary instead of being left
// for a reader to assume.

const reproducibilityBoundary = "two runs of the same named child file (same bytes at -child before each start; what that file itself launches is not observed) from the same plan, in separate state directories, produced the same artifact bytes; this shows the builder was deterministic for that plan, with that binary, on the host that ran it, in these two attempts, and together with the on-disk digest match it supports the claim that the artifact is that plan's output. The same plan was also observed to yield the same artifact bytes on linux/amd64 and windows/amd64 (SB-C0 §4.25). It is not a proof for any other host, another builder version or another plan, and it is not L2-b, acceptance, or runtime authority"

const (
	reproducibilityReproduced  = "reproduced"
	reproducibilityMismatched  = "mismatched"
	reproducibilityUnperformed = "unperformed"
)

// selfBuildReproducibility records whether a second, independent build of the
// same plan produced the same artifact bytes.
type selfBuildReproducibility struct {
	State string `json:"state"`
	// UnperformedReason is set when the check did not run, so a reader never has
	// to infer "no field" as "agreed" (SC0-18's rule, applied here).
	UnperformedReason string `json:"unperformedReason,omitempty"`
	// FirstArtifactDigest is what the first build reported, which is also the
	// digest planArtifactBinding checked against the file on disk.
	FirstArtifactDigest string `json:"firstArtifactDigest,omitempty"`
	// SecondArtifactDigest is what the second build reported.
	SecondArtifactDigest string `json:"secondArtifactDigest,omitempty"`
	// FirstChildDigest and SecondChildDigest are the bytes of the file named by
	// -child, read before each of the two starts. They are not an observation of
	// the process that ran: a -child that launches another executable keeps the
	// same digest whatever it launches (§4.47). Reproducibility across two
	// different named files is not the claim, so the two are recorded and compared
	// rather than assumed equal.
	FirstChildDigest  string `json:"firstChildDigest,omitempty"`
	SecondChildDigest string `json:"secondChildDigest,omitempty"`
	// MismatchReason says what disagreed when State is "mismatched", so a reader
	// learns whether the bytes, the builders or the state directories differed.
	MismatchReason string `json:"mismatchReason,omitempty"`
	// FirstStateDir and SecondStateDir are the state directories the entry
	// passed. SecondStateDir is freshly created for the second build; the claim
	// is that the two runs did not share state.
	FirstStateDir  string `json:"firstStateDir,omitempty"`
	SecondStateDir string `json:"secondStateDir,omitempty"`
	// OnDiskDigest is the artifact this attempt left in place, re-read after the
	// second build so a later overwrite would be visible.
	OnDiskDigest string `json:"onDiskDigest,omitempty"`
	Boundary     string `json:"boundary"`
}

// withArtifactAndState returns the child argv with the values of -artifact-out and
// -state replaced. The outer shape appended both itself, so the second run can
// reuse every caller-supplied argument and change only where output and state go.
func withArtifactAndState(args []string, artifactOut, state string) ([]string, error) {
	out := make([]string, len(args))
	copy(out, args)

	replace := func(flag, value string) error {
		found := false
		for i := 0; i+1 < len(out); i++ {
			if out[i] == flag {
				out[i+1] = value
				found = true
			}
		}
		if !found {
			return fmt.Errorf("child argv has no %s to replace", flag)
		}
		return nil
	}
	if err := replace("-artifact-out", artifactOut); err != nil {
		return nil, err
	}
	if err := replace("-state", state); err != nil {
		return nil, err
	}
	return out, nil
}

// checkArtifactReproducibility builds the same plan a second time, in a fresh
// state directory and to a temporary output path, and compares the artifact
// digest with the first attempt's.
//
// The second run writes outside the caller's -output, so a successful first
// attempt keeps its no-clobber guarantee and the caller's artifact is never
// rewritten by a check.
func checkArtifactReproducibility(
	childPath string,
	childArgs []string,
	firstArtifactDigest string,
	firstChildDigest string,
	firstStateDir string,
	firstReportedStateDir string,
	outputPath string,
) selfBuildReproducibility {
	record := selfBuildReproducibility{Boundary: reproducibilityBoundary}
	unperformed := func(reason string) selfBuildReproducibility {
		record.State = reproducibilityUnperformed
		record.UnperformedReason = reason
		return record
	}

	work, err := os.MkdirTemp("", "tei-self-build-rebuild-")
	if err != nil {
		return unperformed("a temporary directory for the second build could not be created: " + err.Error())
	}
	defer os.RemoveAll(work)

	secondOutput := filepath.Join(work, "artifact")
	secondState := filepath.Join(work, "state")
	args, err := withArtifactAndState(childArgs, secondOutput, secondState)
	if err != nil {
		return unperformed(err.Error())
	}

	evidence, err := driveSelfBuild(childPath, args, secondOutput, "")
	if err != nil {
		return unperformed("the second build could not be started: " + err.Error())
	}
	if evidence.ChildArtifact == "" {
		return unperformed("the second build reported no artifact digest" + reasonSuffix(evidence.FailureCause))
	}

	record.FirstArtifactDigest = firstArtifactDigest
	record.SecondArtifactDigest = evidence.ChildArtifact
	record.FirstChildDigest = firstChildDigest
	record.SecondChildDigest = evidence.Invoked.ExecutableDigest
	record.FirstStateDir = firstStateDir
	record.SecondStateDir = secondState

	// "In separate state" is the reason two runs count as independent. The entry
	// passes a fresh -state to the second run; the child reports which it used, so
	// the separation is checked instead of assumed. A child that reports nothing
	// leaves the observation unsupported, which is not the same as agreeing.
	if reason := reproducibilityStateVerdict(firstStateDir, firstReportedStateDir); reason != "" {
		return unperformed(reason)
	}
	if reason := reproducibilityStateVerdict(secondState, evidence.ChildStateDir); reason != "" {
		return unperformed(reason)
	}

	// The claim is about one builder. If the executable changed between the two
	// runs, matching bytes would say nothing about the builder the caller named.
	if reason := reproducibilityBuilderVerdict(firstChildDigest, record.SecondChildDigest); reason != "" {
		record.State = reproducibilityMismatched
		record.MismatchReason = reason
		return record
	}

	// A child that honours -artifact-out writes to the temporary path; if it does,
	// confirm the bytes it actually wrote so a digest it did not earn cannot
	// manufacture agreement. A child that ignores -artifact-out writes to whatever
	// its own argv says, and the caller's artifact is re-read below instead.
	if _, statErr := os.Stat(secondOutput); statErr == nil {
		secondOnDisk, err := digestOfFile(secondOutput)
		if err != nil {
			return unperformed("the second build's artifact is not readable: " + err.Error())
		}
		if secondOnDisk != evidence.ChildArtifact {
			record.State = reproducibilityMismatched
			record.MismatchReason = "the second build reported an artifact digest that does not match the bytes it wrote"
			return record
		}
	}

	// Re-read the caller's artifact after the second build. If the second run
	// overwrote it, this compares the two builds' output; if it did not, this
	// confirms the artifact this attempt reported is still the one on disk.
	onDisk, err := digestOfFile(outputPath)
	if err != nil {
		return unperformed("the artifact this attempt produced is not readable: " + err.Error())
	}
	record.OnDiskDigest = onDisk

	if firstArtifactDigest != evidence.ChildArtifact || onDisk != firstArtifactDigest {
		record.State = reproducibilityMismatched
		record.MismatchReason = "the two builds of the same plan produced different artifact bytes"
		return record
	}
	record.State = reproducibilityReproduced
	return record
}

// reproducibilityBuilderVerdict reports why two builder digests cannot support the
// claim, or "" when they can. A missing digest is refused rather than treated as
// agreement: an unrecorded builder is not the same builder (SC0-18's rule).
func reproducibilityBuilderVerdict(first, second string) string {
	if first == "" || second == "" {
		return "the builder executable digest was not recorded for both runs"
	}
	if first != second {
		return "the two runs used different builder executables"
	}
	return ""
}

// reproducibilityStateVerdict reports why a run's state directory cannot support
// the claim, or "" when it can. The child must report the directory it used, and
// it must be the one this entry passed: two runs that shared state are one
// observation repeated, not two.
func reproducibilityStateVerdict(passed, reported string) string {
	if strings.TrimSpace(reported) == "" {
		return "the child did not report the state directory it used, so the separation of the two runs is not shown"
	}
	if filepath.Clean(passed) != filepath.Clean(reported) {
		return "the child used a different state directory than the one this entry passed"
	}
	return ""
}

func reasonSuffix(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	return ": " + reason
}
