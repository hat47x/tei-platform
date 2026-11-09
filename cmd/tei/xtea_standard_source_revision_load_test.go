package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// TestXTEAStandardSourceRevisionBoundedLoad compares two historically pinned
// byte versions of one real example Source through the production Workspace
// loader. The fixture is a pinned past revision, intentionally separate from
// the actively evolving examples/xtea-lowcode file. It intentionally does not qualify an entire Function, an external
// Provider, a consumer reply, or an Owner-accepted business meaning (#3252).
func TestXTEAStandardSourceRevisionBoundedLoad(t *testing.T) {
	const (
		sourcePath = "semantic/master/standard.tei"
		oldBlob    = "292188047e7be360e31fec2abbda53cf3d9f0e2f"
		newBlob    = "22588b6e3b68e0a7dc96a6140acb70116aa6bfbf"
	)
	// The checked-in fixture is the *historical B Git blob*, not the mutable
	// examples/xtea-lowcode Source. The latter has continued to gain roles.
	// A dedicated -text attribute prevents checkout EOL conversion.
	newBytes, err := os.ReadFile(filepath.Join("testdata", "xtea_standard_20261108_b.tei"))
	if err != nil {
		t.Fatal(err)
	}
	// Git blob IDs pin historical fixture bytes. They do not confer trust,
	// acceptance, or authority over a live Workspace.
	if got := gitBlobIDForRevisionTest(newBytes); got != newBlob {
		t.Fatalf("historical Source B fixture drifted: Git blob %s, expected %s", got, newBlob)
	}
	bom := []byte{0xef, 0xbb, 0xbf}
	added := []byte("    attribute rule type std.identifier display \"規則\" role numbering-rule\n")
	if !bytes.HasPrefix(newBytes, bom) || bytes.Count(newBytes, added) != 1 ||
		!bytes.HasSuffix(newBytes, []byte("}\n\r\n")) {
		t.Fatal("new Source no longer has the pinned BOM, added numbering-rule and trailing CRLF")
	}
	oldBytes := bytes.TrimPrefix(newBytes, bom)
	oldBytes = bytes.Replace(oldBytes, added, nil, 1)
	oldBytes = bytes.TrimSuffix(oldBytes, []byte("\r\n"))
	if got := gitBlobIDForRevisionTest(oldBytes); got != oldBlob {
		t.Fatalf("reconstructed old Source is not pinned Git blob: %s != %s", got, oldBlob)
	}

	type roleAndType struct {
		role, typeRef string
	}
	type observed struct {
		attributes map[string]roleAndType
		revision   string
	}
	load := func(label string, raw []byte) observed {
		t.Helper()
		root := t.TempDir()
		p := filepath.Join(root, filepath.FromSlash(sourcePath))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := []byte("workspace: source_revision_preflight\nformatVersion: 1\nsources:\n  - semantic\n")
		if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("%s: actual Workspace Load failed: %v", label, err)
		}
		index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatalf("%s: Source Index from Load failed: %v", label, err)
		}
		foundSource := false
		for _, artifact := range index.Artifacts {
			if artifact.Path == sourcePath {
				foundSource = true
				if artifact.Fingerprint != sourceindex.Fingerprint(raw) {
					t.Fatalf("%s: indexed Source bytes did not match the loaded fixture", label)
				}
			}
		}
		if !foundSource {
			t.Fatalf("%s: actual Source Index omitted %s", label, sourcePath)
		}
		attrs := make(map[string]roleAndType)
		for _, ns := range loaded.World.Semantics {
			if string(ns.ID) != "master" {
				continue
			}
			for _, data := range ns.Data {
				for _, attr := range data.Attributes {
					if attr.Role == "" {
						continue
					}
					key := string(attr.ID)
					if _, exists := attrs[key]; exists {
						t.Fatalf("%s: duplicate parsed attribute %s", label, key)
					}
					attrs[key] = roleAndType{string(attr.Role), string(attr.TypeRef)}
				}
			}
		}
		return observed{attributes: attrs, revision: sourceindex.ModelRevision(index)}
	}

	oldResult := load("A", oldBytes)
	newResult := load("B", newBytes)
	if len(oldResult.attributes) != 17 || len(newResult.attributes) != 18 {
		t.Fatalf("role-bearing attributes A=%d B=%d, expected 17/18", len(oldResult.attributes), len(newResult.attributes))
	}
	if oldResult.revision == newResult.revision {
		t.Fatal("two distinct actual Source byte versions produced the same Source Index revision")
	}
	for key, before := range oldResult.attributes {
		after, found := newResult.attributes[key]
		if !found || before != after {
			t.Fatalf("existing parsed attribute changed between A and B: %s before=%+v after=%+v found=%v", key, before, after, found)
		}
	}
	if v, found := oldResult.attributes["master.numbering_rule.rule"]; found {
		t.Fatalf("A unexpectedly contains newly added numbering rule: %+v", v)
	}
	if v, found := newResult.attributes["master.numbering_rule.rule"]; !found ||
		v != (roleAndType{"numbering-rule", "std.identifier"}) {
		t.Fatalf("B numbering-rule role/type missing or changed: %+v found=%v", v, found)
	}
	for label, attrs := range map[string]map[string]roleAndType{
		"A": oldResult.attributes, "B": newResult.attributes,
	} {
		if v, found := attrs["master.tax_rate.rate"]; !found ||
			v != (roleAndType{"tax-value", "std.decimal"}) {
			t.Fatalf("%s: tax-rate role/type not preserved: %+v found=%v", label, v, found)
		}
	}
	// The preserved tax declaration is a bounded local fact. There is no
	// independent Required Function closure, business Rule, row, or Effect here.
}

func gitBlobIDForRevisionTest(raw []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(raw))
	h.Write(raw)
	return hex.EncodeToString(h.Sum(nil))
}
