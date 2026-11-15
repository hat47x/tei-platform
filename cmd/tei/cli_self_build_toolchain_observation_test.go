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
	"tei/internal/plugin"
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
		record := observeChildToolchain(tc.reported, "", false)
		if record.State != tc.wantState {
			t.Fatalf("%s: state = %q (%s)", tc.name, record.State, record.Reason)
		}
		if tc.wantWhy != "" && !strings.Contains(record.Reason, tc.wantWhy) {
			t.Fatalf("%s: reason %q does not contain %q", tc.name, record.Reason, tc.wantWhy)
		}
		if record.NamedByPlan {
			t.Fatalf("%s: the record says the plan names the toolchain; it does not", tc.name)
		}
		if !strings.Contains(record.Boundary, "When the plan names none") || !strings.Contains(record.Boundary, "not that the builder ran it") {
			t.Fatalf("%s: the boundary does not state the limits: %s", tc.name, record.Boundary)
		}
	}
}

// Decision 3 (#2831): when the plan names a toolchain, the builder's report is
// compared with it, independently of the executable re-read.
func TestObserveChildToolchainComparesTheReportWithThePlan(t *testing.T) {
	const planned = "sha256:" + "aa"
	reported := &selfBuildChildToolchain{GoExecutable: "/absent/go", GoExecutableDigest: "sha256:00", ClosureDigest: planned}

	same := observeChildToolchain(reported, planned, true)
	if !same.NamedByPlan || !same.MatchesPlan || same.PlannedClosureDigest != planned || same.PlanReason != "" {
		t.Fatalf("the planned toolchain was reported but not matched: %+v", same)
	}

	other := observeChildToolchain(&selfBuildChildToolchain{ClosureDigest: "sha256:bb"}, planned, true)
	if other.MatchesPlan || !strings.Contains(other.PlanReason, "the builder reported sha256:bb") {
		t.Fatalf("another toolchain was not named: %+v", other)
	}

	silent := observeChildToolchain(nil, planned, true)
	if silent.MatchesPlan || !strings.Contains(silent.PlanReason, "did not report the toolchain") {
		t.Fatalf("a silent builder was not named: %+v", silent)
	}

	unnamed := observeChildToolchain(reported, "", false)
	if unnamed.NamedByPlan || unnamed.MatchesPlan || unnamed.PlanReason != "" {
		t.Fatalf("a plan without a toolchain was treated as naming one: %+v", unnamed)
	}
}

func TestPlannedPlanParameterReadsTheReceiptPlan(t *testing.T) {
	receipt := &PlanReceipt{ReceiptJSON: `{"plan":{"parameters":[{"name":"other","value":"x"},{"name":"go.toolchain.closureDigest","value":"sha256:cc"}]}}`}
	if value, ok := plannedPlanParameter(receipt, "go.toolchain.closureDigest"); !ok || value != "sha256:cc" {
		t.Fatalf("value=%q ok=%v", value, ok)
	}
	if _, ok := plannedPlanParameter(receipt, "absent"); ok {
		t.Fatal("an absent parameter was found")
	}
	if _, ok := plannedPlanParameter(&PlanReceipt{ReceiptJSON: "{"}, "go.toolchain.closureDigest"); ok {
		t.Fatal("an unreadable receipt yielded a parameter")
	}
	if _, ok := plannedPlanParameter(nil, "go.toolchain.closureDigest"); ok {
		t.Fatal("no receipt yielded a parameter")
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
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := buildintent.ParseBuildCatalog(catalogBytes)
	if err != nil {
		t.Fatal(err)
	}
	writePlan := func(name, body string) (string, string) {
		intentFile := filepath.Join(root, name+"-intent.json")
		if err := os.WriteFile(intentFile, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		intent, err := buildintent.ParseIntent([]byte(body))
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
		planFile := filepath.Join(root, name+"-plan.json")
		if err := os.WriteFile(planFile, planJSON, 0o600); err != nil {
			t.Fatal(err)
		}
		return intentFile, planFile
	}
	// One plan file, read by both builds: the toolchain is the only thing varied.
	intentPath, planPath := writePlan("unpinned", intentBody)

	attempt := func(name, goExe, intentPath, planPath string) (toolchainChainReport, string) {
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
		if report.L2B == nil || report.ToolchainObservation == nil {
			t.Fatalf("%s: the report is incomplete: %s\n%s", name, stdout.String(), stderr.String())
		}
		return report, stderr.String()
	}
	run := func(name, goExe string) toolchainChainReport {
		report, stderr := attempt(name, goExe, intentPath, planPath)
		if !report.OK || !report.L2B.Claimed {
			t.Fatalf("%s: the build with %s did not satisfy the conditions: l2b=%+v error=%+v\n%s", name, goExe, report.L2B, report.Error, stderr)
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
	// A plan that is silent on the toolchain: both attempts hold every condition
	// while producing different bytes. Naming the toolchain is optional (#2831
	// decision 3), so this stays true for such plans.
	if first.ChildRun.ArtifactDigest == second.ChildRun.ArtifactDigest {
		t.Fatalf("two toolchains produced the same artifact (%s); the measurement this test keeps no longer holds", first.ChildRun.ArtifactDigest)
	}
	if first.ToolchainObservation.Reported.ClosureDigest == "" || first.ToolchainObservation.Reported.ClosureDigest == second.ToolchainObservation.Reported.ClosureDigest {
		t.Fatalf("the closure digests do not tell the toolchains apart: %+v / %+v", first.ToolchainObservation.Reported, second.ToolchainObservation.Reported)
	}
	t.Logf("plan %s: artifact %s with %s, artifact %s with %s", first.PlanReceipt.PlanFingerprint,
		first.ChildRun.ArtifactDigest, first.ToolchainObservation.Reported.GoExecutable,
		second.ChildRun.ArtifactDigest, second.ToolchainObservation.Reported.GoExecutable)

	// The same declaration, now naming the first toolchain (#2831 decision 3).
	// The name is a plan parameter, so the plan fingerprint changes with it.
	pinnedBody := strings.Replace(intentBody, `"profile": "release"`,
		`"profile": "release",
  "parameters": [{"name": "`+plugin.BuildParameterGoToolchainClosureDigest+`", "value": "`+first.ToolchainObservation.Reported.ClosureDigest+`"}]`, 1)
	if pinnedBody == intentBody {
		t.Fatal("the declaration could not be extended with the toolchain parameter")
	}
	pinnedIntent, pinnedPlan := writePlan("pinned", pinnedBody)

	withNamed, stderr := attempt("pinned-first", firstGo, pinnedIntent, pinnedPlan)
	if !withNamed.OK || !withNamed.L2B.Claimed {
		t.Fatalf("the toolchain the plan names was refused: l2b=%+v error=%+v\n%s", withNamed.L2B, withNamed.Error, stderr)
	}
	if !withNamed.ToolchainObservation.NamedByPlan || !withNamed.ToolchainObservation.MatchesPlan {
		t.Fatalf("toolchainObservation = %+v", withNamed.ToolchainObservation)
	}
	if withNamed.PlanReceipt.PlanFingerprint == first.PlanReceipt.PlanFingerprint {
		t.Fatal("naming the toolchain did not change the plan fingerprint")
	}

	withOther, _ := attempt("pinned-second", secondGo, pinnedIntent, pinnedPlan)
	if withOther.OK || withOther.L2B.Claimed {
		t.Fatalf("a build with another toolchain than the plan names was accepted: ok=%v l2b=%+v", withOther.OK, withOther.L2B)
	}
	if !withOther.ToolchainObservation.NamedByPlan || withOther.ToolchainObservation.MatchesPlan {
		t.Fatalf("toolchainObservation = %+v", withOther.ToolchainObservation)
	}
	if !strings.Contains(strings.Join(withOther.L2B.Unmet, "\n"), "did not report the Go toolchain the plan names") {
		t.Fatalf("unmet does not name the toolchain: %v", withOther.L2B.Unmet)
	}
	if withOther.ChildRun != nil && withOther.ChildRun.ArtifactDigest != "" {
		t.Fatalf("the builder produced an artifact with a toolchain the plan does not name: %s", withOther.ChildRun.ArtifactDigest)
	}
}
