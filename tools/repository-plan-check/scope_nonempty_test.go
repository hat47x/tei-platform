package plancheck

import (
	"os"
	"path/filepath"
	"testing"
)

// A check that reads an empty scope passes without examining anything: "every
// entry is fine" is true over zero entries. That happened to the issue-number
// and six-state checks once plan/issue stopped carrying documents (#3931), and
// to two other checks through a wrong walk root and a broken regex quantifier in
// rounds 33 and 34.
//
// This test makes the input size visible. Every scope the package reads must
// hold something, and a scope that is legitimately empty has to say so here
// rather than fail silently.
func TestEveryCheckScopeIsNonEmpty(t *testing.T) {
	root := repositoryRoot(t)

	scopes := []struct {
		name     string
		rel      string
		digits   int
		exempt   bool
		whyEmpty string
	}{
		{name: "plan/adr", rel: "plan/adr", digits: 4},
		{name: "plan/evidence", rel: "plan/evidence"},
		{name: "plan/workstreams", rel: "plan/workstreams"},
		{name: "tools", rel: "tools"},
		{
			name:     "plan/issue",
			rel:      "plan/issue",
			digits:   6,
			exempt:   true,
			whyEmpty: "Issue records live in GitHub Issues; the directory keeps only a README (#3931)",
		},
	}

	for _, scope := range scopes {
		dir := filepath.Join(root, filepath.FromSlash(scope.rel))
		if _, err := os.ReadDir(dir); err != nil {
			t.Errorf("%s: %v", scope.name, err)
			continue
		}
		// A directory that holds only a README is not empty as a directory but
		// is empty to the checks, which count numbered documents. Measure what
		// the checks read, not what the directory contains.
		examined := len(numberedFiles(t, dir, scope.digits))
		if scope.digits == 0 {
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Errorf("%s: %v", scope.name, err)
				continue
			}
			examined = len(entries)
		}
		if examined == 0 {
			if scope.exempt {
				t.Logf("%s yields 0 documents by design: %s", scope.name, scope.whyEmpty)
				continue
			}
			t.Errorf("%s yields 0 documents; the checks over it would pass without examining anything", scope.name)
		}
	}

	// The document-level checks read whole trees. Report their size so a walk
	// that reaches nothing is visible instead of looking like a pass.
	markdown := 0
	if err := walkMarkdown(root, func(rel, text string) { markdown++ }); err != nil {
		t.Fatal(err)
	}
	if markdown == 0 {
		t.Fatal("the Markdown walk reached no files; the link, title and backtick checks would pass vacuously")
	}
	t.Logf("Markdown files examined by the document checks: %d", markdown)

	requirements := requirementIDs(t)
	if len(requirements) == 0 {
		t.Fatal("no Product Requirement ids were read; the traceability check would pass vacuously")
	}
	t.Logf("Product Requirement ids examined: %d", len(requirements))
}
