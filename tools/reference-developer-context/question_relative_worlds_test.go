// A question-relative, three-world characterization of candidate sufficiency.
// The independent R sets belong exclusively to the test-side evaluator.
// There is no LLM call, business meaning acceptance, or actual Product Probe.
package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestQuestionRelativeSameReportDifferentHiddenWorlds(t *testing.T) {
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	doc := "docs/target-reference.md"
	// W0: no doc; W1: doc mentions target; W2: doc exists but mentions
	// a different source. The target and all _test.go bytes are identical.
	makeWorld := func(docMode string) string {
		t.Helper()
		root := t.TempDir()
		writeCoverageFixture(t, root, target, "package validation\n")
		for i := 0; i < maxCandidates; i++ {
			writeCoverageFixture(t, root,
				fmt.Sprintf("internal/validation/a%02d_test.go", i),
				"package validation\n")
		}
		writeCoverageFixture(t, root, direct, "package validation_test\n")
		switch docMode {
		case "exact":
			writeCoverageFixture(t, root, doc, "# Source citation\n"+target+"\n")
		case "other":
			writeCoverageFixture(t, root, doc,
				"# Different source\ninternal/validation/other.go\n")
		}
		return root
	}
	w0 := makeWorld("absent")
	w1 := makeWorld("exact")
	w2 := makeWorld("other")

	base0, err := Discover(w0, target)
	if err != nil {
		t.Fatal(err)
	}
	base1, err := Discover(w1, target)
	if err != nil {
		t.Fatal(err)
	}
	base2, err := Discover(w2, target)
	if err != nil {
		t.Fatal(err)
	}
	// A method receiving only this initial report cannot distinguish the
	// three worlds. It must not use the test oracle or assume no docs exist.
	if !reflect.DeepEqual(base0, base1) || !reflect.DeepEqual(base1, base2) ||
		len(base0.Candidates) != maxCandidates ||
		comparisonHasPath(base1.Candidates, doc) {
		t.Fatalf("the initial Discover reports must be indistinguishable")
	}

	type world struct {
		name     string
		root     string
		docReads int
		docFound bool
	}
	worlds := []world{
		{"W0-no-document", w0, 0, false},
		{"W1-exact-document", w1, 1, true},
		{"W2-different-document", w2, 1, false},
	}
	for _, w := range worlds {
		t.Run(w.name, func(t *testing.T) {
			// A bounded alternative is intentionally a NEW source read,
			// not a model magically recovering a missing candidate.
			out, err := compareLocalLexicalCollection(w.root, target, 13, 12)
			if err != nil {
				t.Fatal(err)
			}
			display := factorialDisplay(out.Candidates, true)
			if out.TestRead != 13 || out.TestTruncated || out.DocRead != w.docReads ||
				out.ReadFailures != 0 ||
				comparisonHasPath(display, doc) != w.docFound ||
				!comparisonHasPath(display, direct) ||
				!comparisonHasPath(out.Candidates, direct) {
				t.Fatalf("failed to account for actual reads and lexical observations: %+v", out)
			}
			// The independent question-specific requirements are not an
			// input to collection, ranking, or the model. Treat absence of
			// required evidence as "not established", NOT "nonexistent".
			questions := []struct {
				name     string
				required []string
				found    int
			}{
				{"Q-test-change", []string{direct}, 1},
				{"Q-citation-check", []string{doc}, boolToCount(w.docFound)},
				{"Q-test-and-doc", []string{direct, doc}, 1 + boolToCount(w.docFound)},
			}
			for _, q := range questions {
				if got := countExpectedPaths(display, q.required); got != q.found {
					t.Errorf("%s: found %d/%d want %d, from %+v",
						q.name, got, len(q.required), q.found, display)
				}
			}
		})
	}
	// No inference about semantic relevance, stale citations, or a global
	// negative claim follows from these lexical candidate-path observations.
}

func boolToCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestQuestionRelativeSameCollectedCandidatesDifferentQuestions(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	doc := "docs/target-reference.md"
	writeCoverageFixture(t, root, target, "package validation\n")
	for i := 0; i < maxCandidates; i++ {
		writeCoverageFixture(t, root,
			fmt.Sprintf("internal/validation/a%02d_test.go", i),
			"package validation\n")
	}
	writeCoverageFixture(t, root, direct, "package validation_test\n")
	writeCoverageFixture(t, root, doc, "# Source citation\n"+target+"\n")

	// Freeze C and read counts. Q changes only the sealed test-side R.
	collected, err := compareLocalLexicalCollection(root, target, 13, 12)
	if err != nil {
		t.Fatal(err)
	}
	if collected.TestRead != 13 || collected.DocRead != 1 ||
		len(collected.Candidates) != 14 {
		t.Fatalf("unexpected observed collection: %+v", collected)
	}
	display := factorialDisplay(collected.Candidates, false)
	if !comparisonHasPath(display, direct) || comparisonHasPath(display, doc) {
		t.Fatalf("test-first presentation did not create the expected tradeoff: %+v", display)
	}
	questions := []struct {
		name      string
		required  []string
		satisfied bool
	}{
		{"Q-test-change", []string{direct}, true},
		{"Q-citation-check", []string{doc}, false},
		{"Q-test-and-doc", []string{direct, doc}, false},
	}
	for _, q := range questions {
		got := countExpectedPaths(display, q.required)
		satisfied := got == len(q.required)
		if satisfied != q.satisfied {
			t.Errorf("same C/V, different Q must have distinct sufficiency: %s=%v", q.name, satisfied)
		}
	}
	// All three Q share the SAME acquired candidate set and display.
	// An AI ranking label or C length alone cannot establish sufficiency.
}
