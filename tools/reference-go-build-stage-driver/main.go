// reference-go-build-stage-driver は、Buildを「別プロセスの起動」として駆動し、
// 起動した側（invoker）と起動された側（invoked）の同一性を別々に記録します
// （GitHub #2831 SB-C0/C1 の前提確認）。
//
// この参照は child を実起動し、成果物も作りますが、受理・署名・権限・配備は行いません。
// また L3（stage1が生成したTEI自身がstage2を駆動した）を主張しません。ここで示すのは
// 「誰が、どの実行ファイルを、どのargvで起動したか」の記録が取れることだけです。
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

const driverSchema = "tei.reference.go-build-stage-driver/v0"

const driverBoundary = "one driven child process on one machine; the invoker and invoked identities are recorded separately, but this is not L3, not acceptance, not build or runtime authority"

// InvokedProcess is what the driver actually started.
type InvokedProcess struct {
	Executable       string   `json:"executable"`
	ExecutableDigest string   `json:"executableDigest"`
	Argv             []string `json:"argv"`
	PID              int      `json:"pid,omitempty"`
	ExitCode         int      `json:"exitCode"`
	Stderr           string   `json:"stderr,omitempty"`
	StdoutDigest     string   `json:"stdoutDigest"`
}

// StageDriverEvidence is one driven attempt.
type StageDriverEvidence struct {
	SchemaVersion        string         `json:"schemaVersion"`
	Nature               string         `json:"nature"`
	DriverSelfDigest     string         `json:"driverSelfDigest"`
	Invoked              InvokedProcess `json:"invoked"`
	ChildSchema          string         `json:"childSchema,omitempty"`
	ChildArtifact        string         `json:"childArtifactDigest,omitempty"`
	CrossCheckedArtifact string         `json:"artifactCrossCheckedDigest,omitempty"`
	CrossCheckMatched    bool           `json:"artifactCrossCheckMatched"`
	InvokerRecorded      bool           `json:"invokerIdentityRecorded"`
	InvokedRecorded      bool           `json:"invokedIdentityRecorded"`
	// ClaimedInvokerDigest is what the driving stage says it is. It is never
	// accepted on its own: SB-N13 rejects a manifest entry that names one
	// executable while the process that actually ran is a different one.
	ClaimedInvokerDigest string `json:"claimedInvokerDigest,omitempty"`
	// InvokerClaimChecked is true only when a claim was supplied and compared
	// against the digest of the running process image.
	InvokerClaimChecked bool `json:"invokerClaimChecked"`
	// InvokerClaimMatched is meaningful only when InvokerClaimChecked is true.
	InvokerClaimMatched bool   `json:"invokerClaimMatched"`
	L3Claimed           bool   `json:"l3Claimed"`
	FailureCause        string `json:"failureCause,omitempty"`
	Boundary            string `json:"claimBoundary"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-go-build-stage-driver", flag.ContinueOnError)
	fs.SetOutput(stderr)
	child := fs.String("child", "", "起動するchild executable（必須。PATH解決しない）")
	self := fs.String("self", "", "invoker自身のexecutable（省略時は現在のプロセス）")
	claimedSelf := fs.String("claimed-self", "", "駆動主体が自称するdigest（SB-N13の照合用。省略時は照合しない）")
	childArg := fs.String("child-arg", "", "childへ渡す引数一式をJSON配列で指定")
	childArgFile := fs.String("child-arg-file", "", "childへ渡す引数一式のJSON配列を書いたファイル（シェルの引用符問題を避ける）")
	artifactPath := fs.String("artifact", "", "childが作るartifactのパス（cross-check用。必須）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *child == "" || *artifactPath == "" {
		fmt.Fprintln(stderr, "-child and -artifact are required")
		return 2
	}
	rawArgs := *childArg
	if *childArgFile != "" {
		raw, err := os.ReadFile(*childArgFile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		rawArgs = string(raw)
	}
	if strings.TrimSpace(rawArgs) == "" {
		fmt.Fprintln(stderr, "one of -child-arg or -child-arg-file is required")
		return 2
	}
	var childArgs []string
	if err := json.Unmarshal([]byte(rawArgs), &childArgs); err != nil {
		fmt.Fprintf(stderr, "-child-arg is not a JSON array of strings: %v\n", err)
		return 2
	}

	evidence, err := Drive(*child, childArgs, *artifactPath, *self, *claimedSelf)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	// 終了コードは観測が行えたかだけを示す。childの失敗は証拠として記録し、
	// 「対象が不合格」とは読ませない。
	return 0
}

// Drive starts the child with the exact argv, records both identities, then
// independently re-reads the artifact to cross-check the child's own report.
//
// claimedSelf is an optional digest the driving stage asserts about itself. It
// is compared against the digest of the process image that actually ran, so
// SB-N13 ("the manifest names one executable, a different one ran") is
// detectable instead of silently accepted.
func Drive(childPath string, childArgs []string, artifactPath, selfPath, claimedSelf string) (StageDriverEvidence, error) {
	exactChild, err := filepath.Abs(childPath)
	if err != nil {
		return StageDriverEvidence{}, err
	}
	info, err := os.Stat(exactChild)
	if err != nil {
		return StageDriverEvidence{}, fmt.Errorf("child executable is not readable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return StageDriverEvidence{}, fmt.Errorf("child executable is not a regular file: %s", exactChild)
	}
	childDigest, err := fileDigest(exactChild)
	if err != nil {
		return StageDriverEvidence{}, err
	}

	evidence := StageDriverEvidence{
		SchemaVersion: driverSchema,
		Nature:        "one driven child process; the driver does not build in-process",
		Boundary:      driverBoundary,
	}

	if selfPath == "" {
		selfPath, err = os.Executable()
		if err != nil {
			return StageDriverEvidence{}, err
		}
	}
	selfDigest, err := fileDigest(selfPath)
	if err != nil {
		return StageDriverEvidence{}, err
	}
	evidence.DriverSelfDigest = selfDigest
	evidence.InvokerRecorded = selfDigest != ""

	// SB-N13 guard: a self-reported invoker digest is only meaningful when it
	// is compared with the image that actually ran. Record the claim, the
	// comparison, and the failure cause; never let the claim stand alone.
	if strings.TrimSpace(claimedSelf) != "" {
		evidence.ClaimedInvokerDigest = strings.TrimSpace(claimedSelf)
		evidence.InvokerClaimChecked = true
		evidence.InvokerClaimMatched = evidence.ClaimedInvokerDigest == selfDigest
		if !evidence.InvokerClaimMatched {
			evidence.FailureCause = "claimed invoker digest differs from the running process image"
		}
	}

	if artifactPath != "" {
		if _, err := os.Stat(artifactPath); err == nil {
			return StageDriverEvidence{}, fmt.Errorf("artifact path already exists; this driver does not overwrite: %s", artifactPath)
		}
	}

	cmd := exec.Command(exactChild, childArgs...)
	cmd.Dir = ""
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return StageDriverEvidence{}, fmt.Errorf("start child: %w", err)
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
	out := stdoutBuf.String()
	evidence.Invoked = InvokedProcess{
		Executable:       exactChild,
		ExecutableDigest: childDigest,
		Argv:             append([]string{exactChild}, childArgs...),
		PID:              pid,
		ExitCode:         exitCode,
		Stderr:           strings.TrimSpace(stderrBuf.String()),
		StdoutDigest:     digestOfBytes([]byte(out)),
	}
	evidence.InvokedRecorded = childDigest != ""

	if exitCode != 0 {
		appendFailure(&evidence, "child exited unsuccessfully")
		return evidence, nil
	}

	var child struct {
		SchemaVersion  string `json:"schemaVersion"`
		ArtifactDigest string `json:"artifactDigest"`
	}
	decoder := json.NewDecoder(strings.NewReader(out))
	if err := decoder.Decode(&child); err != nil {
		appendFailure(&evidence, "child output is not one JSON object: "+err.Error())
		return evidence, nil
	}
	evidence.ChildSchema = child.SchemaVersion
	evidence.ChildArtifact = child.ArtifactDigest

	// Independent cross-check: read the artifact from disk instead of trusting
	// the child's own digest field.
	actualDigest, err := fileDigest(artifactPath)
	if err != nil {
		appendFailure(&evidence, "artifact cross-check could not read the artifact: "+err.Error())
		return evidence, nil
	}
	evidence.CrossCheckedArtifact = actualDigest
	evidence.CrossCheckMatched = child.ArtifactDigest == actualDigest
	if !evidence.CrossCheckMatched {
		appendFailure(&evidence, "child-reported digest differs from the artifact on disk")
	}
	return evidence, nil
}

// appendFailure keeps every independent failure cause. A single overwritten
// field would hide an invoker-claim mismatch as soon as a later check also
// fails, and two failures are not one.
func appendFailure(evidence *StageDriverEvidence, cause string) {
	if evidence.FailureCause == "" {
		evidence.FailureCause = cause
		return
	}
	evidence.FailureCause += "; " + cause
}

func fileDigest(path string) (string, error) {
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
