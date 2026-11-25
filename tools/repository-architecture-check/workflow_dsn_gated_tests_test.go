package architecturecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// This check pins one narrow claim: a test that skips itself unless a
// PostgreSQL DSN is set must be named by a workflow, so the job that sets the
// DSN actually runs it.
//
// Tei's real-database tests read `TEI_VIEW_PG_DSN` (or a sibling) and `t.Skip`
// when it is unset, because a developer's machine has none. The CI job that
// sets the DSN runs `go test -run '^A$|^B$…'`, so a test whose name is not in
// that list never runs there: the job is green, the test is green locally (as a
// skip), and nothing exercises the real database. This hole was reported by
// hand seven times (#5833, #5882, #5886, #5913, #5922, #5970, #5979, #5986).
//
// It does not judge whether the test's expectations are right, which job runs
// it, or whether a `required` name covers its subtests. It reads names and
// gates, not results.

// testFuncStartRe finds a top-level test function declaration and its name.
var testFuncStartRe = regexp.MustCompile(`^func (Test[A-Za-z0-9_]+)\(`)

// otherFuncStartRe finds any other top-level function: the end of a body.
var otherFuncStartRe = regexp.MustCompile(`^func `)

// dsnGateRe matches a read of a PostgreSQL DSN environment variable.
var dsnGateRe = regexp.MustCompile(`TEI[A-Za-z0-9_]*_PG_DSN|POSTGRES_DSN`)

// skipRe matches a t.Skip call.
var skipRe = regexp.MustCompile(`\bt\.Skipf?\(`)

// runListRe finds every -run argument in a workflow.
var runListRe = regexp.MustCompile(`-run '([^']*)'`)

// testNameWordRe finds any Test-like identifier inside a -run argument.
var testNameWordRe = regexp.MustCompile(`\b(Test[A-Za-z0-9_]+)\b`)

// gatedTestsIn returns the test functions directly in dir whose body reads a
// PostgreSQL DSN and skips when it is unset.
func gatedTestsIn(dir string) (map[string]bool, error) {
	gated := map[string]bool{}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		text, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(text), "\n")
		for i := 0; i < len(lines); i++ {
			m := testFuncStartRe.FindStringSubmatch(lines[i])
			if m == nil {
				continue
			}
			name := m[1]
			// Collect the body: every following line up to the next top-level
			// function or the end of the file.
			var body strings.Builder
			j := i + 1
			for ; j < len(lines); j++ {
				if otherFuncStartRe.MatchString(lines[j]) {
					break
				}
				body.WriteString(lines[j])
				body.WriteString("\n")
			}
			i = j - 1
			if dsnGateRe.MatchString(body.String()) && skipRe.MatchString(body.String()) {
				gated[name] = true
			}
		}
	}
	return gated, nil
}

// workflowRunNames returns every test-function name a workflow selects with a
// `-run` argument. A name in `required` alone does not make the test run: the
// job selects tests with `go test -run ...`, so only those names are executed.
func workflowRunNames(text string) map[string]bool {
	names := map[string]bool{}
	for _, m := range runListRe.FindAllStringSubmatch(text, -1) {
		for _, n := range testNameWordRe.FindAllStringSubmatch(m[1], -1) {
			names[n[1]] = true
		}
	}
	return names
}

// TestDSNGatedTestsAreNamedByAWorkflow reports every DSN-gated test function
// that no workflow names in a `-run` argument or a `required` list.
func TestDSNGatedTestsAreNamedByAWorkflow(t *testing.T) {
	root := filepath.Join("..", "..")
	packages := []string{
		filepath.Join(root, "cmd", "tei"),
		filepath.Join(root, "internal", "driver", "validationread"),
	}

	allNamed := map[string]bool{}
	for _, workflow := range readWorkflows(t, root) {
		for name := range workflowRunNames(workflow.text) {
			allNamed[name] = true
		}
	}

	var violations []string
	checked := 0
	for _, pkg := range packages {
		gated, err := gatedTestsIn(pkg)
		if err != nil {
			t.Fatal(err)
		}
		rel, relErr := filepath.Rel(root, pkg)
		if relErr != nil {
			t.Fatal(relErr)
		}
		for name := range gated {
			checked++
			if !allNamed[name] {
				violations = append(violations, fmt.Sprintf(
					"%s skips itself unless a PostgreSQL DSN is set, but no workflow selects it with -run, so it never runs where the DSN is set",
					filepath.ToSlash(filepath.Join(rel, name))))
			}
		}
	}
	if checked == 0 {
		t.Fatal("no DSN-gated test found; the check would pass without reading any")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("%d DSN-gated tests are named by no workflow:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}
