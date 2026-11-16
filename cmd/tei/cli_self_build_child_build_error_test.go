package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A child that exits 0 while reporting that it did not build. The plan-reading
// builder does this when, for example, its Go toolchain is older than go.mod
// asks for. The entry used to report only the consequence (no artifact).
const reportingChildSource = `package main

import (
	"fmt"
	"os"
)

func main() {
	switch os.Getenv("CHILD_MODE") {
	case "reason":
		fmt.Println("{\"schemaVersion\":\"test/v0\",\"buildSucceeded\":false,\"buildError\":\"go.mod requires go >= 9.9\"}")
	case "silent":
		fmt.Println("{\"schemaVersion\":\"test/v0\",\"buildSucceeded\":false}")
	case "success-without-artifact":
		fmt.Println("{\"schemaVersion\":\"test/v0\",\"buildSucceeded\":true}")
	}
}
`

func buildReportingChild(t *testing.T) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "child.go")
	if err := os.WriteFile(source, []byte(reportingChildSource), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "child")
	if runtime.GOOS == "windows" {
		child += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", child, source)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build the child: %v: %s", err, out)
	}
	return child
}

func TestSelfBuildStatesTheChildsOwnBuildErrorFirst(t *testing.T) {
	child := buildReportingChild(t)
	artifact := filepath.Join(t.TempDir(), "artifact")
	cases := []struct {
		mode      string
		wantError string // exact childBuildError; "" means the field must be absent
	}{
		{"reason", "go.mod requires go >= 9.9"},
		{"silent", "the child reported buildSucceeded=false without a reason"},
		{"success-without-artifact", ""},
	}
	for _, c := range cases {
		t.Setenv("CHILD_MODE", c.mode)
		evidence, err := driveSelfBuild(child, nil, artifact, "")
		if err != nil {
			t.Fatalf("%s: %v", c.mode, err)
		}
		if evidence.ChildBuildError != c.wantError {
			t.Errorf("%s: childBuildError = %q, want %q", c.mode, evidence.ChildBuildError, c.wantError)
		}
		// Every mode leaves no artifact, so every mode fails; only the stated
		// cause differs.
		if evidence.CrossCheckOK || evidence.FailureCause == "" {
			t.Errorf("%s: a run without an artifact was not a failure: %+v", c.mode, evidence)
		}
		startsWithChild := strings.HasPrefix(evidence.FailureCause, "the child reported that it did not build: ")
		if (c.wantError != "") != startsWithChild {
			t.Errorf("%s: failureCause = %q", c.mode, evidence.FailureCause)
		}
		if c.wantError != "" && !strings.Contains(evidence.FailureCause, c.wantError) {
			t.Errorf("%s: failureCause does not carry the child's reason: %q", c.mode, evidence.FailureCause)
		}
	}
}
