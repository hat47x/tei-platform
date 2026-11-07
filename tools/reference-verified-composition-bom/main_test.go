package main

import (
	"reflect"
	"strings"
	"testing"
)

func base() Input {
	return Input{
		Axes:           []Axis{{Name: "provider", Options: []string{"1.5", "1.4"}}, {Name: "consumer", Options: []string{"2.1", "2.0"}}},
		MaxCases:       4,
		RequiredChecks: []string{"contract", "integration"},
		Locks: []Lock{
			{Selections: []Selection{{Axis: "provider", Value: "1.4"}, {Axis: "consumer", Value: "2.0"}}, Fingerprint: "lock-14-20", SourceFingerprint: "source-1", ArtifactDigest: "artifact-1", Target: "linux-amd64"},
			{Selections: []Selection{{Axis: "provider", Value: "1.5"}, {Axis: "consumer", Value: "2.0"}}, Fingerprint: "lock-15-20", SourceFingerprint: "source-2", ArtifactDigest: "artifact-2", Target: "linux-amd64"},
			{Selections: []Selection{{Axis: "provider", Value: "1.4"}, {Axis: "consumer", Value: "2.1"}}, Fingerprint: "lock-14-21", SourceFingerprint: "source-3", ArtifactDigest: "artifact-3", Target: "linux-amd64"},
		},
		Checks: []Check{
			{LockFingerprint: "lock-14-20", CheckID: "contract", Status: "pass", EvidenceRef: "e1", IndependentExpectedRef: "expected-1"},
			{LockFingerprint: "lock-14-20", CheckID: "integration", Status: "pass", EvidenceRef: "e2", IndependentExpectedRef: "expected-2"},
			{LockFingerprint: "lock-15-20", CheckID: "contract", Status: "fail", EvidenceRef: "e3", IndependentExpectedRef: "expected-1"},
			{LockFingerprint: "lock-15-20", CheckID: "integration", Status: "pass", EvidenceRef: "e4", IndependentExpectedRef: "expected-2"},
			{LockFingerprint: "lock-14-21", CheckID: "contract", Status: "skip", EvidenceRef: "e5", IndependentExpectedRef: "expected-1"},
		},
	}
}

func TestFourCombinationsWithoutFalseVerification(t *testing.T) {
	got, err := Project(base())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != 4 || got.Qualified || got.MatrixComplete || got.ClaimCeiling != "reference-fixture-only" {
		t.Fatalf("wrong qualification: %+v", got)
	}
	statuses := []string{}
	for _, row := range got.Rows {
		statuses = append(statuses, row.Status)
	}
	want := []string{"observed-pass", "failed", "unverified", "unresolved"}
	if !reflect.DeepEqual(statuses, want) {
		t.Fatalf("got %v want %v", statuses, want)
	}
}
func TestUnresolvedAndMissingAreNotPass(t *testing.T) {
	x := base()
	x.Checks = x.Checks[:1]
	got, err := Project(x)
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows[0].Status != "not-run" || len(got.Rows[0].MissingChecks) != 1 {
		t.Fatalf("got %+v", got.Rows[0])
	}
}
func TestRejectAmbiguousOrExcessMatrix(t *testing.T) {
	x := base()
	x.MaxCases = 3
	if _, err := Project(x); err == nil || !strings.Contains(err.Error(), "maxCases") {
		t.Fatalf("limit error = %v", err)
	}
	x = base()
	x.Locks = append(x.Locks, x.Locks[0])
	if _, err := Project(x); err == nil {
		t.Fatal("duplicate lock accepted")
	}
	x = base()
	x.Checks[0].LockFingerprint = "stale"
	if _, err := Project(x); err == nil {
		t.Fatal("stale evidence accepted")
	}
}
func TestNoIndependentExpectedNoPass(t *testing.T) {
	x := base()
	x.Checks[0].IndependentExpectedRef = ""
	got, err := Project(x)
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows[0].Status != "unverified" {
		t.Fatalf("got %+v", got.Rows[0])
	}
}

func TestProjectDoesNotMutateInput(t *testing.T) {
	x := base()
	orig := append([]string(nil), x.Axes[0].Options...)
	if _, err := Project(x); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x.Axes[0].Options, orig) {
		t.Fatalf("project mutated shared options: %v", x.Axes[0].Options)
	}
}
