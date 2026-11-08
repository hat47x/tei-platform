package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The manifest is a YAML subset that the CLI reports by name, so a quoted
// name must come back without its quotes and a `#` inside it must stay (#3557).
func TestParseWorkspaceManifestReadsQuotedNames(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{name: "plain", src: "workspace: scn\nformatVersion: 1\nsources:\n  - semantic\n", want: "scn"},
		{name: "double quoted", src: "workspace: \"scn\"\nformatVersion: 1\nsources:\n  - semantic\n", want: "scn"},
		{name: "single quoted", src: "workspace: 'scn'\nformatVersion: 1\nsources:\n  - semantic\n", want: "scn"},
		{name: "double quoted with hash", src: "workspace: \"scn#1\"\nformatVersion: 1\nsources:\n  - semantic\n", want: "scn#1"},
		{name: "single quoted with hash", src: "workspace: 'scn#1'\nformatVersion: 1\nsources:\n  - semantic\n", want: "scn#1"},
		{name: "double quoted with space", src: "workspace: \"scn one\"\nformatVersion: 1\nsources:\n  - semantic\n", want: "scn one"},
		{name: "trailing comment", src: "workspace: scn # the name\nformatVersion: 1\nsources:\n  - semantic\n", want: "scn"},
		{name: "quoted source root", src: "workspace: scn\nformatVersion: 1\nsources:\n  - \"semantic\"\n", want: "scn"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := parseWorkspaceManifestYAML(tc.src)
			if err != nil {
				t.Fatalf("manifest was rejected: %v", err)
			}
			if manifest.Name != tc.want {
				t.Fatalf("name=%q want %q", manifest.Name, tc.want)
			}
			if len(manifest.Sources) != 1 || manifest.Sources[0] != "semantic" {
				t.Fatalf("sources=%q want [semantic]", manifest.Sources)
			}
		})
	}
}

// An unquoted `#` is part of the value, not the start of a comment. Silently
// truncating there used to change the reported name.
func TestParseWorkspaceManifestKeepsUnquotedHash(t *testing.T) {
	manifest, err := parseWorkspaceManifestYAML("workspace: scn#1\nformatVersion: 1\nsources:\n  - semantic\n")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "scn#1" {
		t.Fatalf("name=%q want %q", manifest.Name, "scn#1")
	}
}

func TestParseWorkspaceManifestRejectsBrokenQuotes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		src     string
		wantErr string
	}{
		{name: "unterminated name", src: "workspace: \"scn\nformatVersion: 1\nsources:\n  - semantic\n", wantErr: "unterminated quoted value"},
		{name: "text after quote", src: "workspace: \"scn\" extra\nformatVersion: 1\nsources:\n  - semantic\n", wantErr: "unexpected text after quoted value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseWorkspaceManifestYAML(tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// The same rules apply to imports, where the path and namespace are also
// reported back to callers.
func TestParseWorkspaceManifestReadsQuotedImportValues(t *testing.T) {
	src := "workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: 'sui'\n    path: \"../provider\"\n    namespace: sui # owner\n"
	manifest, err := parseWorkspaceManifestYAML(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Imports) != 1 {
		t.Fatalf("imports=%+v", manifest.Imports)
	}
	imp := manifest.Imports[0]
	if imp.Alias != "sui" || imp.Path != "../provider" || imp.Namespace != "sui" {
		t.Fatalf("import parsed incorrectly: %+v", imp)
	}
}

func TestLoadReportsTheNameWrittenInTheManifest(t *testing.T) {
	manifest := "workspace: \"scn #1\"  # display name\nformatVersion: 1\nsources:\n  - semantic\n"
	root := writeFormatVersionWorkspace(t, manifest)
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Identity().Name; got != "scn #1" {
		t.Fatalf("workspace name=%q want %q", got, "scn #1")
	}
}

// The shipped manifests are the ones every caller reads first; they must parse
// with the same rules the tests above pin.
func TestShippedManifestsParse(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, dir := range []string{"reference/workspace", "examples/first-workspace", "reference/product-contracts"} {
		raw, err := os.ReadFile(filepath.Join(root, dir, "tei.workspace.yaml"))
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		if _, err := parseWorkspaceManifestYAML(string(raw)); err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
	}
}
