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
)

// These tests pin §4.21: a second, independent build of the same plan must
// produce the same artifact bytes before this entry will say the artifact is that
// plan's output. The lock (§4.11) ties a plan fingerprint to an artifact digest;
// only reproduction shows the builder actually yields those bytes.

// The fixture child writes the bytes the caller names, so the same invocation
// twice is reproducible and a changed argument is not.
const reproducibleChildSource = `package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The child reports the digest of the bytes it actually wrote, so the entry's
// own disk check can pass; the -counter file makes the second run differ.
func main() {
	artifact, counter, state := "", "", ""
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-artifact", "-counter", "-state":
			value := os.Args[i+1]
			switch os.Args[i] {
			case "-artifact":
				artifact = value
			case "-counter":
				counter = value
			case "-state":
				state = value
			}
			i++
		}
	}
	body := "artifact-bytes"
	if counter != "" {
		previous := ""
		if raw, err := os.ReadFile(counter); err == nil {
			previous = string(raw)
		}
		if previous != "" {
			body = "artifact-bytes-" + strings.TrimSpace(previous)
		}
		_ = os.WriteFile(counter, []byte(previous+"x"), 0o600)
	}
	if artifact != "" {
		if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := os.WriteFile(artifact, []byte(body), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	sum := sha256.Sum256([]byte(body))
	if state != "" {
		fmt.Printf("{\"schemaVersion\":\"tei.reference.go-build-with-supply/v0\",\"artifactDigest\":\"sha256:%s\",\"stateDir\":%q}\n", hex.EncodeToString(sum[:]), state)
	} else {
		fmt.Printf("{\"schemaVersion\":\"tei.reference.go-build-with-supply/v0\",\"artifactDigest\":\"sha256:%s\"}\n", hex.EncodeToString(sum[:]))
	}
}
`

func buildReproducibleChild(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "reproduciblechild.go")
	if err := os.WriteFile(source, []byte(reproducibleChildSource), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "reproduciblechild")
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

type selfBuildReproducibilityResult struct {
	OK                      bool `json:"ok"`
	Qualified               bool `json:"qualified"`
	ArtifactReproducibility *struct {
		State             string `json:"state"`
		UnperformedReason string `json:"unperformedReason"`
		FirstDigest       string `json:"firstArtifactDigest"`
		SecondDigest      string `json:"secondArtifactDigest"`
		SecondChildDigest string `json:"secondChildDigest"`
		OnDiskDigest      string `json:"onDiskDigest"`
		Boundary          string `json:"boundary"`
	} `json:"artifactReproducibility"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runReproducibilityShape(t *testing.T, args ...string) (selfBuildReproducibilityResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil && code == 0 {
		t.Fatalf("run: %v", err)
	}
	var report selfBuildReproducibilityResult
	if out.Len() > 0 {
		if decodeErr := json.Unmarshal(out.Bytes(), &report); decodeErr != nil {
			t.Fatalf("decode report: %v: %s", decodeErr, out.String())
		}
	}
	return report, code
}

// A deterministic child asked to build twice is reported as reproduced, and the
// record names the artifact the attempt actually left on disk.
func TestSelfBuildRebuildCheckReportsReproduced(t *testing.T) {
	dir := t.TempDir()
	child := buildReproducibleChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	report, code := runReproducibilityShape(t,
		"-child", child,
		"-workspace", workspace,
		"-intent", "intent.sample",
		"-intent-file", outerIntentFile(t, dir),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output),
		"-rebuild-check",
	)
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	record := report.ArtifactReproducibility
	if record == nil {
		t.Fatal("no reproducibility record")
	}
	if record.State != reproducibilityReproduced {
		t.Fatalf("state=%q reason=%q", record.State, record.UnperformedReason)
	}
	if record.FirstDigest == "" || record.FirstDigest != record.SecondDigest ||
		record.FirstDigest != record.OnDiskDigest {
		t.Fatalf("digests do not agree: %+v", record)
	}
	if record.SecondChildDigest == "" {
		t.Fatal("the second run's executable digest is missing")
	}
	if record.Boundary == "" {
		t.Fatal("the boundary is missing")
	}
	if report.Qualified {
		t.Fatal("reproducibility reported the attempt as qualified")
	}
}

// The boundary is machine-readable output and the only place a caller learns how far
// the reproduction result reaches. It said cross-host agreement had been seen on two
// hosts only, after arm64 had been observed, and it said nothing about arm64's artifacts
// differing from amd64's. Pin both: the third host is named, and the refusal to claim
// cross-architecture equality is stated.
func TestSelfBuildReproducibilityBoundaryStatesTheObservedScope(t *testing.T) {
	boundary := reproducibilityBoundary

	for _, want := range []string{
		"linux/amd64 and windows/amd64",
		"linux/arm64 under emulation",
		"cross-architecture equality of artifact bytes is not claimed",
	} {
		if !strings.Contains(boundary, want) {
			t.Fatalf("the boundary no longer states %q: %s", want, boundary)
		}
	}

	// The old sentence implied that the two hosts were the whole of what had been
	// seen. arm64 was observed, so that reading no longer holds.
	if strings.Contains(boundary, "It is not a proof for any other host") &&
		!strings.Contains(boundary, "linux/arm64 under emulation") {
		t.Fatalf("the boundary still treats the two hosts as exhaustive: %s", boundary)
	}
}

// A child that writes different bytes the second time is not reproducible, and
// the attempt fails: without reproduction the artifact cannot be attributed to
// the plan.
func TestSelfBuildRebuildCheckReportsMismatch(t *testing.T) {
	dir := t.TempDir()
	child := buildReproducibleChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	report, code := runReproducibilityShape(t,
		"-child", child,
		"-workspace", workspace,
		"-intent", "intent.sample",
		"-intent-file", outerIntentFile(t, dir),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output,
			"-counter", filepath.Join(dir, "invocations")),
		"-rebuild-check",
	)
	if code != 1 {
		t.Fatalf("a differing second build did not fail the attempt: code=%d", code)
	}
	if report.OK {
		t.Fatal("the attempt was reported as ok")
	}
	record := report.ArtifactReproducibility
	if record == nil || record.State != reproducibilityMismatched {
		t.Fatalf("state=%+v", record)
	}
}

// Without the flag the record still appears, and says why it did not run: a
// missing field would let a reader take silence for agreement.
func TestSelfBuildRebuildCheckRecordsWhyItDidNotRun(t *testing.T) {
	dir := t.TempDir()
	child := buildReproducibleChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	report, code := runReproducibilityShape(t,
		"-child", child,
		"-workspace", workspace,
		"-intent", "intent.sample",
		"-intent-file", outerIntentFile(t, dir),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output),
	)
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	record := report.ArtifactReproducibility
	if record == nil {
		t.Fatal("no reproducibility record without the flag")
	}
	if record.State != reproducibilityUnperformed || record.UnperformedReason == "" {
		t.Fatalf("state=%q reason=%q", record.State, record.UnperformedReason)
	}
	if record.FirstDigest != "" || record.SecondDigest != "" {
		t.Fatalf("digests were reported for a check that did not run: %+v", record)
	}
}

func TestWithArtifactAndStateReplacesOnlyItsOwnValues(t *testing.T) {
	args := []string{"-plan", "p.json", "-artifact-out", "old", "-state", "old-state", "-intent", "i.json"}
	got, err := withArtifactAndState(args, "new", "new-state")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-plan", "p.json", "-artifact-out", "new", "-state", "new-state", "-intent", "i.json"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("got %q want %q", got, want)
	}
	if strings.Join(args, "\x00") == strings.Join(got, "\x00") {
		t.Fatal("the caller's argv was modified in place")
	}
}

func TestWithArtifactAndStateRequiresBothFlags(t *testing.T) {
	if _, err := withArtifactAndState([]string{"-plan", "p.json"}, "out", "state"); err == nil {
		t.Fatal("argv with neither flag was accepted")
	}
	if _, err := withArtifactAndState([]string{"-artifact-out", "a"}, "out", "state"); err == nil {
		t.Fatal("argv with no -state was accepted")
	}
}
