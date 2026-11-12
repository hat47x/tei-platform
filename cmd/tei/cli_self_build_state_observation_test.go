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

// SB-C0 §4.55: the entry reads the two state directories itself. These tests pin
// what that reading can tell apart and, just as deliberately, what it cannot.
//
// None of the children here is a bundled builder. They are fixtures that behave
// in the ways the check has to distinguish.

// stateObservationChildSource is a child with a build "cache" in its state
// directory. -mode chooses where the cache really lives; the report always names
// the directory the child was passed.
const stateObservationChildSource = `package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	artifact, state, mode, shared, prev := "", "", "honest", "", ""
	for i := 1; i+1 < len(os.Args); i += 2 {
		switch os.Args[i] {
		case "-artifact-out":
			artifact = os.Args[i+1]
		case "-state":
			state = os.Args[i+1]
		case "-mode":
			mode = os.Args[i+1]
		case "-shared":
			shared = os.Args[i+1]
		case "-prev":
			prev = os.Args[i+1]
		}
	}
	time.Sleep(150 * time.Millisecond)
	used := state
	switch mode {
	case "share":
		used = shared
	case "copy":
		if raw, err := os.ReadFile(prev); err == nil && string(raw) != state {
			if cached, err := os.ReadFile(filepath.Join(string(raw), "cache", "object")); err == nil {
				_ = os.MkdirAll(filepath.Join(state, "cache"), 0o755)
				_ = os.WriteFile(filepath.Join(state, "cache", "object"), cached, 0o600)
			}
		}
		_ = os.WriteFile(prev, []byte(state), 0o600)
	}
	body := []byte("artifact-bytes")
	if mode != "silent" {
		object := filepath.Join(used, "cache", "object")
		if cached, err := os.ReadFile(object); err == nil {
			body = cached // a cache hit: nothing is built again
		} else {
			_ = os.MkdirAll(filepath.Dir(object), 0o755)
			_ = os.WriteFile(object, body, 0o600)
		}
	}
	if err := os.WriteFile(artifact, body, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	sum := sha256.Sum256(body)
	fmt.Printf("{\"schemaVersion\":\"fixture\",\"artifactDigest\":\"sha256:%s\",\"stateDir\":%q}\n", hex.EncodeToString(sum[:]), state)
}
`

type stateObservationReport struct {
	OK        bool `json:"ok"`
	Qualified bool `json:"qualified"`
	ChildRun  *struct {
		Invoked struct {
			ProcessImage *selfBuildProcessImage `json:"processImage"`
		} `json:"invoked"`
	} `json:"childRun"`
	ArtifactReproducibility *struct {
		State                string                     `json:"state"`
		FirstArtifactDigest  string                     `json:"firstArtifactDigest"`
		SecondArtifactDigest string                     `json:"secondArtifactDigest"`
		StateObservation     *selfBuildStateObservation `json:"stateObservation"`
	} `json:"artifactReproducibility"`
	L2B *struct {
		Claimed         bool     `json:"claimed"`
		Unmet           []string `json:"unmet"`
		StateSeparation string   `json:"stateSeparation"`
	} `json:"l2b"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func buildStateObservationChild(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "stateobservationchild.go")
	if err := os.WriteFile(source, []byte(stateObservationChildSource), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "stateobservationchild")
	if runtime.GOOS == "windows" {
		child += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", child, source)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build child fixture: %v: %s", err, out)
	}
	return child
}

func runStateObservation(t *testing.T, dir, child string, childArgs []string) stateObservationReport {
	t.Helper()
	encoded, err := json.Marshal(childArgs)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, runErr := runSelfBuildCommand([]string{
		"-child", child,
		"-child-arg", string(encoded),
		"-workspace", outerWorkspace(t, dir),
		"-intent", "intent.sample",
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "artifact"),
		"-rebuild-check",
	}, &out)
	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}
	var report stateObservationReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v: %s", err, out.String())
	}
	if report.Qualified {
		t.Fatal("the attempt was reported as qualified")
	}
	if report.ArtifactReproducibility == nil || report.ArtifactReproducibility.StateObservation == nil {
		t.Fatalf("no state observation was recorded: %s", out.String())
	}
	if len(report.ArtifactReproducibility.StateObservation.NotObserved) == 0 {
		t.Fatal("the observation does not say what it did not watch")
	}
	return report
}

func findingsContain(observation *selfBuildStateObservation, want string) bool {
	for _, finding := range observation.Findings {
		if strings.Contains(finding, want) {
			return true
		}
	}
	return false
}

// Positive case: a child that keeps its cache in the directory it was passed is
// seen to have used it, in both runs, in two different directories.
func TestStateObservationSeesAChildThatUsesTheStateItWasPassed(t *testing.T) {
	dir := t.TempDir()
	report := runStateObservation(t, dir, buildStateObservationChild(t, dir), []string{"-mode", "honest"})

	reproduction := report.ArtifactReproducibility
	if !report.OK || reproduction.State != reproducibilityReproduced {
		t.Fatalf("the honest child was not reproduced: ok=%v state=%q error=%+v", report.OK, reproduction.State, report.Error)
	}
	observation := reproduction.StateObservation
	if observation.Kind != stateObservationEffectObserved || len(observation.Findings) != 0 {
		t.Fatalf("kind = %q, findings = %v", observation.Kind, observation.Findings)
	}
	if observation.SameDirectory || observation.First.FilesAfter == 0 || observation.Second.FilesAfter == 0 {
		t.Fatalf("the two directories were not each seen in use: %+v", observation)
	}
}

// The negative case PR #4299 measured: both runs keep their cache in one shared
// directory and report the passed value. The artifact bytes agree, so the existing
// verdict still says reproduced — that is the property of the check, pinned here
// rather than changed. What changes is that the entry no longer has only the
// child's word: it saw that neither passed directory was used.
func TestStateObservationDoesNotAcceptAReportThatHidesSharedState(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared")
	report := runStateObservation(t, dir, buildStateObservationChild(t, dir), []string{"-mode", "share", "-shared", shared})

	reproduction := report.ArtifactReproducibility
	if reproduction.State != reproducibilityReproduced || reproduction.FirstArtifactDigest != reproduction.SecondArtifactDigest {
		t.Fatalf("the fixture is meant to agree byte for byte: %+v", reproduction)
	}
	observation := reproduction.StateObservation
	if observation.Kind != stateObservationDeclaredOnly {
		t.Fatalf("shared state behind a matching report was summarised as %q", observation.Kind)
	}
	if !findingsContain(observation, "the first run") || !findingsContain(observation, "the second run") {
		t.Fatalf("the findings do not name both runs: %v", observation.Findings)
	}
	if !findingsContain(observation, "does not exist after the run") {
		t.Fatalf("the contradiction with the child's report is not named: %v", observation.Findings)
	}
	if _, err := os.Stat(filepath.Join(shared, "cache", "object")); err != nil {
		t.Fatalf("the fixture did not actually share state: %v", err)
	}
}

// Control: a child that reports the state it was passed and keeps nothing there is
// what every earlier fixture in this package does. It must keep working, and it is
// summarised as declared-only, not as a failure.
func TestStateObservationDoesNotRejectAChildThatKeepsNoState(t *testing.T) {
	dir := t.TempDir()
	report := runStateObservation(t, dir, buildStateObservationChild(t, dir), []string{"-mode", "silent"})

	reproduction := report.ArtifactReproducibility
	if !report.OK || reproduction.State != reproducibilityReproduced {
		t.Fatalf("a child that keeps no state was refused: ok=%v state=%q error=%+v", report.OK, reproduction.State, report.Error)
	}
	if reproduction.StateObservation.Kind != stateObservationDeclaredOnly {
		t.Fatalf("kind = %q", reproduction.StateObservation.Kind)
	}
}

// The limit, stated as a test so nobody reads effect-observed as isolation: a
// child that copies the first run's cache into the second run's directory leaves
// new files in both, so this entry sees effect-observed. Only an OS boundary that
// hides the first run's state from the second can tell this apart, and this entry
// sets none up. The record says so in notObserved.
func TestStateObservationCannotSeeStateCopiedBetweenRuns(t *testing.T) {
	dir := t.TempDir()
	report := runStateObservation(t, dir, buildStateObservationChild(t, dir), []string{"-mode", "copy", "-prev", filepath.Join(dir, "prev")})

	observation := report.ArtifactReproducibility.StateObservation
	if observation.Kind != stateObservationEffectObserved {
		t.Fatalf("kind = %q, findings = %v; if this entry can now see the copy, update §4.55 and this test together", observation.Kind, observation.Findings)
	}
	joined := strings.Join(observation.NotObserved, " ")
	if !strings.Contains(joined, "no OS boundary") || !strings.Contains(joined, "not excluded") {
		t.Fatalf("the record does not state the limit: %v", observation.NotObserved)
	}
	if !strings.Contains(observation.Boundary, "does not show that no state was shared") {
		t.Fatalf("the boundary does not state the limit: %s", observation.Boundary)
	}
}

// The process image is what the OS ran, the executable digest is the file -child
// named. A script makes the two differ: the image is its interpreter.
func TestSelfBuildRecordsTheProcessImageBesideTheNamedFile(t *testing.T) {
	if runtime.GOOS != "linux" {
		image := observeProcessImage(os.Getpid(), "anything")
		if image.NotObserved == "" || image.MatchesExecutable {
			t.Fatalf("a host without /proc must say the image was not observed: %+v", image)
		}
		t.Skip("the process image is read from /proc; the reading itself is linux-only")
	}
	dir := t.TempDir()
	child := buildStateObservationChild(t, dir)

	direct := runStateObservation(t, dir, child, []string{"-mode", "honest"})
	image := direct.ChildRun.Invoked.ProcessImage
	if image == nil || !image.MatchesExecutable {
		t.Fatalf("a directly started binary was not recorded as its own image: %+v", image)
	}

	wrapped := t.TempDir()
	script := filepath.Join(wrapped, "wrapper")
	body := "#!/bin/sh\nsleep 0.2\nexec " + child + " \"$@\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	viaScript := runStateObservation(t, wrapped, script, []string{"-mode", "honest"})
	image = viaScript.ChildRun.Invoked.ProcessImage
	if image == nil || image.NotObserved != "" || image.MatchesExecutable {
		t.Fatalf("a script was recorded as if the named file were the image: %+v", image)
	}
	if !viaScript.OK {
		t.Fatalf("the image is recorded, not enforced; the attempt should still succeed: %+v", viaScript.Error)
	}
}

// The combined claim repeats the kind and is not decided by it: §4.37's owner has
// not changed the five conditions.
func TestAssessL2BRepeatsStateSeparationWithoutDependingOnIt(t *testing.T) {
	for _, kind := range []string{stateObservationDeclaredOnly, stateObservationEffectObserved} {
		report := satisfiedL2B()
		report.ArtifactReproducibility.StateObservation = &selfBuildStateObservation{Kind: kind}
		claim := assessL2B(report)
		if !claim.Claimed || claim.StateSeparation != kind {
			t.Fatalf("kind %q: claimed=%v stateSeparation=%q unmet=%v", kind, claim.Claimed, claim.StateSeparation, claim.Unmet)
		}
	}
	if !strings.Contains(l2bClaimBoundary, "shared state is not excluded") {
		t.Fatalf("the claim boundary does not limit the state statement: %s", l2bClaimBoundary)
	}
	if strings.Contains(reproducibilityBoundary, "in separate state directories") {
		t.Fatalf("the reproduction boundary still asserts separation: %s", reproducibilityBoundary)
	}
}

func TestAssessStateObservationNamesEachReason(t *testing.T) {
	used := func(path string) stateDirObservation {
		return stateDirObservation{Path: path, ExistsAfter: true, FilesAfter: 3, BytesAfter: 9}
	}
	cases := []struct {
		name          string
		first, second stateDirObservation
		same          bool
		wantKind      string
		wantFinding   string
	}{
		{"both used", used("a"), used("b"), false, stateObservationEffectObserved, ""},
		{"same object", used("a"), used("a"), true, stateObservationDeclaredOnly, "same directory object"},
		{"not fresh", stateDirObservation{Path: "a", ExistedBefore: true, FilesBefore: 2, ExistsAfter: true, FilesAfter: 5}, used("b"), false, stateObservationDeclaredOnly, "was not fresh"},
		{"untouched", stateDirObservation{Path: "a", ExistsAfter: true}, used("b"), false, stateObservationDeclaredOnly, "left no new file"},
		{"unreadable", stateDirObservation{Path: "a", ReadError: "permission denied"}, used("b"), false, stateObservationDeclaredOnly, "could not be read"},
	}
	for _, tc := range cases {
		record := assessStateObservation(tc.first, tc.second, tc.same, "a", "b")
		if record.Kind != tc.wantKind {
			t.Fatalf("%s: kind = %q, findings = %v", tc.name, record.Kind, record.Findings)
		}
		if tc.wantFinding != "" && !findingsContain(&record, tc.wantFinding) {
			t.Fatalf("%s: findings %v do not contain %q", tc.name, record.Findings, tc.wantFinding)
		}
		if len(record.NotObserved) == 0 || record.Boundary == "" {
			t.Fatalf("%s: the record does not state its limits", tc.name)
		}
	}
}

// stateSharingWrapperSource wraps a real builder. It moves -state to one shared
// directory and writes the passed value back into the builder's report, which is
// the construction PR #4299 measured.
const stateSharingWrapperSource = `package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
)

func main() {
	args := append([]string(nil), os.Args[1:]...)
	passed := ""
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-state" {
			passed = args[i+1]
			args[i+1] = os.Getenv("TEI_TEST_SHARED_STATE")
		}
	}
	var stdout bytes.Buffer
	cmd := exec.Command(os.Getenv("TEI_TEST_REAL_BUILDER"), args...)
	cmd.Stdout, cmd.Stderr = &stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		os.Stdout.Write(stdout.Bytes())
		os.Exit(1)
	}
	var report map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		os.Exit(1)
	}
	report["stateDir"] = passed
	_ = json.NewEncoder(os.Stdout).Encode(report)
}
`

// The same pair on the real Go builder and a real go build. It costs two real
// builds per case, so it is opt-in like the full chain (§4.51) and runs where the
// catalog's target does.
func TestStateObservationOnARealGoBuild(t *testing.T) {
	if os.Getenv("TEI_SELF_BUILD_FULL_CHAIN") == "" {
		t.Skip("two real go builds per case; set TEI_SELF_BUILD_FULL_CHAIN=1 to run it")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("the catalog names host/linux/amd64; this host is %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cli := buildToolBinary(t, ".", "tei")
	builder := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-go-build-from-plan"), "go-build-from-plan")

	wrapperDir := t.TempDir()
	wrapperSource := filepath.Join(wrapperDir, "wrapper.go")
	if err := os.WriteFile(wrapperSource, []byte(stateSharingWrapperSource), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(wrapperDir, "wrapper")
	if out, err := exec.Command(goExe, "build", "-trimpath", "-o", wrapper, wrapperSource).CombinedOutput(); err != nil {
		t.Fatalf("build the wrapper: %v\n%s", err, out)
	}

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
	run := func(t *testing.T, child string, shared string) stateObservationReport {
		root := t.TempDir()
		catalogPath, catalogDigest := authorGoCommandCatalog(t, root, builder)
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
		planPath := filepath.Join(root, "plan.json")
		if err := os.WriteFile(planPath, planJSON, 0o600); err != nil {
			t.Fatal(err)
		}
		childArg, err := json.Marshal([]string{"-plan", planPath, "-intent", intentPath, "-repo", repoRoot, "-go", goExe, "-isolate"})
		if err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		cmd := exec.Command(cli, "self-build",
			"-workspace", outerWorkspace(t, root),
			"-intent", "tei.native.runtime.reference",
			"-intent-file", intentPath,
			"-build-catalog", catalogPath,
			"-build-catalog-digest", catalogDigest,
			"-build-source-root", repoRoot,
			"-child", child,
			"-child-arg", string(childArg),
			"-state", filepath.Join(root, "state"),
			"-output", filepath.Join(root, "artifact"),
		)
		cmd.Env = append(os.Environ(), "TEI_TEST_REAL_BUILDER="+builder, "TEI_TEST_SHARED_STATE="+shared)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		_ = cmd.Run()
		var report stateObservationReport
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("decode the report: %v\n%s\n%s", err, stdout.String(), stderr.String())
		}
		if report.Qualified {
			t.Fatal("the attempt was reported as qualified")
		}
		if report.ArtifactReproducibility == nil || report.ArtifactReproducibility.StateObservation == nil || report.L2B == nil {
			t.Fatalf("the report is incomplete: %s\n%s", stdout.String(), stderr.String())
		}
		return report
	}

	t.Run("the bundled builder uses the state it is passed", func(t *testing.T) {
		report := run(t, builder, "")
		observation := report.ArtifactReproducibility.StateObservation
		if !report.OK || !report.L2B.Claimed {
			t.Fatalf("the normal build was refused: ok=%v l2b=%+v error=%+v", report.OK, report.L2B, report.Error)
		}
		if observation.Kind != stateObservationEffectObserved || report.L2B.StateSeparation != stateObservationEffectObserved {
			t.Fatalf("kind = %q, findings = %v", observation.Kind, observation.Findings)
		}
		t.Logf("first: %d files, %d bytes; second: %d files, %d bytes", observation.First.FilesAfter, observation.First.BytesAfter, observation.Second.FilesAfter, observation.Second.BytesAfter)
	})

	t.Run("shared state behind a matching report is not summarised as observed", func(t *testing.T) {
		shared := filepath.Join(t.TempDir(), "shared")
		report := run(t, wrapper, shared)
		reproduction := report.ArtifactReproducibility
		if reproduction.State != reproducibilityReproduced || reproduction.FirstArtifactDigest != reproduction.SecondArtifactDigest {
			t.Fatalf("the two builds were meant to agree byte for byte: %+v error=%+v", reproduction, report.Error)
		}
		// The five conditions are unchanged, so the claim still stands; what the
		// reader now gets is that the separation is declared, not seen.
		if !report.L2B.Claimed {
			t.Fatalf("this test pins the current predicate; if it changed, §4.55 must change with it: %+v", report.L2B)
		}
		if reproduction.StateObservation.Kind != stateObservationDeclaredOnly || report.L2B.StateSeparation != stateObservationDeclaredOnly {
			t.Fatalf("kind = %q, stateSeparation = %q", reproduction.StateObservation.Kind, report.L2B.StateSeparation)
		}
		entries, err := os.ReadDir(shared)
		if err != nil || len(entries) == 0 {
			t.Fatalf("the wrapper did not actually share state: %v", err)
		}
		t.Logf("findings: %v", reproduction.StateObservation.Findings)
	})
}
