package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

// selectedPackageGoFiles asks the Go toolchain for its actual selected
// non-test source files. Discover does not currently include this dependency
// set in its read-only candidate report.
func selectedPackageGoFiles(t *testing.T, root string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-json", ".")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=", "CGO_ENABLED=0")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list selected sources failed: %v\n%s", err, raw)
	}
	var pkg struct {
		GoFiles []string
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("invalid go list output: %v\n%s", err, raw)
	}
	sort.Strings(pkg.GoFiles)
	return pkg.GoFiles
}

// CC-18: the target file and the selected test file can remain byte-identical
// while a separate compiled Go source dependency changes the test outcome.
// This is a deterministic, synthetic single-threaded test; it does not
// establish production dependency closure, a joint snapshot, or a current
// execution receipt for arbitrary TEI workspaces.
func TestDiscoveryProfileCC18UnreportedSourceDependencyChangesTestOutcome(t *testing.T) {
	const target = "qualification.go"
	const testName = "qualification_test.go"
	const dependency = "rules.go"
	const source = "package fixture\nfunc Evaluate() int { return requiredRule() }\n"
	const testSource = "package fixture\nimport \"testing\"\nfunc TestRequiredRule(t *testing.T) { if Evaluate() != 0 { t.Fatal(\"required dependency changed\") } }\n"
	const ruleA = "package fixture\nfunc requiredRule() int { return 0 }\n"
	const ruleB = "package fixture\nfunc requiredRule() int { return 1 }\n"

	root := profileFixture(t, map[string]string{
		target:     source,
		testName:   testSource,
		dependency: ruleA,
	})
	expectedFiles := []string{target, dependency}
	if got := selectedPackageGoFiles(t, root); !reflect.DeepEqual(got, expectedFiles) {
		t.Fatalf("Go compiler sources got %v, want %v", got, expectedFiles)
	}
	if got := selectedTestFiles(t, root, runtime.GOOS); !reflect.DeepEqual(got, []string{testName}) {
		t.Fatalf("test selection changed: %v", got)
	}

	before, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Candidates) != 1 || before.Candidates[0].Path != testName {
		t.Fatalf("initial context unexpected: %+v", before)
	}
	beforeEvents, beforeErr := runSelectedTests(t, root)
	if beforeErr != nil || !hasProfileTestEvent(beforeEvents, "TestRequiredRule", "pass") {
		t.Fatalf("original test must actually pass: err=%v events=%+v", beforeErr, beforeEvents)
	}

	// Change the compiled dependency, not the target or selected test.
	if err := os.WriteFile(filepath.Join(root, dependency), []byte(ruleB), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("the current context report unexpectedly exposed the dependency change: before=%+v after=%+v", before, after)
	}
	if got := selectedPackageGoFiles(t, root); !reflect.DeepEqual(got, expectedFiles) {
		t.Fatalf("Go selected source filenames changed: %v", got)
	}
	afterEvents, afterErr := runSelectedTests(t, root)
	if afterErr == nil || !hasProfileTestEvent(afterEvents, "TestRequiredRule", "run") ||
		!hasProfileTestEvent(afterEvents, "TestRequiredRule", "fail") {
		t.Fatalf("changed dependency must cause named test failure: err=%v events=%+v", afterErr, afterEvents)
	}

	// A -> B -> A restores source bytes but does not make the old
	// execution attempt a receipt for this new execution.
	if err := os.WriteFile(filepath.Join(root, dependency), []byte(ruleA), 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, restored) {
		t.Fatalf("restored fixture should yield the same limited context report: %+v", restored)
	}
	restoredEvents, restoredErr := runSelectedTests(t, root)
	if restoredErr != nil || !hasProfileTestEvent(restoredEvents, "TestRequiredRule", "pass") {
		t.Fatalf("new test execution after restoration must pass: err=%v events=%+v", restoredErr, restoredEvents)
	}
}
