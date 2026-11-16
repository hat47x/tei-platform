package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Plan hand-over: no program other than the entry writes the plan the builder
// builds. In the other self-build tests the test itself (or a resolver tool)
// resolves the plan and writes plan.json for the child; here nothing does.

type planHandoverReport struct {
	OK  bool `json:"ok"`
	L2B *struct {
		Claimed bool     `json:"claimed"`
		Unmet   []string `json:"unmet"`
	} `json:"l2b"`
	PlanReceipt *struct {
		PlanFingerprint string `json:"planFingerprint"`
	} `json:"planReceipt"`
	PlanCrossCheck *struct {
		State string `json:"state"`
	} `json:"planCrossCheck"`
	PlanHandover *struct {
		State               string `json:"state"`
		PlanPath            string `json:"planPath"`
		PlanBytesDigest     string `json:"planBytesDigest"`
		PlanFingerprint     string `json:"planFingerprint"`
		UnchangedAfterChild *bool  `json:"unchangedAfterChild"`
		Reason              string `json:"reason"`
	} `json:"planHandover"`
	ChildRun *struct {
		ChildPlanFingerprint string `json:"childPlanFingerprint"`
		Invoked              *struct {
			Argv []string `json:"argv"`
		} `json:"invoked"`
	} `json:"childRun"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const planHandoverIntent = `{
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

type planHandoverFixture struct {
	cli, builder, root, repoRoot, goExe string
	workspace, intentPath               string
	catalogPath, catalogDigest          string
}

func newPlanHandoverFixture(t *testing.T) planHandoverFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("a real go build is the observation; -short skips it")
	}
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	f := planHandoverFixture{
		cli:      buildToolBinary(t, ".", "tei"),
		builder:  buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-go-build-from-plan"), "go-build-from-plan"),
		root:     t.TempDir(),
		repoRoot: repoRoot,
		goExe:    goExe,
	}
	f.catalogPath, f.catalogDigest = authorGoCommandCatalog(t, f.root, f.builder)
	f.workspace = outerWorkspace(t, f.root)
	f.intentPath = filepath.Join(f.root, "intent.json")
	if err := os.WriteFile(f.intentPath, []byte(planHandoverIntent), 0o600); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f planHandoverFixture) run(t *testing.T, childArgs []string, extra ...string) (planHandoverReport, int, string) {
	t.Helper()
	childArg, err := json.Marshal(childArgs)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{
		"self-build",
		"-workspace", f.workspace,
		"-intent", "tei.native.runtime.reference",
		"-intent-file", f.intentPath,
		"-child", f.builder,
		"-child-arg", string(childArg),
		"-state", filepath.Join(f.root, "state"),
		"-output", filepath.Join(f.root, "artifact"),
	}
	args = append(args, extra...)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(f.cli, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	var report planHandoverReport
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("decode the report: %v\n%s", err, stdout.String())
		}
	}
	return report, code, stderr.String()
}

func (f planHandoverFixture) inProcess() []string {
	return []string{
		"-build-catalog", f.catalogPath,
		"-build-catalog-digest", f.catalogDigest,
		"-build-source-root", f.repoRoot,
	}
}

func TestSelfBuildHandsItsOwnPlanToTheBuilder(t *testing.T) {
	f := newPlanHandoverFixture(t)
	// No plan file exists anywhere before the entry starts, and -child-arg names
	// none.
	report, code, stderr := f.run(t,
		[]string{"-intent", f.intentPath, "-repo", f.repoRoot, "-go", f.goExe, "-isolate"},
		append(f.inProcess(), "-hand-over-plan")...)
	if code != 0 || !report.OK {
		t.Fatalf("the attempt failed (code %d): %+v\n%s", code, report.Error, stderr)
	}
	h := report.PlanHandover
	if h == nil || h.State != "handed-over" {
		t.Fatalf("planHandover = %+v", h)
	}
	if h.UnchangedAfterChild == nil || !*h.UnchangedAfterChild {
		t.Fatalf("the handed-over file was not confirmed unchanged: %+v", h)
	}
	// The child was started with exactly the file the entry wrote.
	argv := report.ChildRun.Invoked.Argv
	found := false
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == "-plan" {
			if found || argv[i+1] != h.PlanPath {
				t.Fatalf("the child argv names another plan or two plans: %v", argv)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("the child argv carries no -plan: %v", argv)
	}
	// Three statements of one fingerprint: what the entry resolved, what it
	// wrote, and what the builder says it built.
	if h.PlanFingerprint == "" || h.PlanFingerprint != report.PlanReceipt.PlanFingerprint ||
		h.PlanFingerprint != report.ChildRun.ChildPlanFingerprint {
		t.Fatalf("fingerprints differ: resolved %q, written %q, built %q",
			report.PlanReceipt.PlanFingerprint, h.PlanFingerprint, report.ChildRun.ChildPlanFingerprint)
	}
	if report.PlanCrossCheck == nil || report.PlanCrossCheck.State != planCrossCheckMatched {
		t.Fatalf("planCrossCheck = %+v", report.PlanCrossCheck)
	}
	if report.L2B == nil || !report.L2B.Claimed {
		t.Fatalf("l2b was not claimed: %+v", report.L2B)
	}
	// The entry cleans up the plan it wrote.
	if _, err := os.Stat(h.PlanPath); !os.IsNotExist(err) {
		t.Fatalf("the handed-over plan file was left behind: %v", err)
	}
}

func TestSelfBuildRefusesTwoPlansForOneChild(t *testing.T) {
	f := newPlanHandoverFixture(t)
	callerPlan := filepath.Join(f.root, "caller-plan.json")
	if err := os.WriteFile(callerPlan, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, spelling := range [][]string{{"-plan", callerPlan}, {"-plan=" + callerPlan}} {
		report, code, stderr := f.run(t,
			append(spelling, "-repo", f.repoRoot, "-go", f.goExe),
			append(f.inProcess(), "-hand-over-plan")...)
		if code != 2 || !strings.Contains(stderr, "-child-arg must not set -plan") {
			t.Fatalf("%v: code %d, stderr %q", spelling, code, stderr)
		}
		if report.ChildRun != nil {
			t.Fatalf("%v: the child was started", spelling)
		}
		if _, err := os.Stat(filepath.Join(f.root, "artifact")); !os.IsNotExist(err) {
			t.Fatalf("%v: an artifact exists after a refusal", spelling)
		}
	}
}

func TestSelfBuildCannotHandOverAPlanItDidNotResolve(t *testing.T) {
	f := newPlanHandoverFixture(t)
	// Without -build-catalog the entry resolves nothing in process.
	report, code, stderr := f.run(t,
		[]string{"-repo", f.repoRoot, "-go", f.goExe},
		"-hand-over-plan")
	if code != 2 || !strings.Contains(stderr, "needs a plan this entry resolved in process") {
		t.Fatalf("code %d, stderr %q", code, stderr)
	}
	if report.ChildRun != nil {
		t.Fatal("the child was started")
	}
}

func TestSelfBuildSaysWhenThePlanCameFromTheCaller(t *testing.T) {
	f := newPlanHandoverFixture(t)
	// The older shape: no hand-over. The child gets no plan here and fails, but
	// the record must still say the plan was not handed over rather than omit it.
	report, _, _ := f.run(t,
		[]string{"-repo", f.repoRoot, "-go", f.goExe},
		f.inProcess()...)
	if report.PlanHandover == nil || report.PlanHandover.State != "not-requested" || report.PlanHandover.Reason == "" {
		t.Fatalf("planHandover = %+v", report.PlanHandover)
	}
}

// The post-run check has a value of its own, so it gets a case that produces it.
func TestConfirmPlanUnchangedDetectsARewrittenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	original := []byte("{\"a\":1}\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	record := &selfBuildPlanHandover{State: planHandoverHanded, PlanPath: path, PlanBytesDigest: digestOfBytes(original)}
	confirmPlanUnchanged(record)
	if record.UnchangedAfterChild == nil || !*record.UnchangedAfterChild || record.Reason != "" {
		t.Fatalf("an untouched file was reported as changed: %+v", record)
	}
	if err := os.WriteFile(path, []byte("{\"a\":2}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	confirmPlanUnchanged(record)
	if record.UnchangedAfterChild == nil || *record.UnchangedAfterChild || record.Reason == "" {
		t.Fatalf("a rewritten file was reported as unchanged: %+v", record)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	confirmPlanUnchanged(record)
	if *record.UnchangedAfterChild {
		t.Fatal("a removed file was reported as unchanged")
	}
}
