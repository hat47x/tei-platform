package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// selfBuildEntrySchema is this entry's own output shape. It is deliberately not
// a `tei.cli.*` envelope: the command is not in the read-only discovery
// catalog, so it must not look like one of those commands (#3414 §4.5).
const selfBuildEntrySchema = "tei.cli.self-build-entry/v0"

const selfBuildEntryBoundary = "one child process started by this TEI binary; the invoker and invoked identities are recorded separately, but this is not L3, not acceptance, not build or runtime authority"

// selfBuildInvoked is the process this entry actually started.
type selfBuildInvoked struct {
	Executable       string   `json:"executable"`
	ExecutableDigest string   `json:"executableDigest"`
	Argv             []string `json:"argv"`
	PID              int      `json:"pid,omitempty"`
	ExitCode         int      `json:"exitCode"`
	StdoutDigest     string   `json:"stdoutDigest"`
}

// selfBuildEntryEvidence is one driven attempt. Every field is a separate
// observation; none of them is a qualification.
type selfBuildEntryEvidence struct {
	SchemaVersion  string           `json:"schemaVersion"`
	Nature         string           `json:"nature"`
	InvokerDigest  string           `json:"invokerDigest"`
	InvokerPath    string           `json:"invokerPath,omitempty"`
	Invoked        selfBuildInvoked `json:"invoked"`
	ChildSchema    string           `json:"childSchema,omitempty"`
	ChildArtifact  string           `json:"childArtifactDigest,omitempty"`
	CrossChecked   string           `json:"artifactCrossCheckedDigest,omitempty"`
	CrossCheckOK   bool             `json:"artifactCrossCheckMatched"`
	ClaimedInvoker string           `json:"claimedInvokerDigest,omitempty"`
	ClaimChecked   bool             `json:"invokerClaimChecked"`
	ClaimMatched   bool             `json:"invokerClaimMatched"`
	L3Claimed      bool             `json:"l3Claimed"`
	FailureCause   string           `json:"failureCause,omitempty"`
	Boundary       string           `json:"claimBoundary"`
	// The child's own report about the declaration it read (§4.8). They are
	// copied from the child's JSON and never compared by the primitive form;
	// the outer shape does the comparison.
	ChildIntentBytesDigest string `json:"childIntentBytesDigest,omitempty"`
	ChildIntentFingerprint string `json:"childIntentFingerprint,omitempty"`
	ChildPlanFingerprint   string `json:"childPlanFingerprint,omitempty"`
	// ChildStateDir is the state directory the child says it used. The entry
	// passes -state; this is the child's own reading of it, so the separation the
	// reproducibility claim rests on can be checked rather than assumed.
	ChildStateDir string `json:"childStateDir,omitempty"`
	// ChildDefinitionDigest is the digest the child reports for the definition
	// file it read. The child checks it itself before building (§4.36); recording
	// it lets this entry check the claim rather than rely on the child's own
	// enforcement.
	ChildDefinitionDigest string `json:"childDefinitionBytesDigest,omitempty"`
	// ChildIsolation is the builder's report that it built from a staged tree.
	// Recorded as a statement about the run, not as a condition of the claim.
	ChildIsolation *selfBuildChildIsolation `json:"childIsolation,omitempty"`
	// ChildToolchain is the Go toolchain the builder says it ran. The plan does
	// not name one; the outer shape re-reads the executable to compare.
	ChildToolchain *selfBuildChildToolchain `json:"childToolchain,omitempty"`
	// ChildLock is the child's own validated plan-to-artifact lock (§4.11),
	// copied verbatim. The entry checks it against the file rather than
	// treating it as proof.
	ChildLock *childSelfLock `json:"childLock,omitempty"`
}

// childSelfLock mirrors the lock a child may form about its own work. Every
// field is the child's claim; the entry compares them with what it can see.
type childSelfLock struct {
	FormatVersion   int    `json:"formatVersion"`
	PlanFingerprint string `json:"planFingerprint"`
	ArtifactDigest  string `json:"artifactDigest"`
	LockFingerprint string `json:"lockFingerprint"`
	Validated       bool   `json:"validated"`
	Boundary        string `json:"boundary,omitempty"`
}

func runSelfBuildCommand(args []string, out io.Writer) (int, error) {
	// The §4 workspace form and the primitive -child/-artifact form share one
	// command. A caller that names any §4 flag is routed to the form that
	// checks the workspace, the state, and no-clobber before starting anything;
	// the primitive form keeps its exact grammar (SB-C0 §4.6).
	if looksLikeOuterShape(args) {
		return runSelfBuildOuterShape(args, out)
	}
	fs := flag.NewFlagSet("self-build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	child := fs.String("child", "", "exact child executable to start (never resolved from PATH)")
	childArg := fs.String("child-arg", "", "child arguments as a JSON array of strings")
	childArgFile := fs.String("child-arg-file", "", "JSON array of child arguments in a file")
	artifact := fs.String("artifact", "", "path the child writes its artifact to (cross-checked from disk)")
	claimedSelf := fs.String("claimed-self", "", "digest this binary claims about itself; compared with the running image")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "self-build")
			return 0, nil
		}
		return 2, fmt.Errorf("INVALID_ARGUMENT: %w", err)
	}
	if fs.NArg() != 0 {
		return 2, errors.New("INVALID_ARGUMENT: self-build does not accept positional arguments")
	}
	if strings.TrimSpace(*child) == "" || strings.TrimSpace(*artifact) == "" {
		return 2, errors.New("INVALID_ARGUMENT: -child and -artifact are required")
	}

	rawArgs := *childArg
	if *childArgFile != "" {
		raw, err := os.ReadFile(*childArgFile)
		if err != nil {
			return 1, fmt.Errorf("read -child-arg-file: %w", err)
		}
		rawArgs = string(raw)
	}
	if strings.TrimSpace(rawArgs) == "" {
		return 2, errors.New("INVALID_ARGUMENT: one of -child-arg or -child-arg-file is required")
	}
	var childArgs []string
	if err := json.Unmarshal([]byte(rawArgs), &childArgs); err != nil {
		return 2, fmt.Errorf("INVALID_ARGUMENT: -child-arg is not a JSON array of strings: %w", err)
	}

	evidence, err := driveSelfBuild(*child, childArgs, *artifact, *claimedSelf)
	if err != nil {
		return 1, err
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		return 1, err
	}
	// The exit code says whether the observation happened. The child's own
	// failure is evidence and stays in the record; it is not "the target
	// failed a gate".
	return 0, nil
}

// driveSelfBuild starts the child with the exact argv, records the invoker and
// the invoked separately, then re-reads the artifact from disk instead of
// trusting the child's own digest field.
func driveSelfBuild(childPath string, childArgs []string, artifactPath, claimedSelf string) (selfBuildEntryEvidence, error) {
	exactChild, err := filepath.Abs(childPath)
	if err != nil {
		return selfBuildEntryEvidence{}, err
	}
	info, err := os.Stat(exactChild)
	if err != nil {
		return selfBuildEntryEvidence{}, fmt.Errorf("child executable is not readable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return selfBuildEntryEvidence{}, fmt.Errorf("child executable is not a regular file: %s", exactChild)
	}
	childDigest, err := digestOfFile(exactChild)
	if err != nil {
		return selfBuildEntryEvidence{}, err
	}

	evidence := selfBuildEntryEvidence{
		SchemaVersion: selfBuildEntrySchema,
		Nature:        "one driven child process; this entry does not build in-process",
		Boundary:      selfBuildEntryBoundary,
	}

	selfPath, err := os.Executable()
	if err != nil {
		return selfBuildEntryEvidence{}, err
	}
	selfDigest, err := digestOfFile(selfPath)
	if err != nil {
		return selfBuildEntryEvidence{}, err
	}
	evidence.InvokerDigest = selfDigest
	evidence.InvokerPath = selfPath

	// SB-N13: a self-reported digest is only meaningful next to the image that
	// actually ran. Record the claim and the comparison; never the claim alone.
	if strings.TrimSpace(claimedSelf) != "" {
		evidence.ClaimedInvoker = strings.TrimSpace(claimedSelf)
		evidence.ClaimChecked = true
		evidence.ClaimMatched = evidence.ClaimedInvoker == selfDigest
		if !evidence.ClaimMatched {
			evidence.FailureCause = "claimed invoker digest differs from the running process image"
		}
	}

	// no-clobber: an existing artifact would make the cross-check read someone
	// else's bytes.
	if _, err := os.Stat(artifactPath); err == nil {
		return selfBuildEntryEvidence{}, fmt.Errorf("artifact path already exists; self-build does not overwrite: %s", artifactPath)
	}

	cmd := exec.Command(exactChild, childArgs...)
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return selfBuildEntryEvidence{}, fmt.Errorf("start child: %w", err)
	}
	pid := cmd.Process.Pid
	runErr := cmd.Wait()

	exitCode := 0
	if runErr != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	stdout := stdoutBuf.String()
	evidence.Invoked = selfBuildInvoked{
		Executable:       exactChild,
		ExecutableDigest: childDigest,
		Argv:             append([]string{exactChild}, childArgs...),
		PID:              pid,
		ExitCode:         exitCode,
		StdoutDigest:     digestOfBytes([]byte(stdout)),
	}

	if exitCode != 0 {
		appendSelfBuildFailure(&evidence, "child exited unsuccessfully: "+strings.TrimSpace(stderrBuf.String()))
		return evidence, nil
	}

	var child struct {
		SchemaVersion  string `json:"schemaVersion"`
		ArtifactDigest string `json:"artifactDigest"`
		// The child's own readings of the declaration it read (§4.8). They are
		// optional: a child that does not interpret a declaration omits them.
		IntentBytesDigest string `json:"intentBytesDigest"`
		IntentFingerprint string `json:"intentFingerprint"`
		PlanFingerprint   string `json:"planFingerprint"`
		StateDir          string `json:"stateDir"`
		DefinitionDigest  string `json:"definitionBytesDigest"`
		// The builder may report that it built from a tree made only of the
		// planned sources. It is a statement about what it did, like the rest.
		IsolatedTree *struct {
			Root        string   `json:"root"`
			ModuleFiles []string `json:"moduleFiles"`
			Boundary    string   `json:"boundary"`
			Members     []struct {
				Locator string `json:"locator"`
			} `json:"members"`
		} `json:"isolatedTree"`
		// The child's own validated plan-to-artifact lock (§4.11). When a child
		// forms and checks it itself, the entry no longer assembles the binding
		// from two separate self-reports.
		BuildArtifactLock *childSelfLock `json:"buildArtifactLock"`
		// The toolchain the builder ran, when it is a Go builder that reports one.
		Toolchain *selfBuildChildToolchain `json:"toolchain"`
	}
	if err := json.Unmarshal([]byte(stdout), &child); err != nil {
		appendSelfBuildFailure(&evidence, "child output is not one JSON object: "+err.Error())
		return evidence, nil
	}
	evidence.ChildSchema = child.SchemaVersion
	evidence.ChildArtifact = child.ArtifactDigest
	evidence.ChildIntentBytesDigest = child.IntentBytesDigest
	evidence.ChildIntentFingerprint = child.IntentFingerprint
	evidence.ChildPlanFingerprint = child.PlanFingerprint
	evidence.ChildStateDir = child.StateDir
	evidence.ChildDefinitionDigest = child.DefinitionDigest
	evidence.ChildToolchain = child.Toolchain
	if child.IsolatedTree != nil {
		evidence.ChildIsolation = &selfBuildChildIsolation{
			Root:        child.IsolatedTree.Root,
			MemberCount: len(child.IsolatedTree.Members),
			ModuleFiles: child.IsolatedTree.ModuleFiles,
			Boundary:    child.IsolatedTree.Boundary,
		}
	}
	if child.BuildArtifactLock != nil {
		evidence.ChildLock = &childSelfLock{
			FormatVersion:   child.BuildArtifactLock.FormatVersion,
			PlanFingerprint: child.BuildArtifactLock.PlanFingerprint,
			ArtifactDigest:  child.BuildArtifactLock.ArtifactDigest,
			LockFingerprint: child.BuildArtifactLock.LockFingerprint,
			Validated:       child.BuildArtifactLock.Validated,
			Boundary:        child.BuildArtifactLock.Boundary,
		}
	}

	actual, err := digestOfFile(artifactPath)
	if err != nil {
		appendSelfBuildFailure(&evidence, "artifact cross-check could not read the artifact: "+err.Error())
		return evidence, nil
	}
	evidence.CrossChecked = actual
	evidence.CrossCheckOK = child.ArtifactDigest == actual
	if !evidence.CrossCheckOK {
		appendSelfBuildFailure(&evidence, "child-reported digest differs from the artifact on disk")
	}
	return evidence, nil
}

// appendSelfBuildFailure keeps every independent cause. One overwritten field
// would hide an invoker-claim mismatch as soon as a later check also failed.
func appendSelfBuildFailure(evidence *selfBuildEntryEvidence, cause string) {
	if evidence.FailureCause == "" {
		evidence.FailureCause = cause
		return
	}
	evidence.FailureCause += "; " + cause
}

func digestOfFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digestOfBytes(raw), nil
}

func digestOfBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
