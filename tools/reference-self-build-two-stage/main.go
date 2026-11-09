// reference-self-build-two-stage は、stage1で生成した実行ファイル自身が
// stage2を駆動することを、実プロセスで確かめます（GitHub #2831 SB-C1）。
//
// 手順:
//
//	stage0 : 外部Go seedで駆動役(driver)を構築する
//	stage1 : stage0のdriverがchildを起動し、driver自身を再構築する
//	stage2 : stage1で得たdriver実行ファイルをinvokerとして起動し、同じchildで再構築する
//
// 判定は三つを別々に見る。stage1/stage2のartifact digestが一致するか、
// stage2のinvoker digestがstage1のartifact digestと一致するか、
// stage1とstage2が同一のchild・同一の引数を使ったか。
//
// この参照はSeed Trustも refreshing も受理も扱いません。stage0の外部Goは
// 明示したseedであり、TEI自身によるGoコンパイラのself-hostingではありません。
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
	"runtime"
	"strings"
)

const twoStageSchema = "tei.reference.self-build-two-stage/v0"

const twoStageBoundary = "one machine, one Go toolchain, one child recipe; stage1 artifact really invoked stage2, but seed trust, acceptance and runtime authority are NOT established"

// StageRecord is one stage's observed facts.
type StageRecord struct {
	Stage           string `json:"stage"`
	InvokerDigest   string `json:"invokerDigest"`
	InvokerSource   string `json:"invokerSource"`
	ChildDigest     string `json:"childDigest"`
	ChildArgvDigest string `json:"childArgvDigest"`
	TargetPackage   string `json:"targetPackage"`
	SupplyRoot      string `json:"supplyRoot"`
	ExitCode        int    `json:"exitCode"`
	ArtifactDigest  string `json:"artifactDigest,omitempty"`
	Error           string `json:"error,omitempty"`
}

// TwoStageEvidence is the comparison record.
type TwoStageEvidence struct {
	SchemaVersion      string        `json:"schemaVersion"`
	Nature             string        `json:"nature"`
	GoOS               string        `json:"goos"`
	GoArch             string        `json:"goarch"`
	SeedGoDigest       string        `json:"seedGoExecutableDigest"`
	TargetPackage      string        `json:"targetPackage"`
	Stages             []StageRecord `json:"stages"`
	Stage1DrivesStage2 bool          `json:"stage1ArtifactIsStage2Invoker"`
	SameChildAndArgs   bool          `json:"sameChildAndArguments"`
	ArtifactsEqual     bool          `json:"artifactsEqual"`
	L3Claimed          bool          `json:"l3Claimed"`
	FailureCause       string        `json:"failureCause,omitempty"`
	Boundary           string        `json:"claimBoundary"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-self-build-two-stage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", ".", "対象リポジトリのcheckout")
	driverChild := fs.String("driver-child", "", "child executable（必須。PATH解決しない）")
	target := fs.String("target", "./tools/reference-go-build-stage-driver", "再構築するpackage（必須）")
	supply := fs.String("supply", "", "子プロセスへ渡すGOPROXY=file供給（必須）")
	work := fs.String("work", "", "作業ディレクトリ（省略時は一時）")
	goExecutable := fs.String("go", "", "stage0 seedのGo実行ファイル（省略時はPATHのgo）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *driverChild == "" || *target == "" || *supply == "" {
		fmt.Fprintln(stderr, "-driver-child, -target and -supply are required")
		return 2
	}
	root := *work
	cleanup := func() {}
	if root == "" {
		dir, err := os.MkdirTemp("", "tei-two-stage-*")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		root = dir
		cleanup = func() { removeWritable(dir) }
	}
	defer cleanup()

	evidence, err := RunTwoStage(*repo, *driverChild, *target, *supply, root, *goExecutable)
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
	return 0
}

// RunTwoStage performs the three steps and reports each judgement separately.
func RunTwoStage(repo, childPath, targetPackage, supply, workDir, goArg string) (TwoStageEvidence, error) {
	exactChild, err := filepath.Abs(childPath)
	if err != nil {
		return TwoStageEvidence{}, err
	}
	childDigest, err := fileDigest(exactChild)
	if err != nil {
		return TwoStageEvidence{}, fmt.Errorf("child executable is not readable: %w", err)
	}
	goPath, err := resolveGo(goArg)
	if err != nil {
		return TwoStageEvidence{}, err
	}
	goDigest, err := fileDigest(goPath)
	if err != nil {
		return TwoStageEvidence{}, err
	}

	evidence := TwoStageEvidence{
		SchemaVersion: twoStageSchema,
		Nature:        "two stages where the stage1 artifact is the stage2 invoker",
		GoOS:          runtime.GOOS,
		GoArch:        runtime.GOARCH,
		SeedGoDigest:  goDigest,
		TargetPackage: targetPackage,
		Boundary:      twoStageBoundary,
	}

	// stage0: the external seed builds the driver. This is a Go build, not a
	// driven process, so it is recorded but not compared as a stage.
	stage0Dir := filepath.Join(workDir, "stage0")
	if err := os.MkdirAll(stage0Dir, 0o755); err != nil {
		return TwoStageEvidence{}, err
	}
	stage0Driver := filepath.Join(stage0Dir, "driver")
	if runtime.GOOS == "windows" {
		stage0Driver += ".exe"
	}
	goBuild := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false", "-mod=readonly", "-o", stage0Driver, "./tools/reference-go-build-stage-driver")
	goBuild.Dir = repo
	if out, err := goBuild.CombinedOutput(); err != nil {
		return TwoStageEvidence{}, fmt.Errorf("stage0 seed build: %v: %s", err, strings.TrimSpace(string(out)))
	}

	// The child writes the artifact the driver cross-checks, so its
	// -artifact-out must be the stage artifact path. Sharing one output path
	// between stages would also collide with the driver's no-clobber rule.
	childArgsFor := func(stage, artifact string) []string {
		return []string{
			"-repo", repo,
			"-package", targetPackage,
			"-supply", supply,
			"-state", filepath.Join(workDir, stage+"-child-state"),
			"-artifact-out", artifact,
		}
	}

	// stage1: the seed driver builds the next driver.
	stage1Artifact := filepath.Join(workDir, "stage1-driver")
	if runtime.GOOS == "windows" {
		stage1Artifact += ".exe"
	}
	stage1Args := childArgsFor("stage1", stage1Artifact)
	stage1Argv := digestOfBytes([]byte(strings.Join(stage1Args, "\x00")))
	stage1, err := driveStage("stage1", stage0Driver, exactChild, childDigest, targetPackage, supply, stage1Args, stage1Argv, stage1Artifact)
	if err != nil {
		return TwoStageEvidence{}, err
	}
	evidence.Stages = append(evidence.Stages, stage1)
	if stage1.ExitCode != 0 {
		evidence.FailureCause = "stage1 did not produce an artifact"
		return evidence, nil
	}

	// stage2: the artifact produced by stage1 is now the invoker. This is the
	// only step that can support the L3-shaped claim.
	stage2Artifact := filepath.Join(workDir, "stage2-driver")
	if runtime.GOOS == "windows" {
		stage2Artifact += ".exe"
	}
	stage2Args := childArgsFor("stage2", stage2Artifact)
	stage2Argv := digestOfBytes([]byte(strings.Join(stage2Args, "\x00")))
	stage2, err := driveStage("stage2", stage1Artifact, exactChild, childDigest, targetPackage, supply, stage2Args, stage2Argv, stage2Artifact)
	if err != nil {
		return TwoStageEvidence{}, err
	}
	evidence.Stages = append(evidence.Stages, stage2)
	if stage2.ExitCode != 0 {
		// The most likely cause is that the stage1 artifact is not a driver at
		// all: the target package was something else. Say so instead of leaving
		// a bare "did not produce an artifact".
		evidence.FailureCause = "stage2 did not produce an artifact: the stage1 artifact could not drive the child"
		if stage2.Error != "" {
			evidence.FailureCause += " (" + firstLine(stage2.Error) + ")"
		}
		return evidence, nil
	}

	evidence.Stage1DrivesStage2, evidence.SameChildAndArgs, evidence.ArtifactsEqual, evidence.FailureCause = Judge(stage1, stage2)
	if evidence.FailureCause == "" {
		evidence.L3Claimed = true
	}
	return evidence, nil
}

// Judge applies the fixed precedence: the invoker identity decides L3, then the
// recipe must match, then the artifacts. A matching artifact produced by a
// different invoker is not convergence; it is two unrelated builds.
func Judge(stage1, stage2 StageRecord) (invokerMatch, sameRecipe, artifactsEqual bool, cause string) {
	invokerMatch = stage2.InvokerDigest == stage1.ArtifactDigest
	// The output and state paths must differ per stage, so the argv digests
	// legitimately differ. The invariant recipe is the child executable, the
	// target package and the supply.
	sameRecipe = stage1.ChildDigest == stage2.ChildDigest &&
		stage1.TargetPackage == stage2.TargetPackage && stage1.SupplyRoot == stage2.SupplyRoot
	artifactsEqual = stage1.ArtifactDigest == stage2.ArtifactDigest
	switch {
	case !invokerMatch:
		cause = "stage2 was not invoked by the stage1 artifact"
	case !sameRecipe:
		cause = "the two stages did not use the same child and recipe"
	case !artifactsEqual:
		cause = "stage1 and stage2 artifacts differ"
	}
	return invokerMatch, sameRecipe, artifactsEqual, cause
}

// driveStage runs one driver binary and reads the artifact it produced.
func driveStage(name, invokerPath, childPath, childDigest, targetPackage, supply string, childArgs []string, argvDigest, artifactOut string) (StageRecord, error) {
	invokerDigest, err := fileDigest(invokerPath)
	if err != nil {
		return StageRecord{}, fmt.Errorf("%s invoker is not readable: %w", name, err)
	}
	record := StageRecord{
		Stage:           name,
		InvokerDigest:   invokerDigest,
		InvokerSource:   invokerPath,
		ChildDigest:     childDigest,
		ChildArgvDigest: argvDigest,
		TargetPackage:   targetPackage,
		SupplyRoot:      supply,
	}
	if _, err := os.Stat(artifactOut); err == nil {
		return StageRecord{}, fmt.Errorf("%s artifact path already exists: %s", name, artifactOut)
	}
	argsFile := artifactOut + ".args.json"
	raw, err := json.Marshal(childArgs)
	if err != nil {
		return StageRecord{}, err
	}
	if err := os.WriteFile(argsFile, raw, 0o600); err != nil {
		return StageRecord{}, err
	}
	cmd := exec.Command(invokerPath,
		"-child", childPath,
		"-child-arg-file", argsFile,
		"-artifact", artifactOut,
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	record.ExitCode = 0
	if runErr != nil {
		record.ExitCode = -1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			record.ExitCode = exitErr.ExitCode()
		}
		record.Error = strings.TrimSpace(stderr.String())
		return record, nil
	}
	digest, err := fileDigest(artifactOut)
	if err != nil {
		record.Error = err.Error()
		return record, nil
	}
	record.ArtifactDigest = digest
	return record, nil
}

// firstLine keeps a child error readable in a one-line cause.

func firstLine(text string) string {
	if index := strings.IndexByte(text, 0x0a); index >= 0 {
		return strings.TrimSpace(text[:index])
	}
	return strings.TrimSpace(text)
}

func resolveGo(argument string) (string, error) {
	trimmed := strings.TrimSpace(argument)
	if trimmed == "" {
		trimmed = "go"
	}
	if !strings.Contains(trimmed, "/") && !strings.Contains(trimmed, string(filepath.Separator)) {
		path, err := exec.LookPath(trimmed)
		if err != nil {
			return "", fmt.Errorf("resolve seed Go executable: %w", err)
		}
		trimmed = path
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve seed Go executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("seed Go executable is not a regular file: %s", absolute)
	}
	return absolute, nil
}

func digestOfBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fileDigest(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digestOfBytes(raw), nil
}

func removeWritable(root string) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			_ = os.Chmod(path, 0o700)
		} else {
			_ = os.Chmod(path, 0o600)
		}
		return nil
	})
	_ = os.RemoveAll(root)
}
