package designoutput

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/ir"
)

func TestTextFormatKeepsSemanticLines(t *testing.T) {
	root := referenceWorkspace(t)
	markdownOut := filepath.Join(t.TempDir(), "markdown")
	textOut := filepath.Join(t.TempDir(), "text")

	markdownManifest, err := WriteWithOptions(root, markdownOut, "postgres", Options{Format: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	textManifest, err := WriteWithOptions(root, textOut, "postgres", Options{Format: "text"})
	if err != nil {
		t.Fatal(err)
	}
	if markdownManifest.Format != "markdown" || textManifest.Format != "text" {
		t.Fatalf("formats = %q, %q", markdownManifest.Format, textManifest.Format)
	}

	markdownBundle, err := buildFromWorkspace(root, "postgres", nil, "", "", Options{Format: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	textBundle, err := buildFromWorkspace(root, "postgres", nil, "", "", Options{Format: "text"})
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckSemanticParity(markdownBundle, textBundle); err != nil {
		t.Fatalf("format change altered the meaning: %v", err)
	}

	markdownPaths := map[string]bool{}
	for _, file := range markdownBundle.Files {
		markdownPaths[file.Path] = true
	}
	if !markdownPaths["table-definitions.md"] {
		t.Fatalf("markdown paths = %v", markdownPaths)
	}
	textPaths := map[string]bool{}
	for _, file := range textBundle.Files {
		textPaths[file.Path] = true
	}
	if !textPaths["table-definitions.txt"] || textPaths["table-definitions.md"] {
		t.Fatalf("text paths = %v", textPaths)
	}
	if !textPaths["access-vs-effects.txt"] {
		t.Fatalf("effects report missing from text bundle: %v", textPaths)
	}
	// DDL is provider output and keeps its own extension in every format.
	for path := range textPaths {
		if strings.HasPrefix(path, "ddl/") && !strings.HasSuffix(path, ".sql") {
			t.Fatalf("DDL was reformatted: %s", path)
		}
	}
}

func TestTextFormatIsPlainText(t *testing.T) {
	for _, line := range SemanticLines(renderText("# Heading\n\n| a | b |\n| --- | --- |\n| `x` | y |\n")) {
		if strings.ContainsAny(line, "#|`") {
			t.Fatalf("text line still carries presentation markers: %q", line)
		}
	}
}

func TestUnknownFormatIsRejected(t *testing.T) {
	_, err := BuildWithOptions(ir.World{}, "rev", "postgres", Options{Format: "pdf"}, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown document format") {
		t.Fatalf("err=%v", err)
	}
}

func TestSemanticParityDetectsADroppedFact(t *testing.T) {
	root := referenceWorkspace(t)
	markdownBundle, err := buildFromWorkspace(root, "postgres", nil, "", "", Options{Format: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	textBundle, err := buildFromWorkspace(root, "postgres", nil, "", "", Options{Format: "text"})
	if err != nil {
		t.Fatal(err)
	}
	for i := range textBundle.Files {
		if textBundle.Files[i].Path == "table-definitions.txt" {
			lines := strings.Split(string(textBundle.Files[i].Content), "\n")
			if len(lines) < 3 {
				t.Fatalf("unexpected fixture: %q", textBundle.Files[i].Content)
			}
			textBundle.Files[i].Content = []byte(strings.Join(append(lines[:1], lines[2:]...), "\n"))
		}
	}
	if err := CheckSemanticParity(markdownBundle, textBundle); err == nil {
		t.Fatal("a dropped fact was not detected")
	}
}

func TestVerifyWithTheSameFormatRoundTrips(t *testing.T) {
	root := referenceWorkspace(t)
	out := filepath.Join(t.TempDir(), "text")
	if _, err := WriteWithOptions(root, out, "postgres", Options{Format: "text"}); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyWithOptions(root, out, "postgres", Options{Format: "text"}); err != nil {
		t.Fatalf("same-format verify failed: %v", err)
	}
	if _, err := VerifyWithOptions(root, out, "postgres", Options{Format: "markdown"}); err == nil {
		t.Fatal("a different format verified against a saved text bundle")
	}
}

func TestSemanticParityDetectsAChangedWord(t *testing.T) {
	root := referenceWorkspace(t)
	markdownBundle, err := buildFromWorkspace(root, "postgres", nil, "", "", Options{Format: "markdown"})
	if err != nil {
		t.Fatal(err)
	}
	textBundle, err := buildFromWorkspace(root, "postgres", nil, "", "", Options{Format: "text"})
	if err != nil {
		t.Fatal(err)
	}
	for i := range textBundle.Files {
		if textBundle.Files[i].Path == "application-data-matrix.txt" {
			textBundle.Files[i].Content = []byte(strings.Replace(string(textBundle.Files[i].Content), "Application", "App", 1))
		}
	}
	if err := CheckSemanticParity(markdownBundle, textBundle); err == nil {
		t.Fatal("a changed word was not detected")
	}
}
