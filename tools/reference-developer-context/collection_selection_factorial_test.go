// A two-factor evaluation: test read budget versus presentation policy.
// All source files are public temp fixtures. The independent expected paths
// are fixed before any Discover/collector call. No production change.
package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// factorialDisplay only projects an already collected candidate slice.
// It performs no Source reads and does not change the caller's slice.
// "Tests first" is an evaluation policy, NOT the current Discover behavior,
// which may stop before reaching the documents scope at all.
func factorialDisplay(collected []Candidate, docsBeforeWeakTests bool) []Candidate {
	candidates := append([]Candidate(nil), collected...)
	priority := map[string]int{
		"filename-matched-test":        0,
		"content-matched-test":         1,
		"exact-file-mention-eval-only": 2,
	}
	if docsBeforeWeakTests {
		priority["content-matched-test"] = 2
		priority["exact-file-mention-eval-only"] = 1
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if priority[a.Basis] != priority[b.Basis] {
			return priority[a.Basis] < priority[b.Basis]
		}
		return a.Path < b.Path
	})
	for i := range candidates {
		candidates[i].BaselineRank = i + 1
	}
	n := len(candidates)
	if n > maxCandidates {
		n = maxCandidates
	}
	return candidates[:n]
}

func countExpectedPaths(display []Candidate, expected []string) int {
	found := 0
	for _, path := range expected {
		if comparisonHasPath(display, path) {
			found++
		}
	}
	return found
}

func TestCollectionFactorialBudgetVersusSelectionPolicy(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	document := "docs/target-reference.md"
	// The evaluation oracle is fixed independently of the compared rankings.
	// These are path witnesses, not validated business meaning or freshness.
	needed := []string{direct, document}
	writeCoverageFixture(t, root, target, "package validation\n")
	for i := 0; i < maxCandidates; i++ {
		writeCoverageFixture(t, root,
			fmt.Sprintf("internal/validation/a%02d_test.go", i),
			"package validation\n")
	}
	writeCoverageFixture(t, root, direct, "package validation_test\n")
	writeCoverageFixture(t, root, document, "# Exact file reference\n"+target+"\n")

	baseline, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if countExpectedPaths(baseline.Candidates, needed) != 0 {
		t.Fatalf("original 12-candidate early stop unexpectedly included oracle: %+v", baseline.Candidates)
	}

	low, err := compareLocalLexicalCollection(root, target, 12, 12)
	if err != nil {
		t.Fatal(err)
	}
	high, err := compareLocalLexicalCollection(root, target, 13, 12)
	if err != nil {
		t.Fatal(err)
	}
	if low.TestRead != 12 || low.DocRead != 1 || !low.TestTruncated ||
		high.TestRead != 13 || high.DocRead != 1 || high.TestTruncated ||
		low.ReadFailures != 0 || high.ReadFailures != 0 ||
		len(low.Candidates) != 13 || len(high.Candidates) != 14 {
		t.Fatalf("unexpected read/retention budgets: low=%+v high=%+v", low, high)
	}
	if comparisonHasPath(low.Candidates, direct) ||
		!comparisonHasPath(low.Candidates, document) ||
		!comparisonHasPath(high.Candidates, direct) ||
		!comparisonHasPath(high.Candidates, document) {
		t.Fatalf("collector did not isolate budget factor: low=%+v high=%+v", low.Candidates, high.Candidates)
	}

	// Each row projects one identical collected C, without further reads.
	// Each column applies exactly the same selection logic at both budgets.
	lowSnapshot := append([]Candidate(nil), low.Candidates...)
	highSnapshot := append([]Candidate(nil), high.Candidates...)
	lowTestsFirst := factorialDisplay(low.Candidates, false)
	lowDocsFirst := factorialDisplay(low.Candidates, true)
	highTestsFirst := factorialDisplay(high.Candidates, false)
	highDocsFirst := factorialDisplay(high.Candidates, true)
	if !reflect.DeepEqual(low.Candidates, lowSnapshot) ||
		!reflect.DeepEqual(high.Candidates, highSnapshot) {
		t.Fatal("presentation policy unexpectedly mutated the collected evidence")
	}
	for label, display := range map[string][]Candidate{
		"low/tests-first":  lowTestsFirst,
		"low/docs-first":   lowDocsFirst,
		"high/tests-first": highTestsFirst,
		"high/docs-first":  highDocsFirst,
	} {
		if len(display) != maxCandidates {
			t.Fatalf("%s: unequal display capacity = %d", label, len(display))
		}
	}
	// Explicit expected 2x2 matrix, not an inferred rank/model score:
	//
	//                     test-first    exact-doc-first
	// 12 test reads            0 / 2            1 / 2
	// 13 test reads            1 / 2            2 / 2
	//
	// Doc reading is one file in both rows. Test read cost is 12 vs 13.
	cases := []struct {
		name string
		out  []Candidate
		want int
	}{
		{"low/tests-first", lowTestsFirst, 0},
		{"low/docs-first", lowDocsFirst, 1},
		{"high/tests-first", highTestsFirst, 1},
		{"high/docs-first", highDocsFirst, 2},
	}
	for _, c := range cases {
		if got := countExpectedPaths(c.out, needed); got != c.want {
			t.Errorf("%s: oracle reach got %d of 2, want %d: %+v",
				c.name, got, c.want, c.out)
		}
	}
	if !comparisonHasPath(lowDocsFirst, document) ||
		!comparisonHasPath(highDocsFirst, document) ||
		!comparisonHasPath(highTestsFirst, direct) ||
		!comparisonHasPath(highDocsFirst, direct) {
		t.Fatal("a different fixture factor unexpectedly explains the result")
	}
	// Reaching a name is not understanding Source or requalifying the citation.
	// This single toy fixture does not estimate average retrieval performance.
}

func TestCollectionFactorialSelectionWithoutDocumentCannotCreateEvidence(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	for i := 0; i < maxCandidates; i++ {
		writeCoverageFixture(t, root,
			fmt.Sprintf("internal/validation/a%02d_test.go", i),
			"package validation\n")
	}
	writeCoverageFixture(t, root, direct, "package validation_test\n")

	for _, limit := range []int{12, 13} {
		collected, err := compareLocalLexicalCollection(root, target, limit, 12)
		if err != nil {
			t.Fatal(err)
		}
		if collected.DocRead != 0 || collected.ReadFailures != 0 {
			t.Fatalf("document-free control has unexpected reads: %+v", collected)
		}
		a := factorialDisplay(collected.Candidates, false)
		b := factorialDisplay(collected.Candidates, true)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("doc priority changed result with no doc, limit=%d: A=%+v B=%+v", limit, a, b)
		}
		if comparisonHasPath(a, direct) != (limit == 13) {
			t.Fatalf("only test budget should change direct-test reach at limit=%d", limit)
		}
	}
	// An unobserved source cannot be restored by display reordering.
}
