package studiohost_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/studiohost"
	"tei/internal/validation"
)

func writeStudioFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestValidateWorkspacePairsProposalsWithTheReadRevision checks that the Studio
// host returns version-bound proposals with the validation, and that none of
// them is accepted: approval is a separate decision.
func TestValidateWorkspacePairsProposalsWithTheReadRevision(t *testing.T) {
	root := t.TempDir()
	writeStudioFile(t, root, "tei.workspace.yaml", "workspace: studio_proposal\nformatVersion: 1\nsources:\n  - applications\n")
	writeStudioFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    database sales {
        target sales_core
        access invalid
    }
}
`)
	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	response := host.ValidateWorkspace("workspace-1")
	if !response.OK || response.Value == nil {
		t.Fatalf("validation failed: %+v", response.Failure)
	}
	expected := len(response.Value.Diagnostics)
	if response.Value.Direct != nil {
		expected++
	}
	if len(response.Proposals) != expected {
		t.Fatalf("diagnostics=%d proposals=%d", expected, len(response.Proposals))
	}
	for _, proposal := range response.Proposals {
		if proposal.Accepted {
			t.Fatalf("proposal was accepted without a decision: %+v", proposal)
		}
		if proposal.Revision != response.Value.WorldFingerprint {
			t.Fatalf("proposal revision=%q want=%q", proposal.Revision, response.Value.WorldFingerprint)
		}
	}
	if len(response.Proposals) > 0 {
		if err := validation.Accept(&response.Proposals[0], "rev-changed"); err == nil {
			t.Fatal("a changed revision accepted the proposal")
		}
	}
	if strings.Contains(response.Value.WorldFingerprint, " ") {
		t.Fatalf("fingerprint=%q", response.Value.WorldFingerprint)
	}
}
