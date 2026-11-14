package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/buildintent"
)

// The plan does not name the Go toolchain. These tests pin the record that tells
// two toolchains apart, and — as a statement of the current contract, not a wish —
// that the same plan built with two toolchains satisfies the five conditions both
// times with different bytes.

func TestObserveChildToolchainComparesTheReportWithTheFile(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "go")
	if err := os.WriteFile(goFile, []byte("go command bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	actual, err := digestOfFile(goFile)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		reported  *selfBuildChildToolchain
		wantState string
		wantWhy   string
	}{
		{"the file has the reported bytes", &selfBuildChildToolchain{GoExecutable: goFile, GoExecutableDigest: actual}, toolchainObservationMatched, ""},
		{"the file has other bytes", &selfBuildChildToolchain{GoExecutable: goFile, GoExecutableDigest: "sha256:" + strings.Repeat("0", 64)}, toolchainObservationMismatched, "does not have the digest"},
		{"the builder said nothing", nil, toolchainObservationUnreported, "did not report a toolchain"},
		{"the builder could not read it", &selfBuildChildToolchain{NotObserved: "not found on PATH"}, toolchainObservationUnreported, "not found on PATH"},
		{"the reported file is gone", &selfBuildChildToolchain{GoExecutable: filepath.Join(dir, "absent"), GoExecutableDigest: actual}, toolchainObservationUnreported, "could not read"},
	}
	for _, tc := range cases {
		record := observeChildToolchain(tc.reported)
		if record.State != tc.wantState {
			t.Fatalf("%s: state = %q (%s)", tc.name, record.State, record.Reason)
		}
		if tc.wantWhy != "" && !strings.Contains(record.Reason, tc.wantWhy) {
			t.Fatalf("%s: reason %q does not contain %q", tc.name, record.Reason, tc.wantWhy)
		}
		if record.NamedByPlan {
			t.Fatalf("%s: the record says the plan names the toolchain; it does not", tc.name)
		}
		if !strings.Contains(record.Boundary, "The plan does not name a toolchain") || !strings.Contains(record.Boundary, "not that the builder ran it") {
			t.Fatalf("%s: the boundary does not state the limits: %s", tc.name, record.Boundary)
		}
	}
}

// A builder that reports no toolchain — every non-Go child, and every fixture in
// this package — must keep working. The record says the toolchain is unrecorded.
func TestSelfBuildDoesNotRequireAToolchainReport(t *testing.T) {
	dir := t.TempDir()
	child := buildReproducibleChild(t, dir)
	var out bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-child-arg", `["-artifact","` + filepath.ToSlash(filepath.Join(dir, "artifact")) + `"]`,
		"-workspace", outerWorkspace(t, dir),
		"-intent", "intent.sample",
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "artifact"),
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("a child without a toolchain report was refused: code=%d err=%v\n%s", code, err, out.String())
	}
	var report struct {
		OK                   bool                           `json:"ok"`
		ToolchainObservation *selfBuildToolchainObservation `json:"toolchainObservation"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("decode: %v\n%s", err, out.String())
	}
	if !report.OK || report.ToolchainObservation == nil || report.ToolchainObservation.State != toolchainObservationUnreported {
		t.Fatalf("ok=%v toolchainObservation=%+v", report.OK, report.ToolchainObservation)
	}
}

type toolchainChainReport struct {
	OK        bool `json:"ok"`
	Qualified bool `json:"qualified"`
	L2B       *struct {
		Claimed bool     `json:"claimed"`
		Unmet   []string `json:"unmet"`
	} `json:"l2b"`
	PlanReceipt *struct {
		PlanFingerprint string `json:"planFingerprint"`
	} `json:"planReceipt"`
	ChildRun *struct {
		ArtifactDigest string `json:"artifactCrossCheckedDigest"`
	} `json:"childRun"`
	ToolchainObservation *selfBuildToolchainObservation `json:"toolchainObservation"`
	Error                *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// One plan, two toolchains. This is the measurement in
// plan/evidence/TEI_GO_SELF_BUILD_SEED_REACH_2026-11-13.md, kept as a test.
//
// It needs a second Go toolchain, which a plain checkout does not have, and two
// real builds per toolchain. So it runs only when the caller opts in and names the
// second executable:
//
//	TEI_SELF_BUILD_FULL_CHAIN=1 TEI_SELF_BUILD_SECOND_GO=/path/to/other/bin/go
func TestSamePlanBuiltByTwoToolchainsIsToldApartOnlyByTheRecord(t *testing.T) {
	if os.Getenv("TEI_SELF_BUILD_FULL_CHAIN") == "" {
		t.Skip("real go builds; set TEI_SELF_BUILD_FULL_CHAIN=1 to run it")
	}
	secondGo := os.Getenv("TEI_SELF_BUILD_SECOND_GO")
	if secondGo == "" {
		t.Skip("a second Go toolchain is required; name its go executable in TEI_SELF_BUILD_SECOND_GO")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("the catalog names host/linux/amd64; this host is %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	firstGo, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	if rootOut, rootErr := exec.Command(firstGo, "env", "GOROOT").Output(); rootErr == nil {
		// PATH may hold a launcher that switches toolchains; the builder needs the
		// executable inside the distribution it will actually run.
		firstGo = filepath.Join(strings.TrimSpace(string(rootOut)), "bin", "go")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cli := buildToolBinary(t, ".", "tei")
	builder := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-go-build-from-plan"), "go-build-from-plan")

	root := t.TempDir()
	catalogPath, catalogDigest := authorGoCommandCatalog(t, root, builder)
	intentBody := `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.native.runtime.reference", "version": "1.0.0", "definitionSource": "tei"},
  "sources": [
    {"kind": "go-package", "id": "tei", "ref": "cmd/tei/testdata/l2b-go-command"},
    {"kind": "go-module", "id": "mod", "ref": "go.mod"},
    {"kind": "go-module", "id": "sum", "ref": "go.sum"}
  ],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-command", "entrypoint": "./cmd/tei/testdata/l2b-go-command"},
  "outputKind": "runtime-artifact",
  "target": "host/linux/amd64",
  "profile": "release"
}
`
	intentPath := filepath.Join(root, "intent.json")
	if err := os.WriteFile(intentPath, []byte(intentBody), 0o600); err != nil {
		t.Fatal(err)
	}
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := buildintent.ParseBuildCatalog(catalogBytes)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := buildintent.ParseIntent([]byte(intentBody))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := buildintent.Resolve(intent, buildintent.NewBuildContext(repoRoot, catalog))
	if err != nil {
		t.Fatalf("derive the plan: %v", err)
	}
	planJSON, err := json.Marshal(receipt.Plan)
	if err != nil {
		t.Fatal(err)
	}
	// One plan file, read by both builds: the toolchain is the only thing varied.
	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, planJSON, 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(name, goExe string) toolchainChainReport {
		out := filepath.Join(root, name)
		if err := os.MkdirAll(out, 0o755); err != nil {
			t.Fatal(err)
		}
		childArg, err := json.Marshal([]string{"-plan", planPath, "-intent", intentPath, "-repo", repoRoot, "-go", goExe, "-isolate"})
		if err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(cli, "self-build",
			"-workspace", outerWorkspace(t, out),
			"-intent", "tei.native.runtime.reference",
			"-intent-file", intentPath,
			"-build-catalog", catalogPath,
			"-build-catalog-digest", catalogDigest,
			"-build-source-root", repoRoot,
			"-child", builder,
			"-child-arg", string(childArg),
			"-state", filepath.Join(out, "state"),
			"-output", filepath.Join(out, "artifact"),
		)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		_ = cmd.Run()
		var report toolchainChainReport
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("%s: decode the report: %v\n%s\n%s", name, err, stdout.String(), stderr.String())
		}
		if report.Qualified {
			t.Fatalf("%s: the attempt was reported as qualified", name)
		}
		if !report.OK || report.L2B == nil || !report.L2B.Claimed {
			t.Fatalf("%s: the build with %s did not satisfy the five conditions: l2b=%+v error=%+v\n%s", name, goExe, report.L2B, report.Error, stderr.String())
		}
		if report.ToolchainObservation == nil || report.ToolchainObservation.State != toolchainObservationMatched {
			t.Fatalf("%s: toolchainObservation = %+v", name, report.ToolchainObservation)
		}
		return report
	}
	first := run("first", firstGo)
	second := run("second", secondGo)

	if first.ToolchainObservation.Reported.GoExecutableDigest == second.ToolchainObservation.Reported.GoExecutableDigest {
		t.Skip("TEI_SELF_BUILD_SECOND_GO names the same toolchain as the first; nothing was varied")
	}
	if first.PlanReceipt == nil || second.PlanReceipt == nil || first.PlanReceipt.PlanFingerprint != second.PlanReceipt.PlanFingerprint {
		t.Fatalf("the plan was meant to be the same for both builds: %+v / %+v", first.PlanReceipt, second.PlanReceipt)
	}
	// The current contract: the plan is silent on the toolchain, so both attempts
	// hold all five conditions while producing different bytes. If a plan ever
	// names its toolchain, the second build must be refused instead, and this
	// expectation changes with that decision.
	if first.ChildRun.ArtifactDigest == second.ChildRun.ArtifactDigest {
		t.Fatalf("two toolchains produced the same artifact (%s); the measurement this test keeps no longer holds", first.ChildRun.ArtifactDigest)
	}
	if first.ToolchainObservation.Reported.ClosureDigest == "" || first.ToolchainObservation.Reported.ClosureDigest == second.ToolchainObservation.Reported.ClosureDigest {
		t.Fatalf("the closure digests do not tell the toolchains apart: %+v / %+v", first.ToolchainObservation.Reported, second.ToolchainObservation.Reported)
	}
	t.Logf("plan %s: artifact %s with %s, artifact %s with %s", first.PlanReceipt.PlanFingerprint,
		first.ChildRun.ArtifactDigest, first.ToolchainObservation.Reported.GoExecutable,
		second.ChildRun.ArtifactDigest, second.ToolchainObservation.Reported.GoExecutable)
}
