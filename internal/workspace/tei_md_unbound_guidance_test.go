package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A *.tei.md source that no selected language claims stops the Load. The message
// has to say what to select: "no selected parser is available" alone leaves the
// author guessing which of the built-in languages to add to the manifest.
func TestUnboundMarkdownSourceNamesASelectableLanguage(t *testing.T) {
	root := t.TempDir()
	writeUnboundCaseFile(t, root, "tei.workspace.yaml", "workspace: unbound_md\nformatVersion: 1\nsources:\n  - semantic\n")
	writeUnboundCaseFile(t, root, "semantic/orphan.tei.md", "# orphan\n")

	_, err := Load(root)
	if err == nil {
		t.Fatal("an unclaimed *.tei.md was admitted")
	}
	message := err.Error()
	if !strings.Contains(message, "TEI_MD_PARSER_UNBOUND") {
		t.Fatalf("error is not the unbound diagnostic: %v", err)
	}
	// It names the file, so the author knows which source stopped the load.
	if !strings.Contains(message, "semantic/orphan.tei.md") {
		t.Errorf("the message does not name the source: %s", message)
	}
	// It names a concrete selection the author can paste.
	if !strings.Contains(message, "languages:") {
		t.Errorf("the message does not say how to select a language: %s", message)
	}
	for _, language := range builtinAuthoringLanguageProviders() {
		name := language.Binding.LanguageID + "/" + language.Binding.ContractVersion
		if !strings.Contains(message, name) {
			t.Errorf("the message omits the available language %q: %s", name, message)
		}
	}
}

// Selecting one of the offered languages admits the source, so the guidance is
// actionable rather than decorative.
func TestSelectingTheNamedLanguageAdmitsTheSource(t *testing.T) {
	root := t.TempDir()
	writeUnboundCaseFile(t, root, "tei.workspace.yaml",
		"workspace: selected_md\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n")
	// tei.authoring/v1 reads a marked Data item TABLE; a `data` block belongs to
	// the .tei surface. Use the form this language actually admits.
	writeUnboundCaseFile(t, root, "semantic/thing.tei.md",
		"# Thing\n\n<!-- tei: data Thing in probe -->\n| attribute | type |\n| --- | --- |\n| id | std.text |\n")

	if _, err := Load(root); err != nil {
		t.Fatalf("the language the diagnostic named did not admit the source: %v", err)
	}
}

func writeUnboundCaseFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
