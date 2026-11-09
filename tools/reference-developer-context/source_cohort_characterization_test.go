package main

import (
	"os"
	"path/filepath"
	"testing"
)

// CC-14 is a public, deterministic counterexample based on the same
// readInside primitive called separately for target and candidates by
// Discover. We deliberately control the mutations between reads: this is
// NOT proof that a concurrent Discover call actually observed this pairing,
// nor proof of an OS race, sandbox escape, or attacker-controlled mutation.
func TestActualReadCC14IndividualDigestsDoNotProveJointCohort(t *testing.T) {
	root := t.TempDir()
	const target = "qualification.go"
	const candidate = "qualification_test.go"
	const t0 = "package fixture\n// target revision T0\nfunc Evaluate() int { return 0 }\n"
	const t1 = "package fixture\n// target revision T1\nfunc Evaluate() int { return 1 }\n"
	const c0 = "package fixture\n// candidate revision C0: checks T0\n"
	const c1 = "package fixture\n// candidate revision C1: checks T1\n"

	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	type epoch struct {
		target    string
		candidate string
	}
	var observedEpochs []epoch
	observe := func(tSource, cSource string) {
		t.Helper()
		targetBytes, err := os.ReadFile(filepath.Join(root, target))
		if err != nil {
			t.Fatal(err)
		}
		candidateBytes, err := os.ReadFile(filepath.Join(root, candidate))
		if err != nil {
			t.Fatal(err)
		}
		if string(targetBytes) != tSource || string(candidateBytes) != cSource {
			t.Fatal("controlled epoch contents did not match the independent fixture")
		}
		observedEpochs = append(observedEpochs, epoch{digest(targetBytes), digest(candidateBytes)})
	}

	// Actual source states are (T0,C0), then (T1,C0), then (T1,C1).
	// There is never a simultaneous state (T0,C1) during this trial.
	write(target, t0)
	write(candidate, c0)
	observe(t0, c0)

	// Discover reads its target first; capture that actual byte read.
	earlyTarget, err := readInside(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if digest(earlyTarget) != digest([]byte(t0)) {
		t.Fatal("early target read was not T0")
	}

	write(target, t1)
	observe(t1, c0)
	write(candidate, c1)
	observe(t1, c1)

	lateCandidate, err := readInside(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if digest(lateCandidate) != digest([]byte(c1)) {
		t.Fatal("late candidate read was not C1")
	}

	// Both bytes and both digests really came from readInside, but the
	// combination is not a state from any of the three observed epochs.
	mixed := epoch{digest(earlyTarget), digest(lateCandidate)}
	if mixed.target == "" || mixed.candidate == "" {
		t.Fatal("a source digest is missing")
	}
	for i, oneEpoch := range observedEpochs {
		if mixed == oneEpoch {
			t.Fatalf("mixed pair unexpectedly equals complete epoch %d", i)
		}
	}
	// The report fields can represent a mixed result without an epoch or
	// joint snapshot qualification. This is schema characterization, not
	// an injected Discover call with a scheduled race.
	report := Report{
		Schema:       "tei.reference.developer-context/v0",
		Target:       target,
		TargetSHA256: mixed.target,
		ReadOnly:     true,
		Candidates: []Candidate{{
			Path:   candidate,
			Kind:   "test",
			SHA256: mixed.candidate,
			Basis:  "filename-matched-test",
		}},
	}
	if report.TargetSHA256 != digest([]byte(t0)) || report.Candidates[0].SHA256 != digest([]byte(c1)) {
		t.Fatalf("report did not retain real read bytes: %+v", report)
	}

	// A new ordinary Discover pass on the now-stable source has a
	// different target digest, confirming that the mixed pair is not the
	// current (T1,C1) world.
	fresh, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.TargetSHA256 != digest([]byte(t1)) || len(fresh.Candidates) != 1 ||
		fresh.Candidates[0].SHA256 != digest([]byte(c1)) {
		t.Fatalf("new Discover did not observe the stable final world: %+v", fresh)
	}
	if fresh.TargetSHA256 == report.TargetSHA256 {
		t.Fatal("fractured and fresh target epochs incorrectly conflated")
	}
}

func TestActualReadCC14StableCohortIsOnlyFixtureQualified(t *testing.T) {
	root := t.TempDir()
	const target = "qualification.go"
	const candidate = "qualification_test.go"
	const t1 = "package fixture\nfunc Evaluate() int { return 1 }\n"
	const c1 = "package fixture\n// qualification current candidate\n"
	for path, body := range map[string]string{target: t1, candidate: c1} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// No changes occur in this fixture, and both individual reads match.
	// That is a fixture-local positive control, not a general proof that
	// arbitrary simultaneous edits would be noticed by Discover.
	first, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if first.TargetSHA256 != digest([]byte(t1)) || second.TargetSHA256 != first.TargetSHA256 ||
		len(first.Candidates) != 1 || len(second.Candidates) != 1 ||
		first.Candidates[0].SHA256 != digest([]byte(c1)) ||
		second.Candidates[0].SHA256 != first.Candidates[0].SHA256 {
		t.Fatalf("stable read expected to repeat fixture bytes: first=%+v second=%+v", first, second)
	}
}
