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

// These checks pin two narrow claims about the test names .github/workflows
// carry. Both are copies of a Go test function name, and both go stale silently
// when a test is renamed or when a name is typed differently.
//
//   - every name in a `required=( … )` list must be a test function
//     (otherwise the job fails with "required test did not PASS" although the
//     test ran and passed under its new name; #5886 measured that);
//   - every name inside a `-run '^A$|^B$ …'` regex must be a test function
//     (otherwise the job runs no test for that name and its receipt is empty).
//
// Neither check judges whether a test is in the right job, whether a SKIP
// counts, or whether a required name also needs its subtests. They read names,
// not results.

var requiredBlockRe = regexp.MustCompile(`(?s)required=\(\s*(.*?)\n\s*\)`)

var quotedNameRe = regexp.MustCompile(`'([^']+)'`)

// runRegexRe finds the anchored names inside a -run '...' argument.
var runRegexRe = regexp.MustCompile(`-run '([^']*)'`)

// runNameRe finds one `^Name$` alternative inside a -run regex.
var runNameRe = regexp.MustCompile(`\^([A-Za-z0-9_]+)\$`)

var testFuncRe = regexp.MustCompile(`^func (Test[A-Za-z0-9_]+)\(`)

// goTestNamesIn returns the test function names declared by the *_test.go files
// directly in dir.
func goTestNamesIn(dir string) (map[string]bool, error) {
	names := map[string]bool{}
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		text, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(text), "\n") {
			if m := testFuncRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				names[m[1]] = true
			}
		}
	}
	return names, nil
}

// testNamesByDir walks the root module and maps each package directory to the
// test functions it declares. Nested modules are their own build units and are
// skipped, matching the paths check's scope.
func testNamesByDir(t *testing.T, root string) map[string]map[string]bool {
	t.Helper()
	byDir := map[string]map[string]bool{}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		top := entry.Name()
		if _, statErr := os.Stat(filepath.Join(root, top, "go.mod")); statErr == nil {
			continue
		}
		walkErr := filepath.Walk(filepath.Join(root, top), func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || !info.IsDir() {
				return nil
			}
			// A workflow may run a nested module's package (cmd/tei-studio
			// has its own go.mod), so its test names are indexed too. The
			// paths check skips nested modules; this one only compares names.
			names, namesErr := goTestNamesIn(path)
			if namesErr != nil {
				return namesErr
			}
			if len(names) > 0 {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				byDir[filepath.ToSlash(rel)] = names
			}
			return nil
		})
		if walkErr != nil {
			t.Fatal(walkErr)
		}
	}
	return byDir
}

func anyPackageDeclares(byDir map[string]map[string]bool, name string) bool {
	for _, names := range byDir {
		if names[name] {
			return true
		}
	}
	return false
}

// testFunctionName strips the subtest suffix a required name may carry.
func testFunctionName(required string) string {
	if at := strings.Index(required, "/"); at >= 0 {
		return required[:at]
	}
	return required
}

func readWorkflows(t *testing.T, root string) []struct {
	name string
	text string
} {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no workflows found; a check that reads none would pass by accident")
	}
	out := make([]struct {
		name string
		text string
	}, 0, len(paths))
	for _, path := range paths {
		text, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		out = append(out, struct {
			name string
			text string
		}{name: filepath.Base(path), text: string(text)})
	}
	return out
}

// TestWorkflowRequiredNamesAreTestFunctions reports every required name that is
// not a test function anywhere in the root module.
func TestWorkflowRequiredNamesAreTestFunctions(t *testing.T) {
	root := filepath.Join("..", "..")
	byDir := testNamesByDir(t, root)

	var violations []string
	checked := 0
	for _, workflow := range readWorkflows(t, root) {
		for _, block := range requiredBlockRe.FindAllStringSubmatch(workflow.text, -1) {
			for _, m := range quotedNameRe.FindAllStringSubmatch(block[1], -1) {
				required := m[1]
				checked++
				if function := testFunctionName(required); !anyPackageDeclares(byDir, function) {
					violations = append(violations, fmt.Sprintf(
						"%s requires %q but no test function %s exists in this repository; a required name is a copy of a test function name, so a rename has to update both",
						workflow.name, required, function))
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no required names found; the check would pass without reading any")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("%d required names are not test functions:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}

// TestWorkflowRunNamesAreTestFunctions reports every name a `-run` regex selects
// that is not a test function anywhere in the root module.
func TestWorkflowRunNamesAreTestFunctions(t *testing.T) {
	root := filepath.Join("..", "..")
	byDir := testNamesByDir(t, root)

	var violations []string
	checked := 0
	for _, workflow := range readWorkflows(t, root) {
		for _, m := range runRegexRe.FindAllStringSubmatch(workflow.text, -1) {
			for _, n := range runNameRe.FindAllStringSubmatch(m[1], -1) {
				name := n[1]
				checked++
				if !anyPackageDeclares(byDir, name) {
					violations = append(violations, fmt.Sprintf(
						"%s runs -run '^%s$' but no test function %s exists in this repository; a -run name is a copy of a test function name, so a rename has to update both",
						workflow.name, name, name))
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no -run names found; the check would pass without reading any")
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Fatalf("%d -run names are not test functions:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}
