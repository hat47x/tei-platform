// Evaluation-only lexical collection comparison for P2-2a. This is NOT a
// replacement Discover, a production resolver, or a full-repository search.
// It deliberately counts reads and exhausted budgets beside each trial.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type boundedComparison struct {
	Candidates    []Candidate
	Display       []Candidate
	TestRead      int
	DocRead       int
	ReadFailures  int
	TestTruncated bool
	DocTruncated  bool
}

// compareLocalLexicalCollection intentionally explores only _test.go files in
// the target directory and flat docs/*.md files in a public temp workspace.
// Each scope has an independent, explicit upper bound. The fixture's oracle
// remains independent of this classifier.
func compareLocalLexicalCollection(root, target string, testLimit, docLimit int) (boundedComparison, error) {
	var result boundedComparison
	dir := filepath.Dir(target)
	stem := strings.TrimSuffix(filepath.Base(target), ".go")
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return result, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if result.TestRead >= testLimit {
			result.TestTruncated = true
			break
		}
		rel := filepath.ToSlash(filepath.Join(dir, e.Name()))
		result.TestRead++
		body, readErr := readInside(root, rel)
		if readErr != nil {
			result.ReadFailures++
			continue
		}
		basis := ""
		if strings.HasPrefix(e.Name(), stem+"_") || strings.HasPrefix(e.Name(), stem+"test") {
			basis = "filename-matched-test"
		} else if strings.Contains(string(body), stem) {
			basis = "content-matched-test"
		}
		if basis != "" {
			result.Candidates = append(result.Candidates, Candidate{
				Path: rel, Kind: "test", Basis: basis, SHA256: digest(body),
			})
		}
	}

	docsDir := filepath.Join(root, "docs")
	if docs, readErr := os.ReadDir(docsDir); readErr == nil {
		for _, e := range docs {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
				continue
			}
			if result.DocRead >= docLimit {
				result.DocTruncated = true
				break
			}
			rel := "docs/" + e.Name()
			result.DocRead++
			body, err := readInside(root, rel)
			if err != nil {
				result.ReadFailures++
				continue
			}
			// Exact file mention only: not semantic documentation validity.
			if strings.Contains(string(body), filepath.ToSlash(target)) {
				result.Candidates = append(result.Candidates, Candidate{
					Path: rel, Kind: "document", Basis: "exact-file-mention-eval-only", SHA256: digest(body),
				})
			}
		}
	} else if !os.IsNotExist(readErr) {
		return result, readErr
	}
	// A transparent lexical presentation policy, NOT an accepted relevance
	// policy: directly named test, exact-file doc, then content/stem test.
	priority := map[string]int{
		"filename-matched-test": 0,
		"exact-file-mention-eval-only": 1,
		"content-matched-test": 2,
	}
	sort.SliceStable(result.Candidates, func(i, j int) bool {
		a, b := result.Candidates[i], result.Candidates[j]
		if priority[a.Basis] != priority[b.Basis] {
			return priority[a.Basis] < priority[b.Basis]
		}
		return a.Path < b.Path
	})
	for i := range result.Candidates {
		result.Candidates[i].BaselineRank = i + 1
	}
	k := len(result.Candidates)
	if k > maxCandidates {
		k = maxCandidates
	}
	result.Display = append([]Candidate(nil), result.Candidates[:k]...)
	return result, nil
}

func comparisonHasPath(xs []Candidate, path string) bool {
	for _, c := range xs {
		if c.Path == path {
			return true
		}
	}
	return false
}

func TestCollectionComparisonExtraReadBudgetPreservesDirectEvidence(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	for i := 0; i < 12; i++ {
		writeCoverageFixture(t, root, fmt.Sprintf("internal/validation/a%02d_test.go", i), "package validation\n")
	}
	writeCoverageFixture(t, root, direct, "package validation_test\n")
	expectedRead := []string{direct} // independent fixture oracle, not inferred from the algorithm.

	existing, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	limited, err := compareLocalLexicalCollection(root, target, 12, 12)
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := compareLocalLexicalCollection(root, target, 64, 12)
	if err != nil {
		t.Fatal(err)
	}
	if comparisonHasPath(existing.Candidates, expectedRead[0]) ||
		comparisonHasPath(limited.Display, expectedRead[0]) ||
		!limited.TestTruncated || limited.TestRead != 12 {
		t.Fatalf("12 test reads cannot discover the 13th direct test: limited=%+v existing=%+v", limited, existing.Candidates)
	}
	if !comparisonHasPath(expanded.Display, expectedRead[0]) ||
		expanded.Display[0].Path != direct || expanded.TestRead != 13 ||
		expanded.TestTruncated || len(expanded.Display) != maxCandidates ||
		len(expanded.Candidates) != 13 {
		t.Fatalf("separate read budget and display cap failed: %+v", expanded)
	}
	// Improvement here costs one more real file read. It is NOT a pure
	// ranking gain or evidence that the full repo has been covered.
}

func TestCollectionComparisonIndependentDocBudgetRescuesExactMention(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	doc := "docs/target-reference.md"
	writeCoverageFixture(t, root, target, "package validation\n")
	for i := 0; i < maxCandidates; i++ {
		writeCoverageFixture(t, root, fmt.Sprintf("internal/validation/a%02d_test.go", i), "package validation\n")
	}
	writeCoverageFixture(t, root, doc, "# Explicit path\n"+target+"\n")
	existing, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	alt, err := compareLocalLexicalCollection(root, target, 12, 12)
	if err != nil {
		t.Fatal(err)
	}
	if comparisonHasPath(existing.Candidates, doc) ||
		!comparisonHasPath(alt.Candidates, doc) ||
		!comparisonHasPath(alt.Display, doc) || alt.DocRead != 1 ||
		alt.TestRead != 12 || len(alt.Candidates) != 13 ||
		len(alt.Display) != maxCandidates {
		t.Fatalf("separate doc scanning and presentation cap: old=%+v alternative=%+v", existing.Candidates, alt)
	}
	// Exact mention is lexical evidence; this does NOT establish that the
	// document is current, relevant to Q, or accepted by its Owner.
}

func TestCollectionComparisonExhaustedReadBudgetCannotClaimAbsence(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	for i := 0; i < 65; i++ {
		writeCoverageFixture(t, root, fmt.Sprintf("internal/validation/a%02d_test.go", i), "package validation\n")
	}
	writeCoverageFixture(t, root, direct, "package validation_test\n")
	alt, err := compareLocalLexicalCollection(root, target, 64, 12)
	if err != nil {
		t.Fatal(err)
	}
	if !alt.TestTruncated || alt.TestRead != 64 ||
		comparisonHasPath(alt.Candidates, direct) || len(alt.Display) != maxCandidates {
		t.Fatalf("bounded result must retain unknown/unvisited remainder: %+v", alt)
	}
	// Independent fixture witness: outside the scanned prefix is a real
	// direct test. "Absent from selected candidates" != "does not exist".
	if b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(direct))); err != nil ||
		!strings.Contains(string(b), "package validation") {
		t.Fatalf("independent unvisited source witness: %v", err)
	}
}
