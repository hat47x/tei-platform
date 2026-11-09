package main

import (
	"os"
	"path/filepath"
	"testing"
)

// CC-17: an earlier passing Go test event cannot qualify a current candidate
// after the candidate's bytes changed, even while target bytes, test name,
// GOOS, and selected TestGoFiles are unchanged. This uses only synthetic local
// Go files, actual Discover, and actual go test -json; it is not evidence of
// production receipt qualification or concurrent filesystem consistency.
func TestDiscoveryProfileCC17CandidateChangeInvalidatesOldPass(t *testing.T) {
	const source = "package fixture\nfunc Evaluate() int { return 0 }\n"
	const originalTest = "package fixture\nimport \"testing\"\nfunc TestObligation(t *testing.T) { if Evaluate() != 0 { t.Fatal(\"old obligation failed\") } }\n"
	const changedTest = "package fixture\nimport \"testing\"\nfunc TestObligation(t *testing.T) { if Evaluate() != 1 { t.Fatal(\"new obligation failed\") } }\n"
	const candidate = "qualification_test.go"
	root := profileFixture(t, map[string]string{
		"qualification.go": source,
		candidate:          originalTest,
	})
	testPath := filepath.Join(root, candidate)

	initial, err := Discover(root, "qualification.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(initial.Candidates) != 1 || initial.Candidates[0].Path != candidate {
		t.Fatalf("initial candidate discovery unexpected: %+v", initial)
	}
	selected := selectedTestFiles(t, root, "linux")
	if len(selected) != 1 || selected[0] != candidate {
		t.Fatalf("initial Go selection unexpected: %v", selected)
	}
	oldEvents, oldErr := runSelectedTests(t, root)
	if oldErr != nil || !hasProfileTestEvent(oldEvents, "TestObligation", "pass") {
		t.Fatalf("initial test must actually pass: err=%v events=%+v", oldErr, oldEvents)
	}

	// Change just the test's required outcome. The target and name remain
	// fixed, and the test is still selected for exactly the same profile.
	if err := os.WriteFile(testPath, []byte(changedTest), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := Discover(root, "qualification.go")
	if err != nil {
		t.Fatal(err)
	}
	if initial.TargetSHA256 != changed.TargetSHA256 {
		t.Fatal("target source unexpectedly changed")
	}
	if len(changed.Candidates) != 1 || initial.Candidates[0].SHA256 == changed.Candidates[0].SHA256 {
		t.Fatalf("test revision was not distinguished: initial=%+v changed=%+v", initial, changed)
	}
	afterSelection := selectedTestFiles(t, root, "linux")
	if len(afterSelection) != 1 || afterSelection[0] != candidate {
		t.Fatalf("Go file selection unexpectedly changed: %v", afterSelection)
	}
	newEvents, newErr := runSelectedTests(t, root)
	if newErr == nil || !hasProfileTestEvent(newEvents, "TestObligation", "run") ||
		!hasProfileTestEvent(newEvents, "TestObligation", "fail") {
		t.Fatalf("revised candidate must actually fail: err=%v events=%+v", newErr, newEvents)
	}

	// A -> B -> A means the content digest can return to an earlier value,
	// without turning the historical test execution into a current receipt.
	if err := os.WriteFile(testPath, []byte(originalTest), 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := Discover(root, "qualification.go")
	if err != nil {
		t.Fatal(err)
	}
	if restored.TargetSHA256 != initial.TargetSHA256 || len(restored.Candidates) != 1 ||
		restored.Candidates[0].SHA256 != initial.Candidates[0].SHA256 {
		t.Fatalf("restoration should match original content digests: %+v", restored)
	}
	restoredEvents, restoredErr := runSelectedTests(t, root)
	if restoredErr != nil || !hasProfileTestEvent(restoredEvents, "TestObligation", "pass") {
		t.Fatalf("restored source should pass in a new execution: err=%v events=%+v", restoredErr, restoredEvents)
	}
}
