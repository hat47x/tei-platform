package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

// workspaceIdentityOf decodes the `workspace` block of one CLI response. The
// field is present in model catalog, inspect, trace, validate -json and
// workspace-files (#3181), so the same helper serves every case.
type cliWorkspaceIdentity struct {
	Name           string `json:"name"`
	FormatVersion  int    `json:"formatVersion"`
	ManifestSHA256 string `json:"manifestSha256"`
}

func decodeWorkspaceIdentity(t *testing.T, raw []byte) cliWorkspaceIdentity {
	t.Helper()
	var top struct {
		Result    json.RawMessage `json:"result"`
		Workspace json.RawMessage `json:"workspace"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("not one JSON object: %v (%q)", err, string(raw))
	}
	block := top.Workspace
	if len(top.Result) > 0 {
		// The command envelope carries the identity inside `result`; validate
		// puts it at the top level.
		var inner struct {
			Workspace json.RawMessage `json:"workspace"`
		}
		if err := json.Unmarshal(top.Result, &inner); err != nil {
			t.Fatalf("result is not one JSON object: %v", err)
		}
		block = inner.Workspace
	}
	var identity cliWorkspaceIdentity
	if len(block) == 0 {
		return identity
	}
	if err := json.Unmarshal(block, &identity); err != nil {
		t.Fatalf("workspace identity is not an object: %v (%q)", err, string(block))
	}
	return identity
}

// TestCLIAnswersNameTheWorkspaceTheyRead pins #3181: a caller holding several
// Workspaces must be able to tell two results apart without comparing paths.
// `worldFingerprint` identifies declaration content, not the read boundary, so
// the same declarations in two Workspaces have equal fingerprints and must
// still carry different identities.
func TestCLIAnswersNameTheWorkspaceTheyRead(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")

	run := func(name string, run func(out *bytes.Buffer) (int, error)) []byte {
		t.Helper()
		var out bytes.Buffer
		code, err := run(&out)
		if err != nil || code != 0 {
			t.Fatalf("%s: code=%d err=%v body=%s", name, code, err, out.String())
		}
		return out.Bytes()
	}

	responses := map[string][]byte{
		"model catalog": run("model catalog", func(out *bytes.Buffer) (int, error) {
			return runModelCommand([]string{"catalog", "-workspace", reference}, out)
		}),
		"inspect": run("inspect", func(out *bytes.Buffer) (int, error) {
			return inspectWorkspaceJSON(out, reference, "", "semantic-data", "sales.order", 0)
		}),
		"trace": run("trace", func(out *bytes.Buffer) (int, error) {
			return traceWorkspaceJSON(out, reference, "", "reference-observation", "semantic-relation", "sales.order.customer", "both", "summary", "", 0, 0, 0)
		}),
		"validate": run("validate", func(out *bytes.Buffer) (int, error) {
			return validateWorkspaceJSON(out, reference)
		}),
		"workspace-files": run("workspace-files", func(out *bytes.Buffer) (int, error) {
			return workspaceFilesJSON(out, reference, "", 0)
		}),
	}
	identities := map[string]cliWorkspaceIdentity{}
	for name, raw := range responses {
		identity := decodeWorkspaceIdentity(t, raw)
		if identity.Name != "tei_reference" || identity.FormatVersion != 1 || identity.ManifestSHA256 == "" {
			t.Fatalf("%s: workspace identity=%+v body=%s", name, identity, raw)
		}
		identities[name] = identity
	}
	// One read of one Workspace: every command must report the same identity.
	var first cliWorkspaceIdentity
	for name, identity := range identities {
		if first == (cliWorkspaceIdentity{}) {
			first = identity
			continue
		}
		if identity != first {
			t.Fatalf("commands disagree about the Workspace they read: %s=%+v, first=%+v", name, identity, first)
		}
	}
}

// TestWorkspaceIdentityDistinguishesWorkspacesWithIdenticalDeclarations is the
// point of the field: two Workspaces can hold the same declarations (equal
// worldFingerprint) and still be different Workspaces.
func TestWorkspaceIdentityDistinguishesWorkspacesWithIdenticalDeclarations(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	const manifest = "workspace: copy_a\nformatVersion: 1\nsources:\n  - semantic\n"
	writeWorkspaceFile(t, first, "tei.workspace.yaml", manifest)
	writeWorkspaceFile(t, first, "semantic/marker.tei", "namespace shared\n")
	writeWorkspaceFile(t, second, "tei.workspace.yaml", "workspace: copy_b\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, second, "semantic/marker.tei", "namespace shared\n")

	var a, b bytes.Buffer
	if code, err := validateWorkspaceJSON(&a, first); err != nil || code != 0 {
		t.Fatalf("first: code=%d err=%v body=%s", code, err, a.String())
	}
	if code, err := validateWorkspaceJSON(&b, second); err != nil || code != 0 {
		t.Fatalf("second: code=%d err=%v body=%s", code, err, b.String())
	}
	var ra, rb struct {
		WorldFingerprint string `json:"worldFingerprint"`
	}
	if err := json.Unmarshal(a.Bytes(), &ra); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b.Bytes(), &rb); err != nil {
		t.Fatal(err)
	}
	if ra.WorldFingerprint == "" || ra.WorldFingerprint != rb.WorldFingerprint {
		t.Fatalf("fixtures do not share declarations: %q vs %q", ra.WorldFingerprint, rb.WorldFingerprint)
	}
	ia := decodeWorkspaceIdentity(t, a.Bytes())
	ib := decodeWorkspaceIdentity(t, b.Bytes())
	if ia.Name == ib.Name || ia.ManifestSHA256 == ib.ManifestSHA256 {
		t.Fatalf("identical declarations collapsed the Workspace identity: %+v vs %+v", ia, ib)
	}
}
