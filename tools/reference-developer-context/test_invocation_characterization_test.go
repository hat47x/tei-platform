package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"testing"
)

// CC-30: the Go package, discovered test file, selected TestGoFiles and
// target/candidate report can all stay unchanged while the real test command
// executes a different subset (or zero tests). Process exit 0 is not a
// receipt for an independently required named test.
func TestDiscoveryProfileCC30RunFilterDoesNotQualifyRequiredNamedTest(t *testing.T) {
	const target = "qualification.go"
	const candidate = "qualification_test.go"
	root := profileFixture(t, map[string]string{
		target:    "package fixture\nfunc Evaluate() int { return 0 }\n",
		candidate: "package fixture\nimport \"testing\"\nfunc TestRequired(t *testing.T) { if Evaluate()!=0 { t.Fatal(\"invalid\") } }\nfunc TestUnrelated(t *testing.T) {}\n",
	})

	initial, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(initial.Candidates) != 1 || initial.Candidates[0].Path != candidate {
		t.Fatalf("discovery mismatch: %+v", initial)
	}
	if got := selectedTestFiles(t, root, runtime.GOOS); !reflect.DeepEqual(got, []string{candidate}) {
		t.Fatalf("file not selected by actual Go toolchain: %v", got)
	}

	// Only actual named events (run/pass) count for the required obligation.
	// Note that the selected Go package is the same for every invocation.
	run := func(filter string) ([]profileTestEvent, error) {
		t.Helper()
		cmd := exec.Command("go", "test", "-json", "-count=1", "-run", filter, ".")
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=", "CGO_ENABLED=0")
		raw, runErr := cmd.Output()
		decoder := json.NewDecoder(bytes.NewReader(raw))
		var events []profileTestEvent
		for {
			var event profileTestEvent
			if err := decoder.Decode(&event); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("go test -json -run %q produced invalid JSON: %v\n%s", filter, err, raw)
			}
			if event.Test != "" {
				events = append(events, event)
			}
		}
		return events, runErr
	}

	required, requiredErr := run("^TestRequired$")
	if requiredErr != nil ||
		!hasProfileTestEvent(required, "TestRequired", "run") ||
		!hasProfileTestEvent(required, "TestRequired", "pass") ||
		hasProfileTestEvent(required, "TestUnrelated", "run") {
		t.Fatalf("required-filter invocation mismatch: err=%v events=%+v", requiredErr, required)
	}

	unrelated, unrelatedErr := run("^TestUnrelated$")
	if unrelatedErr != nil ||
		!hasProfileTestEvent(unrelated, "TestUnrelated", "pass") ||
		hasProfileTestEvent(unrelated, "TestRequired", "run") ||
		hasProfileTestEvent(unrelated, "TestRequired", "pass") {
		t.Fatalf("success of unrelated test must not qualify TestRequired: err=%v events=%+v", unrelatedErr, unrelated)
	}

	noNamed, noNamedErr := run("^TestNoSuchTest$")
	if noNamedErr != nil || len(noNamed) != 0 {
		t.Fatalf("exit=0 with unmatched run filter must have zero named events: err=%v events=%+v", noNamedErr, noNamed)
	}

	// The unchanged lexical context/selection cannot attest any of these
	// different invocations or their filter/Exit/Event set.
	after, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(initial, after) {
		t.Fatalf("the same bytes must give the same limited context report: before=%+v after=%+v", initial, after)
	}
	if got := selectedTestFiles(t, root, runtime.GOOS); !reflect.DeepEqual(got, []string{candidate}) {
		t.Fatalf("selected TestGoFiles drifted: %v", got)
	}
}
