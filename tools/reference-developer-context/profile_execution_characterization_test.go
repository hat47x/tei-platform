package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

type profileTestEvent struct {
	Action string
	Test   string
}

// runSelectedTests runs Go's actual test runner in a disposable, synthetic
// module. The ExitError and per-test JSON events remain separate evidence;
// a package-level exit 0 is not proof that any named test ran.
func runSelectedTests(t *testing.T, root string, tags ...string) ([]profileTestEvent, error) {
	t.Helper()
	args := []string{"test", "-json", "-count=1"}
	if len(tags) > 0 {
		args = append(args, "-tags="+tags[0])
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=", "CGO_ENABLED=0")
	output, runErr := cmd.Output()
	decoder := json.NewDecoder(bytes.NewReader(output))
	var events []profileTestEvent
	for {
		var event profileTestEvent
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid go test -json output (run err=%v): %v\n%s", runErr, err, output)
		}
		if event.Test != "" {
			events = append(events, event)
		}
	}
	return events, runErr
}

func hasProfileTestEvent(events []profileTestEvent, testName, action string) bool {
	for _, event := range events {
		if event.Test == testName && event.Action == action {
			return true
		}
	}
	return false
}

// CC-16: candidate discovery, Go's selected file set, Go's actual test
// execution, and independent Expected are distinct qualifications. A green
// go test command may also mean that no test ran at all.
func TestDiscoveryProfileCC16SelectionIsNotTestExecutionOrPass(t *testing.T) {
	const source = "package fixture\nfunc Evaluate() int { return 0 }\n"
	root := profileFixture(t, map[string]string{
		"qualification.go":             source,
		"qualification_base_test.go":   "package fixture\nimport \"testing\"\nfunc TestBaseline(t *testing.T) { if Evaluate() != 0 { t.Fatal(\"base contract failed\") } }\n",
		"qualification_strict_test.go": "//go:build strict\n\npackage fixture\nimport \"testing\"\nfunc TestStrict(t *testing.T) { if Evaluate() != 1 { t.Fatal(\"strict contract failed\") } }\n",
	})

	discovered, err := Discover(root, "qualification.go")
	if err != nil {
		t.Fatal(err)
	}
	allCandidates := []string{"qualification_base_test.go", "qualification_strict_test.go"}
	if got := discoveryCandidatePaths(discovered); !reflect.DeepEqual(got, allCandidates) {
		t.Fatalf("lexical candidates got %v, want %v", got, allCandidates)
	}
	if got := selectedTestFiles(t, root, runtime.GOOS); !reflect.DeepEqual(got, []string{"qualification_base_test.go"}) {
		t.Fatalf("base profile selected files: %v", got)
	}
	if got := selectedTestFiles(t, root, runtime.GOOS, "strict"); !reflect.DeepEqual(got, allCandidates) {
		t.Fatalf("strict profile selected files: %v", got)
	}

	baseEvents, baseErr := runSelectedTests(t, root)
	if baseErr != nil || !hasProfileTestEvent(baseEvents, "TestBaseline", "pass") || hasProfileTestEvent(baseEvents, "TestStrict", "run") {
		t.Fatalf("base actual execution: err=%v events=%+v", baseErr, baseEvents)
	}

	strictEvents, strictErr := runSelectedTests(t, root, "strict")
	if strictErr == nil || !hasProfileTestEvent(strictEvents, "TestStrict", "run") || !hasProfileTestEvent(strictEvents, "TestStrict", "fail") || !hasProfileTestEvent(strictEvents, "TestBaseline", "pass") {
		t.Fatalf("strict actual failure must be an executed named test: err=%v events=%+v", strictErr, strictEvents)
	}

	// Same source, same Go version, successful go test command, but zero
	// test-scoped events. This is an explicit counterexample to treating
	// Exit0 alone as C0/C1 or scenario coverage.
	withoutTests := profileFixture(t, map[string]string{"qualification.go": source})
	if got := selectedTestFiles(t, withoutTests, runtime.GOOS); len(got) != 0 {
		t.Fatalf("no-test package unexpectedly selected tests: %v", got)
	}
	noEvents, noErr := runSelectedTests(t, withoutTests)
	if noErr != nil || len(noEvents) != 0 {
		t.Fatalf("go test no-test package should exit 0 with no named test evidence: err=%v events=%+v", noErr, noEvents)
	}

	// Evidence is tied to the fixture's exact original source bytes. It
	// does not qualify other profiles, repository test suites, or Owner intent.
	actual, err := os.ReadFile(filepath.Join(root, "qualification.go"))
	if err != nil || string(actual) != source || discovered.TargetSHA256 != digest(actual) {
		t.Fatalf("source changed during test or discovery fingerprint differs: err=%v", err)
	}
}
