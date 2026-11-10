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

// schemaVersionPattern matches a `tei.*/vN` schema identifier.
var schemaVersionPattern = regexp.MustCompile(`"(tei\.[a-z0-9._-]+/v[0-9]+)"`)

// maxUnmentionedSchemaVersions is a ratchet, not a target. A schema identifier
// written by source but never named in a test is a response shape no test has
// seen; the number is only allowed to go down (#4032).
const maxUnmentionedSchemaVersions = 14

// TestSchemaVersionLiteralsOnlyGainTests keeps the count of unmentioned schema
// identifiers visible. It fails when a new identifier arrives without a test and
// when the count drops without the ratchet being lowered.
func TestSchemaVersionLiteralsOnlyGainTests(t *testing.T) {
	root := repositoryRoot(t)
	written := schemaVersionsWritten(t, root)
	if len(written) == 0 {
		t.Fatal("no schema version was found; the pattern or the layout changed")
	}
	mentioned := schemaVersionsMentioned(t, root)
	var unmentioned []string
	for identifier := range written {
		if !mentioned[identifier] {
			unmentioned = append(unmentioned, identifier)
		}
	}
	sort.Strings(unmentioned)

	switch {
	case len(unmentioned) > maxUnmentionedSchemaVersions:
		t.Fatalf("%d schema identifiers have no test (ratchet is %d); add a test or lower the list:\n%s",
			len(unmentioned), maxUnmentionedSchemaVersions, strings.Join(unmentioned, "\n"))
	case len(unmentioned) < maxUnmentionedSchemaVersions:
		t.Fatalf("only %d schema identifiers have no test (ratchet is %d); lower maxUnmentionedSchemaVersions to %d",
			len(unmentioned), maxUnmentionedSchemaVersions, len(unmentioned))
	}
}

// schemaVersionsWritten collects the identifiers that non-test Go sources under
// internal/ and cmd/ write.
func schemaVersionsWritten(t *testing.T, root string) map[string]bool {
	t.Helper()
	written := map[string]bool{}
	for _, dir := range []string{filepath.Join(root, "internal"), filepath.Join(root, "cmd")} {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
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
			for _, match := range schemaVersionPattern.FindAllStringSubmatch(string(source), -1) {
				written[match[1]] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return written
}

// schemaVersionsMentioned collects the identifiers that appear in any test or
// testdata file, because a gold receipt or a fixture also records a schema.
func schemaVersionsMentioned(t *testing.T, root string) map[string]bool {
	t.Helper()
	mentioned := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", ".local", "reference":
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, "_test.go") && !strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".yaml") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range schemaVersionPattern.FindAllStringSubmatch(string(source), -1) {
			mentioned[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return mentioned
}
