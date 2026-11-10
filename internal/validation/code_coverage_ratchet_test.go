package validation

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// diagnosticCodePatterns match the two ways this package writes a diagnostic
// code: as the second argument of errFor and as the Code field of a Diagnostic.
// Only these are counted, so an ordinary uppercase literal is not mistaken for a
// contract the tests must hold.
var diagnosticCodePatterns = []*regexp.Regexp{
	regexp.MustCompile(`errFor\([^,]+,\s*"([A-Z0-9_]+)"`),
	regexp.MustCompile(`Code:\s*"([A-Z0-9_]+)"`),
}

// anyUppercaseLiteralPattern matches an uppercase literal in a test file.
var anyUppercaseLiteralPattern = regexp.MustCompile(`"([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)"`)

// maxUncoveredDiagnosticCodes is a ratchet, not a target. A diagnostic code that
// no test mentions is a contract the tests do not hold, and the number is only
// allowed to go down: when a code gains a test, lower this value in the same
// change (#4032).
const maxUncoveredDiagnosticCodes = 9

// TestDiagnosticCodeCoverageOnlyImproves keeps the count of untested diagnostic
// codes visible. It fails when a new code arrives without a test, and it fails
// when the count drops without the ratchet being lowered, so the number cannot
// drift upwards unnoticed.
func TestDiagnosticCodeCoverageOnlyImproves(t *testing.T) {
	root := packageRoot(t)
	emitted := emittedDiagnosticCodes(t, root)
	mentioned := testMentionedStrings(t, filepath.Join(root, "..", ".."))

	var uncovered []string
	for _, code := range emitted {
		if !mentioned[code] {
			uncovered = append(uncovered, code)
		}
	}
	sort.Strings(uncovered)

	switch {
	case len(uncovered) > maxUncoveredDiagnosticCodes:
		t.Fatalf("%d diagnostic codes have no test (ratchet is %d); add a test for the new ones: %v",
			len(uncovered), maxUncoveredDiagnosticCodes, uncovered)
	case len(uncovered) < maxUncoveredDiagnosticCodes:
		t.Fatalf("only %d diagnostic codes have no test (ratchet is %d); lower maxUncoveredDiagnosticCodes to %d",
			len(uncovered), maxUncoveredDiagnosticCodes, len(uncovered))
	}
}

func packageRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test source file")
	}
	return filepath.Dir(file)
}

// emittedDiagnosticCodes reads the codes the package writes into a Diagnostic.
func emittedDiagnosticCodes(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, pattern := range diagnosticCodePatterns {
			for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
				seen[match[1]] = true
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("no diagnostic codes were found; the pattern or the package layout changed")
	}
	codes := make([]string, 0, len(seen))
	for code := range seen {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// testMentionedStrings collects every uppercase literal that appears in a
// _test.go file under the repository root, because a test may assert a code
// from another package.
func testMentionedStrings(t *testing.T, root string) map[string]bool {
	t.Helper()
	mentioned := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "testdata", ".local":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range anyUppercaseLiteralPattern.FindAllStringSubmatch(string(source), -1) {
			mentioned[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mentioned) == 0 {
		t.Fatal("no uppercase literals were read from tests; the walk root changed")
	}
	return mentioned
}
