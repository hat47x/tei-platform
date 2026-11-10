//go:build server

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// This is the real Wails Bridge's Go-facing method, not a renderer test.
// Two startup-allowlisted roots declare the same Subject ID. The selected
// Inspector must not silently reuse the first Workspace for the second.
func TestStudioSelectedInspectorDoesNotCrossAllowlistedWorkspaces(t *testing.T) {
	create := func(name string, addCustomer bool) string {
		t.Helper()
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "semantic"), 0755); err != nil {
			t.Fatal(err)
		}
		manifest := "workspace: " + name + "\nformatVersion: 1\nsources:\n  - semantic\n"
		if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
		model := "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n"
		if addCustomer {
			model += "    attribute customerId type std.identifier\n"
		}
		model += "}\n"
		if err := os.WriteFile(filepath.Join(root, "semantic", "order.tei"), []byte(model), 0644); err != nil {
			t.Fatal(err)
		}
		return root
	}
	roots := []string{create("alpha", false), create("beta", true)}
	host, err := newStudioHost(roots)
	if err != nil {
		t.Fatal(err)
	}
	bridge := &StudioBridge{host: host}
	a := bridge.InspectSubject("workspace-1", "semantic-data", "sales.order")
	b := bridge.InspectSubject("workspace-2", "semantic-data", "sales.order")
	if !a.OK || !b.OK || a.Value == nil || b.Value == nil ||
		a.Value.Workspace == nil || b.Value.Workspace == nil {
		t.Fatalf("allowlisted Workspaces were not inspectable: A=%+v B=%+v", a, b)
	}
	if a.Value.Workspace.Name != "alpha" || b.Value.Workspace.Name != "beta" ||
		a.Value.Subject != b.Value.Subject ||
		a.Value.WorldFingerprint == b.Value.WorldFingerprint ||
		bytes.Equal(a.Value.Canonical, b.Value.Canonical) {
		t.Fatalf("Inspector crossed Workspace identity: A=%+v B=%+v", a, b)
	}
	// The unchanged second root must not acquire the first root's Source
	// revision or normalization when the first root is modified.
	beforeB := b.Value.SourceIndexFingerprint
	if err := os.Rename(filepath.Join(roots[0], "semantic", "order.tei"),
		filepath.Join(roots[0], "semantic", "renamed.tei")); err != nil {
		t.Fatal(err)
	}
	a2 := bridge.InspectSubject("workspace-1", "semantic-data", "sales.order")
	b2 := bridge.InspectSubject("workspace-2", "semantic-data", "sales.order")
	if !a2.OK || a2.Value == nil || !b2.OK || b2.Value == nil ||
		a2.Value.WorldFingerprint != a.Value.WorldFingerprint ||
		a2.Value.SourceIndexFingerprint == a.Value.SourceIndexFingerprint ||
		b2.Value.SourceIndexFingerprint != beforeB {
		t.Fatalf("Source move leaked between Studio Workspaces: A=%+v A2=%+v B=%+v B2=%+v", a, a2, b, b2)
	}
}
