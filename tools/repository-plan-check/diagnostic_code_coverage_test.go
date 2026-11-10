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

// diagnosticCodePattern matches the literal a Diagnostic code is written as.
var diagnosticCodePattern = regexp.MustCompile(`Code:\s*"([A-Z][A-Z0-9_]+)"`)

// A diagnostic code is a contract: a caller branches on it, and a reader treats
// it as something that can be returned. A code that no test mentions cannot be
// shown to be reachable, and the run-time value then means less than its name
// suggests (#4032).
//
// The same property is enforced inside internal/validation by a ratchet that
// counts its own codes (maxUncoveredDiagnosticCodes). That ratchet only sees one
// package; this check sees every internal package, because the gap appeared
// package by package (#3906 reported four unreachable values in one of them).
func TestEveryDiagnosticCodeIsMentionedByATest(t *testing.T) {
	root := repositoryRoot(t)
	emitted := map[string][]string{}
	for code, origins := range collectDiagnosticCodes(t, filepath.Join(root, "internal")) {
		emitted[code] = origins
	}
	if len(emitted) == 0 {
		t.Fatal("no diagnostic codes were found; the pattern or the layout changed")
	}
	mentioned := testMentions(t, root)
	var uncovered []string
	for code := range emitted {
		if !mentioned[code] {
			uncovered = append(uncovered, code)
		}
	}
	sort.Strings(uncovered)
	if len(uncovered) == 0 {
		return
	}
	for _, code := range uncovered {
		t.Errorf("%s has no test that mentions it; declared in %s", code, strings.Join(emitted[code], ", "))
	}
	t.Fatalf("%d diagnostic codes are declared but never mentioned by a test", len(uncovered))
}

// collectDiagnosticCodes returns every `Code: "..."` value under internal/, with
// the files it appears in, so a failure can point at the declaration.
func collectDiagnosticCodes(t *testing.T, root string) map[string][]string {
	t.Helper()
	codes := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
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
			relative = path
		}
		seen := map[string]bool{}
		for _, match := range diagnosticCodePattern.FindAllStringSubmatch(string(source), -1) {
			if seen[match[1]] {
				continue
			}
			seen[match[1]] = true
			codes[match[1]] = append(codes[match[1]], filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return codes
}

// testMentions collects the uppercase literals that appear in any _test.go file,
// because a code may be asserted from another package's test.
func testMentions(t *testing.T, root string) map[string]bool {
	t.Helper()
	pattern := regexp.MustCompile(`"([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)"`)
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
		for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
			mentioned[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return mentioned
}
