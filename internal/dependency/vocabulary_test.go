package dependency

import "testing"

// Scope and Kind are the vocabulary of a Requirement: a reader of a derived
// requirement branches on them and a provider resolves by them. The constants
// are exported because other packages build requirements, so the set is part of
// this package's contract (#4032).
func TestScopeAndKindVocabulariesAreCovered(t *testing.T) {
	t.Run("scopes", func(t *testing.T) {
		scopes := []Scope{ScopeEditor, ScopeBuild, ScopeTest, ScopeRuntime, ScopeMigration}
		want := []string{"editor", "build", "test", "runtime", "migration"}
		if len(scopes) != len(want) {
			t.Fatalf("a scope was dropped: %d values", len(scopes))
		}
		seen := map[string]bool{}
		for i, scope := range scopes {
			if string(scope) != want[i] {
				t.Fatalf("scopes[%d]=%q want %q", i, scope, want[i])
			}
			if seen[string(scope)] {
				t.Fatalf("scope %q is declared twice", scope)
			}
			seen[string(scope)] = true
		}
	})
	t.Run("kinds", func(t *testing.T) {
		kinds := []Kind{KindToolchain, KindRuntime, KindLanguagePackage, KindTool}
		want := []string{"toolchain", "runtime", "language-package", "tool"}
		if len(kinds) != len(want) {
			t.Fatalf("a kind was dropped: %d values", len(kinds))
		}
		seen := map[string]bool{}
		for i, kind := range kinds {
			if string(kind) != want[i] {
				t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
			}
			if seen[string(kind)] {
				t.Fatalf("kind %q is declared twice", kind)
			}
			seen[string(kind)] = true
		}
	})
}
