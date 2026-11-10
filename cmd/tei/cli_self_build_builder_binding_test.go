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

// §4.46: these tests pin what "builder identity" means in an attempt.
//
// The plan declares a builder by digest; the entry digests the file it starts.
// The claim holds only when the two agree. The cases below run real child
// processes and cover both ways the started file can differ from the declared
// builder — the named file is replaced directly, or the named file is a proxy
// that launches something else — and they also pin the limit that remains: a
// proxy that is itself the declared builder is one declared builder.

type builderBindingResult struct {
	OK  bool `json:"ok"`
	L2B *struct {
		Claimed bool     `json:"claimed"`
		Unmet   []string `json:"unmet"`
	} `json:"l2b"`
	BuilderBinding *struct {
		State             string `json:"state"`
		DeclaredDigest    string `json:"declaredBuilderDigest"`
		StartedFileDigest string `json:"startedFileDigest"`
		Matched           bool   `json:"matched"`
	} `json:"builderBinding"`
	ArtifactReproducibility *struct {
		State             string `json:"state"`
		MismatchReason    string `json:"mismatchReason"`
		FirstChildDigest  string `json:"firstChildDigest"`
		SecondChildDigest string `json:"secondChildDigest"`
	} `json:"artifactReproducibility"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const placeholderBuilderDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"

// declareBuilderInCatalog rewrites the fixture catalog so its provider names the
// bytes of the given file, the way the authoring tool does from a real artifact
// (§4.33). The catalog digest the caller must pass changes with it.
func declareBuilderInCatalog(t *testing.T, fixture *inProcessFixture, builder string) string {
	t.Helper()
	digest, err := digestOfFile(builder)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(inProcessCatalogBody, placeholderBuilderDigest) {
		t.Fatal("the fixture catalog no longer carries the placeholder builder digest")
	}
	catalog := []byte(strings.Replace(inProcessCatalogBody, placeholderBuilderDigest, digest, 1))
	if err := os.WriteFile(fixture.catalogPath, catalog, 0o600); err != nil {
		t.Fatal(err)
	}
	fixture.catalogSha = digestOfBytes(catalog)
	return digest
}

// buildChildVariant compiles the reporting child with extra source appended, so
// two builders can have the same behaviour and different bytes.
func buildChildVariant(t *testing.T, dir, name, extraSource string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	work := filepath.Join(dir, name+"-src")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(work, "main.go")
	if err := os.WriteFile(source, []byte(intentReportingChild+extraSource), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", binary, source)
	cmd.Dir = work
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v: %s", name, err, out)
	}
	return binary
}

// variantB keeps a string the compiler cannot drop, so the bytes differ from the
// plain reporting child while the behaviour stays the same.
const variantB = `
func init() { _ = os.Getenv("TEI_TEST_BUILDER_VARIANT_B") }
`

// selfReplacing makes the child overwrite its own file with another builder on
// its first run. The running image is renamed aside first, which works on every
// platform this suite runs on.
const selfReplacing = `
func init() {
	replacement := os.Getenv("TEI_TEST_REPLACE_SELF_WITH")
	if replacement == "" {
		return
	}
	self, err := os.Executable()
	if err != nil {
		return
	}
	marker := self + ".replaced"
	if _, err := os.Stat(marker); err == nil {
		return
	}
	data, err := os.ReadFile(replacement)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	if err := os.Rename(self, self+".old"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	if err := os.WriteFile(self, data, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	_ = os.WriteFile(marker, nil, 0o600)
}
`

// proxySource keeps its own bytes and changes which builder it launches: the
// first call goes to builder A, every later call to builder B.
const proxySource = `package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	dir := os.Getenv("TEI_TEST_PROXY_DIR")
	count := 0
	if raw, err := os.ReadFile(filepath.Join(dir, "count")); err == nil {
		count, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
	}
	count++
	_ = os.WriteFile(filepath.Join(dir, "count"), []byte(strconv.Itoa(count)), 0o600)
	target := os.Getenv("TEI_TEST_PROXY_A")
	if count > 1 {
		target = os.Getenv("TEI_TEST_PROXY_B")
	}
	log, err := os.OpenFile(filepath.Join(dir, "launched.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err == nil {
		fmt.Fprintf(log, "%d %s\n", count, filepath.Base(target))
		log.Close()
	}
	cmd := exec.Command(target, os.Args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(4)
	}
}
`

func buildProxy(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	work := filepath.Join(dir, "proxy-src")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(work, "main.go")
	if err := os.WriteFile(source, []byte(proxySource), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "proxy")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", binary, source)
	cmd.Dir = work
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build proxy: %v: %s", err, out)
	}
	return binary
}

// runBuilderBindingAttempt runs one in-process attempt (second build on by
// default, §4.30) with the given file as -child.
func runBuilderBindingAttempt(t *testing.T, fixture inProcessFixture, dir, child string) (builderBindingResult, int) {
	t.Helper()
	derived := fixture.deriveForTest(t)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	var out bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-workspace", workspace,
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digestOfBytes([]byte("artifact-bytes")),
			"-report-intent-bytes", intentBytesDigestOf(t, fixture.intentPath),
			"-report-plan-fingerprint", derived.PlanFingerprint),
	}, &out)
	if err != nil && code == 0 {
		t.Fatalf("run: %v", err)
	}
	var report builderBindingResult
	if decodeErr := json.Unmarshal(out.Bytes(), &report); decodeErr != nil {
		t.Fatalf("decode: %v: %s", decodeErr, out.String())
	}
	if report.L2B == nil || report.BuilderBinding == nil || report.ArtifactReproducibility == nil {
		t.Fatalf("the report lacks a record this test reads: %s", out.String())
	}
	return report, code
}

func requireUnmetNamesTheBuilder(t *testing.T, report builderBindingResult) {
	t.Helper()
	if report.L2B.Claimed {
		t.Fatal("L2-b was claimed although the started file is not the declared builder")
	}
	if joined := strings.Join(report.L2B.Unmet, "\n"); !strings.Contains(joined, "not shown to be the builder the plan declares") {
		t.Fatalf("unmet does not name the builder binding:\n%s", joined)
	}
}

// The positive case: the catalog names the bytes of the file that is started.
func TestBuilderBindingHoldsWhenTheStartedFileIsTheDeclaredBuilder(t *testing.T) {
	fixture := writeInProcessFixture(t)
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	declared := declareBuilderInCatalog(t, &fixture, child)

	report, code := runBuilderBindingAttempt(t, fixture, dir, child)
	if code != 0 || !report.OK {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.BuilderBinding.State != builderBindingBound || !report.BuilderBinding.Matched {
		t.Fatalf("builderBinding = %+v, want bound", report.BuilderBinding)
	}
	if report.BuilderBinding.DeclaredDigest != declared || report.BuilderBinding.StartedFileDigest != declared {
		t.Fatalf("builderBinding records %+v, want both %s", report.BuilderBinding, declared)
	}
	if !report.L2B.Claimed {
		t.Fatalf("the claim did not hold: unmet=%v", report.L2B.Unmet)
	}
}

// A catalog that names other bytes than the started file used to pass: the
// lookup does not match on digest, and nothing compared the two afterwards.
func TestBuilderBindingWithholdsTheClaimWhenTheCatalogNamesOtherBytes(t *testing.T) {
	fixture := writeInProcessFixture(t)
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)

	report, code := runBuilderBindingAttempt(t, fixture, dir, child)
	// The attempt itself succeeded: the child ran and its artifact was checked.
	if code != 0 || !report.OK {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.ArtifactReproducibility.State != reproducibilityReproduced {
		t.Fatalf("reproducibility = %q, want reproduced", report.ArtifactReproducibility.State)
	}
	if report.BuilderBinding.State != builderBindingMismatched {
		t.Fatalf("builderBinding = %+v, want mismatched", report.BuilderBinding)
	}
	if report.BuilderBinding.DeclaredDigest != placeholderBuilderDigest {
		t.Fatalf("declared digest = %s", report.BuilderBinding.DeclaredDigest)
	}
	requireUnmetNamesTheBuilder(t, report)
}

// Proxy: the named file keeps its bytes and launches a different builder on the
// second build. Reproduction cannot see this — both digests are the proxy's and
// the artifacts agree. The binding to the declared builder is what withholds the
// claim.
func TestBuilderBindingWithholdsTheClaimBehindAProxy(t *testing.T) {
	fixture := writeInProcessFixture(t)
	dir := t.TempDir()
	builderA := buildChildVariant(t, dir, "builder-a", "")
	builderB := buildChildVariant(t, dir, "builder-b", variantB)
	proxy := buildProxy(t, dir)
	digestA, _ := digestOfFile(builderA)
	digestB, _ := digestOfFile(builderB)
	digestProxy, _ := digestOfFile(proxy)
	if digestA == digestB {
		t.Fatal("the two builders are byte-identical; the experiment would prove nothing")
	}
	// The catalog declares the real builder. The caller starts the proxy.
	declareBuilderInCatalog(t, &fixture, builderA)
	t.Setenv("TEI_TEST_PROXY_DIR", dir)
	t.Setenv("TEI_TEST_PROXY_A", builderA)
	t.Setenv("TEI_TEST_PROXY_B", builderB)

	report, code := runBuilderBindingAttempt(t, fixture, dir, proxy)
	if code != 0 || !report.OK {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	launched, err := os.ReadFile(filepath.Join(dir, "launched.log"))
	if err != nil {
		t.Fatalf("the proxy left no launch log: %v", err)
	}
	lines := strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(launched)), "\r", ""))
	if len(lines) != 4 || !strings.HasPrefix(lines[1], "builder-a") || !strings.HasPrefix(lines[3], "builder-b") {
		t.Fatalf("the proxy did not launch A then B: %q", string(launched))
	}
	// What reproduction sees: one file, twice, same artifact.
	repro := report.ArtifactReproducibility
	if repro.State != reproducibilityReproduced {
		t.Fatalf("reproducibility = %q (%s), want reproduced", repro.State, repro.MismatchReason)
	}
	if repro.FirstChildDigest != digestProxy || repro.SecondChildDigest != digestProxy {
		t.Fatalf("child digests = %s / %s, want the proxy's %s", repro.FirstChildDigest, repro.SecondChildDigest, digestProxy)
	}
	// What the binding sees: the started file is not the declared builder.
	if report.BuilderBinding.State != builderBindingMismatched {
		t.Fatalf("builderBinding = %+v, want mismatched", report.BuilderBinding)
	}
	if report.BuilderBinding.DeclaredDigest != digestA || report.BuilderBinding.StartedFileDigest != digestProxy {
		t.Fatalf("builderBinding records %+v", report.BuilderBinding)
	}
	requireUnmetNamesTheBuilder(t, report)
}

// The limit that remains, pinned so it cannot be read as more: when the catalog
// declares the proxy itself, the proxy is the declared builder and the claim
// holds, whatever the proxy launches. Identity is the bytes of the started file;
// what that file delegates to is its behaviour, which this entry does not observe.
func TestBuilderBindingTreatsADeclaredProxyAsTheDeclaredBuilder(t *testing.T) {
	fixture := writeInProcessFixture(t)
	dir := t.TempDir()
	builderA := buildChildVariant(t, dir, "builder-a", "")
	builderB := buildChildVariant(t, dir, "builder-b", variantB)
	proxy := buildProxy(t, dir)
	declareBuilderInCatalog(t, &fixture, proxy)
	t.Setenv("TEI_TEST_PROXY_DIR", dir)
	t.Setenv("TEI_TEST_PROXY_A", builderA)
	t.Setenv("TEI_TEST_PROXY_B", builderB)

	report, code := runBuilderBindingAttempt(t, fixture, dir, proxy)
	if code != 0 || !report.OK {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.BuilderBinding.State != builderBindingBound {
		t.Fatalf("builderBinding = %+v, want bound", report.BuilderBinding)
	}
	if !report.L2B.Claimed {
		t.Fatalf("unmet=%v", report.L2B.Unmet)
	}
}

// Direct replacement: the named file itself changes between the two builds. The
// first start is the declared builder, so the binding holds; the second start is
// other bytes, so reproduction refuses, and the attempt fails.
func TestBuilderBindingAndReproductionStopADirectReplacement(t *testing.T) {
	fixture := writeInProcessFixture(t)
	dir := t.TempDir()
	child := buildChildVariant(t, dir, "builder-self-replacing", selfReplacing)
	replacement := buildChildVariant(t, dir, "builder-b", variantB)
	declared := declareBuilderInCatalog(t, &fixture, child)
	digestReplacement, _ := digestOfFile(replacement)
	if declared == digestReplacement {
		t.Fatal("the replacement is byte-identical to the declared builder")
	}
	t.Setenv("TEI_TEST_REPLACE_SELF_WITH", replacement)

	report, code := runBuilderBindingAttempt(t, fixture, dir, child)
	if code == 0 || report.OK {
		t.Fatalf("the attempt succeeded although the builder was replaced: code=%d", code)
	}
	if report.BuilderBinding.State != builderBindingBound {
		t.Fatalf("builderBinding = %+v, want bound for the first start", report.BuilderBinding)
	}
	repro := report.ArtifactReproducibility
	if repro.State != reproducibilityMismatched || !strings.Contains(repro.MismatchReason, "different builder executables") {
		t.Fatalf("reproducibility = %q (%s), want a builder mismatch", repro.State, repro.MismatchReason)
	}
	if repro.FirstChildDigest != declared || repro.SecondChildDigest != digestReplacement {
		t.Fatalf("child digests = %s / %s", repro.FirstChildDigest, repro.SecondChildDigest)
	}
	if report.L2B.Claimed {
		t.Fatal("L2-b was claimed across a replaced builder")
	}
}

// The record itself: a missing value on either side is unperformed, never bound.
func TestBindBuilderToPlan(t *testing.T) {
	receipt := func(digest string) *PlanReceipt {
		return &PlanReceipt{ReceiptJSON: `{"plan":{"builder":{"artifactDigest":"` + digest + `"}}}`}
	}
	const d = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const e = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	cases := []struct {
		name    string
		receipt *PlanReceipt
		started string
		want    string
	}{
		{"same bytes", receipt(d), d, builderBindingBound},
		{"digest case differs only", receipt(strings.ToUpper(d)), d, builderBindingBound},
		{"other bytes", receipt(d), e, builderBindingMismatched},
		{"no receipt", nil, d, builderBindingNotRun},
		{"plan without a builder digest", receipt(""), d, builderBindingNotRun},
		{"unreadable receipt", &PlanReceipt{ReceiptJSON: "{"}, d, builderBindingNotRun},
		{"started file not digested", receipt(d), "", builderBindingNotRun},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := bindBuilderToPlan(tc.receipt, tc.started)
			if record.State != tc.want {
				t.Fatalf("state = %q, want %q (%+v)", record.State, tc.want, record)
			}
			if record.Matched != (tc.want == builderBindingBound) {
				t.Fatalf("matched = %v for state %q", record.Matched, record.State)
			}
			if record.Boundary == "" {
				t.Fatal("the record has no boundary")
			}
		})
	}
}
