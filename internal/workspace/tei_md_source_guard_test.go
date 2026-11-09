package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTEIMDTestFile(t *testing.T, root, path, content string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testTEIMDWorkspace(t *testing.T, sources ...string) string {
	t.Helper()
	root := t.TempDir()
	var manifest strings.Builder
	manifest.WriteString("workspace: tei_md_guard\nformatVersion: 1\nsources:\n")
	for _, source := range sources {
		manifest.WriteString("  - " + source + "\n")
	}
	writeTEIMDTestFile(t, root, "tei.workspace.yaml", manifest.String())
	return root
}

const legacyOrgSource = `namespace organization

type OrgUnitId from std.identifier

data OrgUnit {
    attribute orgUnitId type organization.org_unit_id
}
`

func TestWorkspaceTEIMarkdownUnboundIsExplicitlyRejected(t *testing.T) {
	cases := []struct {
		name, root, file string
	}{
		{"semantic", "semantic", "semantic/organization/model.tei.md"},
		{"query", "applications", "applications/org-view/queries/case_read.tei.md"},
		{"persistence", "databases", "databases/org-store/persistence/case_record.tei.md"},
		{"plugin", "plugins", "plugins/plugin.tei.md"},
		{"bundle", "bundles", "bundles/bundle.tei.md"},
		{"release", "releases", "releases/applications/org-view/app.tei.md"},
		{"mixed-case-extension", "semantic", "semantic/organization/model.tei.MD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := testTEIMDWorkspace(t, tc.root)
			writeTEIMDTestFile(t, root, tc.file, "# TEI model\n\n```tei\nnamespace organization\n```\n")

			// The current loader has no selected parser for these authored
			// Markdown files; success with an empty/partial World is unsafe.
			_, err := Load(root)
			if err == nil {
				t.Fatal("silently skipped a .tei.md definition instead of diagnosing the unbound parser")
			}
			if !strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") ||
				!strings.Contains(err.Error(), filepath.ToSlash(tc.file)) {
				t.Fatalf("unexpected error; require a source-specific parser diagnostic: %v", err)
			}
		})
	}
}

func TestWorkspaceTEIMarkdownAndLegacyDuplicateCannotChooseSilently(t *testing.T) {
	root := testTEIMDWorkspace(t, "semantic")
	writeTEIMDTestFile(t, root, "semantic/model.tei", legacyOrgSource)
	writeTEIMDTestFile(t, root, "semantic/model.tei.md", legacyOrgSource)
	if _, err := Load(root); err == nil ||
		!strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") {
		t.Fatalf("the legacy file must not silently win over a second .tei.md source: %v", err)
	}
}

func TestWorkspaceLegacyTEIAndOrdinaryMarkdownRemainReadable(t *testing.T) {
	root := testTEIMDWorkspace(t, "semantic")
	writeTEIMDTestFile(t, root, "semantic/model.tei", legacyOrgSource)
	writeTEIMDTestFile(t, root, "semantic/README.md", "# Notes\nOrdinary Markdown is not itself a TEI model.\n")
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("legacy native source or ordinary Markdown was unexpectedly rejected: %v", err)
	}
	var seen bool
	for _, ns := range loaded.World.Semantics {
		if ns.ID == "organization" && len(ns.Data) == 1 && ns.Data[0].ID == "organization.org_unit" {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("recognized native TEI meaning was dropped: %+v", loaded.World.Semantics)
	}
}

func TestWorkspaceTEIMarkdownOutsideDeclaredSourceRootIsNotAdmitted(t *testing.T) {
	root := testTEIMDWorkspace(t, "semantic")
	writeTEIMDTestFile(t, root, "semantic/model.tei", legacyOrgSource)
	writeTEIMDTestFile(t, root, "drafts/other.tei.md", "# Outside source roots\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("files outside declared roots should not enter the Workspace: %v", err)
	}
}
