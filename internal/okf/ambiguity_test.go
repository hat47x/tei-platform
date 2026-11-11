package okf

import "testing"

// TestStructuralBundleReportsAmbiguity exercises the one case where the
// specification leaves the base open: a relative value that exists under both
// interpretations. The reader must label it rather than pick silently.
//
// The containing concept lives in a subdirectory so the two interpretations name
// different files: document-relative "concepts/shared.md" and bundle-root
// "shared.md".
func TestStructuralBundleReportsAmbiguity(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"index.md":           "# Concepts\n\n* [Top](concepts/top.md) - the top concept\n",
		"concepts/top.md":    "---\ntype: Metric\ntitle: Top\nsources:\n  - { id: shared, resource: shared.md }\n---\n\nbody\n\n[^shared]: shared\n",
		"concepts/shared.md": "---\ntype: Reference\ntitle: Document-relative shared\n---\n\ndocument copy\n",
		"shared.md":          "---\ntype: Reference\ntitle: Root shared\n---\n\nroot copy\n",
	})
	bundle, err := Read(dir, Options{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	top := findConcept(bundle, "concepts/top")
	if top == nil {
		t.Fatalf("missing concepts/top; got %v", conceptIDs(bundle))
	}
	if len(top.Sources) != 1 {
		t.Fatalf("sources = %+v", top.Sources)
	}
	if top.Sources[0].Resolution != ResolutionAmbiguous {
		t.Fatalf("source resolution = %q, want ambiguous for a target present under both bases", top.Sources[0].Resolution)
	}
	if !hasCode(top.Diagnostics, CodePathAmbiguousBase) {
		t.Fatalf("codes = %v, want %s", codes(top.Diagnostics), CodePathAmbiguousBase)
	}
	// Ambiguity is an observation: the format is still conformant and both
	// candidates are listed so a consumer can choose with its own policy.
	if !bundle.Format.Conformant {
		t.Fatalf("ambiguity must not break conformance; errors: %+v", bundle.Format.FormatErrors)
	}
	if len(top.Sources[0].Resolved) != 2 {
		t.Fatalf("resolved = %v, want both candidates", top.Sources[0].Resolved)
	}
	// No consumer policy is applied here: the reader does not treat the
	// document-relative candidate as authoritative merely because it is nearer.
	if top.Sources[0].Resolved[0] != "concepts/shared.md" || top.Sources[0].Resolved[1] != "shared.md" {
		t.Fatalf("resolved order = %v, want [concepts/shared.md shared.md]", top.Sources[0].Resolved)
	}
}
