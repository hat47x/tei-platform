package plancheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// exportedStringConstantPattern matches a const declaration whose name is
// exported and whose value is a lowercase word: a declared vocabulary word rather
// than a schema identifier or a message.
var exportedStringConstantPattern = regexp.MustCompile(`(?m)^\s*(?:(\w+)\s+\w+\s*=\s*|(\w+)\s*=\s*)"([a-z][a-z0-9-]{3,})"`)

// unmentionedExportedStringConstants returns “package/file.go/Name“ for every
// exported string constant under internal/ that no test mentions.
func unmentionedExportedStringConstants(t *testing.T, root string) []string {
	t.Helper()
	mentioned := constantNamesMentionedByTests(t, root)
	declared := 0
	var out []string
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		for _, match := range exportedStringConstantPattern.FindAllStringSubmatch(string(source), -1) {
			constant := match[1]
			if constant == "" {
				constant = match[2]
			}
			if isExported(constant) {
				declared++
			}
			if !isExported(constant) || mentioned[constant] {
				continue
			}
			out = append(out, filepath.ToSlash(relative)+"/"+constant)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The guard counts declarations, not unmentioned ones: an empty result is the
	// goal (every constant has a test), while an empty scan means the pattern or
	// the layout changed and the check would pass for the wrong reason.
	if declared == 0 {
		t.Fatal("no exported string constant was found; the pattern or the layout changed")
	}
	// A file may declare the same constant twice only via separate blocks; keeping
	// the first occurrence makes the list stable.
	deduped := map[string]bool{}
	var kept []string
	for _, entry := range out {
		if deduped[entry] {
			continue
		}
		deduped[entry] = true
		kept = append(kept, entry)
	}
	sort.Strings(kept)
	return kept
}

// isExported reports whether a Go identifier is exported.
func isExported(name string) bool {
	if name == "" {
		return false
	}
	first := name[0]
	return first >= 'A' && first <= 'Z'
}

// constantNamesMentionedByTests returns every exported identifier that appears in
// any test file under the repository, because a constant may be asserted from
// another package's test.
func constantNamesMentionedByTests(t *testing.T, root string) map[string]bool {
	t.Helper()
	mentioned := map[string]bool{}
	identifier := regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\b`)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", ".local", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		// The coverage list names every entry as a string, so reading this file
		// would make every constant look mentioned. Skip it: the list is the
		// claim, not the evidence.
		if entry.Name() == "string_constant_coverage_test.go" {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range identifier.FindAllStringSubmatch(string(source), -1) {
			mentioned[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mentioned) == 0 {
		t.Fatal("no identifier was read from the tests; the walk root changed")
	}
	return mentioned
}
