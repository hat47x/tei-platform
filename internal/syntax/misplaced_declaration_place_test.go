package syntax

import (
	"strings"
	"testing"
)

// semanticNamespaceBodies lists the declarations a semantic file accepts inside
// a namespace body. They are not "misplaced" after a namespace, so the
// namespace-scope test skips them; the table still carries their directory.
var semanticNamespaceBodies = map[string]bool{
	"type":        true,
	"data":        true,
	"capability":  true,
	"validation":  true,
	"calendar":    true,
	"period":      true,
	"effect-rule": true,
	"run-plan":    true,
}

// placeholderBodies gives each declaration keyword arguments its own parser
// accepts, so the test reaches the placement branch rather than a narrower
// error about a malformed body. What is asserted is the placement answer, so
// the body only has to be well formed enough to get there.
var placeholderBodies = map[string]string{
	"type":        " x from y",
	"persist":     " sales.order total",
	"realize":     " sales.order total",
	"data":        " x { attribute a type std.text }",
	"capability":  " x { }",
	"validation":  " x { }",
	"namespace":   " x",
	"application": " x { }",
	"pattern":     " x browse { }",
	"query":       " x { }",
	"contract":    " x { }",
	"database":    " x { kind relational ownership managed-here }",
	"migration":   " x { database d recovery forward-fix }",
}

// TestMisplacedDeclarationNamesADirectoryAndAnExample is the mechanical half of
// #4169. A caller who writes a declaration in the wrong file is told which file
// to move it to, without reading the source: the message names the directory
// and an example path inside it, and both come from the same table that names
// the declaration, so the answer cannot drift away from what the loader scans.
func TestMisplacedDeclarationNamesADirectoryAndAnExample(t *testing.T) {
	// Each keyword is placed after a semantic namespace, which is the widest
	// scope, so every row of the table is exercised.
	for _, declaration := range topLevelDeclarations {
		if semanticNamespaceBodies[declaration.keyword] {
			continue
		}
		body, ok := placeholderBodies[declaration.keyword]
		if !ok {
			body = " x { }"
		}
		src := "namespace sales\n\n" + declaration.keyword + body + "\n"
		_, err := NewParser(src).ParseSemanticFile()
		if err == nil {
			t.Fatalf("%q: expected a parse error", declaration.keyword)
		}
		message, ok := ErrorMessage(err)
		if !ok {
			t.Fatalf("%q: error has no message", declaration.keyword)
		}
		if declaration.directory == "" {
			t.Fatalf("%q has no directory in the table", declaration.keyword)
		}
		if len(declaration.fileNames) == 0 || declaration.fileNames[0] == "" {
			t.Fatalf("%q has no example path in the table", declaration.keyword)
		}
		if !strings.Contains(message, "declare it under "+declaration.directory) {
			t.Fatalf("%q: message does not name %q: %s",
				declaration.keyword, declaration.directory, message)
		}
		if !strings.Contains(message, "for example "+declaration.fileNames[0]) {
			t.Fatalf("%q: message does not name %q: %s",
				declaration.keyword, declaration.fileNames[0], message)
		}
	}
}

// A directory the loader never scans would be a wrong answer, so every entry
// must name a source root the workspace loader recognises.
func TestMisplacedDeclarationDirectoriesAreScannedSourceRoots(t *testing.T) {
	roots := map[string]bool{}
	for _, root := range []string{
		"semantic", "applications", "databases", "environments", "plugins", "bundles",
		"solutions", "reference",
	} {
		roots[root] = true
	}
	for _, declaration := range topLevelDeclarations {
		first := declaration.directory
		if i := strings.Index(first, "/"); i >= 0 {
			first = first[:i]
		}
		if !roots[first] {
			t.Fatalf("%q names directory %q, whose first component %q is not a scanned source root",
				declaration.keyword, declaration.directory, first)
		}
	}
}
