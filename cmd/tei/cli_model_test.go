package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIModelCatalogReadOnly(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", root, "-namespace", "sales"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model catalog failed: exit=%d err=%v response=%s", code, err, stdout.String())
	}
	var response modelCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Error != nil || response.Result == nil ||
		response.SchemaVersion != "tei.cli.model/v0" ||
		response.Result.Effect != "read-only" || response.Result.NamespaceFilter != "sales" {
		t.Fatalf("unexpected model catalog output: %+v", response)
	}
	found := false
	for _, entry := range response.Result.Entries {
		if entry.Subject.Kind == "application" {
			t.Fatalf("application included in filtered semantic namespace: %+v", entry)
		}
		if entry.Subject.Kind == "semantic-data" && entry.Subject.ID == "sales.order" {
			found = true
		}
	}
	if !found {
		t.Fatal("sales.order is missing")
	}
}

func TestCLIModelCatalogErrorContract(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"unknown"},
		{"catalog"},
		{"catalog", "-workspace", " "},
		{"catalog", "-workspace", "x", "unexpected"},
		{"catalog", "-workspace", "x", "-world", "forbidden"},
		{"catalog", "-workspace", "x", "-namespace", " sales "},
	} {
		var stdout bytes.Buffer
		code, err := runModelCommand(args, &stdout)
		if err != nil || code != 2 {
			t.Fatalf("invalid args %q: code=%d err=%v", args, code, err)
		}
		var response modelCLIResponse
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("non-JSON error for %q: %v %s", args, err, stdout.String())
		}
		if response.OK || response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" {
			t.Fatalf("unexpected invalid argument response: %+v", response)
		}
		if strings.Count(strings.TrimSpace(stdout.String()), "\n") > 0 {
			t.Fatalf("expected one JSON document, got %s", stdout.String())
		}
	}
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", filepath.Join(t.TempDir(), "missing")}, &stdout)
	if err != nil || code != 2 {
		t.Fatalf("missing workspace code=%d err=%v", code, err)
	}
	var response modelCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.OK || response.Error == nil || response.Error.Code != "WORKSPACE_LOAD_FAILED" {
		t.Fatalf("workspace load not fail-closed: %+v", response)
	}
}

func TestCLIModelCatalogUnknownNamespaceIsNotAnEmptySuccess(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for name, args := range map[string][]string{
		"misspelled namespace": {"catalog", "-workspace", root, "-namespace", "no.such.ns"},
		"application id":       {"catalog", "-workspace", root, "-namespace", "order_entry"},
		// A query is a real workspace subject but not a semantic namespace, so
		// -namespace selects nothing for it. It must fail the same way and name
		// the kind, not read as "this namespace declares no entries" (#2992).
		"query id": {"catalog", "-workspace", root, "-namespace", "order_entry.recent_orders"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			code, err := runModelCommand(args, &stdout)
			if err != nil || code != 1 {
				t.Fatalf("code=%d err=%v body=%s", code, err, stdout.String())
			}
			var response modelCLIResponse
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.OK || response.Error == nil || response.Error.Code != "NAMESPACE_NOT_FOUND" {
				t.Fatalf("unknown namespace returned a success: %+v", response)
			}
			if !strings.Contains(response.Error.Message, "not a semantic namespace") {
				t.Fatalf("message does not separate a namespace filter from the subject it named: %q", response.Error.Message)
			}
			switch name {
			case "application id", "query id":
				kind := strings.TrimSuffix(name, " id")
				if !strings.HasPrefix(response.Error.Message, kind+" ") {
					t.Fatalf("%s message should name the %s kind: %q", name, kind, response.Error.Message)
				}
			}
		})
	}
}

func TestCLIModelCatalogEmptyNamespaceIsADeclaredSuccess(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: empty_ns\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/empty/empty.tei", "namespace empty\n")
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", root, "-namespace", "empty"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("declared namespace code=%d err=%v body=%s", code, err, stdout.String())
	}
	var response modelCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Result == nil || !response.Result.NamespaceMatched ||
		response.Result.NamespaceMatchKind != "namespace" {
		t.Fatalf("declared empty namespace was not reported as a match: %+v", response.Result)
	}
	// The namespace subject itself is indexed; what matters is that the
	// request matched a declared namespace instead of being reported unknown.
	if len(response.Result.Entries) != 1 || response.Result.Entries[0].Subject.Kind != "semantic" {
		t.Fatalf("empty namespace catalog=%+v", response.Result.Entries)
	}
}

// TestCLIModelCatalogEmptyWorkspaceRevisionIsTheEmptyDigest pins what
// modelRevision means when the Workspace manages no Source (#2822). The value
// is the revision of the managed Source set, not of the loaded World: the
// standard prelude still contributes entries, while the revision stays at the
// empty-input digest. Without this, an empty Workspace looks either like a
// model identity or like a failure, and neither is what the field reports.
func TestCLIModelCatalogEmptyWorkspaceRevisionIsTheEmptyDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: empty_source\nformatVersion: 1\nsources:\n  - semantic\n")
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"catalog", "-workspace", root}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("empty workspace code=%d err=%v body=%s", code, err, stdout.String())
	}
	var response modelCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Result == nil {
		t.Fatalf("empty workspace catalog is not a success: %+v", response)
	}
	const emptyInput = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if response.Result.ModelRevision != emptyInput {
		t.Fatalf("empty workspace revision=%q", response.Result.ModelRevision)
	}
	// The entries are not empty: the prelude is loaded. That is why the
	// revision cannot be read as "the World declares nothing".
	prelude := 0
	for _, entry := range response.Result.Entries {
		if entry.Subject.ID == "std" || strings.HasPrefix(entry.Subject.ID, "std.") {
			prelude++
		}
		if entry.Subject.ID != "std" && !strings.HasPrefix(entry.Subject.ID, "std.") {
			t.Fatalf("an empty workspace declared a non-prelude subject: %+v", entry.Subject)
		}
	}
	if prelude == 0 {
		t.Fatal("expected the standard prelude in the catalog entries")
	}
	// The empty Workspace and the reference Workspace share neither the
	// revision nor the declaration counts, so the two cannot be confused.
	reference := filepath.Join("..", "..", "reference", "workspace")
	var refOut bytes.Buffer
	if code, err := runModelCommand([]string{"catalog", "-workspace", reference}, &refOut); err != nil || code != 0 {
		t.Fatalf("reference catalog code=%d err=%v", code, err)
	}
	var ref modelCLIResponse
	if err := json.Unmarshal(refOut.Bytes(), &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Result == nil || ref.Result.ModelRevision == emptyInput ||
		ref.Result.DeclarationCounts.Total == 0 {
		t.Fatalf("reference workspace collapsed into the empty revision: %+v", ref.Result)
	}
}
