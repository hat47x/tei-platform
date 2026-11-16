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

// diagnosticCodeConstPattern matches a code declared as a named constant, which
// is how internal/plugin writes its resolver vocabulary. Without this the check
// saw only struct-literal codes and reported success while five declared codes
// had no test at all, so "every code is mentioned" meant "every code written in
// one of the two styles is mentioned".
//
// The trailing underscore is excluded so a concatenation prefix such as
// "SAVE_HOOK_" is not mistaken for a code.
var diagnosticCodeConstPattern = regexp.MustCompile(`(?m)^\s*[A-Za-z0-9_]+\s*=\s*"([A-Z][A-Z0-9]*_[A-Z0-9_]*[A-Z0-9])"`)

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
	named := testMentionedConstantNames(t, root)
	var uncovered []string
	for code := range emitted {
		if mentioned[code] {
			continue
		}
		// A test may name the constant instead of repeating the literal; both
		// pin the code, and the repository does both.
		if names := constantNamesByCode(t, root, code); len(names) > 0 {
			coveredByName := false
			for _, name := range names {
				if named[name] {
					coveredByName = true
					break
				}
			}
			if coveredByName {
				continue
			}
		}
		if _, allowed := uncoveredDiagnosticCodesAllowed[code]; allowed {
			continue
		}
		uncovered = append(uncovered, code)
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
// uncoveredDiagnosticCodesAllowed names the codes that are declared and emitted
// but that no test reaches yet. Widening the pattern revealed them; they are
// listed rather than removed so the gap is visible and cannot grow. Remove an
// entry when a test pins the code it names.
var uncoveredDiagnosticCodesAllowed = map[string]string{
	"APPLICATION_AMBIGUOUS":                 "declared in internal/documentdraft/draft.go; needs a draft whose subject matches two Applications",
	"ARTIFACT_REVISION_MISMATCH":            "declared in internal/artifact/consistency.go; needs a revision skew that no fixture builds",
	"EXTERNAL_CANDIDATE_INCOMPLETE":         "declared in internal/operations/external_candidates.go; needs a partially answered candidate",
	"PROVISIONAL_GRANT_TRUST_SCOPE_INVALID": "declared in internal/security/provisionaleffectgrant/trustscope.go; the constructor rejects the value before a code can be emitted",
	"RELATION_NOT_OF_SUBJECT":               "declared in internal/documentdraft/draft.go; needs a relation that exists but belongs to another subject",
}

// TestTheUncoveredCodeAllowlistDoesNotGrow pins the size of the allowlist so a
// new gap has to be argued for rather than appended.
func TestTheUncoveredCodeAllowlistDoesNotGrow(t *testing.T) {
	if len(uncoveredDiagnosticCodesAllowed) > 5 {
		t.Fatalf("the allowlist has %d entries; remove a fixed one before adding another", len(uncoveredDiagnosticCodesAllowed))
	}
	for code, reason := range uncoveredDiagnosticCodesAllowed {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s is allowed without a reason", code)
		}
	}
}

// constantIdentPattern captures the identifier a code literal is bound to.
var constantIdentPattern = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s*=\s*"([A-Z][A-Z0-9]*_[A-Z0-9_]*[A-Z0-9])"`)

// constantNamesByCode maps a code literal to the identifiers bound to it. It is
// rebuilt per call because a test binary runs the check once; the map is small
// and keeping it out of package state avoids an init-order dependency.
func constantNamesByCode(t *testing.T, root, code string) []string {
	t.Helper()
	base := filepath.Join(root, "internal")
	byCode := map[string][]string{}
	filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, match := range constantIdentPattern.FindAllStringSubmatch(string(source), -1) {
			byCode[match[2]] = append(byCode[match[2]], match[1])
		}
		return nil
	})
	return byCode[code]
}

// testMentionedConstantNames collects the identifiers that appear in a _test.go
// file, so a test that writes plugin.DiagnosticDependencyCycle covers the code
// that constant holds just as a test repeating the literal does.
func testMentionedConstantNames(t *testing.T, root string) map[string]bool {
	t.Helper()
	identifier := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\b`)
	named := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err := skipVanishedPath(err); err != nil {
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
		for _, match := range identifier.FindAllStringSubmatch(string(source), -1) {
			named[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return named
}

func collectDiagnosticCodes(t *testing.T, root string) map[string][]string {
	t.Helper()
	codes := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err := skipVanishedPath(err); err != nil {
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
		for _, pattern := range []*regexp.Regexp{diagnosticCodePattern, diagnosticCodeConstPattern} {
			for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
				if seen[match[1]] {
					continue
				}
				seen[match[1]] = true
				codes[match[1]] = append(codes[match[1]], filepath.ToSlash(relative))
			}
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
		if err := skipVanishedPath(err); err != nil {
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
