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

// unmentionedSchemaVersionsAllowed lists the identifiers that no test names yet.
// Each entry names the package that owns the response shape, so the list stays a
// to-do rather than a place to park new identifiers (#4032).
var unmentionedSchemaVersionsAllowed = map[string]string{
	"tei.cli-scenario-plan/v0":                                     "internal/scenario: no test asserts the plan schema",
	"tei.desktop.draft-marker/v0":                                  "internal/desktophost: draft markers are written to disk",
	"tei.host-acceptance-store/v0":                                 "internal/desktophost-acceptance: store header",
	"tei.knowledge.data-design-candidate-review/v1":                "internal/knowledge: candidate review record",
	"tei.knowledge.data-design.business-identity-candidate/v2":     "internal/knowledge: candidate record",
	"tei.knowledge.data-design.functional-dependency-candidate/v2": "internal/knowledge: candidate record",
	"tei.modelref.impact-review/v0":                                "internal/modelref: impact review response",
	"tei.operations.model-location/v0":                             "internal/operations: model location response",
	"tei.operations.xead-observation/v0":                           "internal/operations: xead observation response",
	"tei.plugin-definition-json/v2":                                "internal/plugin: plugin definition file",
	"tei.plugin-package-input-binding/v1":                          "internal/plugin: package admission input",
	"tei.plugin-package-output-observation/v1":                     "internal/plugin: package admission output",
	"tei.xeadsource.cross-view/v4":                                 "internal/xeadsource: cross-view source",
}

// TestSchemaVersionLiteralsHaveATest keeps the set of unnamed schema identifiers
// visible: a new identifier fails here until it gets a test or a listed reason,
// and a listed one that gains a test has to be removed from the list.
//
// The list lives in this file, so the scan skips this file when collecting what
// the tests mention; otherwise listing an identifier would be the same as
// testing it.
func TestSchemaVersionLiteralsHaveATest(t *testing.T) {
	root := repositoryRoot(t)
	written := schemaVersionsWritten(t, root)
	if len(written) == 0 {
		t.Fatal("no schema version was found; the pattern or the layout changed")
	}
	mentioned := schemaVersionsMentioned(t, root)
	var unexpected []string
	for identifier := range written {
		if mentioned[identifier] {
			if _, listed := unmentionedSchemaVersionsAllowed[identifier]; listed {
				t.Errorf("%s is named by a test now; remove it from unmentionedSchemaVersionsAllowed", identifier)
			}
			continue
		}
		if _, listed := unmentionedSchemaVersionsAllowed[identifier]; !listed {
			unexpected = append(unexpected, identifier)
		}
	}
	sort.Strings(unexpected)
	for _, identifier := range unexpected {
		t.Errorf("%s is written by source but no test names it; add a test or list it with a reason", identifier)
	}
	if len(unexpected) != 0 {
		t.Fatalf("%d schema identifiers have neither a test nor a listed reason", len(unexpected))
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

// schemaVersionsMentioned collects the identifiers that appear in a test or a
// fixture, because a gold receipt or a fixture also records a schema.
func schemaVersionsMentioned(t *testing.T, root string) map[string]bool {
	t.Helper()
	self := "schema_version_coverage_test.go"
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
		if name == self {
			return nil
		}
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
