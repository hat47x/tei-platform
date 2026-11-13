package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

const reproducibilityBoundary = "two runs of the same child executable from the same plan, in separate state directories, produced the same artifact bytes; this shows the builder was deterministic for that plan, with that binary, on the host that ran it, in these two attempts, and together with the on-disk digest match it supports the claim that the artifact is that plan's output. The same plan was also observed to yield the same artifact bytes on linux/amd64 and windows/amd64 (SB-C0 §4.25). linux/arm64 under emulation was observed too (SB-C0 §4.37, plan/evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10.md); there the two runs agreed with each other but the artifact bytes differed from amd64, because the builder binary differed, so cross-architecture equality of artifact bytes is not claimed. It is not a proof for any other host, another builder version or another plan, and it is not L2-b, acceptance, or runtime authority"

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
	// FirstChildDigest and SecondChildDigest are the builder executables the two
	// runs actually launched. Reproducibility across two different builders is not
	// the claim, so the two are recorded and compared rather than assumed equal.
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
	// StateSeparation names what the separation rests on, so a reader can tell a
	// child's statement apart from something the entry itself looked at. The
	// claim is unchanged by this field; only its basis becomes visible.
	StateSeparation *selfBuildStateSeparation `json:"stateSeparation,omitempty"`
	// OnDiskDigest is the artifact this attempt left in place, re-read after the
	// second build so a later overwrite would be visible.
	OnDiskDigest string `json:"onDiskDigest,omitempty"`
	Boundary     string `json:"boundary"`
}

// The three bases §4.37 can name for state separation. Only the first two are
// produced here: host-boundary needs the Linux namespace launch (#4486's S2),
// which this build does not do, so the entry never reports it.
const (
	stateSeparationChildStatement = "child-statement"
	stateSeparationEntryPostHoc   = "entry-post-hoc"
	stateSeparationHostBoundary   = "host-boundary"
)

// selfBuildStateSeparation records which basis the separation check reached.
//
// The claim's meaning is unchanged: two runs that shared state are still caught
// the same way they were. What changes is that a reader can see whether the entry
// looked at the file system or only at what the child said, because a child's
// statement and an entry-side check are not the same evidence (#4486).
type selfBuildStateSeparation struct {
	Basis string `json:"basis"`
	// Scope names exactly what was checked, so "entry-post-hoc" is not read as
	// "the write was prevented". Preventing it needs the host boundary.
	Scope string `json:"scope,omitempty"`
}

const (
	stateSeparationChildStatementScope = "the state directory the child reported is the one this entry passed"
	stateSeparationPostHocScope        = "the second build created the state directory this entry passed, and the listing of the first build's state directory was unchanged across the second build"
	// The wording names what the mechanism does and stops there: the read-only
	// remount covers the root tree, and filesystems mounted separately inside the
	// namespace (/proc, /sys, /dev) are outside it, so this is not "nothing else was
	// writable".
	stateSeparationHostBoundaryScope = "the second build ran in a mount namespace where the root tree was remounted read-only, the passed state directory and the artifact directory were reopened for writing, and the first build's state directory was masked by an empty tmpfs; filesystems mounted separately from the root tree are not covered"
)

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
	stateBoundary bool,
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

	// The listing is taken before the second build so a write into the first run's
	// state shows up as a difference, not as an untested possibility.
	firstListingBefore, err := listStateDirEntries(firstStateDir)
	if err != nil {
		return unperformed("the first build's state directory could not be listed: " + err.Error())
	}

	// With -state-boundary the second build runs inside a mount namespace in which
	// only the state directory and the artifact directory are writable. The allow
	// paths have to exist before the namespace is entered, because the root tree is
	// read-only by the time the launcher tries to create them.
	var boundary *selfBuildBoundary
	if stateBoundary {
		if mkErr := os.MkdirAll(secondState, 0o755); mkErr != nil {
			return unperformed("the second build's state directory could not be prepared: " + mkErr.Error())
		}
		if mkErr := os.MkdirAll(filepath.Dir(secondOutput), 0o755); mkErr != nil {
			return unperformed("the second build's artifact directory could not be prepared: " + mkErr.Error())
		}
		made, boundaryErr := stateBoundaryLauncher(secondState, filepath.Dir(secondOutput), firstStateDir)
		if boundaryErr != nil {
			return unperformed(boundaryErr.Error())
		}
		boundary = made
	}

	evidence, err := driveSelfBuildIn(childPath, args, secondOutput, "", boundary)
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
	record.StateSeparation = &selfBuildStateSeparation{
		Basis: stateSeparationChildStatement,
		Scope: stateSeparationChildStatementScope,
	}

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

	// What the child said is not the only thing the entry can see. It passed a
	// directory the second build should have created, and it can look at the first
	// run's directory before and after. Both are entry-side facts, so a child that
	// reports the right path while writing elsewhere no longer passes unnoticed.
	//
	// This does not show the write was prevented: a child that creates an empty
	// directory at the passed path still satisfies it (#4486 measured exactly
	// that). "entry-post-hoc" is therefore weaker than "host-boundary", and the
	// scope string says which of the two ran.
	firstListingAfter, err := listStateDirEntries(firstStateDir)
	if err != nil {
		return unperformed("the first build's state directory could not be listed: " + err.Error())
	}
	if reason := stateSeparationListingVerdict(firstListingBefore, firstListingAfter); reason != "" {
		return unperformed(reason)
	}
	if reason := stateSeparationSecondDirVerdict(secondState); reason != "" {
		return unperformed(reason)
	}
	record.StateSeparation = &selfBuildStateSeparation{
		Basis: stateSeparationEntryPostHoc,
		Scope: stateSeparationPostHocScope,
	}
	if boundary != nil {
		// The namespace is stronger than looking afterwards: it refused the write
		// instead of noticing it. The basis says which of the two ran.
		record.StateSeparation = &selfBuildStateSeparation{
			Basis: stateSeparationHostBoundary,
			Scope: stateSeparationHostBoundaryScope,
		}
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

// listStateDirEntries returns the sorted names in a state directory. A directory
// that does not exist is reported as empty rather than as an error: a child that
// ignores -state never creates it, and that has to reach the verdict as "nothing
// was written", not as a reason the check could not run.
func listStateDirEntries(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// stateSeparationListingVerdict reports why an unchanged first-run state listing
// cannot be shown, or "" when it can. A second build that writes into the first
// run's state made the two runs share state, whatever each one reported.
func stateSeparationListingVerdict(before, after []string) string {
	if len(before) != len(after) {
		return "the second build changed the first build's state directory, so the two runs were not separate"
	}
	for i := range before {
		if before[i] != after[i] {
			return "the second build changed the first build's state directory, so the two runs were not separate"
		}
	}
	return ""
}

// stateSeparationSecondDirVerdict reports why the second run's state directory
// cannot be taken as evidence, or "" when it can. The entry passed a fresh path;
// a run that never created it wrote its state somewhere else.
func stateSeparationSecondDirVerdict(secondState string) string {
	if strings.TrimSpace(secondState) == "" {
		return "no state directory was passed for the second build, so its separation is not shown"
	}
	if _, err := os.Stat(secondState); err != nil {
		if os.IsNotExist(err) {
			return "the second build did not create the state directory this entry passed, so it wrote its state somewhere else"
		}
		return "the second build's state directory could not be inspected: " + err.Error()
	}
	return ""
}

func reasonSuffix(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	return ": " + reason
}
