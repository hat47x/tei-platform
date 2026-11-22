package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const selfBuildReportedFailureChild = `package main
import (
 "fmt"
 "os"
)
func main() {
 switch os.Getenv("CHILD_MODE") {
 case "reason":
  fmt.Printf("{%q:%q,%q:%v,%q:%q}\n", "schemaVersion", "test/v0", "buildSucceeded", false, "buildError", "go.mod requires go >= 9.9")
 case "silent":
  fmt.Printf("{%q:%q,%q:%v}\n", "schemaVersion", "test/v0", "buildSucceeded", false)
 case "success-without-artifact":
  fmt.Printf("{%q:%q,%q:%v}\n", "schemaVersion", "test/v0", "buildSucceeded", true)
 }
}
`

func TestSelfBuildPreservesChildReportedFailureBeforeMissingArtifact(t *testing.T) {
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "reporting-child.go")
	if err := os.WriteFile(source, []byte(selfBuildReportedFailureChild), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "reporting-child")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", binary, source)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build test child: %v: %s", err, out)
	}
	cases := []struct {
		mode     string
		expected string
	}{
		{mode: "reason", expected: "go.mod requires go >= 9.9"},
		{mode: "silent", expected: "the child reported buildSucceeded=false without a reason"},
		{mode: "success-without-artifact", expected: ""},
	}
	for _, tt := range cases {
		t.Run(tt.mode, func(t *testing.T) {
			t.Setenv("CHILD_MODE", tt.mode)
			artifact := filepath.Join(t.TempDir(), "artifact")
			evidence, err := driveSelfBuild(binary, nil, artifact, "")
			if err != nil {
				t.Fatal(err)
			}
			if evidence.ChildBuildError != tt.expected {
				t.Fatalf("childBuildError=%q; want %q", evidence.ChildBuildError, tt.expected)
			}
			if evidence.CrossCheckOK || evidence.FailureCause == "" {
				t.Fatalf("a missing artifact was incorrectly accepted: %+v", evidence)
			}
			prefix := "the child reported that it did not build: "
			if tt.expected != "" {
				if !strings.HasPrefix(evidence.FailureCause, prefix+tt.expected) {
					t.Fatalf("child failure was not reported before its consequences: %q", evidence.FailureCause)
				}
			} else if strings.Contains(evidence.FailureCause, prefix) {
				t.Fatalf("the child did not report failure: %q", evidence.FailureCause)
			}
			wire, err := json.Marshal(evidence)
			if err != nil {
				t.Fatal(err)
			}
			hasField := strings.Contains(string(wire), "\"childBuildError\":")
			if hasField != (tt.expected != "") {
				t.Fatalf("wrong wire failure field: %s", wire)
			}
		})
	}
}
