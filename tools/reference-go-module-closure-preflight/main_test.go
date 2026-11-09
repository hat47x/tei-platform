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

func decodeOne(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var out map[string]any
	if err := decoder.Decode(&out); err != nil {
		t.Fatalf("preflight output is not one JSON object: %v\n%s", err, raw)
	}
	return out
}

func TestRunReportsMissingModuleSupplyForThrowawayTarget(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	root := t.TempDir()
	write := func(name, content string) {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.invalid/missing\n\ngo 1.23\n\nrequire example.invalid/absent v1.0.0\n")
	write("go.sum", "")
	write("cmd/app/main.go", "package main\n\nimport _ \"example.invalid/absent\"\n\nfunc main() {}\n")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-repo", root, "-package", "./cmd/app"}, &stdout, &stderr); code != 0 {
		t.Fatalf("preflight did not complete: code=%d stderr=%s", code, stderr.String())
	}
	out := decodeOne(t, stdout.Bytes())
	if out["classification"] != "missing-module" {
		t.Fatalf("offline supply gap was not classified: %v", out["classification"])
	}
	if out["sourceClosureReachable"] != false {
		t.Fatal("unreachable closure was reported reachable")
	}
}

func TestRunRejectsMissingPackageFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("missing -package did not fail usage: code=%d", code)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "-package is required") {
		t.Fatalf("usage failure leaked output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

// This asserts the observation the plan's SB-0a asks for against the current
// repository: whether cmd/tei reaches an external module offline. The wanted
// value is NOT hard-coded; the test only pins that the classification is one
// of the defined values and that an unreachable closure never reads as
// supported.
func TestRunClassifiesCurrentCmdTeiWithoutClaimingAuthority(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git needed for the repository fixture: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-repo", repo, "-package", "./cmd/tei"}, &stdout, &stderr); code != 0 {
		t.Fatalf("cmd/tei preflight did not complete: code=%d stderr=%s", code, stderr.String())
	}
	out := decodeOne(t, stdout.Bytes())
	classification, _ := out["classification"].(string)
	switch classification {
	case "supported", "external-unresolved", "missing-module", "unknown":
	default:
		t.Fatalf("preflight returned an undefined classification: %q", classification)
	}
	if classification != "supported" && out["sourceClosureReachable"] == true {
		t.Fatal("unreachable closure was reported reachable")
	}
	if out["nature"] != "offline go list -e observation; not a build attempt" {
		t.Fatalf("preflight lost its scope statement: %v", out["nature"])
	}
	if out["goExecutableDigest"] == "" || out["goos"] != runtime.GOOS || out["goarch"] != runtime.GOARCH {
		t.Fatalf("preflight identity fields are incomplete: %v", out)
	}
}
