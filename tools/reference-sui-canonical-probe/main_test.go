package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func workspacePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repositoryRoot(t), "reference", "sui-adoption", "workspace-m2")
}

func TestM2WorkspaceResolvesTheDefinition(t *testing.T) {
	got, err := run(workspacePath(t))
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got.TEIAuthority != "definition-only" {
		t.Fatalf("TEIAuthority = %q, want definition-only", got.TEIAuthority)
	}
	if got.Canonicalized {
		t.Fatal("instance data was canonicalized; the M2 probe must stay read-only")
	}
	if got.InstanceCount != 0 {
		t.Fatalf("InstanceCount = %d, want 0 (no instance is promoted)", got.InstanceCount)
	}

	// Every required definition must resolve exactly once. A duplicate is an
	// error in the probe, so a non-nil Data slice is enough here.
	wanted := []string{
		"sui.card",
		"sui.containment",
		"sui.affiliation",
		"sui.document",
		"sui.edge",
		"sui.island",
		"sui.island_parent",
	}
	seen := map[string]bool{}
	for _, d := range got.Data {
		seen[d.ID] = true
	}
	for _, id := range wanted {
		if !seen[id] {
			t.Errorf("definition %s did not resolve", id)
		}
	}
}

func TestM2KeepsMembershipKindsSeparate(t *testing.T) {
	got, err := run(workspacePath(t))
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !got.ContainmentAndAffiliationSeparate {
		t.Fatal("containment and affiliation resolved into one membership")
	}

	// The two memberships must keep their own occurrence identity attributes.
	for _, d := range got.Data {
		if d.ID == "sui.containment" || d.ID == "sui.affiliation" {
			if len(d.Attributes) != 1 {
				t.Errorf("%s has %d attributes, want 1 occurrence id", d.ID, len(d.Attributes))
			}
		}
	}
}

func TestM2KeepsParallelEdgesDistinguishable(t *testing.T) {
	got, err := run(workspacePath(t))
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !got.ParallelEdgesDistinguishable {
		t.Fatal("a directed edge cannot be told apart from its parallel edge")
	}
}

func TestM2DeclaresWhatItDoesNotCarry(t *testing.T) {
	got, err := run(workspacePath(t))
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	// The losses are part of the report: an empty list would claim the
	// definition carries everything, which it does not.
	if len(got.Undeclared) == 0 {
		t.Fatal("Undeclared is empty; the definition must name its losses")
	}
	var treeCanvasReview bool
	for _, u := range got.Undeclared {
		switch {
		case strings.HasPrefix(u, "tree."), strings.HasPrefix(u, "canvas."), strings.HasPrefix(u, "review."):
			treeCanvasReview = true
		}
	}
	if !treeCanvasReview {
		t.Errorf("Undeclared does not name tree/canvas/review losses: %v", got.Undeclared)
	}
	if got.ParentPresenceSemantics != "unresolved" {
		t.Errorf("ParentPresenceSemantics = %q, want unresolved (0..1 is not expressible)", got.ParentPresenceSemantics)
	}
}

func TestM2RejectsAMissingDefinition(t *testing.T) {
	root := repositoryRoot(t)
	dir := t.TempDir()
	// A workspace without the sui namespace resolves to no definition, so the
	// probe must fail closed instead of reporting an empty success.
	writeWorkspaceFiles(t, dir, "")
	if _, err := run(dir); err == nil {
		t.Fatalf("run() with no sui definitions returned no error (root=%s)", root)
	}
}

// writeWorkspaceFiles writes a minimal workspace whose semantic source holds
// the given definition body. An empty body means the workspace has no sui
// definitions at all.
func writeWorkspaceFiles(t *testing.T, dir, body string) {
	t.Helper()
	manifest := "workspace: sui_m2_probe_fixture\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(dir, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	semanticDir := filepath.Join(dir, "semantic", "sui")
	if err := os.MkdirAll(semanticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(semanticDir, "sui.tei"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
