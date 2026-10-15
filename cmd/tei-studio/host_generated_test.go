//go:build studio_generated && server

package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
)

func TestGeneratedGoServicesRunThroughRealHost(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	h, err := newStudioHost([]string{root})
	if err != nil { t.Fatal(err) }
	if refs := h.ListWorkspaces(); !refs.OK || len(refs.Value) != 1 || refs.Value[0].ID != "workspace-1" {
		t.Fatalf("allowlist lost: %+v", refs)
	}
	catalog := h.CatalogWorkspace("workspace-1", "")
	wantCatalog, err := operations.CatalogWorkspace(root, "")
	if err != nil { t.Fatal(err) }
	if !catalog.OK || !reflect.DeepEqual(catalog.Value, &wantCatalog) {
		t.Fatalf("generated catalog does not match: %+v", catalog)
	}
	validation := h.ValidateWorkspace("workspace-1")
	wantValidation, err := operations.ValidateWorkspace(root)
	if err != nil { t.Fatal(err) }
	if !validation.OK || !reflect.DeepEqual(validation.Value, &wantValidation) {
		t.Fatalf("generated validation differs, including diagnostics: %+v", validation)
	}
	if len(validation.Value.Diagnostics) == 0 {
		t.Fatal("reference diagnostics were silently discarded")
	}
	for _, id := range []string{"", "../reference/workspace", "workspace-2", "/tmp/arbitrary", " workspace-1"} {
		if res := h.CatalogWorkspace(id, ""); res.OK || res.Failure == nil || res.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("generated catalog admits %q: %+v", id, res)
		}
		if res := h.ValidateWorkspace(id); res.OK || res.Failure == nil || res.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("generated validation admits %q: %+v", id, res)
		}
	}
	if len(wantCatalog.Entries) > 0 {
		subject := wantCatalog.Entries[0].Subject
		actual := h.InspectSubject("workspace-1", subject.Kind, subject.ID)
		if !actual.OK || actual.Value == nil { t.Fatalf("unchanged inspect failed: %+v", actual) }
	}
}
