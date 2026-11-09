// Independent characterization of source relocation and typed-model scoring.
// These are bounded public fixtures, not production improvements or real LLM use.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverySameBytesSourceRenameChangesLexicalEvidence(t *testing.T) {
	root := t.TempDir()
	oldTarget := "internal/crossview/qualification.go"
	newTarget := "internal/crossview/assessment.go"
	body := "package crossview\nfunc Evaluate() {}\n"
	writeCoverageFixture(t, root, oldTarget, body)
	direct := "internal/crossview/qualification_test.go"
	indirect := "internal/crossview/revision_test.go"
	writeCoverageFixture(t, root, direct, "package crossview\n")
	writeCoverageFixture(t, root, indirect, "package crossview\n// qualification revision review\n")
	exactDoc := "docs/01-exact-old.md"
	dirDoc := "docs/02-directory-only.md"
	writeCoverageFixture(t, root, exactDoc, "# Old file\n"+oldTarget+"\n")
	writeCoverageFixture(t, root, dirDoc, "# Directory\ninternal/crossview/\n")

	before, err := Discover(root, oldTarget)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Candidates) != 4 {
		t.Fatalf("expected 2 tests and 2 docs before rename: %+v", before.Candidates)
	}
	if c, ok := candidateOf(before, direct); !ok || c.Basis != "filename-matched-test" {
		t.Fatalf("expected direct filename match before rename: %+v", before.Candidates)
	}
	if _, ok := candidateOf(before, indirect); !ok {
		t.Fatalf("expected old-stem content match before rename: %+v", before.Candidates)
	}
	if err := os.Rename(filepath.Join(root, filepath.FromSlash(oldTarget)),
		filepath.Join(root, filepath.FromSlash(newTarget))); err != nil {
		t.Fatal(err)
	}
	after, err := Discover(root, newTarget)
	if err != nil {
		t.Fatal(err)
	}
	if before.TargetSHA256 != after.TargetSHA256 || before.Target == after.Target {
		t.Fatalf("same bytes, different path required: old=%s %s new=%s %s",
			before.Target, before.TargetSHA256, after.Target, after.TargetSHA256)
	}
	if _, ok := candidateOf(after, direct); ok {
		t.Fatalf("old-stem direct test should not match new stem: %+v", after.Candidates)
	}
	if _, ok := candidateOf(after, indirect); ok {
		t.Fatalf("old-stem content-only test should not match new stem: %+v", after.Candidates)
	}
	for _, path := range []string{exactDoc, dirDoc} {
		oldCandidate, okOld := candidateOf(before, path)
		newCandidate, okNew := candidateOf(after, path)
		if !okOld || !okNew || oldCandidate.SHA256 != newCandidate.SHA256 ||
			newCandidate.Basis != "explicit-path-reference" {
			t.Fatalf("directory-granularity markdown evidence was not preserved: %s old=%+v new=%+v",
				path, oldCandidate, newCandidate)
		}
	}
	// This file still matches via the *parent directory* string. It does
	// NOT explicitly cite the relocated target file.
	docBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(exactDoc)))
	if err != nil || !strings.Contains(string(docBytes), oldTarget) ||
		strings.Contains(string(docBytes), newTarget) {
		t.Fatalf("independent citation fixture changed: %v", err)
	}
	if len(after.Candidates) != 2 {
		t.Fatalf("expected only two coarse document references after rename: %+v", after.Candidates)
	}
	// Content SHA sameness is not location/claim/meaning equivalence. The
	// Model/WorldFP/SourceIndexFP of a different TEI CLI are not present here.
}

func TestRerankChoicePriorityDiffersFromLargestProbability(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	other := "internal/validation/other_test.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	writeCoverageFixture(t, root, direct, "package validation_test\n")
	writeCoverageFixture(t, root, other, "package validation\n")
	report, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 2 || report.Candidates[0].Path != direct {
		t.Fatalf("unexpected baseline candidates: %+v", report.Candidates)
	}
	// This deliberately disagrees between choice and argmax. The current
	// parser only checks type, option names, bounds, and probability sum.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Errorf("unexpected model request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"answers":{"c0":{"type":"choice","choice":"uncertain","probabilities":{"relevant":0.99,"uncertain":0.01,"not_relevant":0}},"c1":{"type":"choice","choice":"relevant","probabilities":{"relevant":0.01,"uncertain":0.99,"not_relevant":0}}}}`)
	}))
	defer server.Close()
	Rerank(context.Background(), &report, server.URL+"/v1/systemone", "fixture-only-model")
	if report.Evaluation.State != "model-ranked" || report.Candidates[0].Path != other {
		t.Fatalf("expected class choice to sort before relevant probability: %+v", report)
	}
	if report.Candidates[0].ModelChoice != "relevant" ||
		report.Candidates[0].ModelProbabilities["relevant"] != 0.01 ||
		report.Candidates[1].ModelChoice != "uncertain" ||
		report.Candidates[1].ModelProbabilities["relevant"] != 0.99 {
		t.Fatalf("did not preserve contradictory typed fields: %+v", report.Candidates)
	}
	// A formally accepted model-ranked result does not establish calibration
	// or relevant evidence; the Provider's choice/argmax contract is unknown.
}

func TestRerankAssignmentAccountingRetainsFailures(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	writeCoverageFixture(t, root, "internal/validation/validation_test.go", "package validation_test\n")
	badSends := 0
	goodSends := 0
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		badSends++
		http.Error(w, "fixture server failure", http.StatusServiceUnavailable)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		goodSends++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"answers":{"c0":{"type":"choice","choice":"uncertain","probabilities":{"relevant":0,"uncertain":1,"not_relevant":0}}}}`)
	}))
	defer good.Close()
	noCandidatesRoot := t.TempDir()
	writeCoverageFixture(t, noCandidatesRoot, target, "package validation\n")

	type assignment struct {
		sourceRoot string
		endpoint   string
		expected   string
	}
	trials := []assignment{
		{root, "http://example.invalid/v1/systemone", "model-failed"},
		{root, bad.URL + "/v1/systemone", "model-failed"},
		{noCandidatesRoot, good.URL + "/v1/systemone", "no-candidates"},
		{root, good.URL + "/v1/systemone", "model-ranked"},
	}
	ranked := 0
	for i, trial := range trials {
		// Every allocated model-mode trial starts from a new Discover Report.
		report, err := Discover(trial.sourceRoot, target)
		if err != nil {
			t.Fatalf("trial %d: %v", i, err)
		}
		Rerank(context.Background(), &report, trial.endpoint, "fixture-only-model")
		if report.Evaluation.State != trial.expected {
			t.Fatalf("trial %d: state=%s want=%s", i, report.Evaluation.State, trial.expected)
		}
		if report.Evaluation.State == "model-ranked" {
			ranked++
		}
	}
	if len(trials) != 4 || ranked != 1 || badSends != 1 || goodSends != 1 {
		t.Fatalf("accounting mismatch assigned=%d ranked=%d sent503=%d sent200=%d",
			len(trials), ranked, badSends, goodSends)
	}
	// This is a fixture's 1/4 accepted-response count, NOT an empirical
	// model accuracy/uptime estimate. There is no independent relevance R,
	// actual inference, token cost, or consumer outcome in this test.
}
