package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeUnreadSourceFixture writes a file into the fixture workspace.
func writeUnreadSourceFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A .tei file the loader inventories but no domain loader reads is invisible to
// the model: round 71 showed a misplaced persistence mapping validating as `ok`
// while changing nothing (#4128).
func TestUnreadTIESourceIsReported(t *testing.T) {
	root := t.TempDir()
	writeUnreadSourceFixture(t, root, "tei.workspace.yaml", "workspace: unread_source_test\nformatVersion: 1\nsources:\n  - databases\n")
	writeUnreadSourceFixture(t, root, "databases/core/database.tei", "database core {\n    kind relational\n    ownership managed-here\n}\n")
	// A persistence mapping one directory too high: inventoried, never parsed.
	writeUnreadSourceFixture(t, root, "databases/core/stray.tei", "persist core.thing {\n    database core\n}\n")

	result, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var found bool
	for _, finding := range result.SourceRootFindings {
		if finding.File == "databases/core/stray.tei" {
			found = true
			if finding.Severity != "warning" {
				t.Errorf("severity = %q, want warning", finding.Severity)
			}
			if finding.Message == "" {
				t.Error("message is empty")
			}
		}
	}
	if !found {
		t.Fatalf("the unread source was not reported: %+v", result.SourceRootFindings)
	}
}

// The same file in the location the loader reads produces no warning: the report
// must follow what was actually read, not the file name.
func TestReadTIESourceIsNotReported(t *testing.T) {
	root := t.TempDir()
	writeUnreadSourceFixture(t, root, "tei.workspace.yaml", "workspace: unread_source_test\nformatVersion: 1\nsources:\n  - databases\n")
	writeUnreadSourceFixture(t, root, "databases/core/database.tei", "database core {\n    kind relational\n    ownership managed-here\n}\n")
	writeUnreadSourceFixture(t, root, "databases/core/persistence/thing.tei", "persist core.thing {\n    database core\n    semantic std.text\n    table public.things\n}\n")

	result, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, finding := range result.SourceRootFindings {
		if finding.File == "databases/core/persistence/thing.tei" {
			t.Fatalf("a file the loader read was reported as unread: %+v", finding)
		}
	}
}

// A near-miss name is already explained by the name check, so the unread report
// must not duplicate it.
func TestUnreadSourceDoesNotDuplicateTheNameFinding(t *testing.T) {
	root := t.TempDir()
	writeUnreadSourceFixture(t, root, "tei.workspace.yaml", "workspace: unread_source_test\nformatVersion: 1\nsources:\n  - databases\n")
	writeUnreadSourceFixture(t, root, "databases/core/Database.TEI", "database core {\n    kind relational\n    ownership managed-here\n}\n")

	result, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Count only this report's message. The name check has its own wording,
	// and a near-miss name can match two of its branches.
	for _, finding := range result.SourceRootFindings {
		if finding.File == "databases/core/Database.TEI" && strings.Contains(finding.Message, "no loader reads") {
			t.Fatalf("the unread report duplicated the name finding: %+v", finding)
		}
	}
}

// Non-TEI managed files are outside this report: the finding is about TEI
// sources whose declarations never reach the model.
func TestNonTEIUnreadFilesAreNotReported(t *testing.T) {
	root := t.TempDir()
	writeUnreadSourceFixture(t, root, "tei.workspace.yaml", "workspace: unread_source_test\nformatVersion: 1\nsources:\n  - databases\n")
	writeUnreadSourceFixture(t, root, "databases/core/database.tei", "database core {\n    kind relational\n    ownership managed-here\n}\n")
	writeUnreadSourceFixture(t, root, "databases/core/notes.txt", "not a TEI source\n")

	result, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, finding := range result.SourceRootFindings {
		if finding.File == "databases/core/notes.txt" {
			t.Fatalf("a non-TEI file was reported: %+v", finding)
		}
	}
}
