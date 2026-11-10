package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const manifestProbeBody = "namespace sales\n"

func writeManifestProbe(t *testing.T, manifest string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte(manifestProbeBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// A leading UTF-8 BOM is accepted, matching the syntax lexer (#3556). Windows
// editors and PowerShell redirection add it, and refusing the manifest while
// accepting a BOM in a .tei source was an unexplained asymmetry.
func TestManifestAcceptsOneLeadingBOM(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{name: "BOM then workspace", src: "\ufeffworkspace: bom_probe\nformatVersion: 1\nsources:\n  - semantic\n"},
		{name: "BOM without workspace name", src: "\ufeffformatVersion: 1\nsources:\n  - semantic\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeManifestProbe(t, tc.src)
			result, err := Load(root)
			if err != nil {
				t.Fatalf("manifest with a BOM was refused: %v", err)
			}
			if strings.HasPrefix(result.Identity().Name, "\ufeff") {
				t.Fatalf("the BOM leaked into the name: %q", result.Identity().Name)
			}
			if tc.name == "BOM then workspace" && result.Identity().Name != "bom_probe" {
				t.Fatalf("name=%q", result.Identity().Name)
			}
		})
	}
}

// A BOM is only stripped at the start. A second one is an invalid character
// rather than a silently ignored one.
func TestManifestRejectsBOMAfterTheFirstCharacter(t *testing.T) {
	root := writeManifestProbe(t, "workspace: probe\ufeff\nformatVersion: 1\nsources:\n  - semantic\n")
	if _, err := Load(root); err == nil {
		t.Fatal("a BOM inside the manifest was accepted")
	}
}

// Quoted values are accepted and returned without the quotes, so the name in
// the response equals the name in the manifest (#3557).
func TestManifestUnquotesSingleAndDoubleQuotedValues(t *testing.T) {
	for _, tc := range []struct{ src, want string }{
		{src: "workspace: \"quoted\"\nformatVersion: 1\nsources:\n  - semantic\n", want: "quoted"},
		{src: "workspace: 'quoted'\nformatVersion: 1\nsources:\n  - semantic\n", want: "quoted"},
	} {
		root := writeManifestProbe(t, tc.src)
		result, err := Load(root)
		if err != nil {
			t.Fatalf("%q: %v", tc.src, err)
		}
		if got := result.Identity().Name; got != tc.want {
			t.Fatalf("name=%q want %q", got, tc.want)
		}
	}
}

// A quoted source root names the same root as the unquoted spelling.
func TestManifestUnquotesSourceRoots(t *testing.T) {
	root := writeManifestProbe(t, "workspace: probe\nformatVersion: 1\nsources:\n  - \"semantic\"\n")
	result, err := Load(root)
	if err != nil {
		t.Fatalf("quoted source root was refused: %v", err)
	}
	if result.World.Semantics == nil {
		t.Fatal("no semantic namespace was read")
	}
}

// A `#` inside an unquoted value is part of the value. Only a `#` at the start
// of a line or after whitespace begins a comment (#3557).
func TestManifestKeepsHashInsideUnquotedName(t *testing.T) {
	root := writeManifestProbe(t, "workspace: scn#1\nformatVersion: 1\nsources:\n  - semantic\n")
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Identity().Name; got != "scn#1" {
		t.Fatalf("name=%q want %q", got, "scn#1")
	}
	for _, tc := range []struct{ src, want string }{
		{src: "workspace: probe # trailing comment\nformatVersion: 1\nsources:\n  - semantic\n", want: "probe"},
		{src: "# leading comment\nworkspace: probe\nformatVersion: 1\nsources:\n  - semantic\n", want: "probe"},
	} {
		root := writeManifestProbe(t, tc.src)
		result, err := Load(root)
		if err != nil {
			t.Fatalf("%q: %v", tc.src, err)
		}
		if got := result.Identity().Name; got != tc.want {
			t.Fatalf("%q: name=%q want %q", tc.src, got, tc.want)
		}
	}
}

// Malformed quoting is refused with a reason instead of being reinterpreted.
func TestManifestRefusesMalformedQuoting(t *testing.T) {
	for _, tc := range []struct{ name, src, wantErr string }{
		{name: "missing closing quote", src: "workspace: \"probe\nformatVersion: 1\nsources:\n  - semantic\n", wantErr: "missing its closing"},
		{name: "text after the closing quote", src: "workspace: \"probe\" extra\nformatVersion: 1\nsources:\n  - semantic\n", wantErr: "followed by"},
		{name: "quoted formatVersion", src: "workspace: probe\nformatVersion: \"1\"\nsources:\n  - semantic\n", wantErr: "positive integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeManifestProbe(t, tc.src)
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// The refused spellings stay refused, and the refusal names the line.
func TestManifestStillRefusesUnsupportedSpellings(t *testing.T) {
	for _, tc := range []struct{ name, src, wantErr string }{
		{name: "inline array", src: "workspace: probe\nformatVersion: 1\nsources: [semantic]\n", wantErr: "unsupported workspace manifest entry"},
		{name: "unknown key", src: "workspace: probe\nformatVersion: 1\ndescription: hi\nsources:\n  - semantic\n", wantErr: "unsupported workspace manifest entry"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeManifestProbe(t, tc.src)
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want %q, got %v", tc.wantErr, err)
			}
			if !strings.Contains(err.Error(), "line 3") && !strings.Contains(err.Error(), "line 4") {
				t.Fatalf("the refusal does not name the line: %v", err)
			}
		})
	}
}
